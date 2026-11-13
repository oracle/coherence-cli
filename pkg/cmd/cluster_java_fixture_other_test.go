// Copyright (c) 2026 Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at
// https://oss.oracle.com/licenses/upl.

//go:build !windows

package cmd

import "testing"

func installWindowsJavaFixture(t *testing.T, _ javaFixture) {
	t.Helper()
	t.Fatal("native Windows fixture selected on another platform")
}
