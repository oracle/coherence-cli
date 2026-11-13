/*
 * Copyright (c) 2026 Oracle and/or its affiliates.
 * Licensed under the Universal Permissive License v 1.0 as shown at
 * https://oss.oracle.com/licenses/upl.
 */

package cmd

import (
	"os"

	"golang.org/x/sys/windows"
)

func restrictMigrationFile(file migrationFile, _ os.FileMode) error {
	// Windows Chmod(0600) does not restrict access. Protect the empty file with a
	// DACL granting access only to the current user, before writing credentials.
	user, err := windows.GetCurrentProcessToken().GetTokenUser()
	if err != nil {
		return err
	}
	descriptor, err := windows.SecurityDescriptorFromString("D:P(A;;FA;;;" + user.User.Sid.String() + ")")
	if err != nil {
		return err
	}
	dacl, _, err := descriptor.DACL()
	if err != nil {
		return err
	}
	// os.CreateTemp's handle does not request WRITE_DAC. SetNamedSecurityInfo
	// opens with that right; the current empty-file handle prevents delete sharing.
	return windows.SetNamedSecurityInfo(file.Name(), windows.SE_FILE_OBJECT,
		windows.DACL_SECURITY_INFORMATION|windows.PROTECTED_DACL_SECURITY_INFORMATION, nil, nil, dacl, nil)
}

func replaceMigrationFile(staged, target string) error {
	from, err := windows.UTF16PtrFromString(staged)
	if err != nil {
		return err
	}
	to, err := windows.UTF16PtrFromString(target)
	if err != nil {
		return err
	}
	// Same-volume replacement, with no remove-first fallback. Windows does not
	// promise Unix rename atomicity; the flushed original backup is retained.
	return windows.MoveFileEx(from, to, windows.MOVEFILE_REPLACE_EXISTING|windows.MOVEFILE_WRITE_THROUGH)
}

func syncMigrationDirectory(_ string) error {
	// Windows directory handles cannot be synced with os.File.Sync; replacement
	// uses MOVEFILE_WRITE_THROUGH and each backup/staging file is synced separately.
	return nil
}
