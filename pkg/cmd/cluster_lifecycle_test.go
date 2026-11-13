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
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/oracle/coherence-cli/pkg/utils"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const lifecycleSavePhase = "save"

// Boundary fixtures bypass runtime/endpoint preflight; command tests exercise
// those separately. The real transaction and its owned child handles are used.
func testManagedConnection(t *testing.T, connection ClusterConnection, m ClusterManagement) ClusterConnection {
	t.Helper()
	args, err := serverBaseOptions(connection)
	if err != nil {
		t.Fatal(err)
	}
	return managedConnection(connection, m, args)
}

func testStartPlan(t *testing.T, connection ClusterConnection, count, existing int32, m ClusterManagement) managedStartPlan {
	t.Helper()
	options, err := validateManagedOptions(connection)
	if err != nil {
		t.Fatal(err)
	}
	delay, err := utils.GetStartupDelayInMillis(startupDelayParam)
	if err != nil {
		t.Fatal(err)
	}
	return prepareManagedMembers(connection, m, options, count, existing, time.Duration(delay)*time.Millisecond)
}

func runTestStartup(t *testing.T, cmd *cobra.Command, connection ClusterConnection, count, existing int32, m ClusterManagement) ([]*managedProcess, error) {
	t.Helper()
	return runPreparedStartup(cmd, testStartPlan(t, connection, count, existing, m), managedNoSave)
}

func TestManagementLifecycleTransaction(t *testing.T) {
	for _, tc := range []struct {
		name              string
		save              managedSaveMode
		failure, cancelAt string
	}{
		{"partial-launch", managedCreate, cleanupLaunchFailure, ""},
		{managementTestReadiness, managedUpdate, managementTestReadiness, ""},
		{"cancel-readiness", managedNoSave, "", managementTestReadiness},
		{"cancel-before-save", managedUpdate, "", "before-save"},
		{"create-save-failure", managedCreate, lifecycleSavePhase, ""},
		{"update-save-failure", managedUpdate, lifecycleSavePhase, ""},
		{"cancel-failed-save", managedUpdate, lifecycleSavePhase, lifecycleSavePhase},
		{"accepted-create", managedCreate, "", ""},
		{"accepted-update-during-save", managedUpdate, "", lifecycleSavePhase},
		{"accepted-scale", managedNoSave, "", ""},
	} {
		failure := tc.failure != "" || tc.cancelAt != "" && tc.cancelAt != lifecycleSavePhase
		variants := []bool{false}
		if failure {
			variants = append(variants, true)
		}
		for _, cleanupFails := range variants {
			t.Run(fmt.Sprintf("%s/cleanup-failure-%t", tc.name, cleanupFails), func(t *testing.T) {
				managedPersistenceState(t)
				cleanupTestSeams(t)
				cleanupTestJava(t)
				// Give ordinary native completion enough time, including race builds.
				if !cleanupFails {
					managedCleanupTimeout = 3 * time.Second
				}
				filename := filepath.Join(t.TempDir(), "cohctl.json")
				persistenceFixture(t, filename, managementTestJSON, "")
				connection := testManagedConnection(t, ClusterConnection{Name: managementTestConnectionName, ClusterName: managementTestConnectionName, ManuallyCreated: true, ManagementPort: 30000, ClusterVersion: defaultCoherenceVersion}, ClusterManagement{localHost, 9999, localHost, localHost})
				if err := persistManagedConnection(connection, true); err != nil {
					t.Fatal(err)
				}
				before, _ := os.ReadFile(filename)
				oldConfig, oldSettings := Config, viper.AllSettings()
				control, _, _ := cleanupTestChild(t, false, nil)
				proposed := connection
				if tc.save == managedCreate {
					proposed.Name = managementTestNewConnectionName
				}
				m := ClusterManagement{RESTAddress: managementTestIP}
				count, existing := int32(1), int32(0)
				if tc.failure == cleanupLaunchFailure {
					count = 2
					if err := os.MkdirAll(filepath.Join(logsDirectory, proposed.Name, "storage-1.log"), 0700); err != nil {
						t.Fatal(err)
					}
				}
				if tc.save == managedNoSave {
					existing = 1
				}
				plan := testStartPlan(t, proposed, count, existing, m)
				cmd := managementTestCommand(t)
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				cmd.SetContext(ctx)
				restore := managedOperationSignals(cmd)
				defer restore()
				cause := errors.New("injected transaction failure")
				awaitManagedStartup = func(_ *cobra.Command, actual ClusterConnection, children []*managedProcess) error {
					if actual.ConnectionURL != managementURL(m, proposed.ManagementPort) {
						t.Fatal("readiness used old URL")
					}
					if len(children) != 1 {
						t.Fatal("lost launched children")
					}
					if tc.cancelAt == managementTestReadiness || tc.cancelAt == "before-save" {
						cancel()
					}
					if tc.failure == managementTestReadiness {
						return cause
					}
					if tc.cancelAt == managementTestReadiness {
						return ctx.Err()
					}
					return nil
				}
				writes, kills := 0, 0
				renameManagedConfig = func(from, to string) error {
					writes++
					if tc.cancelAt == lifecycleSavePhase {
						cancel()
					}
					if tc.failure == lifecycleSavePhase {
						return cause
					}
					return os.Rename(from, to)
				}
				nativeKill := killManagedProcess
				killManagedProcess = func(p *os.Process) error {
					kills++
					if p == control.command.Process {
						t.Fatal("cleanup targeted preexisting member")
					}
					if cleanupFails {
						cleanupTestRescue(t, p)
						return os.ErrPermission
					}
					return nativeKill(p)
				}
				children, err := runPreparedStartup(cmd, plan, tc.save)
				cleanupTestReap(t, children)
				restore()
				if cmd.Context() != ctx {
					t.Fatal("caller context not restored")
				}
				if len(children) != 1 {
					t.Fatalf("collected %d children: %v", len(children), err)
				}
				assertManagedChildAlive(t, control)
				if failure {
					if err == nil {
						t.Fatal("failure accepted")
					}
					if tc.failure == managementTestReadiness || tc.failure == lifecycleSavePhase {
						if !errors.Is(err, cause) {
							t.Fatal("lost transaction failure", err)
						}
					} else if tc.cancelAt != "" {
						if !errors.Is(err, context.Canceled) {
							t.Fatal("lost cancellation", err)
						}
					} else if !strings.Contains(err.Error(), "unable to start member storage-1") {
						t.Fatal("lost partial launch failure", err)
					}
					if kills != 1 {
						t.Fatalf("cleanup requests=%d", kills)
					}
					if cleanupFails {
						if !errors.Is(err, os.ErrPermission) || !errors.Is(err, context.DeadlineExceeded) {
							t.Fatal("lost cleanup failures", err)
						}
						assertCleanupDiagnostic(t, err, children[0], managementTestKill)
						assertCleanupDiagnostic(t, err, children[0], "wait for")
					} else {
						waitStopTestChild(t, children[0])
					}
				} else {
					if err != nil || kills != 0 {
						t.Fatalf("accepted startup: cleanup=%d error=%v", kills, err)
					}
					cancel()
					restore()
					assertManagedChildAlive(t, children[0])
				}
				wantWrites := 0
				if tc.failure == lifecycleSavePhase || !failure && tc.save != managedNoSave {
					wantWrites = 1
				}
				if writes != wantWrites {
					t.Fatalf("writes=%d want %d", writes, wantWrites)
				}
				if failure || tc.save == managedNoSave {
					after, _ := os.ReadFile(filename)
					if !bytes.Equal(before, after) || !reflect.DeepEqual(Config, oldConfig) || !reflect.DeepEqual(viper.AllSettings(), oldSettings) {
						t.Fatal("failure/scale changed disk or live settings")
					}
				} else {
					_, loaded := readManagedPersistence(t, filename, managementTestJSON)
					if !reflect.DeepEqual(loaded, Config) {
						t.Fatal("accepted save did not round trip")
					}
					found, saved := GetClusterConnection(plan.connection.Name)
					if !found || !reflect.DeepEqual(saved, plan.connection) {
						t.Fatal("accepted proposed record missing")
					}
				}
				assertNoManagedReplacement(t, filename)
			})
		}
	}
}
