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
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type managedCancelWriter struct {
	cancel context.CancelFunc
	text   string
}

func (w managedCancelWriter) Write(p []byte) (int, error) {
	if strings.Contains(string(p), w.text) {
		w.cancel()
	}
	return len(p), nil
}

type managedCancelReader struct {
	cancel context.CancelFunc
	reader *strings.Reader
}

func (r managedCancelReader) Read(p []byte) (int, error) { r.cancel(); return r.reader.Read(p) }

func assertManagedChildAlive(t *testing.T, p *managedProcess) {
	t.Helper()
	select {
	case <-p.done:
		t.Fatalf("pre-existing/accepted PID %d was stopped", p.command.Process.Pid)
	default:
	}
}

func TestManagementCancellationBeforeSideEffects(t *testing.T) {
	for _, operation := range []string{stopClusterCommand, restartClusterCommand, startClusterCommand, scaleClusterCommand} {
		t.Run(operation, func(t *testing.T) {
			managedTestState(t)
			child := stopTestChild(t)
			connection := stopTestConnection(t, []int{child.command.Process.Pid}, nil)
			before, _ := os.ReadFile(viper.ConfigFileUsed())
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			cmd := managementTestCommand(t)
			cmd.SetContext(ctx)
			if err := runClusterOperation(cmd, connection.Name, operation); !errors.Is(err, context.Canceled) {
				t.Fatal(err)
			}
			assertManagedChildAlive(t, child)
			after, _ := os.ReadFile(viper.ConfigFileUsed())
			if !bytes.Equal(before, after) || !reflect.DeepEqual(connection, Config.Clusters[0]) {
				t.Fatal("cancel changed config")
			}
		})
	}
}

func TestManagementCancellationAtStopBoundary(t *testing.T) {
	for _, operation := range []string{stopClusterCommand, restartClusterCommand} {
		t.Run(operation, func(t *testing.T) {
			managedTestState(t)
			if operation == restartClusterCommand {
				fakeJava(t, javaFixture{Version: javaVersion17})
			}
			child := stopTestChild(t)
			connection := stopTestConnection(t, []int{child.command.Process.Pid}, nil)
			before, _ := os.ReadFile(viper.ConfigFileUsed())
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			cmd := managementTestCommand(t)
			cmd.SetContext(ctx)
			cmd.SetIn(managedCancelReader{cancel, strings.NewReader("y\n")})
			if err := runClusterOperation(cmd, connection.Name, operation); !errors.Is(err, context.Canceled) {
				t.Fatal(err)
			}
			assertManagedChildAlive(t, child)
			if cmd.Context() != ctx {
				t.Fatal("caller context not restored")
			}
			after, _ := os.ReadFile(viper.ConfigFileUsed())
			if !bytes.Equal(before, after) || !reflect.DeepEqual(connection, Config.Clusters[0]) {
				t.Fatal("cancel at stop changed config")
			}
		})
	}
}

func TestManagementCancellationBeforeEachLaunch(t *testing.T) {
	for _, at := range []string{"before-first", "before-second"} {
		t.Run(at, func(t *testing.T) {
			managedTestState(t)
			old := stopTestChild(t)
			fakeJava(t, javaFixture{Sleep: true})
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			cmd := managementTestCommand(t)
			cmd.SetContext(ctx)
			if at == "before-first" {
				cancel()
			} else {
				cmd.SetOut(managedCancelWriter{cancel, "Starting cluster member storage-2"})
			}
			conn := ClusterConnection{Name: "scale-cancel", ManagementPort: 30000, ClusterVersion: defaultCoherenceVersion}
			children, err := runTestStartup(t, cmd, conn, 2, 1, ClusterManagement{RESTAddress: localHost})
			if !errors.Is(err, context.Canceled) {
				t.Fatal(err)
			}
			want := 0
			if at == "before-second" {
				want = 1
			}
			if len(children) != want {
				t.Fatalf("launched %d, want %d", len(children), want)
			}
			for _, p := range children {
				waitStopTestChild(t, p)
			}
			assertManagedChildAlive(t, old)
		})
	}
}

func TestManagementRestartCancellationFinishesStopPass(t *testing.T) {
	managedTestState(t)
	stopTestSeams(t)
	fakeJava(t, javaFixture{Version: javaVersion17})
	children := []*managedProcess{stopTestChild(t), stopTestChild(t), stopTestChild(t)}
	var pids []int
	for _, p := range children {
		pids = append(pids, p.command.Process.Pid)
	}
	conn := stopTestConnection(t, pids, nil)
	before, _ := os.ReadFile(viper.ConfigFileUsed())
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	attempts, writes, readiness := 0, 0, 0
	killProcessForStop = func(p *os.Process) error { attempts++; cancel(); return p.Kill() }
	writeManagedConfig = func() error { writes++; return nil }
	awaitManagedStartup = func(_ *cobra.Command, _ ClusterConnection, _ []*managedProcess) error { readiness++; return nil }
	automaticallyConfirm = true
	cmd := managementTestCommand(t)
	cmd.SetContext(ctx)
	err := runClusterOperation(cmd, conn.Name, restartClusterCommand)
	if !errors.Is(err, context.Canceled) || !strings.Contains(err.Error(), "cannot be restored automatically") {
		t.Fatal(err)
	}
	if attempts != 3 || writes != 0 || readiness != 0 {
		t.Fatalf("attempts=%d writes=%d readiness=%d", attempts, writes, readiness)
	}
	for _, p := range children {
		waitStopTestChild(t, p)
	}
	after, _ := os.ReadFile(viper.ConfigFileUsed())
	if !bytes.Equal(before, after) || !reflect.DeepEqual(conn, Config.Clusters[0]) {
		t.Fatal("cancel changed config")
	}
}

func TestManagementRestartCancellationDuringShutdownBarrier(t *testing.T) {
	managedTestState(t)
	stopTestSeams(t)
	restartTestSeams(t)
	fakeJava(t, javaFixture{Version: javaVersion17})
	child := stopTestChild(t)
	conn := stopTestConnection(t, []int{child.command.Process.Pid}, nil)
	before, _ := os.ReadFile(viper.ConfigFileUsed())
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	closed := 0
	var canceledAt time.Time
	observeProcessForRestart = func(*os.Process) (processExitObserver, error) {
		return &restartTestObserver{check: func() (bool, error) {
			if canceledAt.IsZero() {
				canceledAt = time.Now()
			}
			cancel()
			return false, nil
		}, closed: &closed}, nil
	}
	attempts, writes, readiness := 0, 0, 0
	killProcessForStop = func(*os.Process) error { attempts++; return nil } // modeled delayed exit
	writeManagedConfig = func() error { writes++; return nil }
	awaitManagedStartup = func(*cobra.Command, ClusterConnection, []*managedProcess) error { readiness++; return nil }
	automaticallyConfirm = true
	cmd := managementTestCommand(t)
	cmd.SetContext(ctx)
	started := time.Now()
	err := runClusterOperation(cmd, conn.Name, restartClusterCommand)
	// Measure prompt return from the actual injected cancellation. Native Java
	// preflight runs before that event and must not consume this assertion's budget.
	elapsed := time.Since(canceledAt)
	if canceledAt.IsZero() || !errors.Is(err, context.Canceled) || elapsed > time.Second || closed != 1 || attempts != 1 || writes != 0 || readiness != 0 {
		t.Fatalf("err=%v after-cancel=%s total=%s closed=%d attempts=%d writes=%d readiness=%d", err, elapsed, time.Since(started), closed, attempts, writes, readiness)
	}
	assertManagedChildAlive(t, child)
	after, _ := os.ReadFile(viper.ConfigFileUsed())
	if !bytes.Equal(before, after) || !reflect.DeepEqual(conn, Config.Clusters[0]) || !reflect.DeepEqual(viper.Get(clusterKey), Config.Clusters) {
		t.Fatal("cancel changed prior state")
	}
}
