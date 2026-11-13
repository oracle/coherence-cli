/*
 * Copyright (c) 2026 Oracle and/or its affiliates.
 * Licensed under the Universal Permissive License v 1.0 as shown at
 * https://oss.oracle.com/licenses/upl.
 */

package cmd

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	stopLookupFailure = "find"

	stopLookupAndKillFailure = "both"
)

// A portable task-owned sleeper, including for future native Windows runs.
func TestManagementStopChild(_ *testing.T) {
	if os.Getenv("COHCTL_STOP_TEST_CHILD") == "1" {
		time.Sleep(30 * time.Second)
	}
}

func stopTestChild(t *testing.T) *managedProcess {
	t.Helper()
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	command := exec.Command(executable, "-test.run=^TestManagementStopChild$")
	command.Env = append(os.Environ(), "COHCTL_STOP_TEST_CHILD=1")
	if err = command.Start(); err != nil {
		t.Fatal(err)
	}
	child := &managedProcess{command: command, done: make(chan struct{})}
	go func() { _ = command.Wait(); close(child.done) }()
	t.Cleanup(func() {
		select {
		case <-child.done:
			return
		default:
		}
		_ = child.command.Process.Kill()
		waitStopTestChild(t, child)
	})
	return child
}

func waitStopTestChild(t *testing.T, child *managedProcess) {
	t.Helper()
	select {
	case <-child.done:
	case <-time.After(5 * time.Second):
		t.Fatalf("task-owned PID %d did not exit", child.command.Process.Pid)
	}
}

func stopTestSeams(t *testing.T) {
	t.Helper()
	oldFind, oldKill := findProcessForStop, killProcessForStop
	t.Cleanup(func() { findProcessForStop, killProcessForStop = oldFind, oldKill })
}

func stopTestConnection(t *testing.T, pids []int, stopped *atomic.Bool) ClusterConnection {
	t.Helper()
	connection, _ := stopTestConnectionEndpoint(t, pids, stopped)
	return connection
}

func stopTestConnectionEndpoint(t *testing.T, pids []int, stopped *atomic.Bool) (ClusterConnection, *httptest.Server) {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		running := pids
		if stopped != nil && stopped.Load() {
			running = nil
		}
		if strings.HasSuffix(strings.TrimSuffix(r.URL.Path, "/"), "/members") {
			items := make([]string, 0, len(running))
			for _, pid := range running {
				items = append(items, fmt.Sprintf(`{"processName":"%d"}`, pid))
			}
			fmt.Fprintf(w, `{"items":[%s]}`, strings.Join(items, ","))
		} else {
			fmt.Fprintf(w, `{"clusterName":%q,"clusterSize":%d}`, managementFixtureName, len(running))
		}
	}))
	t.Cleanup(server.Close)
	connection := ClusterConnection{Name: managementFixtureName, ClusterName: managementFixtureName, ManuallyCreated: true,
		ConnectionType: httpType, ConnectionURL: server.URL, ManagementPort: managementListenerPort(t, server.Listener),
		ClusterVersion: defaultCoherenceVersion, Management: &ClusterManagement{RESTAddress: localHost}}
	Config.Clusters = []ClusterConnection{connection}
	viper.Set(clusterKey, Config.Clusters)
	viper.SetConfigFile(filepath.Join(t.TempDir(), "cohctl.yaml"))
	if err := writeManagedConfigAtomically(); err != nil {
		t.Fatal(err)
	}
	return connection, server
}

func TestManagementStopExitedMembers(t *testing.T) {
	for _, staleIndex := range []int{0, 1} {
		t.Run(fmt.Sprintf("stale-%d", staleIndex), func(t *testing.T) {
			managedTestState(t)
			children := []*managedProcess{stopTestChild(t), stopTestChild(t), stopTestChild(t)}
			if err := children[staleIndex].command.Process.Kill(); err != nil {
				t.Fatal(err)
			}
			waitStopTestChild(t, children[staleIndex])
			pids := make([]int, len(children))
			for i, child := range children {
				pids[i] = child.command.Process.Pid
			}
			connection := stopTestConnection(t, pids, nil)
			before, err := os.ReadFile(viper.ConfigFileUsed())
			if err != nil {
				t.Fatal(err)
			}
			// A confirmed stop must bypass every startup-only validation path.
			t.Setenv("PATH", t.TempDir())
			profileValueParam, startupDelayParam = managementTestMissingProfile, "invalid-delay"
			for _, name := range []string{managementTestJavaToolOptions, managementTestLegacyJavaOptions, managementTestJDKJavaOptions} {
				t.Setenv(name, "@invalid-options")
			}
			automaticallyConfirm = true
			cmd := managementTestCommand(t)
			if err := runClusterOperation(cmd, connection.Name, stopClusterCommand); err != nil {
				t.Fatal(err)
			}
			for _, child := range children {
				waitStopTestChild(t, child)
			}
			output := cmd.OutOrStdout().(*bytes.Buffer).String()
			if !strings.Contains(output, "3 of 3 processes stopped") || !strings.Contains(output, "2 killed, 1 already exited, 0 failed") || strings.Contains(output, "replacement") {
				t.Fatalf("incorrect stop summary: %s", output)
			}
			after, err := os.ReadFile(viper.ConfigFileUsed())
			if err != nil || !bytes.Equal(before, after) || !reflect.DeepEqual(Config.Clusters[0], connection) {
				t.Fatalf("stop changed configuration: %v", err)
			}
		})
	}
}

func TestManagementStopFailuresAttemptAllMembers(t *testing.T) {
	for _, operation := range []string{stopClusterCommand, restartClusterCommand} {
		for _, failureAt := range []string{stopLookupFailure, managementTestKill, stopLookupAndKillFailure} {
			t.Run(operation+"/"+failureAt, func(t *testing.T) {
				managedTestState(t)
				stopTestSeams(t)
				children := []*managedProcess{stopTestChild(t), stopTestChild(t), stopTestChild(t)}
				pids := []int{children[0].command.Process.Pid, children[1].command.Process.Pid, children[2].command.Process.Pid}
				connection := stopTestConnection(t, pids, nil)
				connection.Management = &ClusterManagement{RESTAddress: localHost, JMXPort: 9999, JMXAddress: localHost, JMXHost: localHost}
				Config.Clusters[0] = connection
				viper.Set(clusterKey, Config.Clusters)
				if err := writeManagedConfigAtomically(); err != nil {
					t.Fatal(err)
				}
				before, err := os.ReadFile(viper.ConfigFileUsed())
				if err != nil {
					t.Fatal(err)
				}
				if operation == restartClusterCommand {
					fakeJava(t, javaFixture{Version: javaVersion17, Query: true, Marker: "COHCTL_STOP_TEST_LAUNCH"})
				} else {
					t.Setenv("PATH", t.TempDir())
				}
				launchMarker := filepath.Join(t.TempDir(), "launched")
				t.Setenv("COHCTL_STOP_TEST_LAUNCH", launchMarker)
				var attempted []int
				findProcessForStop = func(pid int) (*os.Process, error) {
					attempted = append(attempted, pid)
					if pid == pids[1] && failureAt != managementTestKill {
						return nil, fmt.Errorf("injected lookup: %w", os.ErrPermission)
					}
					return os.FindProcess(pid)
				}
				killProcessForStop = func(p *os.Process) error {
					if p.Pid == pids[1] || failureAt == stopLookupAndKillFailure && p.Pid == pids[0] {
						return fmt.Errorf("injected kill: %w", os.ErrPermission)
					}
					return p.Kill()
				}
				writes, starts := 0, 0
				writeManagedConfig = func() error { writes++; return nil }
				awaitManagedStartup = func(_ *cobra.Command, _ ClusterConnection, processes []*managedProcess) error {
					starts++
					return cleanupManagedProcesses(processes)
				}
				automaticallyConfirm = true
				cmd := managementTestCommand(t, "--jmx-port=0")
				err = runClusterOperation(cmd, connection.Name, operation)
				if !errors.Is(err, os.ErrPermission) || !reflect.DeepEqual(attempted, pids) {
					t.Fatalf("failure/attempts: %v, %v; want %v", err, attempted, pids)
				}
				waitStopTestChild(t, children[2])
				if failureAt != stopLookupAndKillFailure {
					waitStopTestChild(t, children[0])
				}
				select {
				case <-children[1].done:
					t.Fatal("simulated failed member unexpectedly stopped")
				default:
				}
				killed, failed := 2, 1
				if failureAt == stopLookupAndKillFailure {
					killed, failed = 1, 2
				}
				wantSummary := fmt.Sprintf("%d of 3 processes stopped for cluster %s (%d killed, 0 already exited, %d failed)", killed, connection.Name, killed, failed)
				if operation == restartClusterCommand {
					wantSummary = fmt.Sprintf("%d of 3 stop requests completed for cluster %s (%d killed, 0 already exited, %d errors)", killed, connection.Name, killed, failed)
				}
				for _, want := range []string{operation, "stop incomplete", "partial stop", wantSummary, fmt.Sprintf("process %d", pids[1])} {
					if !strings.Contains(err.Error(), want) {
						t.Fatalf("missing %q: %v", want, err)
					}
				}
				if failureAt == stopLookupAndKillFailure && !strings.Contains(err.Error(), fmt.Sprintf("unable to kill process %d", pids[0])) {
					t.Fatalf("earlier error missing from aggregate: %v", err)
				}
				if operation == restartClusterCommand {
					if !strings.Contains(err.Error(), "replacement was not started") || !strings.Contains(err.Error(), "previous configuration retained") {
						t.Fatal(err)
					}
				} else if strings.Contains(err.Error(), "replacement") || strings.Contains(err.Error(), managementTestRestart) {
					t.Fatalf("incorrect pure-stop diagnostic: %v", err)
				}
				if _, statErr := os.Stat(launchMarker); !errors.Is(statErr, os.ErrNotExist) || starts != 0 || writes != 0 {
					t.Fatalf("replacement/persistence reached: marker=%v starts=%d writes=%d", statErr, starts, writes)
				}
				after, err := os.ReadFile(viper.ConfigFileUsed())
				if err != nil || !bytes.Equal(before, after) || !reflect.DeepEqual(Config.Clusters[0], connection) || !reflect.DeepEqual(viper.Get(clusterKey), Config.Clusters) {
					t.Fatalf("failed stop changed saved endpoints: %v", err)
				}
			})
		}
	}
}

func TestManagementStopDisappearanceErrors(t *testing.T) {
	for _, action := range []string{stopLookupFailure, managementTestKill} {
		t.Run(action, func(t *testing.T) {
			stopTestSeams(t)
			stale, live := stopTestChild(t), stopTestChild(t)
			if err := stale.command.Process.Kill(); err != nil {
				t.Fatal(err)
			}
			waitStopTestChild(t, stale)
			pid := stale.command.Process.Pid
			if action == stopLookupFailure {
				findProcessForStop = func(candidate int) (*os.Process, error) {
					if candidate == pid {
						return nil, fmt.Errorf("departed lookup: %w", os.ErrProcessDone)
					}
					return os.FindProcess(candidate)
				}
			} else {
				killProcessForStop = func(process *os.Process) error {
					if process.Pid == pid {
						return fmt.Errorf("departed kill: %w", os.ErrProcessDone)
					}
					return process.Kill()
				}
			}
			cmd := managementTestCommand(t)
			if err := stopClusterProcesses(cmd, managementFixtureName, []int{pid, live.command.Process.Pid}); err != nil {
				t.Fatal(err)
			}
			waitStopTestChild(t, live)
			if output := cmd.OutOrStdout().(*bytes.Buffer).String(); !strings.Contains(output, "1 killed, 1 already exited, 0 failed") {
				t.Fatal(output)
			}
		})
	}
}

func TestManagementStopRestartAfterExitedMember(t *testing.T) {
	managedTestState(t)
	stopTestSeams(t)
	stale, live := stopTestChild(t), stopTestChild(t)
	if err := stale.command.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	waitStopTestChild(t, stale)
	var stopped atomic.Bool
	connection, endpoint := stopTestConnectionEndpoint(t, []int{stale.command.Process.Pid, live.command.Process.Pid}, &stopped)
	killProcessForStop = func(p *os.Process) error {
		err := p.Kill()
		if p.Pid == live.command.Process.Pid && err == nil {
			stopped.Store(true)
			endpoint.Close()
		}
		return err
	}
	fakeJava(t, javaFixture{Version: javaVersion17, Query: true, Sleep: true})
	var replacement []*managedProcess
	awaitManagedStartup = func(_ *cobra.Command, _ ClusterConnection, processes []*managedProcess) error {
		replacement = processes
		return nil
	}
	t.Cleanup(func() {
		if err := cleanupManagedProcesses(replacement); err != nil {
			t.Errorf("replacement fixture cleanup: %v", err)
		}
	})
	automaticallyConfirm = true
	cmd := managementTestCommand(t)
	if err := runClusterOperation(cmd, connection.Name, restartClusterCommand); err != nil {
		t.Fatal(err)
	}
	waitStopTestChild(t, live)
	if len(replacement) != 1 || Config.Clusters[0].ConnectionURL == connection.ConnectionURL {
		t.Fatalf("successful replacement was not persisted: children=%d connection=%+v", len(replacement), Config.Clusters[0])
	}
	if output := cmd.OutOrStdout().(*bytes.Buffer).String(); !strings.Contains(output, "1 killed, 1 already exited, 0 errors") || !strings.Contains(output, "restart cluster completed") {
		t.Fatal(output)
	}
}
