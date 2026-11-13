// Copyright (c) 2026 Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at
// https://oss.oracle.com/licenses/upl.

package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

const (
	logTestCluster = "log-cluster"
	logTestMember  = "storage-0"
	logTestFile    = logTestMember + ".log"
	logTestBackup  = logTestFile + ".old"
	logTestContent = "previous member output\n"
	logTestOutside = "outside sentinel\n"

	logTestClusterComponent = "cluster"
	logTestAbsolutePath     = "/absolute"
	logTestEnabled          = "enabled"
	logTestDisabled         = "disabled"
	logTestWriteFailure     = "write-failure"
)

func clusterLogTestDirectory(t *testing.T) string {
	t.Helper()
	old := logsDirectory
	logsDirectory = t.TempDir()
	t.Cleanup(func() { logsDirectory = old })
	return logsDirectory
}

func writeClusterLogTestFile(t *testing.T, name, content string) {
	t.Helper()
	if err := os.WriteFile(name, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
}

func requireClusterLogTestFile(t *testing.T, name, content string) {
	t.Helper()
	data, err := os.ReadFile(name)
	if err != nil || !bytes.Equal(data, []byte(content)) {
		t.Fatalf("file %s = %q, err=%v; want %q", name, data, err, content)
	}
}

func clusterLogTestSymlink(t *testing.T, target, name string) {
	t.Helper()
	if err := os.Symlink(target, name); err != nil {
		if runtime.GOOS == managementTestWindows && os.IsPermission(err) {
			t.Skipf("symlink privilege unavailable: %v", err)
		}
		t.Fatal(err)
	}
}

func TestManagementLogBackupRejectsTraversalBeforeLaunch(t *testing.T) {
	managedTestState(t)
	root := t.TempDir()
	logsDirectory = filepath.Join(root, "logs")
	outside := filepath.Join(root, "outside")
	for _, dir := range []string{logsDirectory, outside} {
		if err := os.Mkdir(dir, 0700); err != nil {
			t.Fatal(err)
		}
	}
	files := map[string]string{logTestFile: logTestContent, logTestBackup: logTestOutside}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(outside, name), []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
	}
	backupLogFilesParam = true
	// Even an unavailable Java executable must not permit earlier log writes.
	t.Setenv("PATH", t.TempDir())
	connection := ClusterConnection{Name: "../outside", ManuallyCreated: true, ClusterVersion: defaultCoherenceVersion}
	children, err := runTestStartup(t, managementTestCommand(t), connection, 1, 0, ClusterManagement{RESTAddress: localHost})
	if err == nil || !strings.Contains(err.Error(), "invalid cluster log path component") || len(children) != 0 {
		t.Fatalf("children=%d, err=%v", len(children), err)
	}
	for name, content := range files {
		data, err := os.ReadFile(filepath.Join(outside, name))
		if err != nil || !bytes.Equal(data, []byte(content)) {
			t.Errorf("outside file %s changed: %q, err=%v", name, data, err)
		}
	}
}

func unsafeClusterLogNames() []string {
	return []string{"", ".", "..", "../escape", "part/../escape", "part/child", logTestAbsolutePath, `..\escape`, `\rooted`, `\\server\share`, `C:\absolute`, "C:relative", "name:stream", "nul\x00name"}
}

func TestManagementLogPathRejectsUnsafeNames(t *testing.T) {
	clusterLogTestDirectory(t)
	if path, err := getLogFile("local-cluster", "storage-0"); err != nil || path != filepath.Join(logsDirectory, "local-cluster", "storage-0.log") {
		t.Fatalf("valid log path=%q error=%v", path, err)
	}

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
				if path, err := getLogFile(cluster, member); err == nil || path != "" {
					t.Fatalf("accepted path %q, err=%v", path, err)
				}
				if backedUp, err := backupClusterLogFile(cluster, member); err == nil || backedUp {
					t.Fatalf("accepted backup: %v, err=%v", backedUp, err)
				}
				entries, err := os.ReadDir(dir)
				if err != nil || len(entries) != 0 {
					t.Fatalf("invalid name caused directory writes: %v, err=%v", entries, err)
				}
			})
		}
	}
}

func TestManagementLogBackupLegitimatePaths(t *testing.T) {
	for _, mode := range []string{"custom-absolute", "relative-root", "symlink-root"} {
		t.Run(mode, func(t *testing.T) {
			dir := clusterLogTestDirectory(t)
			switch mode {
			case "relative-root":
				t.Chdir(filepath.Dir(dir))
				logsDirectory = filepath.Base(dir)
			case "symlink-root":
				logsDirectory = filepath.Join(t.TempDir(), "selected-root")
				clusterLogTestSymlink(t, dir, logsDirectory)
			}
			// Keep punctuation, spaces and non-ASCII names that are local components.
			cluster, member := "cluster.v1-ž_1", "member 0"
			path, err := getLogFile(cluster, member)
			if err != nil || path != filepath.Join(logsDirectory, cluster, member+".log") {
				t.Fatalf("path=%q, err=%v", path, err)
			}
			writeClusterLogTestFile(t, path, logTestContent)
			for _, existing := range []bool{false, true} {
				if existing {
					writeClusterLogTestFile(t, path+".old", logTestOutside)
				}
				backedUp, err := backupClusterLogFile(cluster, member)
				if err != nil || !backedUp {
					t.Fatalf("backedUp=%v, err=%v", backedUp, err)
				}
				requireClusterLogTestFile(t, path, logTestContent)
				requireClusterLogTestFile(t, path+".old", logTestContent)
			}
			if runtime.GOOS != managementTestWindows {
				info, err := os.Stat(path + ".old")
				if err != nil || info.Mode().Perm() != 0600 {
					t.Fatalf("backup permissions: %v, err=%v", info, err)
				}
			}
		})
	}
}

func TestManagementLogBackupMissingAndEmpty(t *testing.T) {
	clusterLogTestDirectory(t)
	path, err := getLogFile(logTestCluster, logTestMember)
	if err != nil {
		t.Fatal(err)
	}
	writeClusterLogTestFile(t, path+".old", logTestOutside)
	if backedUp, err := backupClusterLogFile(logTestCluster, logTestMember); err != nil || backedUp {
		t.Fatalf("missing source: backedUp=%v, err=%v", backedUp, err)
	}
	requireClusterLogTestFile(t, path+".old", logTestOutside)
	writeClusterLogTestFile(t, path, "")
	if backedUp, err := backupClusterLogFile(logTestCluster, logTestMember); err != nil || !backedUp {
		t.Fatalf("empty source: backedUp=%v, err=%v", backedUp, err)
	}
	requireClusterLogTestFile(t, path+".old", "")
}

func TestManagementLogBackupRejectsSymlinkEscapes(t *testing.T) {
	for _, mode := range []string{logTestClusterComponent, "source", "backup", "source-sibling", "backup-sibling", "source-missing-target"} {
		t.Run(mode, func(t *testing.T) {
			dir := clusterLogTestDirectory(t)
			outside := t.TempDir()
			if strings.HasSuffix(mode, "-sibling") {
				outside = filepath.Join(dir, "sibling")
				if err := os.Mkdir(outside, 0700); err != nil {
					t.Fatal(err)
				}
			}
			sentinel := filepath.Join(outside, logTestFile)
			writeClusterLogTestFile(t, sentinel, logTestOutside)
			if mode == logTestClusterComponent {
				clusterLogTestSymlink(t, outside, filepath.Join(dir, logTestCluster))
				if path, err := getLogFile(logTestCluster, logTestMember); err == nil || path != "" {
					t.Fatalf("accepted escaping cluster: %q, %v", path, err)
				}
			} else {
				path, err := getLogFile(logTestCluster, logTestMember)
				if err != nil {
					t.Fatal(err)
				}
				link := path
				if strings.HasPrefix(mode, "backup") {
					writeClusterLogTestFile(t, path, logTestContent)
					link += ".old"
				}
				target := sentinel
				if strings.HasSuffix(mode, "-sibling") {
					target = filepath.Join("..", "sibling", logTestFile)
				} else if mode == "source-missing-target" {
					target = filepath.Join(outside, "missing.log")
				}
				clusterLogTestSymlink(t, target, link)
			}
			if backedUp, err := backupClusterLogFile(logTestCluster, logTestMember); err == nil || backedUp {
				t.Fatalf("accepted escaping %s: backedUp=%v, err=%v", mode, backedUp, err)
			}
			requireClusterLogTestFile(t, sentinel, logTestOutside)
			if strings.HasPrefix(mode, "backup") {
				requireClusterLogTestFile(t, filepath.Join(dir, logTestCluster, logTestFile), logTestContent)
			}
		})
	}
}

func TestManagementLogBackupAllowsContainedSymlinks(t *testing.T) {
	dir := clusterLogTestDirectory(t)
	realCluster := filepath.Join(dir, "real-cluster")
	if err := os.Mkdir(realCluster, 0700); err != nil {
		t.Fatal(err)
	}
	clusterLogTestSymlink(t, filepath.Base(realCluster), filepath.Join(dir, logTestCluster))
	writeClusterLogTestFile(t, filepath.Join(realCluster, "actual.log"), logTestContent)
	clusterLogTestSymlink(t, "actual.log", filepath.Join(realCluster, logTestFile))
	clusterLogTestSymlink(t, "actual.old", filepath.Join(realCluster, logTestBackup))
	if _, err := getLogFile(logTestCluster, logTestMember); err != nil {
		t.Fatal(err)
	}
	if backedUp, err := backupClusterLogFile(logTestCluster, logTestMember); err != nil || !backedUp {
		t.Fatalf("backedUp=%v, err=%v", backedUp, err)
	}
	requireClusterLogTestFile(t, filepath.Join(realCluster, "actual.log"), logTestContent)
	requireClusterLogTestFile(t, filepath.Join(realCluster, "actual.old"), logTestContent)
}

func TestManagementLogBackupPropagatesIOErrors(t *testing.T) {
	for _, mode := range []string{"root-missing", "cluster-file", "source-directory", "backup-directory"} {
		t.Run(mode, func(t *testing.T) {
			dir := clusterLogTestDirectory(t)
			switch mode {
			case "root-missing":
				logsDirectory = filepath.Join(dir, "absent")
			case "cluster-file":
				writeClusterLogTestFile(t, filepath.Join(dir, logTestCluster), logTestOutside)
			default:
				path, err := getLogFile(logTestCluster, logTestMember)
				if err != nil {
					t.Fatal(err)
				}
				if mode == "backup-directory" {
					writeClusterLogTestFile(t, path, logTestContent)
					path += ".old"
				}
				if err := os.Mkdir(path, 0700); err != nil {
					t.Fatal(err)
				}
			}
			if backedUp, err := backupClusterLogFile(logTestCluster, logTestMember); err == nil || backedUp {
				t.Fatalf("lost %s error: backedUp=%v, err=%v", mode, backedUp, err)
			}
		})
	}
}

func TestManagementLogBackupStartupOrdering(t *testing.T) {
	for _, mode := range []string{logTestEnabled, logTestDisabled, logTestWriteFailure} {
		t.Run(mode, func(t *testing.T) {
			managedTestState(t)
			path, err := getLogFile(logTestCluster, logTestMember)
			if err != nil {
				t.Fatal(err)
			}
			writeClusterLogTestFile(t, path, logTestContent)
			if mode == logTestWriteFailure {
				if err := os.Mkdir(path+".old", 0700); err != nil {
					t.Fatal(err)
				}
			}
			backupLogFilesParam = mode != logTestDisabled
			t.Setenv("PATH", t.TempDir())
			connection := ClusterConnection{Name: logTestCluster, ManuallyCreated: true, ClusterVersion: defaultCoherenceVersion}
			children, err := runTestStartup(t, managementTestCommand(t), connection, 1, 0, ClusterManagement{RESTAddress: localHost})
			if err == nil || len(children) != 0 {
				t.Fatalf("children=%d, err=%v", len(children), err)
			}
			switch mode {
			case logTestEnabled:
				requireClusterLogTestFile(t, path+".old", logTestContent)
				requireClusterLogTestFile(t, path, "")
			case logTestDisabled:
				if _, err := os.Lstat(path + ".old"); !os.IsNotExist(err) {
					t.Fatalf("backup created with flag disabled: %v", err)
				}
			case logTestWriteFailure:
				if !strings.Contains(err.Error(), "unable to write backup log file") {
					t.Fatal("backup failure did not stop launch", err)
				}
				requireClusterLogTestFile(t, path, logTestContent)
			}
		})
	}
}
