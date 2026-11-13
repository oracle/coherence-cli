/*
 * Copyright (c) 2026 Oracle and/or its affiliates.
 * Licensed under the Universal Permissive License v 1.0 as shown at
 * https://oss.oracle.com/licenses/upl.
 */

package cmd

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	cleanupLaunchFailure = "launch"

	cleanupPending      = "pending"
	cleanupNormalExit   = "normal-exit"
	cleanupNonzeroExit  = "nonzero-exit"
	cleanupAlreadyDone  = "already-done"
	cleanupCancellation = "cancellation"
)

// Independent fixture backstops must allow Windows runtime preflight to finish.
// Product cleanup/shutdown budgets remain unchanged.
func managementFixtureRescueDelay() time.Duration {
	if runtime.GOOS == managementTestWindows {
		return 10 * time.Second
	}
	return 2 * time.Second
}

// Self-limiting real child; the independent rescue precedes its natural exit.
func TestManagementCleanupChild(_ *testing.T) {
	if os.Getenv("COHCTL_CLEANUP_TEST_CHILD") == "1" {
		switch os.Getenv("COHCTL_CLEANUP_TEST_CHILD_EXIT") {
		case cleanupNormalExit:
			return
		case cleanupNonzeroExit:
			os.Exit(7)
		}
		lifetime := 8 * time.Second
		if runtime.GOOS == managementTestWindows {
			lifetime = 30 * time.Second
		}
		time.Sleep(lifetime)
	}
}

func cleanupTestSeams(t *testing.T) {
	t.Helper()
	kill, timeout := killManagedProcess, managedCleanupTimeout
	t.Cleanup(func() { killManagedProcess, managedCleanupTimeout = kill, timeout })
	managedCleanupTimeout = 100 * time.Millisecond
}

// Independent native rescue never uses the cleanup/stop seams. Join a running
// timer callback before returning so fault fixtures leave no rescue goroutines.
func cleanupTestRescue(t *testing.T, process *os.Process) {
	t.Helper()
	finished := make(chan struct{})
	timer := time.AfterFunc(2*time.Second, func() { _ = process.Kill(); close(finished) })
	t.Cleanup(func() {
		if !timer.Stop() {
			<-finished
		}
		_ = process.Kill()
	})
}

func cleanupTestReap(t *testing.T, children []*managedProcess) {
	t.Helper()
	t.Cleanup(func() {
		for _, p := range children {
			_ = p.command.Process.Kill()
		}
		for _, p := range children {
			waitStopTestChild(t, p)
		}
	})
}

// This fixture has exactly one actual command.Wait, with the same file-backed
// streams as launchManagedProcess. holdDone delays only the notification after
// native Wait; waitFault models an unexpected Wait error after real reaping.
func cleanupTestChild(t *testing.T, holdDone bool, waitFault error) (*managedProcess, <-chan struct{}, func()) {
	t.Helper()
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	logFile := filepath.Join(t.TempDir(), "child.log")
	f, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	command := exec.Command(executable, "-test.run=^TestManagementCleanupChild$")
	command.Env = append(os.Environ(), "COHCTL_CLEANUP_TEST_CHILD=1")
	command.Stdout, command.Stderr = f, f
	setForkProcess(command)
	if err = command.Start(); err != nil {
		t.Fatal(err)
	}
	p := &managedProcess{command: command, done: make(chan struct{}), logFile: logFile}
	waited, gate := make(chan struct{}), make(chan struct{})
	var once sync.Once
	release := func() { once.Do(func() { close(gate) }) }
	if !holdDone {
		release()
	}
	go func() {
		p.waitErr = command.Wait()
		if waitFault != nil {
			p.waitErr = waitFault
		}
		close(waited)
		<-gate
		close(p.done)
	}()
	// Also release a withheld notification independently if an assertion fails.
	finished := make(chan struct{})
	rescue := time.AfterFunc(managementFixtureRescueDelay(), func() { _ = command.Process.Kill(); release(); close(finished) })
	t.Cleanup(func() {
		if !rescue.Stop() {
			<-finished
		}
		release()
		_ = command.Process.Kill()
		waitStopTestChild(t, p)
	})
	return p, waited, release
}

func cleanupTestJava(t *testing.T) {
	t.Helper()
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("COHCTL_CLEANUP_EXE", executable)
	t.Setenv("COHCTL_CLEANUP_TEST_CHILD", "1")
	fakeJava(t, javaFixture{Version: javaVersion17, Query: true, Cleanup: true})
}

func assertCleanupDiagnostic(t *testing.T, err error, p *managedProcess, action string) {
	t.Helper()
	for _, text := range []string{fmt.Sprintf("%s owned child PID %d", action, p.command.Process.Pid), fmt.Sprintf("%q", p.logFile)} {
		if err == nil || !strings.Contains(err.Error(), text) {
			t.Fatalf("missing %q in cleanup error: %v", text, err)
		}
	}
}

func TestManagementCleanupAttemptsAll(t *testing.T) {
	for _, failed := range []int{0, 1} {
		t.Run(fmt.Sprintf("failed-%d", failed), func(t *testing.T) {
			cleanupTestSeams(t)
			stopTestSeams(t)
			killProcessForStop = func(_ *os.Process) error { t.Fatal("cleanup used stop seam"); return nil }
			var children []*managedProcess
			for i := 0; i < 3; i++ {
				p, _, _ := cleanupTestChild(t, false, nil)
				children = append(children, p)
			}
			control, _, _ := cleanupTestChild(t, false, nil)
			var attempts []*os.Process
			killManagedProcess = func(p *os.Process) error {
				attempts = append(attempts, p)
				if p == children[failed].command.Process {
					return os.ErrPermission // injected; child stays alive until native rescue
				}
				return p.Kill()
			}
			start := time.Now()
			err := cleanupManagedProcesses(children)
			if elapsed := time.Since(start); elapsed > 500*time.Millisecond {
				t.Fatalf("cleanup waited before later attempts: %s", elapsed)
			}
			if !errors.Is(err, os.ErrPermission) || !errors.Is(err, context.DeadlineExceeded) || len(attempts) != len(children) {
				t.Fatalf("err=%v attempts=%d", err, len(attempts))
			}
			for i, p := range children {
				if attempts[i] != p.command.Process {
					t.Fatal("cleanup did not use each original owned handle exactly once")
				}
				if i == failed {
					assertCleanupDiagnostic(t, err, p, managementTestKill)
					assertCleanupDiagnostic(t, err, p, "wait for")
					assertManagedChildAlive(t, p)
				} else {
					waitStopTestChild(t, p)
				}
			}
			assertManagedChildAlive(t, control)
		})
	}
}

func TestManagementCleanupSharedDeadline(t *testing.T) {
	cleanupTestSeams(t)
	unknown := errors.New("injected unknown kill state")
	var children []*managedProcess
	for i := 0; i < 6; i++ {
		p, _, _ := cleanupTestChild(t, true, nil)
		children = append(children, p)
	}
	attempts := 0
	killManagedProcess = func(p *os.Process) error {
		attempts++
		if p == children[0].command.Process {
			return os.ErrPermission
		}
		if p == children[2].command.Process {
			return unknown
		}
		return p.Kill()
	}
	start := time.Now()
	err := cleanupManagedProcesses(children)
	elapsed := time.Since(start)
	if elapsed < managedCleanupTimeout/2 || elapsed > 400*time.Millisecond {
		t.Fatalf("collection did not share one %s deadline: %s", managedCleanupTimeout, elapsed)
	}
	if attempts != len(children) || !errors.Is(err, os.ErrPermission) || !errors.Is(err, unknown) || !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("attempts=%d err=%v", attempts, err)
	}
	for _, p := range children {
		assertCleanupDiagnostic(t, err, p, "wait for")
	}
	assertCleanupDiagnostic(t, err, children[0], managementTestKill)
	assertCleanupDiagnostic(t, err, children[2], managementTestKill)
	if !strings.Contains(err.Error(), "0 of 6 children completed") {
		t.Fatal(err)
	}
	t.Logf("MODEL: six unresolved notifications, two injected kill failures; shared budget=%s elapsed=%s", managedCleanupTimeout, elapsed)
}

func TestManagementCleanupDelayedCompletion(t *testing.T) {
	cleanupTestSeams(t)
	managedCleanupTimeout = time.Second
	first, waited, release := cleanupTestChild(t, true, nil)
	second, _, _ := cleanupTestChild(t, false, nil)
	third, _, _ := cleanupTestChild(t, false, nil)
	children := []*managedProcess{first, second, third}
	attempts := make(chan *os.Process, len(children))
	killManagedProcess = func(p *os.Process) error { attempts <- p; return p.Kill() }
	returned := make(chan error, 1)
	go func() { returned <- cleanupManagedProcesses(children) }()
	// Every path joins this bounded call, even if an assertion fails.
	var once sync.Once
	join := func() {
		once.Do(func() {
			release()
			select {
			case err := <-returned:
				if err != nil {
					t.Error(err)
				}
			case <-time.After(2 * time.Second):
				t.Error("bounded cleanup call did not return")
			}
		})
	}
	t.Cleanup(join)
	for _, p := range children {
		select {
		case attempted := <-attempts:
			if attempted != p.command.Process {
				t.Fatal("unexpected cleanup handle")
			}
		case <-time.After(500 * time.Millisecond):
			t.Fatal("delayed notification prevented later termination request")
		}
	}
	select {
	case <-waited: // real command.Wait has finished; only notification is held
	case <-time.After(500 * time.Millisecond):
		t.Fatal("native Wait did not finish")
	}
	select {
	case <-first.done:
		t.Fatal("fixture did not withhold completion")
	default:
	}
	join() // notification arrives within the shared deadline: successful cleanup
	for _, p := range children {
		waitStopTestChild(t, p)
	}
}

func TestManagementCleanupNativeCompletion(t *testing.T) {
	cleanupTestSeams(t)
	managedCleanupTimeout = time.Second
	if err := cleanupManagedProcesses(nil); err != nil {
		t.Fatal(err)
	}
	for _, mode := range []string{"live", cleanupAlreadyDone, "exit-during-kill", cleanupNormalExit, cleanupNonzeroExit} {
		t.Run(mode, func(t *testing.T) {
			t.Setenv("COHCTL_CLEANUP_TEST_CHILD_EXIT", mode)
			p, _, _ := cleanupTestChild(t, false, nil)
			if mode == cleanupAlreadyDone {
				if err := p.command.Process.Kill(); err != nil {
					t.Fatal(err)
				}
				waitStopTestChild(t, p)
			}
			if mode == cleanupNormalExit || mode == cleanupNonzeroExit {
				waitStopTestChild(t, p)
			}
			attempts := 0
			killManagedProcess = func(process *os.Process) error {
				attempts++
				if mode == "exit-during-kill" {
					if err := process.Kill(); err != nil {
						t.Fatal(err)
					}
					waitStopTestChild(t, p)
				}
				return process.Kill()
			}
			if err := cleanupManagedProcesses([]*managedProcess{p}); err != nil {
				t.Fatal(err)
			}
			want := 1
			if mode == cleanupAlreadyDone || mode == cleanupNormalExit || mode == cleanupNonzeroExit {
				want = 0
			}
			if attempts != want {
				t.Fatalf("attempts=%d want=%d", attempts, want)
			}
			var exitErr *exec.ExitError
			if mode == cleanupNormalExit && p.waitErr != nil || mode != cleanupNormalExit && !errors.As(p.waitErr, &exitErr) || p.command.ProcessState == nil {
				t.Fatalf("native Kill did not exercise expected nonzero/signal Wait: %v", p.waitErr)
			}
			t.Logf("NATIVE %s: Wait=%v cleanup successful", mode, p.waitErr)
		})
	}
}

func TestManagementCleanupRetainsWaitError(t *testing.T) {
	cleanupTestSeams(t)
	managedCleanupTimeout = time.Second
	fault := errors.New("injected unknown Wait state after actual reaping")
	p, _, _ := cleanupTestChild(t, false, fault)
	err := cleanupManagedProcesses([]*managedProcess{p})
	if !errors.Is(err, fault) || !strings.Contains(err.Error(), "exit unresolved") || !strings.Contains(err.Error(), "0 of 1 children completed") {
		t.Fatal(err)
	}
	assertCleanupDiagnostic(t, err, p, "wait for")
}

func TestManagementCleanupRetainsKillErrorAfterCompletion(t *testing.T) {
	cleanupTestSeams(t)
	managedCleanupTimeout = time.Second
	p, _, _ := cleanupTestChild(t, false, nil)
	killManagedProcess = func(process *os.Process) error {
		if err := process.Kill(); err != nil {
			t.Fatal(err)
		}
		// Model a reported permission failure even though native termination
		// succeeds. A later completion must not erase the genuine diagnostic.
		return os.ErrPermission
	}
	err := cleanupManagedProcesses([]*managedProcess{p})
	if !errors.Is(err, os.ErrPermission) || errors.Is(err, context.DeadlineExceeded) || !strings.Contains(err.Error(), "1 of 1 children completed") {
		t.Fatal(err)
	}
	assertCleanupDiagnostic(t, err, p, managementTestKill)
}

func TestManagementCleanupOperationFailure(t *testing.T) {
	for _, operation := range []string{restartClusterCommand, scaleClusterCommand} {
		t.Run(operation, func(t *testing.T) {
			managedPersistenceState(t)
			cleanupTestSeams(t)
			stopTestSeams(t)
			cleanupTestJava(t)
			existing, _, _ := cleanupTestChild(t, false, nil)
			var stopped atomic.Bool
			connection, endpoint := stopTestConnectionEndpoint(t, []int{existing.command.Process.Pid}, &stopped)
			before, _ := os.ReadFile(viper.ConfigFileUsed())
			oldConfig, oldSettings := Config, viper.AllSettings()
			killProcessForStop = func(p *os.Process) error {
				if p.Pid != existing.command.Process.Pid {
					t.Fatal("stop targeted nonfixture")
				}
				stopped.Store(true)
				endpoint.Close()
				return p.Kill()
			}
			automaticallyConfirm, replicaCountParam = true, 2
			cause := errors.New("injected replacement/scale readiness failure")
			var children []*managedProcess
			awaitManagedStartup = func(_ *cobra.Command, _ ClusterConnection, p []*managedProcess) error { children = p; return cause }
			killManagedProcess = func(p *os.Process) error { cleanupTestRescue(t, p); return os.ErrPermission }
			err := runClusterOperation(managementTestCommand(t), connection.Name, operation)
			cleanupTestReap(t, children)
			if !errors.Is(err, cause) || !errors.Is(err, os.ErrPermission) || !errors.Is(err, context.DeadlineExceeded) || len(children) == 0 {
				t.Fatal(err)
			}
			for _, p := range children {
				assertCleanupDiagnostic(t, err, p, "wait for")
			}
			if operation == restartClusterCommand {
				waitStopTestChild(t, existing)
				if strings.Contains(err.Error(), "cluster is stopped") || !strings.Contains(err.Error(), "cannot be restored automatically") {
					t.Fatal("misleading restart cleanup diagnostic", err)
				}
			} else {
				assertManagedChildAlive(t, existing)
				if stopped.Load() {
					t.Fatal("failed scale stopped existing member")
				}
			}
			after, _ := os.ReadFile(viper.ConfigFileUsed())
			if !bytes.Equal(before, after) || !reflect.DeepEqual(Config, oldConfig) || !reflect.DeepEqual(viper.AllSettings(), oldSettings) {
				t.Fatal("failed operation changed saved state")
			}
		})
	}
}

// The exact Windows EINVAL race is benign only with positive owned Wait proof.
// Unknown Wait results and other kill errors remain diagnostic after reaping.
func TestManagementCleanupReleasedHandleProof(t *testing.T) {
	for _, mode := range []string{"completed", "unknown-wait", cleanupPending, "permission", "invalid-file", "unknown-kill"} {
		t.Run(mode, func(t *testing.T) {
			cleanupTestSeams(t)
			var waitFault error
			if mode == "unknown-wait" {
				waitFault = errors.New("unknown owned Wait result")
			}
			p, waited, _ := cleanupTestChild(t, mode == cleanupPending, waitFault)
			killErr := error(syscall.EINVAL)
			switch mode {
			case "permission":
				killErr = os.ErrPermission
			case "invalid-file":
				killErr = os.ErrInvalid
			case "unknown-kill":
				killErr = errors.New("unknown kill failure")
			}
			attempts := 0
			killManagedProcess = func(process *os.Process) error {
				attempts++
				if process != p.command.Process {
					t.Fatal("cleanup replaced the owned process reference")
				}
				if err := process.Kill(); err != nil {
					t.Fatal(err)
				}
				<-waited // The fixture owns the only native Wait.
				return fmt.Errorf("owned kill: %w", killErr)
			}
			err := cleanupManagedProcesses([]*managedProcess{p})
			if attempts != 1 {
				t.Fatalf("owned kill attempts=%d", attempts)
			}
			if mode == "completed" && runtime.GOOS == managementTestWindows {
				if err != nil {
					t.Fatal(err)
				}
				return
			}
			if !errors.Is(err, killErr) {
				t.Fatalf("lost kill failure: %v", err)
			}
			assertCleanupDiagnostic(t, err, p, managementTestKill)
			if waitFault != nil && !errors.Is(err, waitFault) || mode == cleanupPending && !errors.Is(err, context.DeadlineExceeded) {
				t.Fatalf("lost unresolved Wait evidence: %v", err)
			}
		})
	}
}
