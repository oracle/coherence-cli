// Copyright (c) 2026 Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at
// https://oss.oracle.com/licenses/upl.

//go:build !windows

package cmd

import "os"

func restrictMigrationFile(file migrationFile, mode os.FileMode) error {
	return file.Chmod(mode.Perm() & 0600)
}

func replaceMigrationFile(staged, target string) error {
	// Both files are in the same directory/filesystem: rename replaces atomically.
	return os.Rename(staged, target)
}

func syncMigrationDirectory(dir string) error {
	file, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()
	return file.Sync()
}
