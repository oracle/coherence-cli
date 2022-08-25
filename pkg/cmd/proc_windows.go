//go:build windows

/*
 * Copyright (c) 2022, Oracle and/or its affiliates.
 * Licensed under the Universal Permissive License v 1.0 as shown at
 * https://oss.oracle.com/licenses/upl.
 */

package cmd

import (
	"os"
	"os/exec"
	"syscall"
)

// setForkProcess set the process to be forked for windows
func setForkProcess(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP,
	}
}

func signalProcess(proc *os.Process) error {
	// no-op
	return nil
}

func handleCTRLC() {
	// no-op
}
