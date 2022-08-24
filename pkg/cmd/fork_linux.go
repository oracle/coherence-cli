//go:build linux

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

// setForkProcess set the process to be forked for linux
func setForkProcess(process *exec.Cmd) {
	process.SysProcAttr = &syscall.SysProcAttr{
		Setsid: true,
	}
}

func signalProcess(proc *os.Process) error {
	return proc.Signal(syscall.SIGCONT)
}
