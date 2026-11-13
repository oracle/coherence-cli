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
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type restartTestObserver struct {
	check   func() (bool, error)
	closed  *int
	err     error
	onClose func()
}

const (
	restartNewJMX           = "new-jmx"
	restartDisabledJMX      = "disabled-old-jmx"
	restartDelayedAvailable = "delayed-available"
	restartCancelRelease    = "cancel-release"
	restartProcessDeadline  = "processes"
	restartProbeOverrun     = "probe-overrun"
	restartProbeClose       = "probe-close"

	restartChangedREST        = "changed-rest"
	restartCancel             = "cancel"
	restartEndpointPermission = "endpoint-permission"

	restartEndpointAvailable = "available"
	restartObserverAcquire   = "acquire"
	restartEndpointCancel    = "endpoint-cancel"
	restartEmptyMembers      = "empty"
	restartSavedPath         = "/old"
	restartNullSize          = "null"
	restartDifferentCluster  = "different-cluster"
)

func (o *restartTestObserver) exited() (bool, error) { return o.check() }
func (o *restartTestObserver) close() error {
	(*o.closed)++
	if o.onClose != nil {
		o.onClose()
	}
	return o.err
}

type restartCountingObserver struct {
	processExitObserver
	released *int
}

func (o *restartCountingObserver) close() error {
	(*o.released)++
	return o.processExitObserver.close()
}

func assertRestartReferenceReleased(t *testing.T, p *os.Process) {
	t.Helper()
	if runtime.GOOS == managementTestWindows {
		if err := p.WithHandle(func(uintptr) {}); err == nil {
			t.Error("original process handle not released")
		}
	} else if p.Pid != -1 {
		t.Errorf("original process reference not released: %d", p.Pid)
	}
}

func restartTestSeams(t *testing.T) {
	t.Helper()
	observe, budget, poll := observeProcessForRestart, managedShutdownTimeout, managedShutdownPollInterval
	t.Cleanup(func() {
		observeProcessForRestart, managedShutdownTimeout, managedShutdownPollInterval = observe, budget, poll
	})
	managedShutdownTimeout, managedShutdownPollInterval = 150*time.Millisecond, 5*time.Millisecond
}

func restartTestRescue(t *testing.T, children ...*managedProcess) {
	t.Helper()
	rescue := time.AfterFunc(managementFixtureRescueDelay(), func() {
		for _, child := range children {
			_ = child.command.Process.Kill()
		}
	})
	t.Cleanup(func() { rescue.Stop() })
}

func assertRestartState(t *testing.T, connection ClusterConnection, before []byte) {
	t.Helper()
	after, err := os.ReadFile(viper.ConfigFileUsed())
	if err != nil || !bytes.Equal(before, after) || !reflect.DeepEqual(connection, Config.Clusters[0]) || !reflect.DeepEqual(viper.Get(clusterKey), Config.Clusters) {
		t.Fatalf("barrier changed prior disk/Config/Viper state: %v", err)
	}
}

// Successful requests with one delayed live member are injected, not evidence
// of ignored native SIGKILL. Both members/sockets are real task-owned fixtures.
func TestManagementRestartBarrierDelayedExitHTTP(t *testing.T) {
	for _, mode := range []string{"loss", "malformed", restartEmptyMembers} {
		t.Run(mode, func(t *testing.T) {
			managedTestState(t)
			stopTestSeams(t)
			restartTestSeams(t)
			managementPortSeam(t)
			children := []*managedProcess{stopTestChild(t), stopTestChild(t)}
			restartTestRescue(t, children...)
			pids := []int{children[0].command.Process.Pid, children[1].command.Process.Pid}
			var stopped atomic.Bool
			var postStopRequests atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if stopped.Load() {
					postStopRequests.Add(1)
					if mode == "malformed" {
						fmt.Fprint(w, "{")
						return
					}
					fmt.Fprint(w, `{"items":[]}`)
					return
				}
				if strings.Contains(r.URL.Path, "members") {
					fmt.Fprintf(w, `{"items":[{"processName":"%d"},{"processName":"%d"}]}`, pids[0], pids[1])
				} else {
					fmt.Fprintf(w, `{"clusterName":%q,"clusterSize":2}`, managementFixtureName)
				}
			}))
			t.Cleanup(server.Close)
			busyJMX := managementPortListener(t)
			conn := ClusterConnection{Name: managementFixtureName, ClusterName: managementFixtureName, ManuallyCreated: true, ConnectionType: httpType, ConnectionURL: server.URL + "/old", ManagementPort: managementListenerPort(t, server.Listener), ClusterVersion: defaultCoherenceVersion, Management: &ClusterManagement{RESTAddress: localHost, JMXPort: managementListenerPort(t, busyJMX), JMXAddress: localHost, JMXHost: localHost}}
			Config.Clusters = []ClusterConnection{conn}
			viper.SetConfigFile(filepath.Join(t.TempDir(), "config.yaml"))
			before := saveManagementFixture(t, conn)
			marker := filepath.Join(t.TempDir(), cleanupLaunchFailure)
			t.Setenv("COHCTL_RESTART_MARKER", marker)
			fakeJava(t, javaFixture{Version: javaVersion17, Query: true, Marker: "COHCTL_RESTART_MARKER"})
			var attempts []int
			killProcessForStop = func(p *os.Process) error {
				attempts = append(attempts, p.Pid)
				stopped.Store(true)
				if mode == "loss" {
					server.Close()
				}
				if p.Pid == pids[0] {
					return p.Kill()
				}
				return nil
			}
			probes, writes, readiness := 0, 0, 0
			listenManagementEndpoint = func(network, address string) (net.Listener, error) {
				if stopped.Load() {
					probes++
				}
				return net.Listen(network, address)
			}
			writeManagedConfig = func() error { writes++; return nil }
			awaitManagedStartup = func(*cobra.Command, ClusterConnection, []*managedProcess) error { readiness++; return nil }
			automaticallyConfirm = true
			err := runClusterOperation(managementTestCommand(t), conn.Name, restartClusterCommand)
			if !errors.Is(err, context.DeadlineExceeded) || !strings.Contains(err.Error(), fmt.Sprint(pids[1])) || !strings.Contains(err.Error(), "partial or unknown") || !reflect.DeepEqual(attempts, pids) {
				t.Fatalf("err=%v attempts=%v", err, attempts)
			}
			if _, statErr := os.Stat(marker); !errors.Is(statErr, os.ErrNotExist) || probes != 0 || readiness != 0 || writes != 0 || postStopRequests.Load() != 0 {
				t.Fatalf("barrier bypassed: marker=%v probes=%d readiness=%d writes=%d requests=%d", statErr, probes, readiness, writes, postStopRequests.Load())
			}
			assertManagedChildAlive(t, children[1])
			assertRestartState(t, conn, before)
		})
	}
}

func TestManagementRestartBarrierEndpoints(t *testing.T) {
	for _, mode := range []string{"unchanged-rest", restartChangedREST, "unchanged-jmx", restartNewJMX, restartDisabledJMX, restartEndpointAvailable, restartDelayedAvailable} {
		t.Run(mode, func(t *testing.T) {
			managedTestState(t)
			stopTestSeams(t)
			restartTestSeams(t)
			child := stopTestChild(t)
			restartTestRescue(t, child)
			conn, endpoint := stopTestConnectionEndpoint(t, []int{child.command.Process.Pid}, nil)
			conn.ConnectionURL += "/old"
			jmx := managementPortListener(t)
			jmxPort := managementListenerPort(t, jmx)
			conn.Management = &ClusterManagement{RESTAddress: localHost, JMXPort: jmxPort, JMXAddress: localHost, JMXHost: localHost}
			flags := []string{}
			if mode == restartChangedREST {
				conn.Management.RESTAddress = wildcardIPv4
				flags = append(flags, "--management-address="+localHost)
			}
			if mode == restartNewJMX {
				conn.Management.JMXPort = 0
				flags = append(flags, fmt.Sprintf("--jmx-port=%d", jmxPort))
				_ = jmx.Close()
			}
			if mode == restartDisabledJMX {
				flags = append(flags, "--jmx-port=0")
			}
			if mode == restartEndpointAvailable {
				_ = jmx.Close()
			}
			before := saveManagementFixture(t, conn)
			fakeJava(t, javaFixture{Version: javaVersion17, Query: true, Sleep: true})
			released, captures := 0, 0
			var retained []*os.Process
			observeProcessForRestart = func(p *os.Process) (processExitObserver, error) {
				observer, err := newProcessExitObserver(p)
				if err != nil {
					return nil, err
				}
				captures++
				retained = append(retained, p)
				return &restartCountingObserver{processExitObserver: observer, released: &released}, nil
			}
			attempts := 0
			killProcessForStop = func(p *os.Process) error {
				attempts++
				if mode == restartDelayedAvailable {
					killTimer := time.AfterFunc(40*time.Millisecond, func() { _ = child.command.Process.Kill() })
					portTimer := time.AfterFunc(80*time.Millisecond, func() { endpoint.Close(); _ = jmx.Close() })
					t.Cleanup(func() { killTimer.Stop(); portTimer.Stop() })
					return nil
				}
				err := p.Kill()
				if mode != "unchanged-rest" && mode != restartChangedREST {
					endpoint.Close()
				}
				if mode == restartNewJMX {
					var bindErr error
					jmx, bindErr = net.Listen("tcp4", net.JoinHostPort(localHost, fmt.Sprint(jmxPort)))
					if bindErr != nil {
						return errors.Join(err, bindErr)
					}
					t.Cleanup(func() { _ = jmx.Close() })
				}
				return err
			}
			readiness := 0
			var replacement []*managedProcess
			awaitManagedStartup = func(_ *cobra.Command, proposed ClusterConnection, ps []*managedProcess) error {
				readiness++
				replacement = ps
				select {
				case <-child.done:
				default:
					t.Error("replacement before old completion")
				}
				if proposed.ConnectionURL == conn.ConnectionURL || proposed.ConnectionURL != managementURL(*proposed.Management, proposed.ManagementPort) {
					t.Error("old/proposed URL conflated")
				}
				// Probes must already be closed when launch/readiness is reached.
				selected, resolveErr := resolveManagement(managementTestCommand(t, flags...), conn)
				if resolveErr != nil {
					t.Fatal(resolveErr)
				}
				if err := checkManagementPorts(proposed, selected, false); err != nil {
					t.Error(err)
				}
				return nil
			}
			t.Cleanup(func() { _ = cleanupManagedProcesses(replacement) })
			automaticallyConfirm = true
			if mode == restartDelayedAvailable {
				managedShutdownTimeout = 400 * time.Millisecond
			}
			err := runClusterOperation(managementTestCommand(t, flags...), conn.Name, restartClusterCommand)
			if captures != 1 || released != 1 {
				t.Fatalf("captures=%d released=%d", captures, released)
			}
			for _, p := range retained {
				assertRestartReferenceReleased(t, p)
			}
			if attempts != 1 {
				t.Fatalf("termination attempts=%d", attempts)
			}
			success := mode == restartEndpointAvailable || mode == restartDisabledJMX || mode == restartDelayedAvailable
			if success {
				if err != nil || readiness != 1 || Config.Clusters[0].ConnectionURL == conn.ConnectionURL {
					t.Fatalf("err=%v readiness=%d", err, readiness)
				}
				assertManagedChildAlive(t, replacement[0])
			} else {
				label := "JMX endpoint"
				if strings.Contains(mode, "rest") {
					label = "Management REST endpoint"
				}
				if !errors.Is(err, context.DeadlineExceeded) || !strings.Contains(err.Error(), label) || readiness != 0 {
					t.Fatalf("err=%v readiness=%d", err, readiness)
				}
				assertRestartState(t, conn, before)
			}
		})
	}
}

func TestManagementRestartBarrierObserverFailures(t *testing.T) {
	for _, mode := range []string{restartObserverAcquire, "state", "release", managementTestKill, restartCancel, restartCancelRelease} {
		t.Run(mode, func(t *testing.T) {
			stopTestSeams(t)
			restartTestSeams(t)
			children := []*managedProcess{stopTestChild(t), stopTestChild(t)}
			restartTestRescue(t, children...)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			cmd := managementTestCommand(t)
			cmd.SetContext(ctx)
			closed, acquired, attempts := 0, 0, 0
			var retained []*os.Process
			observeProcessForRestart = func(p *os.Process) (processExitObserver, error) {
				retained = append(retained, p)
				acquired++
				if mode == restartObserverAcquire && acquired == 1 {
					return nil, os.ErrPermission
				}
				o := &restartTestObserver{closed: &closed, check: func() (bool, error) {
					if mode == "state" {
						return false, os.ErrPermission
					}
					return true, nil
				}}
				if mode == restartCancelRelease {
					o.onClose = cancel
				}
				if mode == "release" {
					o.err = os.ErrPermission
				}
				return o, nil
			}
			killProcessForStop = func(p *os.Process) error {
				attempts++
				if acquired != attempts {
					t.Error("termination preceded original observer acquisition")
				}
				if mode == restartCancel {
					cancel()
				}
				if mode == managementTestKill && attempts == 1 {
					return os.ErrPermission
				}
				return p.Kill()
			}
			listener := managementPortListener(t)
			port := managementListenerPort(t, listener)
			_ = listener.Close()
			conn := ClusterConnection{Name: "observer-errors", ManagementPort: port}
			err := stopClusterForRestart(cmd, conn, ClusterManagement{RESTAddress: localHost}, []int{children[0].command.Process.Pid, children[1].command.Process.Pid})
			want := os.ErrPermission
			if mode == restartCancel || mode == restartCancelRelease {
				want = context.Canceled
			}
			wantClosed := 2
			if mode == restartObserverAcquire {
				wantClosed = 1
			}
			if !errors.Is(err, want) || attempts != 2 || closed != wantClosed {
				t.Fatalf("err=%v attempts=%d closed=%d", err, attempts, closed)
			}
			for _, p := range retained {
				assertRestartReferenceReleased(t, p)
			}
		})
	}
}

func TestManagementRestartBarrierSharedDeadline(t *testing.T) {
	for _, mode := range []string{restartProcessDeadline, "endpoint", restartProbeOverrun, restartEndpointPermission, restartEndpointCancel, restartProbeClose} {
		t.Run(mode, func(t *testing.T) {
			stopTestSeams(t)
			restartTestSeams(t)
			managementPortSeam(t)
			children := []*managedProcess{stopTestChild(t), stopTestChild(t), stopTestChild(t)}
			restartTestRescue(t, children...)
			budget := 120 * time.Millisecond
			managedShutdownTimeout = budget
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			cmd := managementTestCommand(t)
			cmd.SetContext(ctx)
			start := time.Now()
			closed, probes, calls := 0, 0, 0
			observeProcessForRestart = func(*os.Process) (processExitObserver, error) {
				calls++
				if calls == 1 {
					time.Sleep(25 * time.Millisecond)
				} // collection consumes same budget
				return &restartTestObserver{closed: &closed, check: func() (bool, error) {
					return mode != restartProcessDeadline && time.Since(start) > 60*time.Millisecond, nil
				}}, nil
			}
			killProcessForStop = func(*os.Process) error { return nil } // explicitly modeled delayed exits
			listenManagementEndpoint = func(network, address string) (net.Listener, error) {
				probes++
				if mode == restartEndpointCancel {
					cancel()
					return nil, os.NewSyscallError("bind", errManagementAddressInUse)
				}
				if mode == restartProbeClose {
					listener, err := net.Listen(network, address)
					if err != nil {
						return nil, err
					}
					return restartCloseErrorListener{listener}, nil
				}
				if mode == restartEndpointPermission {
					return nil, os.ErrPermission
				}
				if mode == restartProbeOverrun {
					time.Sleep(budget)
					return net.Listen(network, address)
				}
				return nil, os.NewSyscallError("bind", errManagementAddressInUse)
			}
			err := stopClusterForRestart(cmd, ClusterConnection{Name: "deadline", ManagementPort: 30000}, ClusterManagement{RESTAddress: localHost}, []int{children[0].command.Process.Pid, children[1].command.Process.Pid, children[2].command.Process.Pid})
			want := context.DeadlineExceeded
			if mode == restartEndpointPermission || mode == restartProbeClose {
				want = os.ErrPermission
			}
			if mode == restartEndpointCancel {
				want = context.Canceled
			}
			if !errors.Is(err, want) || time.Since(start) > 300*time.Millisecond || closed != 3 {
				t.Fatalf("err=%v elapsed=%s closed=%d", err, time.Since(start), closed)
			}
			if mode == restartProcessDeadline && probes != 0 || mode == restartProbeOverrun && probes != 1 {
				t.Fatalf("probes=%d", probes)
			}
		})
	}
}

type restartCloseErrorListener struct{ net.Listener }

func (l restartCloseErrorListener) Close() error { _ = l.Listener.Close(); return os.ErrPermission }

// Discovery failure used to look like a stopped cluster and bypass the exit
// barrier. Restart now rejects the whole snapshot before any stop or launch.
func TestManagementRestartRequiresOldSnapshot(t *testing.T) {
	for _, tc := range []struct {
		name, body  string
		status      int
		unreachable bool
	}{
		{name: "unavailable", status: 503},
		{name: "unreachable", unreachable: true},
		{name: "malformed-snapshot", body: `{"items":[`},
		{name: "missing-items", body: `{}`},
		{name: "null-response", body: `null`},
		{name: restartEmptyMembers, body: `{"items":[]}`},
		{name: "missing-pid", body: `{"items":[{}]}`},
		{name: "invalid-pid", body: `{"items":[{"processName":"not-a-pid"}]}`},
		{name: "zero-pid", body: `{"items":[{"processName":"0"}]}`},
		{name: "negative-pid", body: `{"items":[{"processName":"-1"}]}`},
		{name: "mixed-pids", body: `{"items":[{"processName":"12345"},{"processName":"bad"}]}`},
		{name: "duplicate-pid", body: `{"items":[{"processName":"12345"},{"processName":"12345"}]}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			managedTestState(t)
			stopTestSeams(t)
			restartTestSeams(t)
			old := stopTestChild(t)
			connection := stopTestConnection(t, []int{old.command.Process.Pid}, nil)
			queries := 0
			endpoint := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				queries++
				if !strings.HasSuffix(strings.TrimRight(r.URL.Path, "/"), "/members") {
					t.Error("queried cluster identity after failed snapshot")
				}
				if tc.status != 0 {
					w.WriteHeader(tc.status)
				}
				fmt.Fprint(w, tc.body)
			}))
			defer endpoint.Close()
			connection.ConnectionURL = endpoint.URL
			if tc.unreachable {
				endpoint.Close()
			}
			before := saveManagementFixture(t, connection)
			marker := filepath.Join(t.TempDir(), "launch")
			t.Setenv(windowsEndpointLaunchKey, marker)
			fakeJava(t, javaFixture{Version: javaVersion17, Query: true, Marker: windowsEndpointLaunchKey})
			stopCalls, writes, observers, readiness := 0, 0, 0, 0
			killProcessForStop = func(*os.Process) error { stopCalls++; return nil }
			observeProcessForRestart = func(*os.Process) (processExitObserver, error) { observers++; return exitedProcessObserver{}, nil }
			writeManagedConfig = func() error { writes++; return nil }
			awaitManagedStartup = func(*cobra.Command, ClusterConnection, []*managedProcess) error { readiness++; return nil }
			automaticallyConfirm = true
			err := runClusterOperation(managementTestCommand(t), connection.Name, restartClusterCommand)
			if err == nil || !strings.Contains(err.Error(), "old-member snapshot") || !strings.Contains(err.Error(), "use start cluster") {
				t.Fatal(err)
			}
			if stopCalls+writes+observers+readiness != 0 {
				t.Fatalf("side effects: stops=%d writes=%d observers=%d readiness=%d", stopCalls, writes, observers, readiness)
			}
			if !tc.unreachable && queries != 1 {
				t.Fatalf("snapshot queried %d times", queries)
			}
			if _, err := os.Stat(marker); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("replacement launched: %v", err)
			}
			assertManagedChildAlive(t, old)
			assertRestartState(t, connection, before)
		})
	}
}

// A nonempty list of valid PIDs can still omit an original member. Both old
// processes stay alive here; the stop seam only records an attempted stop.
func TestManagementRestartRequiresConsistentSnapshot(t *testing.T) {
	for _, tc := range []struct {
		name, size string
		members    int
	}{
		{"incomplete-members", "2", 1},
		{"excess-members", "1", 2},
		{"missing-size", "", 2},
		{"null-size", restartNullSize, 2},
		{"zero-size", "0", 2},
		{"negative-size", "-1", 2},
		{"string-size", `"2"`, 2},
		{"fractional-size", "2.5", 2},
		{"decimal-size", "2.0", 2},
		{"boolean-size", stringTrue, 2},
		{"array-size", "[2]", 2},
		{"object-size", "{}", 2},
		{"overflow-size", "9223372036854775808", 2},
		{restartDifferentCluster, "2", 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			managedTestState(t)
			stopTestSeams(t)
			restartTestSeams(t)
			children := []*managedProcess{stopTestChild(t), stopTestChild(t)}
			var membersQueries, detailsQueries atomic.Int32
			endpoint := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if strings.TrimRight(r.URL.Path, "/") == restartSavedPath+"/members" {
					membersQueries.Add(1)
					items := make([]string, tc.members)
					for i := range items {
						items[i] = fmt.Sprintf(`{"processName":"%d"}`, children[i].command.Process.Pid)
					}
					fmt.Fprintf(w, `{"items":[%s]}`, strings.Join(items, ","))
				} else if strings.TrimRight(r.URL.Path, "/") == restartSavedPath {
					detailsQueries.Add(1)
					size := ""
					if tc.size != "" {
						size = `,"clusterSize":` + tc.size
					}
					name := managementFixtureName
					if tc.name == restartDifferentCluster {
						name = "other-cluster"
					}
					fmt.Fprintf(w, `{"clusterName":%q%s}`, name, size)
				} else {
					t.Errorf("inspection used unexpected URL %s", r.URL)
					w.WriteHeader(http.StatusNotFound)
				}
			}))
			defer endpoint.Close()
			connection := ClusterConnection{Name: managementFixtureName, ClusterName: managementFixtureName, ManuallyCreated: true,
				ConnectionType: httpType, ConnectionURL: endpoint.URL + restartSavedPath, ManagementPort: managementListenerPort(t, endpoint.Listener),
				ClusterVersion: defaultCoherenceVersion, Management: &ClusterManagement{RESTAddress: localHost}}
			Config.Clusters = []ClusterConnection{connection}
			viper.SetConfigFile(filepath.Join(t.TempDir(), "cohctl.yaml"))
			before := saveManagementFixture(t, connection)
			marker := filepath.Join(t.TempDir(), "launch")
			t.Setenv(windowsEndpointLaunchKey, marker)
			fakeJava(t, javaFixture{Version: javaVersion17, Query: true, Marker: windowsEndpointLaunchKey})
			stopCalls, observers, readiness, writes := 0, 0, 0, 0
			killProcessForStop = func(*os.Process) error { stopCalls++; return errors.New("intercepted stop") }
			observeProcessForRestart = func(p *os.Process) (processExitObserver, error) { observers++; return newProcessExitObserver(p) }
			awaitManagedStartup = func(*cobra.Command, ClusterConnection, []*managedProcess) error { readiness++; return nil }
			writeManagedConfig = func() error { writes++; return nil }
			automaticallyConfirm = true
			err := runClusterOperation(managementTestCommand(t), connection.Name, restartClusterCommand)
			want := "clusterSize"
			if tc.name == restartDifferentCluster {
				want = "different cluster"
			}
			if err == nil || !strings.Contains(err.Error(), want) {
				t.Errorf("unusable/inconsistent clusterSize accepted: %v", err)
			}
			if stopCalls+observers+readiness+writes != 0 {
				t.Errorf("side effects: stops=%d observers=%d readiness=%d writes=%d", stopCalls, observers, readiness, writes)
			}
			if membersQueries.Load() != 1 || detailsQueries.Load() != 1 {
				t.Errorf("old endpoint queries: members=%d details=%d", membersQueries.Load(), detailsQueries.Load())
			}
			if _, err := os.Stat(marker); !errors.Is(err, os.ErrNotExist) {
				t.Errorf("replacement launched: %v", err)
			}
			for _, child := range children {
				assertManagedChildAlive(t, child)
			}
			assertRestartState(t, connection, before)
		})
	}
}

func TestManagementRestartCompleteSnapshot(t *testing.T) {
	managedTestState(t)
	stopTestSeams(t)
	restartTestSeams(t)
	managedShutdownTimeout = 3 * time.Second
	children := []*managedProcess{stopTestChild(t), stopTestChild(t)}
	pids := []int{children[0].command.Process.Pid, children[1].command.Process.Pid}
	connection, endpoint := stopTestConnectionEndpoint(t, pids, nil)
	connection.ConnectionURL += restartSavedPath
	before := saveManagementFixture(t, connection)
	replicaCountParam = 2
	fakeJava(t, javaFixture{Version: javaVersion17, Query: true, Sleep: true})
	acquired, released, readiness, writes := 0, 0, 0, 0
	var retained []*os.Process
	observeProcessForRestart = func(p *os.Process) (processExitObserver, error) {
		acquired++
		retained = append(retained, p)
		observer, err := newProcessExitObserver(p)
		if err != nil {
			return nil, err
		}
		return &restartCountingObserver{processExitObserver: observer, released: &released}, nil
	}
	var attempts []int
	killProcessForStop = func(p *os.Process) error {
		attempts = append(attempts, p.Pid)
		err := p.Kill()
		if len(attempts) == len(pids) {
			endpoint.Close()
		}
		return err
	}
	var replacement []*managedProcess
	t.Cleanup(func() {
		if err := cleanupManagedProcesses(replacement); err != nil {
			t.Errorf("replacement fixture cleanup: %v", err)
		}
	})
	awaitManagedStartup = func(_ *cobra.Command, proposed ClusterConnection, processes []*managedProcess) error {
		readiness++
		replacement = processes
		if acquired != 2 || released != 2 || len(processes) != 2 {
			t.Errorf("acquired=%d released=%d replacements=%d", acquired, released, len(processes))
		}
		for _, child := range children {
			select {
			case <-child.done:
			default:
				t.Error("replacement preceded original process completion")
			}
		}
		for _, p := range retained {
			assertRestartReferenceReleased(t, p)
		}
		if proposed.ConnectionURL == connection.ConnectionURL || proposed.ConnectionURL != managementURL(*proposed.Management, proposed.ManagementPort) {
			t.Error("readiness did not use proposed URL")
		}
		return nil // Native children/barrier; simulated Coherence readiness.
	}
	writeManagedConfig = func() error { writes++; return writeManagedConfigAtomically() }
	automaticallyConfirm = true
	if err := runClusterOperation(managementTestCommand(t), connection.Name, restartClusterCommand); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(attempts, pids) || readiness != 1 || writes != 1 {
		t.Fatalf("stops=%v want=%v readiness=%d writes=%d", attempts, pids, readiness, writes)
	}
	for _, child := range replacement {
		assertManagedChildAlive(t, child)
	}
	after, err := os.ReadFile(viper.ConfigFileUsed())
	if err != nil || bytes.Equal(before, after) || Config.Clusters[0].ConnectionURL == connection.ConnectionURL {
		t.Fatalf("completed replacement did not save proposed settings: %v", err)
	}
}
