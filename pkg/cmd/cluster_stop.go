/*
 * Copyright (c) 2026 Oracle and/or its affiliates.
 * Licensed under the Universal Permissive License v 1.0 as shown at
 * https://oss.oracle.com/licenses/upl.
 */

package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// These seams apply only to the stop phase of stop/restart operations.
var findProcessForStop = os.FindProcess
var killProcessForStop = (*os.Process).Kill

// A departed member is already stopped. Genuine failures must not prevent
// attempts on later members, but must prevent replacement startup.
func stopClusterProcesses(cmd *cobra.Command, clusterName string, processIDs []int) error {
	return requestClusterStop(cmd, clusterName, processIDs, nil)
}

// Restart retains each original reference and captures its exit observer before
// its sole termination request. An observation failure must not skip later stop
// attempts; it does, however, prohibit replacement. Pure stop retains its API.
func requestClusterStop(cmd *cobra.Command, clusterName string, processIDs []int, observed *[]*oldClusterProcess) error {
	killed, exited := 0, 0
	var failures []error
	for _, pid := range processIDs {
		process, err := findProcessForStop(pid)
		var old *oldClusterProcess
		if observed != nil {
			old = &oldClusterProcess{pid: pid, process: process}
			*observed = append(*observed, old)
			if err == nil {
				old.observer, old.observationErr = observeProcessForRestart(process)
				if old.observationErr != nil {
					failures = append(failures, fmt.Errorf("observe original PID %d before stop: %w", pid, old.observationErr))
				}
				// Confirmed absence at acquisition needs no numeric-PID signal.
				if _, done := old.observer.(exitedProcessObserver); done {
					err = os.ErrProcessDone
				}
			}
		}
		action := "find"
		if err == nil {
			action = "kill"
			err = killProcessForStop(process)
		}
		switch {
		case err == nil:
			killed++
			cmd.Printf("killed process %d\n", pid)
		case processAlreadyExited(pid, process, err):
			exited++
			if old != nil {
				old.exited = true
			}
			cmd.Printf("process %d already exited\n", pid)
		default:
			failure := fmt.Errorf("unable to %s process %d: %w", action, pid, err)
			failures = append(failures, failure)
			cmd.Println(failure)
		}
		if process != nil && observed == nil {
			_ = process.Release()
		}
	}
	summary := fmt.Sprintf("%d of %d processes stopped for cluster %s (%d killed, %d already exited, %d failed)",
		killed+exited, len(processIDs), clusterName, killed, exited, len(failures))
	if observed != nil {
		summary = fmt.Sprintf("%d of %d stop requests completed for cluster %s (%d killed, %d already exited, %d errors); original-process exit still requires observation",
			killed+exited, len(processIDs), clusterName, killed, exited, len(failures))
	}
	if len(failures) > 0 {
		return fmt.Errorf("partial stop: %s: %w", summary, errors.Join(failures...))
	}
	cmd.Println(summary)
	return nil
}
