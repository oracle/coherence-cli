// Copyright (c) 2026, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at
// https://oss.oracle.com/licenses/upl.

//go:build darwin || linux

package cmd

import (
	"fmt"
	"os"
	"syscall"
	"testing"
)

func TestManagementStopUnixExitClassification(t *testing.T) {
	for _, tt := range []struct {
		err    error
		exited bool
	}{
		{os.ErrProcessDone, true},
		{fmt.Errorf("wrapped: %w", os.ErrProcessDone), true},
		{os.NewSyscallError("kill", syscall.ESRCH), true},
		{os.NewSyscallError("kill", syscall.EPERM), false},
		{os.NewSyscallError("kill", syscall.EACCES), false},
		{os.ErrPermission, false},
		{syscall.EINVAL, false},
		{fmt.Errorf("other failure"), false},
	} {
		if got := processAlreadyExited(1, nil, tt.err); got != tt.exited {
			t.Errorf("%v: exited=%t want %t", tt.err, got, tt.exited)
		}
	}
}
