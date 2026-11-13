/*
 * Copyright (c) 2026 Oracle and/or its affiliates.
 * Licensed under the Universal Permissive License v 1.0 as shown at
 * https://oss.oracle.com/licenses/upl.
 */

package cmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

// A separate owner starts/reaps the member. The observing test is NOT its
// parent. Both helper levels have independent rescue limits and task-only PIDs.
func TestManagementObserverHelper(t *testing.T) {
	mode := os.Getenv("COHCTL_OBSERVER_HELPER")
	if mode == "" {
		return
	}
	dir := os.Getenv("COHCTL_OBSERVER_DIR")
	if mode == "member" {
		deadline := time.Now().Add(8 * time.Second)
		for time.Now().Before(deadline) {
			if _, err := os.Stat(filepath.Join(dir, "exit")); err == nil { // #nosec G703 -- Guarded subprocess fixture; dir comes only from the parent test private t.TempDir, never CLI input.
				os.Exit(0)
			}
			time.Sleep(5 * time.Millisecond)
		}
		os.Exit(0)
	}
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	child := exec.Command(executable, "-test.run=^TestManagementObserverHelper$")
	child.Env = append(os.Environ(), "COHCTL_OBSERVER_HELPER=member")
	if err := child.Start(); err != nil {
		t.Fatal(err)
	}
	rescue := time.AfterFunc(9*time.Second, func() { _ = child.Process.Kill() })
	defer rescue.Stop()
	if err := os.WriteFile(filepath.Join(dir, "pid"), []byte(strconv.Itoa(child.Process.Pid)), 0600); err != nil { // #nosec G703 -- Guarded subprocess fixture; dir comes only from the parent test private t.TempDir, never CLI input.
		_ = child.Process.Kill()
		_ = child.Wait()
		t.Fatal(err)
	}
	if mode == "zombie-owner" {
		deadline := time.Now().Add(6 * time.Second)
		for time.Now().Before(deadline) {
			if _, err := os.Stat(filepath.Join(dir, "reap")); err == nil { // #nosec G703 -- Guarded subprocess fixture; dir comes only from the parent test private t.TempDir, never CLI input.
				break
			}
			time.Sleep(5 * time.Millisecond)
		}
	}
	_ = child.Wait()
	_ = os.WriteFile(filepath.Join(dir, "reaped"), nil, 0600) // #nosec G703 -- Guarded subprocess fixture; dir comes only from the parent test private t.TempDir, never CLI input.
	// Keep the parent alive until the root explicitly releases it.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(filepath.Join(dir, "owner-exit")); err == nil { // #nosec G703 -- Guarded subprocess fixture; dir comes only from the parent test private t.TempDir, never CLI input.
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func observerTestEventually(t *testing.T, description string, check func() bool) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if check() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("timed out: " + description)
}

func observerTestNonChild(t *testing.T, zombie bool) (int, string) {
	t.Helper()
	dir := t.TempDir()
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	mode := "owner"
	if zombie {
		mode = "zombie-owner"
	}
	owner := exec.Command(executable, "-test.run=^TestManagementObserverHelper$")
	owner.Env = append(os.Environ(), "COHCTL_OBSERVER_HELPER="+mode, "COHCTL_OBSERVER_DIR="+dir)
	if err := owner.Start(); err != nil {
		t.Fatal(err)
	}
	done := make(chan struct{})
	go func() { _ = owner.Wait(); close(done) }()
	t.Cleanup(func() {
		_ = os.WriteFile(filepath.Join(dir, "exit"), nil, 0600)
		_ = os.WriteFile(filepath.Join(dir, "reap"), nil, 0600)
		_ = os.WriteFile(filepath.Join(dir, "owner-exit"), nil, 0600)
		select {
		case <-done:
		case <-time.After(3 * time.Second):
			_ = owner.Process.Kill()
			<-done
			t.Error("non-child owner required rescue")
		}
	})
	pid := 0
	observerTestEventually(t, "non-child identity", func() bool {
		b, err := os.ReadFile(filepath.Join(dir, "pid"))
		if err != nil {
			return false
		}
		pid, err = strconv.Atoi(strings.TrimSpace(string(b)))
		return err == nil && pid > 0
	})
	return pid, dir
}

func TestManagementObserverNativeNonChild(t *testing.T) {
	for _, mode := range []string{"normal", "signal"} {
		t.Run(mode, func(t *testing.T) {
			pid, dir := observerTestNonChild(t, false)
			process, err := os.FindProcess(pid)
			if err != nil {
				t.Fatal(err)
			}
			defer func() {
				if err := process.Release(); err != nil {
					t.Errorf("release non-child process reference: %v", err)
				}
			}()
			observer, err := newProcessExitObserver(process)
			if err != nil {
				t.Fatal(err)
			}
			defer observer.close()
			if exited, err := observer.exited(); exited || err != nil {
				t.Fatalf("live non-child: exited=%v err=%v", exited, err)
			}
			if mode == "normal" {
				err = os.WriteFile(filepath.Join(dir, "exit"), nil, 0600)
			} else {
				err = process.Kill()
			}
			if err != nil {
				t.Fatal(err)
			}
			observerTestEventually(t, "owner reaping before observer polls", func() bool { _, err := os.Stat(filepath.Join(dir, "reaped")); return err == nil })
			observerTestEventually(t, "original non-child exit", func() bool {
				exited, err := observer.exited()
				if err != nil {
					t.Fatal(err)
				}
				return exited
			})
			t.Logf("native %s exit observed for non-child PID %d using %T; no Process.Wait by observer", mode, pid, observer)
		})
	}
}

func TestManagementObserverNativeExitedRace(t *testing.T) {
	for trial := 0; trial < 5; trial++ {
		child := stopTestChild(t)
		pid := child.command.Process.Pid
		if err := child.command.Process.Kill(); err != nil {
			t.Fatal(err)
		}
		waitStopTestChild(t, child)
		process, err := os.FindProcess(pid)
		if err != nil {
			if processAlreadyExited(pid, process, err) {
				continue
			}
			t.Fatal(err)
		}
		observer, err := newProcessExitObserver(process)
		if err != nil {
			if releaseErr := process.Release(); releaseErr != nil {
				t.Errorf("release exited process reference: %v", releaseErr)
			}
			t.Fatal(err)
		}
		exited, stateErr := observer.exited()
		closeErr, releaseErr := observer.close(), process.Release()
		if !exited || stateErr != nil || closeErr != nil || releaseErr != nil {
			t.Fatalf("exited race: %v %v %v %v", exited, stateErr, closeErr, releaseErr)
		}
	}
}
