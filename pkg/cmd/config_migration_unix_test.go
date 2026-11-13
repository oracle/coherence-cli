// Copyright (c) 2026 Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at
// https://oss.oracle.com/licenses/upl.

//go:build !windows

package cmd

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"testing"
)

func assertMigrationPermissions(t *testing.T, file string, want os.FileMode) {
	t.Helper()
	// #nosec G703 -- Paths identify only task-owned fixture, backup, and staging files.
	info, err := os.Stat(file)
	if err != nil || info.Mode().Perm() != want {
		t.Fatalf("permissions on %s = %v, error %v; want %v", file, info, err, want)
	}
}

func TestLegacyTLSMigrationReadOnlyFile(t *testing.T) {
	file := writeMigrationFixture(t, migrationFixture)
	if err := os.Chmod(file, 0400); err != nil {
		t.Fatal(err)
	}
	result, err := migrateLegacyTLSConfig(file, defaultConfigMigrationOps())
	if err != nil {
		t.Fatal(err)
	}
	assertMigrationPermissions(t, file, 0400)
	assertMigrationPermissions(t, result.backup, 0400)
}

func TestLegacyTLSStartupUnreadableFileStopsWork(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	file := writeMigrationFixture(t, migrationFixture)
	if err := os.Chmod(file, 0200); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(file, 0600) })
	if _, err := os.ReadFile(file); err == nil {
		t.Skip("current user can read a write-only file")
	}
	stdout, stderr, err := runMigrationStartup(t, file, server.URL)
	if restoreErr := os.Chmod(file, 0600); restoreErr != nil {
		t.Fatal(restoreErr)
	}
	if err == nil || requests.Load() != 0 || len(stdout) != 0 || !strings.Contains(string(stderr), "unable to read config") ||
		!bytes.Equal(readMigrationBytes(t, file), []byte(migrationFixture)) {
		t.Fatalf("unreadable existing file was truncated or startup continued: %v, %s, %s", err, stdout, stderr)
	}
}
