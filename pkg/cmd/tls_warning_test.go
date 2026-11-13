/*
 * Copyright (c) 2026 Oracle and/or its affiliates.
 * Licensed under the Universal Permissive License v 1.0 as shown at
 * https://oss.oracle.com/licenses/upl.
 */

package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/oracle/coherence-cli/internal/testtls"
	"github.com/oracle/coherence-cli/pkg/fetcher"
)

const (
	commandBypassLog = "TLS certificate verification disabled"
	warningURLFlag   = "--url"
)

func warningCommandConfig(t *testing.T) string {
	t.Helper()
	oldLogger, oldFetcherLogger := Logger, fetcher.Logger
	t.Cleanup(func() {
		Logger, fetcher.Logger = oldLogger, oldFetcherLogger
		fetcher.BeginCommandTLSWarnings(os.Stderr)
	})
	return certificateCommandConfig(t, "version: fixture\ncolor: off\nignoreinvalidcerts: false\n")
}

func warningLogCount(t *testing.T, file string) int {
	t.Helper()
	if err := Logger.Sync(); err != nil {
		t.Fatal(err)
	}
	contents, err := os.ReadFile(filepath.Join(filepath.Dir(file), logFile))
	if err != nil {
		t.Fatal(err)
	}
	return strings.Count(string(contents), commandBypassLog)
}

func TestTLSWarningCommandExecutionReset(t *testing.T) {
	file := warningCommandConfig(t)
	server := connectionRequestServer(t, true)
	contents := fmt.Sprintf("version: fixture\ncolor: off\ncurrentcontext: excepted\nclusters:\n- name: excepted\n  connectiontype: http\n  connectionurl: %s\n  ignoreinvalidcerts: true\n", server.URL)
	if err := os.WriteFile(file, []byte(contents), 0600); err != nil {
		t.Fatal(err)
	}
	command := Initialize(nil)
	var stdout, stderr bytes.Buffer
	command.SetOut(&stdout)
	command.SetErr(&stderr)
	// Reuse exactly the same Cobra tree. No manual warning reset and no
	// Initialize call occurs between executions.
	for execution := 1; execution <= 2; execution++ {
		stdout.Reset()
		stderr.Reset()
		command.SetArgs([]string{configOption, file, certificateGetAction, "health", "-o", "json"})
		if err := command.Execute(); err != nil {
			t.Fatal(err)
		}
		if !json.Valid(stdout.Bytes()) || strings.TrimSpace(stdout.String()) != "{}" {
			t.Fatalf("structured stdout corrupted: %q", stdout.String())
		}
		if strings.Count(stderr.String(), "WARNING:") != 1 || !strings.Contains(stderr.String(), exceptedConnection) || !strings.Contains(stderr.String(), server.URL) {
			t.Fatalf("execution %d warning: %q", execution, stderr.String())
		}
		if count := warningLogCount(t, file); count != execution {
			t.Fatalf("execution %d: got %d bypass log records", execution, count)
		}
	}
}

func TestTLSWarningAddProbeAndSettingSilence(t *testing.T) {
	file := warningCommandConfig(t)
	server := connectionRequestServer(t, true)
	command := Initialize(nil)
	var stdout, stderr bytes.Buffer
	command.SetOut(&stdout)
	command.SetErr(&stderr)
	for _, test := range []struct {
		args     []string
		warnings int
	}{
		{[]string{certificateAddAction, certificateClusterCommand, exceptedConnection, warningURLFlag, server.URL, certificateFlag}, 1},
		{[]string{certificateGetAction, certificateCommand}, 0},
		{[]string{certificateSetAction, certificateCommand, stringFalse}, 0},
		{[]string{certificateSetAction, certificateCommand, stringTrue}, 0},
		{[]string{certificateGetAction, "config"}, 0},
	} {
		stdout.Reset()
		stderr.Reset()
		command.SetArgs(append([]string{configOption, file}, test.args...))
		if err := command.Execute(); err != nil {
			t.Fatalf("%v: %v", test.args, err)
		}
		if strings.Count(stderr.String(), "WARNING:") != test.warnings || strings.Contains(stdout.String(), "WARNING:") {
			t.Fatalf("%v: stdout %q, stderr %q", test.args, stdout.String(), stderr.String())
		}
		if count := warningLogCount(t, file); count != 1 {
			t.Fatalf("%v logged setting/selection as use: %d", test.args, count)
		}
	}
	// A general download remains verified and produces no bypass warning/log
	// even immediately after an actual opted-in cluster probe.
	if _, err := GetURLContents(server.URL); err == nil {
		t.Fatal("download accepted untrusted certificate")
	}
	if stderr.Len() != 0 || warningLogCount(t, file) != 1 {
		t.Fatalf("download emitted a bypass diagnostic: %q", stderr.String())
	}
}

func TestTLSWarningVerifiedCommandSilence(t *testing.T) {
	file := warningCommandConfig(t)
	ca := testtls.NewCA(t, "warning-trusted")
	server := downloadTLSServer(t, ca.Issue(t, localHost, false), nil)
	t.Setenv("COHERENCE_TLS_CERTS_PATH", writeDownloadFixture(t, "warning-roots.pem", ca.PEM))
	command := Initialize(nil)
	var stdout, stderr bytes.Buffer
	command.SetOut(&stdout)
	command.SetErr(&stderr)
	command.SetArgs([]string{configOption, file, certificateAddAction, certificateClusterCommand, verifiedConnection, warningURLFlag, server.URL, certificateFalseFlag})
	if err := command.Execute(); err != nil {
		t.Fatal(err)
	}
	if stderr.Len() != 0 || strings.Contains(stdout.String(), "WARNING:") || warningLogCount(t, file) != 0 {
		t.Fatalf("verified add emitted bypass warning: %q / %q", stdout.String(), stderr.String())
	}
}
