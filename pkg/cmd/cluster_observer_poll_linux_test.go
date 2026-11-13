// Copyright (c) 2026 Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at
// https://oss.oracle.com/licenses/upl.

package cmd

import (
	"errors"
	"math"
	"os"
	"testing"

	"golang.org/x/sys/unix"
)

func TestManagementObserverPollRange(t *testing.T) {
	for handle, descriptor := range map[uintptr]int32{0: 0, 1: 1, math.MaxInt32: math.MaxInt32} {
		called := false
		exited, err := pollProcessHandle(handle, func(fds []unix.PollFd, timeout int) (int, error) {
			called = true
			if fds[0].Fd != descriptor || fds[0].Events != unix.POLLIN || timeout != 0 {
				t.Fatalf("descriptor/observation changed: %+v %d", fds, timeout)
			}
			return 0, nil
		})
		if !called || exited || err != nil {
			t.Fatalf("valid boundary %d: called=%t exit=%t err=%v", handle, called, exited, err)
		}
	}
	for _, handle := range []uintptr{math.MaxInt32 + 1, ^uintptr(0)} {
		exited, err := pollProcessHandle(handle, func([]unix.PollFd, int) (int, error) { t.Fatal("invalid handle reached Poll"); return 0, nil })
		if exited || err == nil {
			t.Fatalf("invalid boundary %d became exit proof: %t %v", handle, exited, err)
		}
	}
}

func TestManagementObserverPollOutcomes(t *testing.T) {
	for _, tt := range []struct {
		name           string
		events         int16
		cause          error
		exited, failed bool
	}{
		{"pending", 0, nil, false, false}, {"readable", unix.POLLIN, nil, true, false}, {"hangup", unix.POLLHUP, nil, true, false},
		{"interrupted", unix.POLLIN, unix.EINTR, false, false}, {"poll-error", unix.POLLIN, unix.EBADF, false, true},
		{"invalid-descriptor", unix.POLLIN | unix.POLLNVAL, nil, false, true},
		{"interrupted-invalid", unix.POLLNVAL, unix.EINTR, false, true}, {"event-error", unix.POLLIN | unix.POLLERR, nil, false, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			exited, err := pollProcessHandle(7, func(fds []unix.PollFd, _ int) (int, error) { fds[0].Revents = tt.events; return 1, tt.cause })
			if exited != tt.exited || (err != nil) != tt.failed {
				t.Fatalf("exit=%t err=%v", exited, err)
			}
			if tt.failed && tt.cause != nil && tt.cause != unix.EINTR && !errors.Is(err, tt.cause) {
				t.Fatalf("lost cause: %v", err)
			}
		})
	}
}

func TestManagementObserverPollRealDescriptor(t *testing.T) {
	read, write, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer read.Close()
	defer write.Close()
	if exited, err := pollProcessHandle(read.Fd(), unix.Poll); exited || err != nil {
		t.Fatalf("empty pipe: %t %v", exited, err)
	}
	if _, err := write.Write([]byte{1}); err != nil {
		t.Fatal(err)
	}
	if exited, err := pollProcessHandle(read.Fd(), unix.Poll); !exited || err != nil {
		t.Fatalf("readable descriptor: %t %v", exited, err)
	}
	fd := read.Fd()
	if err := read.Close(); err != nil {
		t.Fatal(err)
	}
	if exited, err := pollProcessHandle(fd, unix.Poll); exited || err == nil {
		t.Fatalf("closed descriptor became exit proof: %t %v", exited, err)
	}
}
