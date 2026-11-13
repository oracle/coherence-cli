// Copyright (c) 2026 Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at
// https://oss.oracle.com/licenses/upl.

//go:build windows

package main

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

const (
	phaseDelay      = "delay"
	phaseReadiness  = "readiness"
	operationStart  = "start"
	operationCreate = "create"
	fixtureBin      = "bin"

	requestName      = "request.json"
	signalName       = "signal"
	outputName       = "cli.log"
	errInvalidHandle = syscall.Errno(6) // ERROR_INVALID_HANDLE: no console.
)

// The parent copies this controller and the approved CLI into a fresh private
// task's bin directory. The running executable anchors that task root; neither
// environment variables nor JSON requests select a filesystem root or program.
func executableTaskDirectory(executable string) (string, error) {
	bin := filepath.Dir(executable)
	if !filepath.IsAbs(executable) || filepath.Base(bin) != fixtureBin {
		return "", errors.New("fixture executable must be in private task bin")
	}
	return filepath.Dir(bin), nil
}

func openTaskRoot(directory string) (*os.Root, error) {
	if !filepath.IsAbs(directory) || filepath.Clean(directory) != directory {
		return nil, errors.New("absolute clean private task root required")
	}
	info, err := os.Lstat(directory)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return nil, errors.New("private task root must be a directory, not a link")
	}
	return os.OpenRoot(directory)
}

type consoleRequest struct {
	Operation   string
	Phase       string
	Cluster     string
	Port        int
	ClusterPort int
	Event       uint32
}

func readConsoleRequest(root *os.Root) (request consoleRequest, resultErr error) {
	file, err := root.Open(requestName)
	if err != nil {
		return request, err
	}
	defer func() { resultErr = errors.Join(resultErr, file.Close()) }()
	data, err := io.ReadAll(io.LimitReader(file, 65537))
	if err != nil {
		return request, err
	}
	if len(data) > 65536 {
		return request, errors.New("console request exceeds 64 KiB")
	}
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		return request, err
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return request, errors.New("trailing or oversized console request")
	}
	return request, nil
}

func (r consoleRequest) arguments(directory string) ([]string, error) {
	marker := strings.TrimPrefix(r.Cluster, "console-test-")
	if len(marker) != 12 || r.Cluster == marker {
		return nil, errors.New("private cluster marker required")
	}
	if _, err := hex.DecodeString(marker); err != nil {
		return nil, err
	}
	if r.Event > 1 || r.Port < 1 || r.Port > 65535 {
		return nil, errors.New("invalid console event or management port")
	}
	allowed := false
	switch r.Operation {
	case operationStart:
		allowed = r.Phase == phaseDelay || r.Phase == phaseReadiness || r.Phase == "success"
	case operationCreate:
		allowed = r.Phase == phaseDelay || r.Phase == phaseReadiness
	case "restart":
		allowed = r.Phase == phaseReadiness || r.Phase == "confirmation"
	case "scale":
		allowed = r.Phase == phaseReadiness
	}
	if !allowed {
		return nil, errors.New("unapproved console operation/phase")
	}
	replicas := "1"
	if r.Operation == "scale" || r.Phase == phaseDelay {
		replicas = "2"
	}
	args := []string{"--config-dir", directory, "--config", filepath.Join(directory, "cohctl.json"), r.Operation, "cluster", r.Cluster, "-r", replicas}
	if r.Operation == operationCreate {
		if r.ClusterPort < 1 || r.ClusterPort > 65535 || r.ClusterPort == r.Port {
			return nil, errors.New("distinct valid cluster port required")
		}
		args = append(args, "-y", "-I", "-H", strconv.Itoa(r.Port), "-p", strconv.Itoa(r.ClusterPort))
	} else if r.ClusterPort != 0 {
		return nil, errors.New("cluster port is only valid for create")
	}
	if r.Operation == "restart" && r.Phase != "confirmation" {
		args = append(args, "-y")
	}
	if r.Phase == phaseDelay {
		args = append(args, "-D", "30s")
	}
	return args, nil
}

func approvedCLI(root *os.Root) (string, error) {
	// Reject links for the fixed executable and its immediate directory. The
	// private root is trusted against same-principal replacement during launch;
	// CreateProcess has no executable-descriptor API on Windows.
	for _, name := range []string{fixtureBin, "bin/cohctl.exe"} {
		info, err := root.Lstat(name)
		if err != nil {
			return "", err
		}
		if info.Mode()&os.ModeSymlink != 0 || (name == fixtureBin && !info.IsDir()) || (name != fixtureBin && !info.Mode().IsRegular()) {
			return "", fmt.Errorf("unapproved CLI file %s", name)
		}
	}
	return filepath.Join(root.Name(), fixtureBin, "cohctl.exe"), nil
}

func openConsoleOutput(root *os.Root) (*os.File, error) {
	return root.OpenFile(outputName, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
}

func consoleSignalReady(root *os.Root) (bool, error) {
	_, err := root.Stat(signalName)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	return err == nil, err
}

func writeFixtureMarker(root *os.Root, memberDirectory, name string, data []byte) error {
	if memberDirectory != "" && memberDirectory != "old" {
		return errors.New("unapproved fixture member directory")
	}
	if name != phaseReadiness {
		pid, err := strconv.Atoi(strings.TrimPrefix(name, "child-"))
		if err != nil || pid <= 0 || name != "child-"+strconv.Itoa(pid) {
			return errors.New("invalid fixture marker")
		}
	}
	return root.WriteFile(filepath.Join(memberDirectory, name), data, 0600)
}

func fixtureIdentityHandler(root *os.Root, memberDirectory, cluster string, pid int, ready bool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		if err := writeFixtureMarker(root, memberDirectory, phaseReadiness, nil); err != nil {
			http.Error(w, "fixture marker failure", http.StatusInternalServerError)
			return
		}
		if !ready {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		var value any = struct {
			ClusterName string `json:"clusterName"`
		}{cluster}
		if strings.Contains(r.URL.Path, "members") {
			value = struct {
				Items []map[string]string `json:"items"`
			}{[]map[string]string{{"processName": strconv.Itoa(pid)}}}
		}
		if err := json.NewEncoder(w).Encode(value); err != nil {
			return
		} // The client may have disconnected; no second response.
	})
}

// FreeConsole also succeeds when no console is attached. A zero return is
// always a failure; the last-error value is meaningful only on that path.
func detachConsole(call func() (uintptr, uintptr, error)) error {
	detached, _, err := call()
	if detached != 0 {
		return nil
	} // Win32 last-error is undefined after success.
	if err == nil || errors.Is(err, syscall.Errno(0)) {
		err = errors.New("FreeConsole failed without an error code")
	}
	return fmt.Errorf("FreeConsole: %w", err)
}

func rescueWaitResult(waitErr error) error {
	if waitErr == nil {
		return errors.New("owned CLI rescue Wait returned status 0")
	}
	return fmt.Errorf("owned CLI rescue Wait: %w", waitErr)
}
