// Copyright (c) 2026, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at
// https://oss.oracle.com/licenses/upl.

//go:build windows

package cmd

import (
	"fmt"
	"os"
	"testing"

	"golang.org/x/sys/windows"
)

func TestManagementStopWindowsExitClassification(t *testing.T) {
	for _, tt := range []struct {
		pid    int
		err    error
		exited bool
	}{
		{1, fmt.Errorf("wrapped: %w", os.ErrProcessDone), true},
		{1, os.NewSyscallError("OpenProcess", windows.ERROR_INVALID_PARAMETER), true},
		{0, os.NewSyscallError("OpenProcess", windows.ERROR_INVALID_PARAMETER), false},
		{-1, os.NewSyscallError("OpenProcess", windows.ERROR_INVALID_PARAMETER), false},
		{1, os.NewSyscallError("other", windows.ERROR_INVALID_PARAMETER), false},
		{1, os.NewSyscallError("OpenProcess", windows.ERROR_ACCESS_DENIED), false},
		{1, os.ErrPermission, false},
	} {
		if got := processAlreadyExited(tt.pid, nil, tt.err); got != tt.exited {
			t.Errorf("PID %d, %v: exited=%t want %t", tt.pid, tt.err, got, tt.exited)
		}
	}
	child := stopTestChild(t)
	pid := child.command.Process.Pid
	process, err := os.FindProcess(pid)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := process.Release(); err != nil {
			t.Errorf("release Windows process reference: %v", err)
		}
	}()
	denied := os.NewSyscallError("TerminateProcess", windows.ERROR_ACCESS_DENIED)
	if processAlreadyExited(pid, process, denied) {
		t.Fatal("permission failure on live process classified as exited")
	}
	if err := child.command.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	waitStopTestChild(t, child)
	if !processAlreadyExited(pid, process, denied) {
		t.Fatal("terminated process with signalled handle not classified as exited")
	}
}
