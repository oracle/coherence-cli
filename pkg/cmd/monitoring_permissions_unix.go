// Copyright (c) 2026 Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at
// https://oss.oracle.com/licenses/upl.

//go:build !windows

package cmd

import (
	"fmt"
	"os"
	"syscall"
)

func validateMonitoringSecretFile(file string) error {
	info, err := os.Lstat(file)
	if err != nil {
		return fmt.Errorf("unable to inspect Grafana password file %s: %w", file, err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return fmt.Errorf("Grafana password file %s must be a regular, non-symbolic-link file", file)
	}
	if info.Mode().Perm()&0077 != 0 {
		return fmt.Errorf("Grafana password file %s has insecure permissions %04o; expected no group or other access", file, info.Mode().Perm())
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if ok && int64(stat.Uid) != int64(os.Geteuid()) {
		return fmt.Errorf("Grafana password file %s is owned by user ID %d, expected %d", file, stat.Uid, os.Geteuid())
	}
	contents, err := os.ReadFile(file)
	if err != nil {
		return fmt.Errorf("Grafana password file %s is not readable: %w", file, err)
	}
	if err = validateGrafanaPasswordContents(contents); err != nil {
		return fmt.Errorf("Grafana password file %s is invalid: %w", file, err)
	}
	return nil
}
