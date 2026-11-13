/*
 * Copyright (c) 2026 Oracle and/or its affiliates.
 * Licensed under the Universal Permissive License v 1.0 as shown at
 * https://oss.oracle.com/licenses/upl.
 */

package cmd

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
)

// managedOperationSignals starts the cancellable, side-effecting portion of a
// managed operation. Keep the caller's context (including embedding deadlines)
// and restore it and the signal registrations when this invocation returns.
// On Windows Go delivers console Ctrl-C/Ctrl-Break as os.Interrupt; SIGTERM is
// also supported by Notify for console close/logoff/shutdown, subject to the OS
// termination deadline. Native Windows behavior needs separate qualification.
func managedOperationSignals(cmd *cobra.Command) func() {
	previous := cmd.Context()
	ctx, stop := signal.NotifyContext(previous, os.Interrupt, syscall.SIGTERM)
	cmd.SetContext(ctx)
	return func() {
		stop()
		cmd.SetContext(previous)
	}
}
