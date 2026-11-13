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

	"golang.org/x/sys/unix"
)

type kqueueProcessObserver struct {
	fd  int
	pid uint64
}

// NOTE_EXIT is registered before termination, follows the original process,
// works for non-children and does not reap. Never re-register a numeric PID
// during waiting: an exit event remains evidence even if that PID is reused.
func newProcessExitObserver(process *os.Process) (processExitObserver, error) {
	if process.Pid <= 0 {
		return nil, fmt.Errorf("invalid process PID %d", process.Pid)
	}
	fd, err := unix.Kqueue()
	if err != nil {
		return nil, err
	}
	unix.CloseOnExec(fd)
	change := unix.Kevent_t{Ident: uint64(process.Pid), Filter: unix.EVFILT_PROC, Flags: unix.EV_ADD | unix.EV_ONESHOT, Fflags: unix.NOTE_EXIT}
	_, err = unix.Kevent(fd, []unix.Kevent_t{change}, nil, nil)
	if err != nil {
		closeErr := unix.Close(fd)
		if errors.Is(err, unix.ESRCH) && closeErr == nil {
			return exitedProcessObserver{}, nil
		}
		return nil, errors.Join(err, closeErr)
	}
	return &kqueueProcessObserver{fd: fd, pid: change.Ident}, nil
}

func (o *kqueueProcessObserver) exited() (bool, error) {
	events := make([]unix.Kevent_t, 1)
	n, err := unix.Kevent(o.fd, nil, events, &unix.Timespec{})
	if errors.Is(err, unix.EINTR) {
		return false, nil
	}
	if err != nil || n == 0 {
		return false, err
	}
	event := events[0]
	if event.Flags&unix.EV_ERROR != 0 {
		// Darwin Kevent_t.Data carries a nonnegative errno for EV_ERROR.
		// Both supported Darwin architectures have a 64-bit uintptr.
		if event.Data < 0 {
			return false, fmt.Errorf("invalid kqueue error data: %d", event.Data)
		}
		return false, fmt.Errorf("kqueue exit event: %w", unix.Errno(event.Data))
	}
	if event.Ident != o.pid || event.Filter != unix.EVFILT_PROC || event.Fflags&unix.NOTE_EXIT == 0 {
		return false, fmt.Errorf("unexpected kqueue exit event for PID %d", o.pid)
	}
	return true, nil
}

func (o *kqueueProcessObserver) close() error { return unix.Close(o.fd) }
