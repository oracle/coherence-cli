// Copyright (c) 2026, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at
// https://oss.oracle.com/licenses/upl.

//go:build windows

package cmd

import "golang.org/x/sys/windows"

// Winsock bind errors use WSAEADDRINUSE, not syscall.EADDRINUSE.
const errManagementAddressInUse = windows.WSAEADDRINUSE
