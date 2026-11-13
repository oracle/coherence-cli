// Copyright (c) 2026 Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at
// https://oss.oracle.com/licenses/upl.

//go:build linux || darwin

package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/spf13/cobra"
)

func TestClassifyMonitoringBundleFIFO(t *testing.T) {
	for _, name := range []string{generatedDockerComposeYAML, monitoringEnvFile, grafanaPasswordFile} {
		t.Run(name, func(t *testing.T) {
			directory := monitoringValidationFixture(t)
			if err := syscall.Mkfifo(filepath.Join(directory, name), 0600); err != nil {
				if errors.Is(err, syscall.ENOTSUP) || errors.Is(err, syscall.ENOSYS) {
					t.Skipf("FIFO creation unavailable: %v", err)
				}
				t.Fatal(err)
			}
			mode, err := classifyMonitoringBundle(directory)
			if mode != monitoringBundleInvalid {
				t.Errorf("expected invalid mode, got %v", mode)
			}
			assertInvalidMonitoringArtifact(t, err, name)
		})
	}
}

func TestClassifyMonitoringBundleInspectionFailure(t *testing.T) {
	// ENOTDIR remains an inspection failure even when running as root.
	blocked := filepath.Join(t.TempDir(), "not-a-directory")
	if err := os.WriteFile(blocked, nil, 0600); err != nil {
		t.Fatal(err)
	}
	mode, err := classifyMonitoringBundle(blocked)
	var pathError *os.PathError
	if mode != monitoringBundleInvalid || !errors.Is(err, syscall.ENOTDIR) || !errors.As(err, &pathError) {
		t.Fatalf("expected wrapped inspection failure, got mode=%v error=%v", mode, err)
	}
}

// A subprocess deadline bounds regressions in os.ReadFile on a FIFO with no writer.
func TestMonitoringRecoveryFIFO(t *testing.T) {
	if directory := os.Getenv("COHCTL_TEST_MONITORING_RECOVERY_DIR"); directory != "" {
		// Write directly to stderr so completed phases survive a killed subprocess.
		fmt.Fprintln(os.Stderr, "monitoring recovery phase: child entered")
		wantBase := os.Getenv("COHCTL_TEST_MONITORING_RECOVERY_BASE")
		mon := newMonitoringAt(&cobra.Command{}, directory)
		switch os.Getenv("COHCTL_TEST_MONITORING_RECOVERY_PROBE") {
		case "arguments":
			fmt.Fprintln(os.Stderr, "monitoring recovery phase: arguments probe started")
			assertMonitoringRecoveryArgs(t, mon, filepath.Join(directory, generatedDockerComposeYAML), defaultMonitoringAddress, false)
			fmt.Fprintln(os.Stderr, "monitoring recovery phase: arguments probe completed")
		case "stopped":
			assertMonitoringRecoveryStopped(t, mon, wantBase)
		default:
			fmt.Fprintln(os.Stderr, "monitoring recovery phase: recovery started")
			assertMonitoringRecovery(t, directory, wantBase, defaultMonitoringAddress, false)
			fmt.Fprintln(os.Stderr, "monitoring recovery phase: recovery completed")
			assertMonitoringRecoveryStopped(t, mon, wantBase)
		}
		fmt.Fprintln(os.Stderr, "monitoring recovery phase: child completed")
		return
	}
	for _, symlink := range []bool{false, true} {
		kind := "FIFO"
		if symlink {
			kind = "symlink to FIFO"
		}
		t.Run(kind, func(t *testing.T) {
			for _, test := range []struct {
				name, generated, legacy, env, password, wantBase, probe string
			}{
				{"generated fallback", "fifo", "regular", "missing", "missing", dockerComposeYAML, ""},
				{"environment fallback", "regular", "regular", "fifo", "missing", generatedDockerComposeYAML, ""},
				{"legacy only", "missing", "fifo", "missing", "missing", "", ""},
				{"corrupt generated and special legacy", "invalid", "fifo", "missing", "missing", "", ""},
				{"both special", "fifo", "fifo", "missing", "missing", "", ""},
				{"generated priority", "regular", "fifo", "missing", "missing", generatedDockerComposeYAML, ""},
				{"password ignored", "regular", "regular", "missing", "fifo", generatedDockerComposeYAML, ""},
				{"environment arguments", "regular", "regular", "fifo", "missing", generatedDockerComposeYAML, "arguments"},
				{"environment stopped check", "regular", "regular", "fifo", "missing", generatedDockerComposeYAML, "stopped"},
				{"generated stopped check", "fifo", "regular", "missing", "missing", dockerComposeYAML, "stopped"},
			} {
				t.Run(test.name, func(t *testing.T) {
					directory := t.TempDir()
					for name, artifact := range map[string]string{generatedDockerComposeYAML: test.generated, dockerComposeYAML: test.legacy, monitoringEnvFile: test.env, grafanaPasswordFile: test.password} {
						file := filepath.Join(directory, name)
						switch artifact {
						case "missing":
							continue
						case "fifo":
							if symlink {
								target := file + "-fifo"
								monitoringTestSymlink(t, target, file)
								file = target
							}
							if err := syscall.Mkfifo(file, 0600); err != nil {
								if errors.Is(err, syscall.ENOTSUP) || errors.Is(err, syscall.ENOSYS) {
									t.Skipf("FIFO creation unavailable: %v", err)
								}
								t.Fatal(err)
							}
							continue
						}
						contents := minimalLegacyCompose()
						if name == generatedDockerComposeYAML {
							contents = secureCompose(t)
						}
						if artifact == "invalid" {
							contents = []byte("services: {}")
						}
						if err := os.WriteFile(file, contents, 0600); err != nil {
							t.Fatal(err)
						}
					}
					// Only this stub is reachable as docker; no daemon is contacted.
					bin := t.TempDir()
					stub := "#!/bin/sh\nprintf '%s\\n' \"$@\" > \"$COHCTL_TEST_DOCKER_LOG\"\nif [ \"$COHCTL_TEST_DOCKER_RUNNING\" = 1 ]; then printf 'container-id\\n'; fi\n"
					// #nosec G306 -- owner-only executable Docker stub in t.TempDir.
					if err := os.WriteFile(filepath.Join(bin, "docker"), []byte(stub), 0700); err != nil {
						t.Fatal(err)
					}
					executable, err := os.Executable()
					if err != nil {
						t.Fatal(err)
					}
					ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
					defer cancel()
					command := exec.CommandContext(ctx, executable, "-test.run=^TestMonitoringRecoveryFIFO$", "-test.v")
					command.Env = append(os.Environ(),
						"COHCTL_TEST_MONITORING_RECOVERY_DIR="+directory,
						"COHCTL_TEST_MONITORING_RECOVERY_BASE="+test.wantBase,
						"COHCTL_TEST_MONITORING_RECOVERY_PROBE="+test.probe,
						"COHCTL_TEST_DOCKER_LOG="+filepath.Join(bin, "arguments"),
						"COHCTL_TEST_DOCKER_RUNNING=0",
						monitoringBindAddressEnv+"=0.0.0.0", "PATH="+bin)
					output, err := command.CombinedOutput()
					if ctx.Err() != nil {
						t.Fatalf("recovery subprocess exceeded deadline: %v\n%s", ctx.Err(), output)
					}
					if err != nil {
						t.Fatalf("recovery subprocess failed: %v\n%s", err, output)
					}
				})
			}
		})
	}
}

func assertMonitoringRecoveryStopped(t *testing.T, mon *monitoring, wantBase string) {
	t.Helper()
	useDockerComposeParam = false
	fmt.Fprintln(os.Stderr, "monitoring recovery phase: stopped check (no containers) started")
	err := mon.ensureMonitoringIsStopped()
	fmt.Fprintln(os.Stderr, "monitoring recovery phase: stopped check (no containers) returned")
	if err != nil {
		t.Fatal(err)
	}
	// #nosec G703 -- parent test sets this subprocess-only path to its private temporary log.
	args, err := os.ReadFile(os.Getenv("COHCTL_TEST_DOCKER_LOG"))
	if err != nil {
		t.Fatal(err)
	}
	wantArgs := []string{"ps", "--all", "-q", "--filter", "label=com.docker.compose.project=" + monitoringDirectory}
	if wantBase != "" {
		wantArgs = []string{"compose", "-f", filepath.Join(mon.monitoringDir, wantBase), "ps", "--all", "-q"}
	}
	if string(args) != strings.Join(wantArgs, "\n")+"\n" {
		t.Fatalf("unexpected stopped-check command: %q, want %q", args, wantArgs)
	}
	t.Setenv("COHCTL_TEST_DOCKER_RUNNING", "1")
	fmt.Fprintln(os.Stderr, "monitoring recovery phase: stopped check (containers present) started")
	err = mon.ensureMonitoringIsStopped()
	fmt.Fprintln(os.Stderr, "monitoring recovery phase: stopped check (containers present) returned")
	if err == nil || !strings.Contains(err.Error(), "monitoring containers still exist") {
		t.Fatalf("expected running monitoring containers to prevent reinitialization, got %v", err)
	}
}
