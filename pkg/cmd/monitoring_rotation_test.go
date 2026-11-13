/*
 * Copyright (c) 2026 Oracle and/or its affiliates.
 * Licensed under the Universal Permissive License v 1.0 as shown at
 * https://oss.oracle.com/licenses/upl.
 */

package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// The stub models Docker's default running-only filter, so stopped-container
// cases fail if either discovery path omits --all. Rejection exercises init's
// real preflight and verifies it never stages or replaces the active files.
func TestMonitoringInitializationRequiresContainerRemoval(t *testing.T) {
	if runtime.GOOS == managementTestWindows {
		t.Skip("Docker command stub requires a POSIX shell")
	}
	originalDirectory, originalVersion := cfgDirectory, Version
	originalConfirm, originalStandalone := automaticallyConfirm, useDockerComposeParam
	t.Cleanup(func() {
		cfgDirectory, Version = originalDirectory, originalVersion
		automaticallyConfirm, useDockerComposeParam = originalConfirm, originalStandalone
	})
	Version, automaticallyConfirm = "1.10.11", true
	t.Setenv(monitoringBindAddressEnv, defaultMonitoringAddress)
	t.Setenv("COMPOSE_PROJECT_NAME", "")

	bin := t.TempDir()
	stub := `#!/bin/sh
printf '%s\n' "$*" >> "$COHCTL_TEST_DOCKER_LOG"
case "$*" in
  *"version --short") printf '2.39.0\n'; exit 0 ;;
esac
if [ "$COHCTL_TEST_DOCKER_STATE" = error ]; then
  echo 'simulated container discovery failure' >&2
  exit 1
fi
all=false
for arg do
  if [ "$arg" = --all ]; then all=true; fi
done
case "$COHCTL_TEST_DOCKER_STATE" in
  removed) ;;
  running) printf 'monitoring-container-id\n' ;;
  *) if "$all"; then printf 'monitoring-container-id\n'; fi ;;
esac
`
	for _, executable := range []string{"docker", dockerCompose} {
		// #nosec G306 -- owner-only executable Docker stub in t.TempDir; execution is required by this test.
		if err := os.WriteFile(filepath.Join(bin, executable), []byte(stub), 0700); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("PATH", bin)
	for _, selection := range []struct {
		name, generated, legacy string
		standalone              bool
	}{
		{"generated", "valid", "valid", false},
		{"standalone generated", "valid", "valid", true},
		{"legacy", "missing", "valid", false},
		{"corrupt generated fallback", "corrupt", "valid", false},
		{"unusable models", "corrupt", "corrupt", false},
		{"missing models", "missing", "missing", false},
		{"missing directory", "missing", "missing", false},
		{"custom fallback project", "corrupt", "corrupt", false},
	} {
		t.Run(selection.name, func(t *testing.T) {
			for _, state := range []string{"running", "exited", "created", "paused", "dead", "removed", "error"} {
				t.Run(state, func(t *testing.T) {
					cfgDirectory = t.TempDir()
					directory := filepath.Join(cfgDirectory, monitoringDirectory)
					useDockerComposeParam = selection.standalone
					project := monitoringDirectory
					if selection.name == "custom fallback project" {
						project = "custom-monitoring"
					}
					t.Setenv("COMPOSE_PROJECT_NAME", project)
					if selection.name != "missing directory" {
						if err := os.Mkdir(directory, 0700); err != nil {
							t.Fatal(err)
						}
						for file, kind := range map[string]string{generatedDockerComposeYAML: selection.generated, dockerComposeYAML: selection.legacy} {
							if kind == "missing" {
								continue
							}
							contents := minimalLegacyCompose()
							if file == generatedDockerComposeYAML {
								contents = secureCompose(t)
							}
							if kind == "corrupt" {
								contents = []byte("not: [yaml")
							}
							if err := os.WriteFile(filepath.Join(directory, file), contents, 0600); err != nil {
								t.Fatal(err)
							}
						}
						for file, contents := range map[string]string{grafanaPasswordFile: strings.Repeat("a", 43), monitoringEnvFile: monitoringBindAddressEnv + "=127.0.0.1\n", "customization": "keep existing settings"} {
							if err := os.WriteFile(filepath.Join(directory, file), []byte(contents), 0600); err != nil {
								t.Fatal(err)
							}
						}
					}
					log := filepath.Join(t.TempDir(), "docker.log")
					t.Setenv("COHCTL_TEST_DOCKER_LOG", log)
					t.Setenv("COHCTL_TEST_DOCKER_STATE", state)
					before := monitoringDirectorySnapshot(t, cfgDirectory)
					mon := newMonitoringAt(&cobra.Command{}, directory)
					err := mon.ensureMonitoringIsStopped()
					if state == "removed" {
						if err != nil {
							t.Fatalf("containers removed by down must permit initialization: %v", err)
						}
					} else {
						assertMonitoringRotationRejection(t, err, state)
						var output bytes.Buffer
						command := &cobra.Command{}
						command.SetOut(&output)
						command.SetErr(&output)
						assertMonitoringRotationRejection(t, initMonitoringCmd.RunE(command, nil), state)
						if strings.Contains(output.String(), "administrator password:") || strings.Contains(output.String(), OperationCompleted) {
							t.Fatal("rejected initialization reported success or a new password")
						}
					}
					if after := monitoringDirectorySnapshot(t, cfgDirectory); !reflect.DeepEqual(before, after) {
						t.Fatal("container preflight changed the existing directory/password or created staging files")
					}
					commands, err := os.ReadFile(log)
					if err != nil {
						t.Fatal(err)
					}
					if selection.generated != "valid" && selection.legacy != "valid" {
						want := "ps --all -q --filter label=com.docker.compose.project=" + project + "\n"
						if !strings.Contains(string(commands), want) {
							t.Fatalf("fallback must query only the monitoring project in all states: %s", commands)
						}
					} else if !strings.Contains(string(commands), " ps --all -q\n") {
						t.Fatalf("Compose must query containers in all states: %s", commands)
					}
				})
			}
		})
	}
}

func assertMonitoringRotationRejection(t *testing.T, err error, state string) {
	t.Helper()
	if err == nil {
		t.Fatalf("%s containers/discovery must prevent reinitialization", state)
	}
	if !strings.Contains(err.Error(), "cohctl stop monitoring") {
		t.Fatalf("missing removal guidance: %v", err)
	}
	if state == "error" && !strings.Contains(err.Error(), "simulated container discovery failure") {
		t.Fatalf("discovery error lost: %v", err)
	}
}

// Include modes and directory entries as well as file bytes, without printing secrets.
func monitoringDirectorySnapshot(t *testing.T, directory string) map[string]string {
	t.Helper()
	snapshot := map[string]string{}
	err := filepath.WalkDir(directory, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		value := info.Mode().String()
		if !entry.IsDir() {
			// #nosec G122 -- test-owned t.TempDir tree, with no concurrent writers or attacker-controlled entries.
			contents, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			value += string(contents)
		}
		snapshot[path] = value
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return snapshot
}
