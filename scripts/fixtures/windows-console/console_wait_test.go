// Copyright (c) 2026 Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at
// https://oss.oracle.com/licenses/upl.

//go:build windows

package main

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"syscall"
	"testing"
	"time"
)

// Exercise main with the actual unsuffixed argv[0] that Go gives a resolved
// mvn.exe, rather than merely testing a name-comparison helper.
func TestMavenInvocation(t *testing.T) {
	if os.Getenv("COHCTL_WINDOWS_MAVEN_CHILD") == "1" {
		main()
		return
	}
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"mvn", "mvn.exe", `C:\private fixture\MVN.EXE`} {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			child := exec.CommandContext(ctx, self, "-test.run=^TestMavenInvocation$")
			child.Args[0] = name
			child.Env = append(os.Environ(), "COHCTL_WINDOWS_MAVEN_CHILD=1", "COHCTL_WINDOWS_FIXTURE="+dir)
			output, err := child.CombinedOutput() // This command alone owns Wait.
			if err != nil || ctx.Err() != nil {
				t.Fatalf("Maven shim failed or required rescue: %v, %v, %s", err, ctx.Err(), output)
			}
			entries, err := os.ReadDir(dir)
			if err != nil || len(entries) != 0 {
				t.Fatalf("Maven shim entered member branch: %v, %v", entries, err)
			}
		})
	}
}

func TestConsoleTransientReadiness(t *testing.T) {
	retry := make(chan time.Time, 2)
	retry <- time.Time{}
	retry <- time.Time{}
	calls := 0
	r, err := waitForOwnedConsole(context.Background(), make(chan error), retry, func() error {
		calls++
		if calls < 3 {
			return errInvalidHandle
		}
		return nil
	})
	if err != nil || r.Attempts != 3 || r.Exited {
		t.Fatalf("transient readiness: %+v, %v", r, err)
	}
}

func TestConsoleTerminalFailure(t *testing.T) {
	for _, cause := range []error{syscall.ERROR_ACCESS_DENIED, syscall.Errno(87), syscall.Errno(9876)} {
		t.Run(cause.Error(), func(t *testing.T) {
			r, err := waitForOwnedConsole(context.Background(), make(chan error), nil, func() error { return cause })
			if !errors.Is(err, cause) || r.Attempts != 1 || r.Exited {
				t.Fatalf("terminal cause changed or retried: %+v, %v", r, err)
			}
		})
	}
}

func TestConsoleEarlyExit(t *testing.T) {
	for _, waitErr := range []error{nil, errors.New("child exit 7")} {
		for _, duringRetry := range []bool{false, true} {
			done := make(chan error, 1)
			if !duringRetry {
				done <- waitErr
			}
			r, err := waitForOwnedConsole(context.Background(), done, nil, func() error {
				done <- waitErr
				return errInvalidHandle
			})
			expectedAttempts := 0
			if duringRetry {
				expectedAttempts = 1
			}
			if err == nil || !r.Exited || r.Attempts != expectedAttempts || (waitErr != nil && !errors.Is(err, waitErr)) {
				t.Fatalf("early exit (%v, %v): %+v, %v", waitErr, duringRetry, r, err)
			}
			select {
			case unexpected := <-done:
				t.Fatalf("Wait result not consumed exactly once: %v", unexpected)
			default:
			}
		}
	}
}

func TestConsoleDeadline(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	r, err := waitForOwnedConsole(ctx, make(chan error), nil, func() error { return errInvalidHandle })
	if !errors.Is(err, context.DeadlineExceeded) || !errors.Is(err, errInvalidHandle) || r.Exited || r.Attempts != 1 {
		t.Fatalf("deadline lost last cause: %+v, %v", r, err)
	}
	// An exhausted overall lifetime prevents attachment setup from starting.
	r, err = waitForOwnedConsole(ctx, make(chan error), nil, func() error { t.Fatal("attached after deadline"); return nil })
	if !errors.Is(err, context.DeadlineExceeded) || r.Attempts != 0 {
		t.Fatalf("expired overall deadline: %+v, %v", r, err)
	}
}
