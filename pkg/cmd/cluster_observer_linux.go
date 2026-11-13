/*
 * Copyright (c) 2026 Oracle and/or its affiliates.
 * Licensed under the Universal Permissive License v 1.0 as shown at
 * https://oss.oracle.com/licenses/upl.
 */

package cmd

import (
	"errors"
	"fmt"
	"math"
	"os"
	"syscall"

	"golang.org/x/sys/unix"
)

type pidfdProcessObserver struct{ process *os.Process }
type absentProcessObserver struct{ process *os.Process }

func newProcessExitObserver(process *os.Process) (processExitObserver, error) {
	err := process.WithHandle(func(uintptr) {
		// Probe successful handle acquisition only; no handle operation is needed.
	})
	switch {
	case err == nil:
		return pidfdProcessObserver{process}, nil
	case errors.Is(err, os.ErrProcessDone):
		return exitedProcessObserver{}, nil
	case errors.Is(err, os.ErrNoHandle):
		// Older/restricted Linux without Go pidfds: only confirmed absence is
		// exit proof. Permission/unknown errors fail closed; PID reuse/zombies
		// may conservatively block until timeout. No repeated destructive signal.
		o := absentProcessObserver{process}
		_, err = o.exited()
		return o, err
	default:
		return nil, err
	}
}

func (o pidfdProcessObserver) exited() (bool, error) {
	var pollErr error
	exited := false
	err := o.process.WithHandle(func(handle uintptr) {
		exited, pollErr = pollProcessHandle(handle, unix.Poll)
	})
	return exited, errors.Join(err, pollErr)
}

// pollProcessHandle must run inside WithHandle so the pidfd cannot be released
// or reused while Poll observes it. A bad descriptor is never exit proof.
func pollProcessHandle(handle uintptr, poll func([]unix.PollFd, int) (int, error)) (bool, error) {
	if handle > math.MaxInt32 {
		return false, fmt.Errorf("pidfd handle %d exceeds Poll descriptor range", handle)
	}
	fds := []unix.PollFd{{Fd: int32(handle), Events: unix.POLLIN}}
	_, err := poll(fds, 0)
	interrupted := errors.Is(err, unix.EINTR)
	if interrupted {
		err = nil
	}
	if fds[0].Revents&(unix.POLLERR|unix.POLLNVAL) != 0 {
		err = errors.Join(err, fmt.Errorf("pidfd poll returned unexpected events %#x", fds[0].Revents))
	}
	if err != nil {
		return false, err
	}
	// Interrupted observation is pending, even if a readiness bit was set.
	return !interrupted && fds[0].Revents&(unix.POLLIN|unix.POLLHUP) != 0, nil
}

func (pidfdProcessObserver) close() error { return nil } // owner releases os.Process

func (o absentProcessObserver) exited() (bool, error) {
	err := o.process.Signal(syscall.Signal(0))
	if errors.Is(err, syscall.ESRCH) || errors.Is(err, os.ErrProcessDone) {
		return true, nil
	}
	return false, err
}

func (absentProcessObserver) close() error { return nil }

func (absentProcessObserver) pendingReason() string {
	return "no stable process handle available; signal-zero observation has not confirmed absence (a present PID may be live, a zombie or reused)"
}
