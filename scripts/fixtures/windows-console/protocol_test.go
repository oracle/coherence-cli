// Copyright (c) 2026 Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at
// https://oss.oracle.com/licenses/upl.

//go:build windows

package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"syscall"
	"testing"
)

const singleReplicaSuffix = "-r 1"
const clusterIdentityPath = "/cluster"

func privateRoot(t *testing.T) *os.Root {
	t.Helper()
	root, err := openTaskRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := root.Close(); err != nil {
			t.Error(err)
		}
	})
	return root
}

func TestConsoleApprovedProtocol(t *testing.T) {
	directory := filepath.Join(t.TempDir(), "private space")
	cases := []struct{ op, phase, tail string }{{operationStart, phaseDelay, "-D 30s"}, {operationStart, phaseReadiness, singleReplicaSuffix}, {operationStart, "success", singleReplicaSuffix}, {operationCreate, phaseDelay, "-D 30s"}, {operationCreate, phaseReadiness, "-p 7574"}, {"restart", phaseReadiness, "-y"}, {"restart", "confirmation", singleReplicaSuffix}, {"scale", phaseReadiness, "-r 2"}}
	for _, tt := range cases {
		t.Run(tt.op+"/"+tt.phase, func(t *testing.T) {
			request := consoleRequest{Operation: tt.op, Phase: tt.phase, Cluster: "console-test-012345abcdef", Port: 30000, Event: 1}
			if tt.op == operationCreate {
				request.ClusterPort = 7574
			}
			args, err := request.arguments(directory)
			if err != nil {
				t.Fatal(err)
			}
			prefix := []string{"--config-dir", directory, "--config", filepath.Join(directory, "cohctl.json"), tt.op, "cluster", request.Cluster}
			if !reflect.DeepEqual(args[:len(prefix)], prefix) || !strings.HasSuffix(strings.Join(args, " "), tt.tail) {
				t.Fatalf("unexpected approved command: %q", args)
			}
		})
	}
}

func TestConsoleRejectsProtocolExpansion(t *testing.T) {
	good := consoleRequest{Operation: operationStart, Phase: phaseReadiness, Cluster: "console-test-012345abcdef", Port: 30000}
	for _, mutate := range []func(*consoleRequest){
		func(r *consoleRequest) { r.Operation = "delete" }, func(r *consoleRequest) { r.Operation = "cmd.exe" }, func(r *consoleRequest) { r.Phase = "--password" },
		func(r *consoleRequest) { r.Cluster = "../../outside" }, func(r *consoleRequest) { r.Cluster = "console-test-012345abcdeg" }, func(r *consoleRequest) { r.Event = 2 },
		func(r *consoleRequest) { r.Port = 0 }, func(r *consoleRequest) { r.Port = 65536 }, func(r *consoleRequest) { r.ClusterPort = 7574 },
		func(r *consoleRequest) { r.Operation = operationCreate; r.ClusterPort = r.Port }, func(r *consoleRequest) { r.Operation = operationCreate; r.Phase = "confirmation"; r.ClusterPort = 7574 },
	} {
		request := good
		mutate(&request)
		if args, err := request.arguments(t.TempDir()); err == nil {
			t.Fatalf("unapproved request became argv: %+v %q", request, args)
		}
	}
	root := privateRoot(t)
	for _, body := range []string{`{"Argv":["cmd.exe","/c","anything"]}`, `{"Output":"../outside"}`, `{"SignalFile":"../outside"}`, `{} {}`, strings.Repeat(" ", 65538) + `{}`, `{}` + strings.Repeat(" ", 65538)} {
		if err := root.WriteFile(requestName, []byte(body), 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := readConsoleRequest(root); err == nil {
			t.Fatalf("untrusted request accepted: %.100s", body)
		}
	}
	data, err := json.Marshal(good)
	if err != nil {
		t.Fatal(err)
	}
	if err := root.WriteFile(requestName, data, 0600); err != nil {
		t.Fatal(err)
	}
	if got, err := readConsoleRequest(root); err != nil || got != good {
		t.Fatalf("valid request changed: %+v %v", got, err)
	}
}

func TestConsolePrivateRootConfinement(t *testing.T) {
	root := privateRoot(t)
	if _, err := openTaskRoot("relative/task"); err == nil {
		t.Fatal("relative task root accepted")
	}
	if err := root.Mkdir("old", 0700); err != nil {
		t.Fatal(err)
	}
	for _, dir := range []string{"", "old"} {
		if err := writeFixtureMarker(root, dir, "child-123", []byte("123")); err != nil {
			t.Fatal(err)
		}
		if err := writeFixtureMarker(root, dir, phaseReadiness, nil); err != nil {
			t.Fatal(err)
		}
	}
	for _, pair := range [][2]string{{"../outside", phaseReadiness}, {"old/..", phaseReadiness}, {"", "../marker"}, {"", "child--1"}, {"", "child-01"}} {
		if err := writeFixtureMarker(root, pair[0], pair[1], nil); err == nil {
			t.Fatalf("marker escape accepted: %q", pair)
		}
	}
	outside := t.TempDir()
	sentinel := filepath.Join(outside, "sentinel")
	if err := os.WriteFile(sentinel, []byte("unchanged"), 0600); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{requestName, outputName, signalName, "child-999"} {
		t.Run(name, func(t *testing.T) {
			if err := root.Remove(name); err != nil && !errors.Is(err, os.ErrNotExist) {
				t.Fatal(err)
			}
			if err := os.Symlink(sentinel, filepath.Join(root.Name(), name)); err != nil {
				t.Skipf("UNRUN symlink capability: %v", err)
			}
			switch name {
			case requestName:
				if _, err := readConsoleRequest(root); err == nil {
					t.Fatal("request escape accepted")
				}
			case outputName:
				file, err := openConsoleOutput(root)
				if err == nil {
					file.Close()
					t.Fatal("output escape accepted")
				}
			case signalName:
				if ready, err := consoleSignalReady(root); err == nil || ready {
					t.Fatal("signal escape accepted")
				}
			default:
				if err := writeFixtureMarker(root, "", name, nil); err == nil {
					t.Fatal("marker escape accepted")
				}
			}
			data, err := os.ReadFile(sentinel)
			if err != nil || string(data) != "unchanged" {
				t.Fatalf("outside sentinel changed: %s %v", data, err)
			}
		})
	}
}

func TestConsoleApprovedExecutable(t *testing.T) {
	root := privateRoot(t)
	if _, err := approvedCLI(root); err == nil {
		t.Fatal("missing executable accepted")
	}
	if err := root.Mkdir(fixtureBin, 0700); err != nil {
		t.Fatal(err)
	}
	if err := root.WriteFile("bin/cohctl.exe", []byte("approved fixture CLI"), 0600); err != nil {
		t.Fatal(err)
	}
	if got, err := approvedCLI(root); err != nil || got != filepath.Join(root.Name(), fixtureBin, "cohctl.exe") {
		t.Fatalf("approved CLI: %q %v", got, err)
	}
	if err := root.Remove("bin/cohctl.exe"); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(t.TempDir(), "other.exe"), filepath.Join(root.Name(), fixtureBin, "cohctl.exe")); err != nil {
		t.Skipf("UNRUN executable link capability: %v", err)
	}
	if _, err := approvedCLI(root); err == nil {
		t.Fatal("request could select a linked executable")
	}
}

func TestFixtureStructuredIdentity(t *testing.T) {
	root := privateRoot(t)
	cluster := "</script><script>alert(1)</script>\"\\\n\u2028"
	for _, path := range []string{clusterIdentityPath, "/cluster/members"} {
		response := httptest.NewRecorder()
		fixtureIdentityHandler(root, "", cluster, 123, true).ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
		if response.Code != 200 || response.Header().Get("Content-Type") != "application/json" || strings.Contains(response.Body.String(), "<script>") {
			t.Fatalf("unsafe JSON response: %+v %s", response.Header(), response.Body)
		}
		var value struct {
			ClusterName string
			ClusterSize int
			Items       []struct{ ProcessName string }
		}
		if err := json.Unmarshal(response.Body.Bytes(), &value); err != nil {
			t.Fatal(err)
		}
		if path == clusterIdentityPath && (value.ClusterName != cluster || value.ClusterSize != 1) {
			t.Fatalf("cluster identity lost: %+v", value)
		}
		if path != clusterIdentityPath && (len(value.Items) != 1 || value.Items[0].ProcessName != "123") {
			t.Fatalf("member identity lost: %+v", value)
		}
	}
	response := httptest.NewRecorder()
	fixtureIdentityHandler(root, "", cluster, 123, false).ServeHTTP(response, httptest.NewRequest(http.MethodGet, clusterIdentityPath, nil))
	if response.Code != 503 {
		t.Fatalf("unready fixture was accepted: %d", response.Code)
	}
	if _, err := root.Stat(phaseReadiness); err != nil {
		t.Fatal(err)
	}
}

func TestConsoleDetachAndRescueOutcomes(t *testing.T) {
	for _, tt := range []struct {
		name   string
		result uintptr
		cause  error
		failed bool
	}{
		{"success-stale-error", 1, syscall.EACCES, false}, {"no-console-success", 1, errInvalidHandle, false},
		{"invalid-handle-failure", 0, errInvalidHandle, true}, {"access-denied", 0, syscall.EACCES, true}, {"missing-error-code", 0, syscall.Errno(0), true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			err := detachConsole(func() (uintptr, uintptr, error) { return tt.result, 0, tt.cause })
			if (err != nil) != tt.failed {
				t.Fatalf("detach error=%v", err)
			}
			if tt.failed && tt.cause != nil && tt.cause != syscall.Errno(0) && !errors.Is(err, tt.cause) {
				t.Fatalf("detach cause lost: %v", err)
			}
		})
	}
	waitErr := errors.New("killed child")
	if err := rescueWaitResult(waitErr); !errors.Is(err, waitErr) {
		t.Fatalf("sole Wait cause lost: %v", err)
	}
	if err := rescueWaitResult(nil); err == nil {
		t.Fatal("unexpected successful rescue hidden")
	}
}

func TestConsoleExecutableRootAnchor(t *testing.T) {
	directory := t.TempDir()
	for _, name := range []string{"controller.exe", "java.exe", "mvn.exe"} {
		got, err := executableTaskDirectory(filepath.Join(directory, fixtureBin, name))
		if err != nil || got != directory {
			t.Fatalf("private executable root: %q %v", got, err)
		}
	}
	for _, path := range []string{"bin/controller.exe", filepath.Join(directory, "elsewhere", "controller.exe")} {
		if got, err := executableTaskDirectory(path); err == nil {
			t.Fatalf("unapproved executable location selected root: %q", got)
		}
	}
}

func TestConsolePrivateSignalAndOutput(t *testing.T) {
	root := privateRoot(t)
	if ready, err := consoleSignalReady(root); ready || err != nil {
		t.Fatalf("missing signal: %t %v", ready, err)
	}
	if err := root.WriteFile(signalName, nil, 0600); err != nil {
		t.Fatal(err)
	}
	if ready, err := consoleSignalReady(root); !ready || err != nil {
		t.Fatalf("private signal: %t %v", ready, err)
	}
	if err := root.WriteFile(outputName, []byte("old data"), 0600); err != nil {
		t.Fatal(err)
	}
	output, err := openConsoleOutput(root)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := output.Write([]byte("new")); err != nil {
		t.Fatal(err)
	}
	if err := output.Close(); err != nil {
		t.Fatal(err)
	}
	if data, err := root.ReadFile(outputName); err != nil || string(data) != "new" {
		t.Fatalf("private output not truncated: %q %v", data, err)
	}
}
