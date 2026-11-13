//go:build windows

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

	"golang.org/x/sys/windows"
)

func processAlreadyExited(pid int, process *os.Process, err error) bool {
	if errors.Is(err, os.ErrProcessDone) {
		return true
	}
	if process == nil {
		// OpenProcess reports an invalid parameter for a departed positive PID.
		// Do not classify invalid PID 0 or an unrelated syscall error this way.
		var syscallErr *os.SyscallError
		return pid > 0 && uint64(pid) <= uint64(^uint32(0)) && errors.As(err, &syscallErr) &&
			syscallErr.Syscall == "OpenProcess" && errors.Is(err, windows.ERROR_INVALID_PARAMETER)
	}
	if !errors.Is(err, windows.ERROR_ACCESS_DENIED) {
		return false
	}
	// TerminateProcess can report access denied after exit. Require a signalled
	// process handle so a real permission failure remains a stop failure.
	exited := false
	checkErr := process.WithHandle(func(handle uintptr) {
		result, waitErr := windows.WaitForSingleObject(windows.Handle(handle), 0)
		exited = waitErr == nil && result == windows.WAIT_OBJECT_0
	})
	return checkErr == nil && exited
}

// processKillAfterWait recognizes only the platform's released-handle error.
// The caller must first establish completion from the existing owned Wait.
func processKillAfterWait(err error) bool {
	return errors.Is(err, syscall.EINVAL)
}

// setForkProcess set the process to be forked for windows.
func setForkProcess(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP,
	}
}
