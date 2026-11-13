// Copyright (c) 2026, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at
// https://oss.oracle.com/licenses/upl.

//go:build darwin || linux

package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"github.com/spf13/viper"
)

const (
	signalStartupDelay = "delay"
	signalConfirmation = "confirmation"
)

// Executed by the Java shim in the actual CLI's detached child. This is a
// process/REST fixture, not a JVM or a Coherence listener qualification test.
func TestManagementSignalExecutableChild(t *testing.T) {
	if os.Getenv("COHCTL_SIGNAL_CHILD") != "1" {
		return
	}
	dir := os.Getenv("COHCTL_SIGNAL_DIR")
	pid := os.Getpid()
	marker := filepath.Join(dir, fmt.Sprintf("child-%d", pid))
	if err := os.WriteFile(marker, []byte(strconv.Itoa(pid)), 0600); err != nil { // #nosec G703 -- Guarded subprocess fixture; dir comes only from the parent test private t.TempDir, never CLI input.
		t.Fatal(err)
	}
	port := "-1"
	for _, arg := range os.Args {
		if strings.HasPrefix(arg, "-Dcoherence.management.http.port=") {
			port = strings.TrimPrefix(arg, "-Dcoherence.management.http.port=")
		}
	}
	if port == "-1" {
		time.Sleep(time.Minute)
		return
	}
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = os.WriteFile(filepath.Join(dir, managementTestReadiness), nil, 0600) // #nosec G703 -- Guarded subprocess fixture; dir comes only from the parent test private t.TempDir, never CLI input.
		if os.Getenv("COHCTL_SIGNAL_READY") != "1" {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		if strings.Contains(r.URL.Path, "members") {
			fmt.Fprintf(w, `{"items":[{"processName":"%d"}]}`, pid)
		} else {
			fmt.Fprintf(w, `{"clusterName":%q,"clusterSize":1}`, managementFixtureName)
		}
	})
	listener, err := net.Listen("tcp4", net.JoinHostPort(localHost, port))
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	if err := serveSignalTestHTTP(listener, handler, time.Minute); err != nil {
		t.Fatal(err)
	}
}

func serveSignalTestHTTP(listener net.Listener, handler http.Handler, lifetime time.Duration) (err error) {
	server := &http.Server{
		Handler:           handler,
		ReadHeaderTimeout: time.Second, ReadTimeout: 2 * time.Second,
		WriteTimeout: 2 * time.Second, IdleTimeout: 2 * time.Second,
	}
	// An accept deadline is temporary to Serve, so it does not end the fixture.
	// Close the server to stop accepting and close active connections as well.
	closed := make(chan error, 1)
	timer := time.AfterFunc(lifetime, func() { closed <- server.Close() })
	defer func() {
		if timer.Stop() {
			err = errors.Join(err, server.Close())
		} else {
			// Join an expired callback and retain its close error before returning.
			err = errors.Join(err, <-closed)
		}
	}()
	err = server.Serve(listener)
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

func TestManagementSignalHTTPLifetime(t *testing.T) {
	listener, err := net.Listen("tcp4", net.JoinHostPort(localHost, "0"))
	if err != nil {
		t.Fatal(err)
	}
	entered, finished := make(chan struct{}), make(chan struct{})
	var serveErr error
	go func() {
		serveErr = serveSignalTestHTTP(listener, http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
			close(entered)
			<-r.Context().Done()
		}), 250*time.Millisecond)
		close(finished)
	}()
	t.Cleanup(func() {
		_ = listener.Close() // Independent rescue if the lifetime assertion fails.
		select {
		case <-finished:
		case <-time.After(3 * time.Second):
			t.Error("HTTP fixture did not finish after independent listener rescue")
		}
	})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://"+listener.Addr().String(), nil)
	if err != nil {
		t.Fatal(err)
	}
	client := &http.Client{Transport: &http.Transport{Proxy: nil}, Timeout: 3 * time.Second}
	defer client.CloseIdleConnections()
	responseDone := make(chan error, 1)
	go func() {
		response, requestErr := client.Do(request)
		if response != nil {
			requestErr = errors.Join(requestErr, response.Body.Close())
		}
		responseDone <- requestErr
	}()
	select {
	case <-entered:
	case <-time.After(3 * time.Second):
		t.Fatal("HTTP fixture did not accept the control request")
	}
	select {
	case <-finished:
		if serveErr != nil {
			t.Fatal(serveErr)
		}
	case <-time.After(time.Second):
		t.Fatal("HTTP fixture outlived its independent lifetime")
	}
	select {
	case requestErr := <-responseDone:
		if requestErr == nil {
			t.Fatal("active request survived the HTTP fixture lifetime")
		}
	case <-time.After(time.Second):
		t.Fatal("HTTP fixture lifetime did not close the active request")
	}
	connection, err := net.DialTimeout("tcp4", listener.Addr().String(), time.Second)
	if err == nil {
		_ = connection.Close()
		t.Fatal("HTTP fixture listener remained open after its lifetime")
	}
}

func signalTestEventually(t *testing.T, description string, condition func() bool) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("timed out: %s", description)
}

func signalTestChildAlive(pid int) bool {
	// A killed orphan can remain a zombie until its host reaper runs on Linux.
	// A zombie has exited and owns no executable/listeners.
	if data, err := os.ReadFile(fmt.Sprintf("/proc/%d/stat", pid)); err == nil {
		if end := strings.LastIndex(string(data), ") "); end >= 0 && strings.HasPrefix(string(data)[end+2:], "Z ") {
			return false
		}
	}
	return syscall.Kill(pid, 0) == nil
}

func signalTestPIDs(dir string) []int {
	paths, _ := filepath.Glob(filepath.Join(dir, "child-*"))
	var pids []int
	for _, p := range paths {
		data, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		pid, err := strconv.Atoi(string(data))
		if err == nil && pid > 0 {
			pids = append(pids, pid)
		}
	}
	return pids
}

func signalTestFreePort(t *testing.T) int32 {
	t.Helper()
	l, err := net.Listen("tcp4", localHost+":0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	return managementListenerPort(t, l)
}

func TestManagementSignalExecutable(t *testing.T) {
	// Build the product entry point, not a substitute Cobra command/test binary.
	dir := t.TempDir()
	binary := filepath.Join(dir, "cohctl")
	build := exec.Command("go", "build", "-o", binary, "./cohctl")
	build.Dir = filepath.Join("..", "..")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build CLI: %v\n%s", err, output)
	}
	fixture, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name, operation, phase string
		signal                 os.Signal
		success                bool
	}{
		{"start-delay-int", start, signalStartupDelay, os.Interrupt, false},
		{"start-readiness-int", start, managementTestReadiness, os.Interrupt, false},
		{"start-delay-term", start, signalStartupDelay, syscall.SIGTERM, false},
		{"start-readiness-term", start, managementTestReadiness, syscall.SIGTERM, false},
		{"create-delay-int", managementTestCreate, signalStartupDelay, os.Interrupt, false},
		{"create-readiness-term", managementTestCreate, managementTestReadiness, syscall.SIGTERM, false},
		{"restart-replacement-int", managementTestRestart, managementTestReadiness, os.Interrupt, false},
		{"scale-readiness-term", managementTestScale, managementTestReadiness, syscall.SIGTERM, false},
		{"restart-confirm-int", managementTestRestart, signalConfirmation, os.Interrupt, false},
		{"start-success-detached", start, "success", nil, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			work := t.TempDir()
			shimDir := filepath.Join(work, "bin")
			if err := os.Mkdir(shimDir, 0700); err != nil {
				t.Fatal(err)
			}
			quote := func(s string) string { return "'" + strings.ReplaceAll(s, "'", "'\"'\"'") + "'" }
			java := "#!/bin/sh\nif [ \"$1\" = -version ]; then echo 'openjdk version \"17.0.21\"' >&2; exit 0; fi\nexec " + quote(fixture) + " -test.run='^TestManagementSignalExecutableChild$' -- \"$@\"\n"
			if err := os.WriteFile(filepath.Join(shimDir, "java"), []byte(java), 0700); err != nil { // #nosec G306 -- Owner-only executable shim in a parent-created private t.TempDir; execute permission is required.
				t.Fatal(err)
			}
			// create uses dependency tool checks/transitive resolution, with no downloads.
			if err := os.WriteFile(filepath.Join(shimDir, "mvn"), []byte("#!/bin/sh\nexit 0\n"), 0700); err != nil { // #nosec G306 -- Owner-only executable shim in a parent-created private t.TempDir; execute permission is required.
				t.Fatal(err)
			}
			var old *managedProcess
			if tc.operation == managementTestRestart || tc.operation == managementTestScale {
				old = stopTestChild(t)
			}
			var cliDone atomic.Bool
			oldServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tc.phase == signalConfirmation {
					_ = os.WriteFile(filepath.Join(work, "inspected"), nil, 0600)
				}
				pids := []int{}
				if old != nil {
					select {
					case <-old.done:
					default:
						pids = append(pids, old.command.Process.Pid)
					}
				}
				if tc.operation == managementTestScale && len(signalTestPIDs(work)) > 0 {
					_ = os.WriteFile(filepath.Join(work, managementTestReadiness), nil, 0600)
				}
				if strings.Contains(r.URL.Path, "members") {
					items := []map[string]string{}
					for _, pid := range pids {
						items = append(items, map[string]string{"processName": strconv.Itoa(pid)})
					}
					_ = json.NewEncoder(w).Encode(map[string]interface{}{"items": items})
				} else {
					fmt.Fprintf(w, `{"clusterName":%q,"clusterSize":%d}`, managementFixtureName, len(pids))
				}
			}))
			defer oldServer.Close()
			port := signalTestFreePort(t)
			if old != nil {
				port = managementListenerPort(t, oldServer.Listener)
			}
			if tc.operation == managementTestRestart && tc.phase != signalConfirmation {
				// Release the old endpoint after its owned member exits, allowing the
				// replacement fixture to bind the same address/port.
				go func() { <-old.done; oldServer.Close() }()
			}
			conn := ClusterConnection{Name: managementFixtureName, ClusterName: managementFixtureName, ManuallyCreated: true, ConnectionType: httpType,
				ConnectionURL: oldServer.URL + "/old", ManagementPort: port, ClusterVersion: defaultCoherenceVersion, BaseClasspath: "fixture-classpath",
				Management: &ClusterManagement{RESTAddress: localHost}}
			configFile := filepath.Join(work, "cohctl.yaml")
			config := viper.New()
			config.SetConfigFile(configFile)
			config.Set(clusterKey, []ClusterConnection{conn})
			config.Set("requestTimeout", 2)
			config.Set("color", "on")
			if err := config.WriteConfigAs(configFile); err != nil {
				t.Fatal(err)
			}
			before, err := os.ReadFile(configFile)
			if err != nil {
				t.Fatal(err)
			}
			args := []string{"--config-dir", work, "--config", configFile, tc.operation, "cluster", managementFixtureName, "-r", "1"}
			if tc.operation == managementTestScale {
				args[len(args)-1] = "2"
			}
			if tc.operation == managementTestRestart && tc.phase != signalConfirmation {
				args = append(args, "--yes")
			}
			if tc.operation == managementTestCreate {
				args = append(args, "--yes", "--skip-deps", "--http-port", strconv.Itoa(int(port)), "--cluster-port", strconv.Itoa(int(signalTestFreePort(t))))
				// Preserve a pre-existing unrelated config record; create a distinct name.
				args[6] = "created-fixture"
			}
			if tc.phase == signalStartupDelay {
				args = append(args, "--startup-delay", "30s", "-r", "2")
			}
			command := exec.Command(binary, args...)
			command.Env = append(os.Environ(), "PATH="+shimDir+string(os.PathListSeparator)+os.Getenv("PATH"), "COHCTL_SIGNAL_CHILD=1", "COHCTL_SIGNAL_DIR="+work, "JAVA_TOOL_OPTIONS=", "_JAVA_OPTIONS=", "JDK_JAVA_OPTIONS=")
			if tc.success {
				command.Env = append(command.Env, "COHCTL_SIGNAL_READY=1")
			} else {
				command.Env = append(command.Env, "COHCTL_SIGNAL_READY=0")
			}
			input, err := command.StdinPipe()
			if err != nil {
				t.Fatal(err)
			}
			defer func() {
				// Cmd.Wait also closes StdinPipe; that specific repeat-close is benign.
				if err := input.Close(); err != nil && !errors.Is(err, os.ErrClosed) {
					t.Errorf("close CLI stdin: %v", err)
				}
			}()
			// File output avoids races between subprocess copies and progress polling.
			outFile, err := os.Create(filepath.Join(work, "cli-output"))
			if err != nil {
				t.Fatal(err)
			}
			defer outFile.Close()
			command.Stdout, command.Stderr = outFile, outFile
			if err := command.Start(); err != nil {
				t.Fatal(err)
			}
			done := make(chan error, 1)
			go func() { err := command.Wait(); cliDone.Store(true); done <- err }()
			t.Cleanup(func() {
				if !cliDone.Load() {
					if err := command.Process.Kill(); err != nil && !errors.Is(err, os.ErrProcessDone) {
						t.Errorf("rescue CLI fixture: %v", err)
					}
					if err := <-done; err != nil {
						var exitErr *exec.ExitError
						if !errors.As(err, &exitErr) || exitErr.ProcessState == nil {
							t.Errorf("wait for rescued CLI fixture: %v", err)
						}
					}
					t.Error("CLI fixture required cleanup rescue")
				}
				for _, pid := range signalTestPIDs(work) {
					if signalTestChildAlive(pid) {
						t.Logf("test cleanup stops still-live owned fixture PID %d", pid)
						_ = syscall.Kill(pid, syscall.SIGKILL)
					}
					signalTestEventually(t, "fixture cleanup", func() bool { return !signalTestChildAlive(pid) })
				}
			})
			output := func() string { b, _ := os.ReadFile(outFile.Name()); return string(b) }
			phaseReached := func() bool {
				if cliDone.Load() && !tc.success {
					t.Fatalf("CLI exited before %s: %s", tc.phase, output())
				}
				if tc.phase == signalConfirmation {
					return strings.Contains(output(), "Are you sure you want to stop")
				}
				if tc.phase == managementTestReadiness {
					_, err := os.Stat(filepath.Join(work, managementTestReadiness))
					return err == nil
				}
				return len(signalTestPIDs(work)) > 0
			}
			signalTestEventually(t, "CLI "+tc.phase, phaseReached)
			if !tc.success {
				if err := command.Process.Signal(tc.signal); err != nil {
					t.Fatal(err)
				}
			}
			var exitErr error
			select {
			case exitErr = <-done:
			case <-time.After(5 * time.Second):
				t.Fatalf("CLI did not return after signal: %s", output())
			}
			pids := signalTestPIDs(work)
			if tc.success {
				if exitErr != nil {
					t.Fatalf("CLI success: %v\n%s", exitErr, output())
				}
				if len(pids) != 1 || !signalTestChildAlive(pids[0]) {
					t.Fatalf("successful detached child did not survive: %v", pids)
				}
				after := viper.New()
				after.SetConfigFile(configFile)
				if err := after.ReadInConfig(); err != nil {
					t.Fatal(err)
				}
				var settings CoherenceCLIConfig
				if err := after.Unmarshal(&settings); err != nil {
					t.Fatal(err)
				}
				if len(settings.Clusters) != 1 || settings.Clusters[0].ConnectionURL != managementURL(*conn.Management, conn.ManagementPort) {
					t.Fatalf("saved config: %+v", settings.Clusters)
				}
			} else {
				if exitErr == nil {
					t.Fatalf("signal reported success: %s", output())
				}
				if tc.phase != signalConfirmation && !strings.Contains(output(), "context canceled") {
					t.Fatalf("signal bypassed lifecycle cleanup: %v\n%s", exitErr, output())
				}
				for _, pid := range pids {
					signalTestEventually(t, "cancelled member exit", func() bool { return !signalTestChildAlive(pid) })
				}
				after, err := os.ReadFile(configFile)
				if err != nil || !bytes.Equal(before, after) {
					t.Fatalf("cancel changed saved bytes: %v", err)
				}
				afterConfig := viper.New()
				afterConfig.SetConfigFile(configFile)
				if err := afterConfig.ReadInConfig(); err != nil {
					t.Fatal(err)
				}
				var saved CoherenceCLIConfig
				if err := afterConfig.Unmarshal(&saved); err != nil {
					t.Fatal(err)
				}
				if len(saved.Clusters) != 1 || saved.Clusters[0].ConnectionURL != conn.ConnectionURL || *saved.Clusters[0].Management != *conn.Management {
					t.Fatal("cancel changed old settings/URL")
				}
				if tc.phase == signalStartupDelay && len(pids) != 1 {
					t.Fatalf("additional launch after cancel: %v", pids)
				}
				if tc.phase == signalConfirmation && len(pids) != 0 {
					t.Fatal("launch during confirmation")
				}
			}
			if old != nil {
				if tc.operation == managementTestScale || tc.phase == signalConfirmation {
					assertManagedChildAlive(t, old)
				} else {
					waitStopTestChild(t, old)
					if !strings.Contains(output(), "cannot be restored automatically") {
						t.Fatalf("restart did not report stopped old cluster: %s", output())
					}
				}
			}
			t.Logf("actual CLI: %s %v, fixture PIDs=%v, exit=%v", tc.phase, tc.signal, pids, exitErr)
		})
	}
}
