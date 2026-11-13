/*
 * Copyright (c) 2026 Oracle and/or its affiliates.
 * Licensed under the Universal Permissive License v 1.0 as shown at
 * https://oss.oracle.com/licenses/upl.
 */

package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"time"

	"github.com/oracle/coherence-cli/pkg/config"
	"github.com/spf13/cobra"
)

type managedProcess struct {
	command *exec.Cmd
	done    chan struct{}
	logFile string
	// Written by the sole Wait goroutine; read only after done closes.
	waitErr error
}

// openManagedProcessLog confines the truncating open itself, including when
// backup is disabled or a link changes after an earlier path/backup check.
func openManagedProcessLog(clusterName, processName string) (*os.File, error) {
	if err := validateClusterLogName(processName); err != nil {
		return nil, err
	}
	root, err := openClusterLogs(clusterName)
	if err != nil {
		return nil, err
	}
	file, openErr := root.OpenFile(processName+".log", os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
	closeErr := root.Close()
	if openErr != nil {
		return nil, errors.Join(openErr, closeErr)
	}
	if closeErr != nil {
		return nil, errors.Join(closeErr, file.Close())
	}
	return file, nil
}

func launchManagedProcess(clusterName, processName string, args []string) (*managedProcess, error) {
	f, err := openManagedProcessLog(clusterName, processName)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	command := exec.Command(javaExec, args...)
	command.Stdout, command.Stderr = f, f
	setForkProcess(command)
	if err = command.Start(); err != nil {
		return nil, err
	}
	p := &managedProcess{command: command, done: make(chan struct{}), logFile: filepath.Join(logsDirectory, clusterName, processName+".log")}
	go func() { p.waitErr = command.Wait(); close(p.done) }()
	return p, nil
}

// Cleanup is independent of operation cancellation and uses the original owned
// handles. Never reuse the stop-phase seam or re-find/re-signal numeric PIDs.
var killManagedProcess = (*os.Process).Kill
var managedCleanupTimeout = 5 * time.Second

func cleanupManagedProcesses(processes []*managedProcess) error {
	deadline := time.NewTimer(managedCleanupTimeout)
	defer deadline.Stop()
	var failures []error
	killErrors := make([]error, len(processes))
	// A failed kill or delayed completion must not prevent later attempts.
	for i, p := range processes {
		select {
		case <-p.done:
			continue
		default:
		}
		process := p.command.Process
		if err := killManagedProcess(process); err != nil && !processAlreadyExited(process.Pid, process, err) {
			killErrors[i] = err
		}
	}
	// All requests precede every wait. A single timer covers the collection;
	// the existing Wait goroutines keep reaping after a bounded return.
wait:
	for _, p := range processes {
		select {
		case <-p.done:
			continue
		default:
		}
		select {
		case <-p.done:
		case <-deadline.C:
			break wait
		}
	}
	completed := 0
	for i, p := range processes {
		completedWait := false
		select {
		case <-p.done:
			// Nonzero/signal exits still establish termination. In particular,
			// ProcessState.Exited alone would reject expected signal kills.
			var exitErr *exec.ExitError
			if p.waitErr == nil || errors.As(p.waitErr, &exitErr) && exitErr.ProcessState != nil {
				completed++
				completedWait = true
			} else {
				failures = append(failures, fmt.Errorf("wait for owned child PID %d (log %q); exit unresolved: %w", p.command.Process.Pid, p.logFile, p.waitErr))
			}
		default:
			failures = append(failures, fmt.Errorf("wait for owned child PID %d (log %q); completion unresolved after shared cleanup budget %s: %w", p.command.Process.Pid, p.logFile, managedCleanupTimeout, context.DeadlineExceeded))
		}
		// Windows Wait can release the owned handle between the request and
		// Kill. Only a trustworthy result from this child's sole Wait can
		// resolve that specific released-handle error. Other kill failures
		// survive completion; no numeric PID is reopened or signalled.
		if err := killErrors[i]; err != nil && (!completedWait || !processKillAfterWait(err)) {
			failures = append(failures, fmt.Errorf("kill owned child PID %d (log %q): %w", p.command.Process.Pid, p.logFile, err))
		}
	}
	if len(failures) > 0 {
		return fmt.Errorf("owned-child cleanup reported errors (%d of %d children completed): %w", completed, len(processes), errors.Join(failures...))
	}
	return nil
}

var awaitManagedStartup = waitManagedReady

var managedReadinessTimeout = 90 * time.Second

type managedSaveMode uint8

const (
	managedNoSave managedSaveMode = iota
	managedCreate
	managedUpdate
)

// This transaction is the sole cleanup owner, including partial launches.
// Entering a successful synchronous save accepts the children even if a signal
// arrives during that save. Without a save, the final cancellation check is the
// acceptance boundary. Restoring signals/context cannot terminate accepted JVMs.
func runPreparedStartup(cmd *cobra.Command, plan managedStartPlan, save managedSaveMode) (processes []*managedProcess, err error) {
	accepted := false
	defer func() {
		if !accepted {
			if cleanupErr := cleanupManagedProcesses(processes); cleanupErr != nil {
				err = errors.Join(err, fmt.Errorf("startup transaction cleanup: %w", cleanupErr))
			}
		}
	}()
	processes, err = launchPreparedMembers(cmd, plan)
	if err != nil {
		return processes, err
	}
	if err = cmd.Context().Err(); err == nil {
		err = awaitManagedStartup(cmd, plan.connection, processes)
	}
	if err != nil {
		return processes, err
	}
	if err = cmd.Context().Err(); err != nil {
		return processes, err
	}
	if save != managedNoSave {
		if err = persistManagedConnection(plan.connection, save == managedCreate); err != nil {
			return processes, err
		}
	}
	accepted = true
	return processes, nil
}

// Return every successfully started child, even on failure. The transaction
// owns cleanup; each child's original Wait goroutine owns reaping.
func launchPreparedMembers(cmd *cobra.Command, plan managedStartPlan) (processes []*managedProcess, err error) {
	for _, member := range plan.members {
		if err = cmd.Context().Err(); err != nil {
			return processes, err
		}
		logFile, logErr := getLogFile(plan.connection.Name, member.name)
		if logErr != nil {
			return processes, logErr
		}
		if plan.backupLogs {
			backedUp, backupErr := backupClusterLogFile(plan.connection.Name, member.name)
			if backupErr != nil {
				return processes, backupErr
			}
			if backedUp {
				cmd.Printf("creating backup log file %s\n", logFile+".old")
			}
		}
		cmd.Printf("Starting cluster member %s...\n", member.name)
		if err = cmd.Context().Err(); err != nil {
			return processes, err
		}
		args := append(append([]string{}, plan.baseArguments...), "-cp", plan.connection.BaseClasspath)
		args = append(args, member.arguments...)
		process, launchErr := launchManagedProcess(plan.connection.Name, member.name, args)
		if launchErr != nil {
			return processes, fmt.Errorf("unable to start member %s: %w", member.name, launchErr)
		}
		processes = append(processes, process)
		if plan.delay > 0 {
			select {
			case <-cmd.Context().Done():
				return processes, cmd.Context().Err()
			case <-time.After(plan.delay):
			}
		}
	}
	return processes, nil
}

// Readiness proves the expected cluster and every child PID are visible through
// the proposed REST endpoint. exec.Start alone is not a successful startup.
func waitManagedReady(cmd *cobra.Command, connection ClusterConnection, processes []*managedProcess) error {
	transport := &http.Transport{Proxy: nil}
	defer transport.CloseIdleConnections()
	client := &http.Client{Timeout: time.Second, Transport: transport, CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }}
	readJSON := func(url string, value interface{}) error {
		request, err := http.NewRequestWithContext(cmd.Context(), http.MethodGet, url, nil)
		if err != nil {
			return err
		}
		response, err := client.Do(request)
		if err != nil {
			return err
		}
		defer response.Body.Close()
		if response.StatusCode != http.StatusOK {
			return fmt.Errorf("HTTP status %d", response.StatusCode)
		}
		return json.NewDecoder(response.Body).Decode(value)
	}
	deadline := time.Now().Add(managedReadinessTimeout)
	var lastErr error
	for time.Now().Before(deadline) {
		if err := cmd.Context().Err(); err != nil {
			return err
		}
		for _, p := range processes {
			select {
			case <-p.done:
				return fmt.Errorf("member PID %d exited during startup; inspect its cluster log", p.command.Process.Pid)
			default:
			}
		}
		var cluster config.Cluster
		var members config.Members
		lastErr = readJSON(connection.ConnectionURL, &cluster)
		if lastErr == nil && cluster.ClusterName != connection.ClusterName {
			return fmt.Errorf("management endpoint belongs to cluster %s, expected %s", cluster.ClusterName, connection.ClusterName)
		}
		if lastErr == nil {
			lastErr = readJSON(connection.ConnectionURL+"/members", &members)
		}
		if lastErr == nil {
			visible := map[string]bool{}
			for _, member := range members.Members {
				visible[member.ProcessName] = true
			}
			ready := true
			for _, p := range processes {
				if !visible[strconv.Itoa(p.command.Process.Pid)] {
					ready = false
				}
			}
			if ready {
				return nil
			}
			lastErr = fmt.Errorf("waiting for launched member PIDs to join")
		}
		select {
		case <-cmd.Context().Done():
			return cmd.Context().Err()
		case <-time.After(250 * time.Millisecond):
		}
	}
	return fmt.Errorf("cluster startup/readiness failed at %s: %v; inspect cluster logs and check for occupied ports", connection.ConnectionURL, lastErr)
}

func showManagedSettings(cmd *cobra.Command, connection ClusterConnection, m ClusterManagement) {
	cmd.Printf("Management REST: %s (bind %s)\n", managementURL(m, connection.ManagementPort), m.RESTAddress)
	if m.JMXPort == 0 {
		cmd.Println("JMX remote connector: disabled")
	} else {
		cmd.Printf("JMX registry/RMI: %s:%d (advertised %s)\n", m.JMXAddress, m.JMXPort, m.JMXHost)
	}
	cmd.Println("These endpoint settings will be saved after successful startup/readiness and configuration write.")
	if connection.Management == nil && connection.BaseClasspath != "" && m.JMXPort == 0 {
		cmd.Println("Legacy configuration: JMX was not saved by older CLI versions; use --jmx-port to re-enable it. The selected REST bind is shown above.")
	}
}
