// Copyright (c) 2026 Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at
// https://oss.oracle.com/licenses/upl.

//go:build windows

package cmd

import (
	"fmt"
	"os"
)

func validateMonitoringSecretFile(file string) error {
	info, err := os.Lstat(file)
	if err != nil {
		return fmt.Errorf("unable to inspect Grafana password file %s: %w", file, err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return fmt.Errorf("Grafana password file %s must be a regular, non-symbolic-link file", file)
	}
	// The file inherits the current user's ACL on Windows. POSIX permission bits
	// are not meaningful there, so regular-file and reparse checks are the
	// portable validations available through os.FileInfo.
	contents, err := os.ReadFile(file)
	if err != nil {
		return fmt.Errorf("Grafana password file %s is not readable: %w", file, err)
	}
	if err = validateGrafanaPasswordContents(contents); err != nil {
		return fmt.Errorf("Grafana password file %s is invalid: %w", file, err)
	}
	return nil
}
