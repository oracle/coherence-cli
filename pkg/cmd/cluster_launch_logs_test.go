// Copyright (c) 2026 Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at
// https://oss.oracle.com/licenses/upl.

package cmd

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

const (
	launchLogTestMemberComponent   = "member"
	launchLogTestRelativeContained = "relative-contained"
	launchLogTestRelativeOutside   = "relative-outside"
	launchLogTestAbsoluteContained = "absolute-contained"
	launchLogTestAbsoluteOutside   = "absolute-outside"
	launchLogTestLoop              = "loop"
	launchLogTestClusterContained  = "cluster-contained"
	launchLogTestRelativeRoot      = "relative-root"
	launchLogTestSymlinkRoot       = "symlink-root"
	launchLogTestMissingRoot       = "missing-root"
	launchLogTestFileRoot          = "file-root"
	launchLogTestClusterFile       = "cluster-file"

	launchLogTestDisabled = "backup-disabled"
	launchLogTestEnabled  = "backup-enabled"
	launchLogTestSwap     = "backup-then-swap"
	launchLogTestPayload  = "new member output\n"
	launchLogTestHelper   = "COHCTL_LOG_LAUNCH_TEST_CHILD"
	launchLogTestStdout   = "member stdout marker"
	launchLogTestStderr   = "member stderr marker"
)

func TestManagementLogLaunchBackupBoundary(t *testing.T) {
	for _, mode := range []string{launchLogTestDisabled, launchLogTestEnabled, launchLogTestSwap} {
		t.Run(mode, func(t *testing.T) {
			managedTestState(t)
			outside := filepath.Join(t.TempDir(), "outside.log")
			writeClusterLogTestFile(t, outside, logTestOutside)
			path, err := getLogFile(logTestCluster, logTestMember)
			if err != nil {
				t.Fatal(err)
			}
			if mode == launchLogTestSwap {
				writeClusterLogTestFile(t, path, logTestContent)
				if backedUp, err := backupClusterLogFile(logTestCluster, logTestMember); err != nil || !backedUp {
					t.Fatalf("backup=%v, err=%v", backedUp, err)
				}
				if err := os.Remove(path); err != nil {
					t.Fatal(err)
				}
			}
			clusterLogTestSymlink(t, outside, path)
			t.Setenv("PATH", t.TempDir())
			if mode == launchLogTestSwap {
				process, err := launchManagedProcess(logTestCluster, logTestMember, nil)
				if err == nil || process != nil {
					t.Fatalf("process=%v, err=%v", process, err)
				}
				requireClusterLogTestFile(t, path+".old", logTestContent)
			} else {
				backupLogFilesParam = mode == launchLogTestEnabled
				connection := ClusterConnection{Name: logTestCluster, ManuallyCreated: true, ClusterVersion: defaultCoherenceVersion}
				processes, err := runTestStartup(t, managementTestCommand(t), connection, 1, 0, ClusterManagement{RESTAddress: localHost})
				if err == nil || len(processes) != 0 {
					t.Fatalf("processes=%d, err=%v", len(processes), err)
				}
				if _, err := os.Lstat(path + ".old"); !errors.Is(err, os.ErrNotExist) {
					t.Fatalf("unexpected backup after rejected source: %v", err)
				}
			}
			requireClusterLogTestFile(t, outside, logTestOutside)
		})
	}
}

func TestManagementLogLaunchRejectsNames(t *testing.T) {
	for _, name := range unsafeClusterLogNames() {
		for _, component := range []string{logTestClusterComponent, launchLogTestMemberComponent} {
			t.Run(component+"/"+name, func(t *testing.T) {
				dir := clusterLogTestDirectory(t)
				cluster, member := logTestCluster, logTestMember
				if component == logTestClusterComponent {
					cluster = name
				} else {
					member = name
				}
				if file, err := openManagedProcessLog(cluster, member); err == nil || file != nil {
					t.Fatalf("unsafe name opened log: file=%v, err=%v", file, err)
				}
				entries, err := os.ReadDir(dir)
				if err != nil || len(entries) != 0 {
					t.Fatalf("unsafe name changed log directory: %v, err=%v", entries, err)
				}
			})
		}
	}
}

func TestManagementLogLaunchSymlinkBoundary(t *testing.T) {
	for _, mode := range []string{launchLogTestRelativeContained, rootLogTestDangling, launchLogTestRelativeOutside, launchLogTestAbsoluteContained, launchLogTestAbsoluteOutside, launchLogTestLoop, launchLogTestClusterContained, "cluster-outside"} {
		t.Run(mode, func(t *testing.T) {
			dir := clusterLogTestDirectory(t)
			outside := filepath.Join(t.TempDir(), "outside.log")
			writeClusterLogTestFile(t, outside, logTestOutside)
			realCluster := filepath.Join(dir, logTestCluster)
			if strings.HasPrefix(mode, "cluster-") {
				realCluster = filepath.Join(dir, "actual-cluster")
			}
			if err := os.Mkdir(realCluster, 0700); err != nil {
				t.Fatal(err)
			}
			actual := filepath.Join(realCluster, "actual.log")
			writeClusterLogTestFile(t, actual, logTestContent)
			target := filepath.Base(actual)
			allowed := mode == launchLogTestRelativeContained || mode == rootLogTestDangling || mode == launchLogTestClusterContained
			switch mode {
			case rootLogTestDangling:
				target = "new.log"
			case launchLogTestRelativeOutside:
				var err error
				target, err = filepath.Rel(realCluster, outside)
				if err != nil {
					t.Fatal(err)
				}
			case launchLogTestAbsoluteContained:
				target = actual
			case launchLogTestAbsoluteOutside:
				target = outside
			case launchLogTestLoop:
				target = logTestFile
			case launchLogTestClusterContained:
				clusterLogTestSymlink(t, filepath.Base(realCluster), filepath.Join(dir, logTestCluster))
			case "cluster-outside":
				clusterLogTestSymlink(t, filepath.Dir(outside), filepath.Join(dir, logTestCluster))
			}
			clusterLogTestSymlink(t, target, filepath.Join(realCluster, logTestFile))
			file, err := openManagedProcessLog(logTestCluster, logTestMember)
			if allowed {
				if err != nil {
					t.Fatal(err)
				}
				_, writeErr := file.WriteString(launchLogTestPayload)
				if err := errors.Join(writeErr, file.Close()); err != nil {
					t.Fatal(err)
				}
				if mode == rootLogTestDangling {
					actual = filepath.Join(realCluster, target)
				}
				requireClusterLogTestFile(t, actual, launchLogTestPayload)
			} else {
				if err == nil || file != nil {
					t.Fatalf("unsafe link opened log: file=%v, err=%v", file, err)
				}
				requireClusterLogTestFile(t, actual, logTestContent)
			}
			requireClusterLogTestFile(t, outside, logTestOutside)
		})
	}
}

func TestManagementLogLaunchLocationsAndModes(t *testing.T) {
	for _, mode := range []string{"custom-absolute", launchLogTestRelativeRoot, launchLogTestSymlinkRoot} {
		t.Run(mode, func(t *testing.T) {
			dir := clusterLogTestDirectory(t)
			selected := filepath.Join(dir, "selected%2e%2e#ž space")
			if err := os.Mkdir(selected, 0700); err != nil {
				t.Fatal(err)
			}
			logsDirectory = selected
			switch mode {
			case launchLogTestRelativeRoot:
				t.Chdir(dir)
				logsDirectory = filepath.Base(selected)
			case launchLogTestSymlinkRoot:
				logsDirectory = filepath.Join(dir, "selected-root")
				clusterLogTestSymlink(t, selected, logsDirectory)
			}
			cluster, member := "cluster%20#ž", "member 0"
			path := filepath.Join(selected, cluster, member+".log")
			for _, existing := range []bool{false, true} {
				if existing {
					writeClusterLogTestFile(t, path, logTestContent)
					if runtime.GOOS != managementTestWindows {
						if err := os.Chmod(path, 0640); err != nil {
							t.Fatal(err)
						}
					}
				}
				file, err := openManagedProcessLog(cluster, member)
				if err != nil {
					t.Fatal(err)
				}
				requireClusterLogTestFile(t, path, "")
				_, writeErr := file.WriteString(launchLogTestPayload)
				if err := errors.Join(writeErr, file.Close()); err != nil {
					t.Fatal(err)
				}
				requireClusterLogTestFile(t, path, launchLogTestPayload)
				if runtime.GOOS != managementTestWindows {
					want := os.FileMode(0600)
					if existing {
						want = 0640
					}
					info, err := os.Stat(path)
					if err != nil || info.Mode().Perm() != want {
						t.Fatalf("permissions=%v, err=%v; want %v", info, err, want)
					}
				}
			}
		})
	}
}

func TestManagementLogLaunchIOErrors(t *testing.T) {
	for _, mode := range []string{launchLogTestMissingRoot, launchLogTestFileRoot, launchLogTestClusterFile, "log-directory", "unwritable-log"} {
		t.Run(mode, func(t *testing.T) {
			dir := clusterLogTestDirectory(t)
			switch mode {
			case launchLogTestMissingRoot:
				logsDirectory = filepath.Join(dir, "absent")
			case launchLogTestFileRoot:
				logsDirectory = filepath.Join(dir, "regular-file")
				writeClusterLogTestFile(t, logsDirectory, logTestOutside)
			case launchLogTestClusterFile:
				writeClusterLogTestFile(t, filepath.Join(dir, logTestCluster), logTestOutside)
			default:
				path, err := getLogFile(logTestCluster, logTestMember)
				if err != nil {
					t.Fatal(err)
				}
				if mode == "log-directory" {
					if err := os.Mkdir(path, 0700); err != nil {
						t.Fatal(err)
					}
				} else {
					if runtime.GOOS == managementTestWindows || os.Geteuid() == 0 {
						t.Skip("Unix unprivileged permission control")
					}
					writeClusterLogTestFile(t, path, logTestOutside)
					if err := os.Chmod(path, 0400); err != nil {
						t.Fatal(err)
					}
				}
			}
			t.Setenv("PATH", t.TempDir())
			for range 2 {
				process, err := launchManagedProcess(logTestCluster, logTestMember, nil)
				var executableError *exec.Error
				if err == nil || process != nil || errors.As(err, &executableError) {
					t.Fatalf("log failure did not precede executable lookup: process=%v, err=%v", process, err)
				}
			}
			if mode == "unwritable-log" {
				requireClusterLogTestFile(t, filepath.Join(dir, logTestCluster, logTestFile), logTestOutside)
			}
		})
	}
}

func TestManagementLogLaunchOpenedFileSurvivesRename(t *testing.T) {
	dir := clusterLogTestDirectory(t)
	file, err := openManagedProcessLog(logTestCluster, logTestMember)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			t.Error(err)
		}
	}()
	moved := filepath.Join(dir, "moved-cluster")
	if err := os.Rename(filepath.Join(dir, logTestCluster), moved); err != nil {
		if runtime.GOOS == managementTestWindows {
			t.Skipf("directory rename with open log unavailable: %v", err)
		}
		t.Fatal(err)
	}
	outside := t.TempDir()
	sentinel := filepath.Join(outside, logTestFile)
	writeClusterLogTestFile(t, sentinel, logTestOutside)
	clusterLogTestSymlink(t, outside, filepath.Join(dir, logTestCluster))
	if _, err := file.WriteString(launchLogTestPayload); err != nil {
		t.Fatal(err)
	}
	requireClusterLogTestFile(t, filepath.Join(moved, logTestFile), launchLogTestPayload)
	requireClusterLogTestFile(t, sentinel, logTestOutside)
}

func TestManagementLogLaunchChild(_ *testing.T) {
	if os.Getenv(launchLogTestHelper) == "1" {
		fmt.Fprintln(os.Stdout, launchLogTestStdout)
		fmt.Fprintln(os.Stderr, launchLogTestStderr)
	}
}

func TestManagementLogLaunchChildOutput(t *testing.T) {
	managedTestState(t)
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(executable)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	name := javaExec
	if runtime.GOOS == managementTestWindows {
		name += ".exe"
	}
	writeErr := root.WriteFile(name, data, 0700)
	if err := errors.Join(writeErr, root.Close()); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
	t.Setenv(launchLogTestHelper, "1")
	t.Setenv("COHCTL_NATIVE_JAVA_FIXTURE", "")
	process, err := launchManagedProcess(logTestCluster, logTestMember, []string{"-test.run=^TestManagementLogLaunchChild$", "-test.count=1"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = process.command.Process.Kill()
		select {
		case <-process.done:
		case <-time.After(5 * time.Second):
			t.Error("log fixture child was not reaped")
		}
	})
	select {
	case <-process.done:
		if process.waitErr != nil {
			t.Fatal(process.waitErr)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("log fixture child did not exit")
	}
	wantPath := filepath.Join(logsDirectory, logTestCluster, logTestFile)
	if process.logFile != wantPath {
		t.Fatalf("log path=%q; want %q", process.logFile, wantPath)
	}
	output, err := os.ReadFile(wantPath)
	if err != nil || !strings.Contains(string(output), launchLogTestStdout) || !strings.Contains(string(output), launchLogTestStderr) {
		t.Fatalf("member log=%q, err=%v", output, err)
	}
}
