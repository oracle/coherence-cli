/*
 * Copyright (c) 2026 Oracle and/or its affiliates.
 * Licensed under the Universal Permissive License v 1.0 as shown at
 * https://oss.oracle.com/licenses/upl.
 */

package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
)

type processExitObserver interface {
	exited() (bool, error) // read-only, nonblocking; never Wait/reap or signal
	close() error
}

type exitedProcessObserver struct{}

func (exitedProcessObserver) exited() (bool, error) { return true, nil }
func (exitedProcessObserver) close() error          { return nil }

type oldClusterProcess struct {
	pid            int
	process        *os.Process
	observer       processExitObserver
	exited         bool
	observationErr error
}

var observeProcessForRestart = newProcessExitObserver
var managedShutdownTimeout = 10 * time.Second
var managedShutdownPollInterval = 25 * time.Millisecond

// The single deadline starts before observer collection/termination requests.
// Every original process must exit before probing proposed endpoints, and every
// probe/retry shares the remaining budget. No post-stop REST requests are made.
func stopClusterForRestart(cmd *cobra.Command, connection ClusterConnection, management ClusterManagement, processIDs []int) (err error) {
	ctx, cancel := context.WithTimeout(cmd.Context(), managedShutdownTimeout)
	defer cancel()
	var observed []*oldClusterProcess
	defer func() {
		for _, old := range observed {
			if old.observer != nil {
				if closeErr := old.observer.close(); closeErr != nil {
					err = errors.Join(err, fmt.Errorf("release exit observer PID %d: %w", old.pid, closeErr))
				}
			}
			if old.process != nil {
				if closeErr := old.process.Release(); closeErr != nil {
					err = errors.Join(err, fmt.Errorf("release original process PID %d: %w", old.pid, closeErr))
				}
			}
		}
		err = errors.Join(err, ctx.Err())
		if err != nil {
			var unresolved []int
			for _, old := range observed {
				if !old.exited {
					unresolved = append(unresolved, old.pid)
					if detail, ok := old.observer.(interface{ pendingReason() string }); ok {
						err = errors.Join(err, fmt.Errorf("original PID %d: %s", old.pid, detail.pendingReason()))
					}
				}
			}
			err = fmt.Errorf("unable to restart cluster %s: stop incomplete or endpoint readiness failed; unresolved original PIDs %v; shutdown may be partial or unknown; replacement was not started and previous configuration retained; stopped members cannot be restored automatically: %w", connection.Name, unresolved, err)
		}
	}()
	if err = requestClusterStop(cmd, connection.Name, processIDs, &observed); err != nil {
		return errors.Join(err, ctx.Err())
	}
	var endpointErr error
	for {
		if err = ctx.Err(); err != nil {
			return errors.Join(err, endpointErr)
		}
		allExited := true
		var observationErrors []error
		for _, old := range observed {
			if old.exited {
				continue
			}
			old.exited, old.observationErr = old.observer.exited()
			if old.observationErr != nil {
				old.exited = false
				observationErrors = append(observationErrors, fmt.Errorf("observe original PID %d; exit unresolved: %w", old.pid, old.observationErr))
			}
			allExited = allExited && old.exited
		}
		if len(observationErrors) > 0 {
			return errors.Join(observationErrors...)
		}
		if allExited {
			probeErr := checkManagementPortsContext(ctx, connection, management, false)
			if ctx.Err() != nil {
				return errors.Join(ctx.Err(), endpointErr, probeErr)
			}
			endpointErr = probeErr
			if endpointErr == nil {
				return ctx.Err()
			}
			if !errors.Is(endpointErr, errManagementAddressInUse) {
				return endpointErr
			}
		}
		timer := time.NewTimer(managedShutdownPollInterval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return errors.Join(ctx.Err(), endpointErr)
		case <-timer.C:
		}
	}
}
