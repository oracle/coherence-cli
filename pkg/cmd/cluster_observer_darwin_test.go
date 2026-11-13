/*
 * Copyright (c) 2026 Oracle and/or its affiliates.
 * Licensed under the Universal Permissive License v 1.0 as shown at
 * https://oss.oracle.com/licenses/upl.
 */

package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/sys/unix"
)

func TestManagementObserverNativeZombie(t *testing.T) {
	for _, capture := range []string{"before-exit", "already-zombie"} {
		t.Run(capture, func(t *testing.T) {
			pid, dir := observerTestNonChild(t, true)
			process, err := os.FindProcess(pid)
			if err != nil {
				t.Fatal(err)
			}
			defer func() {
				if err := process.Release(); err != nil {
					t.Errorf("release non-child process reference: %v", err)
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
			observerTestEventually(t, "native zombie state", func() bool {
				info, err := unix.SysctlKinfoProc("kern.proc.pid", pid)
				return err == nil && int(info.Proc.P_pid) == pid && info.Proc.P_stat == 5 // Darwin SZOMB
			})
			if capture == "already-zombie" {
				observer, err = newProcessExitObserver(process)
				if err != nil {
					t.Fatal(err)
				}
			}
			defer observer.close()
			observerTestEventually(t, "exit of unreaped original non-child", func() bool {
				exited, err := observer.exited()
				if err != nil {
					t.Fatal(err)
				}
				return exited
			})
			t.Logf("native zombie PID %d (%s) exit observed using %T before owner reaps", pid, capture, observer)
		})
	}
}
