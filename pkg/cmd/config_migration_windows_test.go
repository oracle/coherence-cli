/*
 * Copyright (c) 2026 Oracle and/or its affiliates.
 * Licensed under the Universal Permissive License v 1.0 as shown at
 * https://oss.oracle.com/licenses/upl.
 */

package cmd

import (
	"bytes"
	"os"
	"testing"

	"golang.org/x/sys/windows"
)

func assertMigrationPermissions(t *testing.T, file string, _ os.FileMode) {
	t.Helper()
	user, err := windows.GetCurrentProcessToken().GetTokenUser()
	if err != nil {
		t.Fatal(err)
	}
	want, err := windows.SecurityDescriptorFromString("D:P(A;;FA;;;" + user.User.Sid.String() + ")")
	if err != nil {
		t.Fatal(err)
	}
	actual, err := windows.GetNamedSecurityInfo(file, windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION)
	if err != nil || actual.String() != want.String() {
		t.Fatalf("migration file DACL is not protected for the current user: %v, %v", actual, err)
	}
}

func TestLegacyTLSMigrationLockedWindowsTarget(t *testing.T) {
	file := writeMigrationFixture(t, migrationFixture)
	name, err := windows.UTF16PtrFromString(file)
	if err != nil {
		t.Fatal(err)
	}
	// Deny delete/replace sharing while still allowing the migration to read it.
	handle, err := windows.CreateFile(name, windows.GENERIC_READ, windows.FILE_SHARE_READ, nil, windows.OPEN_EXISTING, windows.FILE_ATTRIBUTE_NORMAL, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = windows.CloseHandle(handle) }()
	result, err := migrateLegacyTLSConfig(file, defaultConfigMigrationOps())
	if err == nil || result != nil || !bytes.Equal(readMigrationBytes(t, file), []byte(migrationFixture)) {
		t.Fatalf("locked-target replacement did not fail safely: %v", err)
	}
	backups := migrationBackups(t, file)
	if len(backups) != 1 || !bytes.Equal(readMigrationBytes(t, backups[0]), []byte(migrationFixture)) {
		t.Fatal("locked-target failure lost its recovery backup")
	}
	assertMigrationPermissions(t, backups[0], 0600)
}
