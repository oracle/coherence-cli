// Copyright (c) 2026 Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at
// https://oss.oracle.com/licenses/upl.

package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestManagementObserverLinuxCapability(t *testing.T) {
	child := stopTestChild(t)
	process, err := os.FindProcess(child.command.Process.Pid)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := process.Release(); err != nil {
			t.Errorf("release native Linux process reference: %v", err)
		}
	}()
	observer, err := newProcessExitObserver(process)
	if err != nil {
		t.Fatal(err)
	}
	defer observer.close()
	t.Logf("native Linux observer=%T WithHandle=%v pid=%d", observer, process.WithHandle(func(uintptr) {}), process.Pid)
	if exited, err := observer.exited(); exited || err != nil {
		t.Fatalf("live original: exited=%v err=%v", exited, err)
	}
}

func TestManagementObserverLinuxNoHandle(t *testing.T) {
	child := stopTestChild(t)
	process, err := os.FindProcess(child.command.Process.Pid)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := process.Release(); err != nil {
			t.Errorf("release native Linux process reference: %v", err)
		}
	}()
	if err = process.WithHandle(func(uintptr) {}); !errors.Is(err, os.ErrNoHandle) {
		t.Skipf("UNRUN no-handle kernel/policy branch: actual WithHandle=%v", err)
	}
	observer, err := newProcessExitObserver(process)
	if err != nil {
		t.Fatal(err)
	}
	defer observer.close()
	if _, ok := observer.(absentProcessObserver); !ok {
		t.Fatalf("actual no-handle environment selected %T", observer)
	}
	if exited, err := observer.exited(); exited || err != nil {
		t.Fatalf("present PID must remain pending: %v %v", exited, err)
	}
	if err := child.command.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	waitStopTestChild(t, child)
	observerTestEventually(t, "confirmed absence in actual no-handle environment", func() bool {
		exited, err := observer.exited()
		if err != nil {
			t.Fatal(err)
		}
		return exited
	})
}

func TestManagementObserverLinuxZombie(t *testing.T) {
	for _, capture := range []string{"before-exit", "already-zombie"} {
		t.Run(capture, func(t *testing.T) {
			pid, dir := observerTestNonChild(t, true)
			process, err := os.FindProcess(pid)
			if err != nil {
				t.Fatal(err)
			}
			defer func() {
				if err := process.Release(); err != nil {
					t.Errorf("release native Linux process reference: %v", err)
				}
			}()
			var observer processExitObserver
			if capture == "before-exit" {
				observer, err = newProcessExitObserver(process)
				if err != nil {
					t.Fatal(err)
				}
			}
			if err := os.WriteFile(filepath.Join(dir, "exit"), nil, 0600); err != nil {
				t.Fatal(err)
			}
			observerTestEventually(t, "native Linux zombie state", func() bool {
				data, err := os.ReadFile(fmt.Sprintf("/proc/%d/stat", pid))
				end := strings.LastIndex(string(data), ") ")
				return err == nil && end >= 0 && strings.HasPrefix(string(data)[end+2:], "Z ")
			})
			if capture == "already-zombie" {
				observer, err = newProcessExitObserver(process)
				if err != nil {
					t.Fatal(err)
				}
			}
			defer observer.close()
			exited, err := observer.exited()
			if err != nil {
				t.Fatal(err)
			}
			_, fallback := observer.(absentProcessObserver)
			if exited == fallback {
				t.Fatalf("unreaped original zombie: observer=%T exited=%v", observer, exited)
			}
			t.Logf("native Linux zombie PID=%d capture=%s observer=%T exited=%v; owner has not reaped", pid, capture, observer, exited)
			if err := os.WriteFile(filepath.Join(dir, "reap"), nil, 0600); err != nil {
				t.Fatal(err)
			}
			observerTestEventually(t, "original exit after sole owner reaps", func() bool {
				exited, err := observer.exited()
				if err != nil {
					t.Fatal(err)
				}
				return exited
			})
		})
	}
}
