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

	"golang.org/x/sys/windows"
)

type handleProcessObserver struct{ process *os.Process }

func newProcessExitObserver(process *os.Process) (processExitObserver, error) {
	if err := process.WithHandle(func(uintptr) {
		// Probe successful handle acquisition only; no handle operation is needed.
	}); err != nil {
		return nil, err
	}
	return handleProcessObserver{process}, nil
}

func (o handleProcessObserver) exited() (bool, error) {
	var waitErr error
	var result uint32
	err := o.process.WithHandle(func(handle uintptr) {
		result, waitErr = windows.WaitForSingleObject(windows.Handle(handle), 0)
	})
	if waitErr == nil && err == nil && result != windows.WAIT_OBJECT_0 && result != uint32(windows.WAIT_TIMEOUT) {
		waitErr = fmt.Errorf("unexpected process wait status %#x", result)
	}
	return result == windows.WAIT_OBJECT_0, errors.Join(err, waitErr)
}

func (handleProcessObserver) close() error { return nil } // owner releases os.Process
