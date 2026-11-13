//go:build darwin || linux

/*
 * Copyright (c) 2022, 2026 Oracle and/or its affiliates.
 * Licensed under the Universal Permissive License v 1.0 as shown at
 * https://oss.oracle.com/licenses/upl.
 */

package cmd

import (
	"errors"
	"os"
	"os/exec"
	"syscall"
)

func processAlreadyExited(_ int, _ *os.Process, err error) bool {
	return errors.Is(err, os.ErrProcessDone) || errors.Is(err, syscall.ESRCH)
}

// processKillAfterWait recognizes only the platform's released-handle error.
// The caller must first establish completion from the existing owned Wait.
func processKillAfterWait(_ error) bool {
	return false
}

// setForkProcess set the process to be forked for linux.
func setForkProcess(process *exec.Cmd) {
	process.SysProcAttr = &syscall.SysProcAttr{
		Setsid: true,
	}
}
