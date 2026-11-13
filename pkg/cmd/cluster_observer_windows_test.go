// Copyright (c) 2026 Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at
// https://oss.oracle.com/licenses/upl.

package cmd

import (
	"errors"
	"os"
	"testing"

	"golang.org/x/sys/windows"
)

// Add only a deny entry to a test-owned process object, retaining its original
// full-access owner handle for restoration/rescue. No account or global ACL.
func TestManagementWindowsNativePermissionFailure(t *testing.T) {
	for _, permission := range []uint32{windows.PROCESS_QUERY_INFORMATION, windows.PROCESS_TERMINATE} {
		t.Run(map[uint32]string{windows.PROCESS_QUERY_INFORMATION: "lookup", windows.PROCESS_TERMINATE: "kill"}[permission], func(t *testing.T) {
			protected, later := stopTestChild(t), stopTestChild(t)
			owner := protected.command.Process
			user, err := windows.GetCurrentProcessToken().GetTokenUser()
			if err != nil {
				t.Fatal(err)
			}
			var original *windows.SECURITY_DESCRIPTOR
			var aclErr error
			err = owner.WithHandle(func(handle uintptr) {
				original, aclErr = windows.GetSecurityInfo(windows.Handle(handle), windows.SE_KERNEL_OBJECT, windows.DACL_SECURITY_INFORMATION)
				if aclErr != nil {
					return
				}
				old, _, e := original.DACL()
				if e != nil {
					aclErr = e
					return
				}
				denied, e := windows.ACLFromEntries([]windows.EXPLICIT_ACCESS{{
					AccessPermissions: windows.ACCESS_MASK(permission), AccessMode: windows.DENY_ACCESS,
					Trustee: windows.TRUSTEE{TrusteeForm: windows.TRUSTEE_IS_SID, TrusteeType: windows.TRUSTEE_IS_USER,
						TrusteeValue: windows.TrusteeValueFromSID(user.User.Sid)},
				}}, old)
				if e != nil {
					aclErr = e
					return
				}
				aclErr = windows.SetSecurityInfo(windows.Handle(handle), windows.SE_KERNEL_OBJECT, windows.DACL_SECURITY_INFORMATION, nil, nil, denied, nil)
			})
			if err != nil || aclErr != nil {
				t.Fatalf("test-owned deny setup: %v %v", err, aclErr)
			}
			defer func() {
				var restoreErr error
				err := owner.WithHandle(func(handle uintptr) {
					old, _, e := original.DACL()
					if e == nil {
						e = windows.SetSecurityInfo(windows.Handle(handle), windows.SE_KERNEL_OBJECT, windows.DACL_SECURITY_INFORMATION, nil, nil, old, nil)
					}
					restoreErr = e
				})
				if err != nil || restoreErr != nil {
					t.Errorf("restore owned process DACL: %v %v", err, restoreErr)
				}
			}()
			process, lookupErr := os.FindProcess(owner.Pid)
			if process != nil {
				defer func() {
					if err := process.Release(); err != nil {
						t.Errorf("release native Windows process reference: %v", err)
					}
				}()
			}
			denial := lookupErr
			if denial == nil {
				denial = process.Kill()
			}
			if !errors.Is(denial, windows.ERROR_ACCESS_DENIED) {
				t.Fatalf("actual permission control must deny access; got %v (privileged bypass is not a pass)", denial)
			}
			if processAlreadyExited(owner.Pid, process, denial) {
				t.Fatal("genuine native access denial classified as exit")
			}
			assertManagedChildAlive(t, protected)
			if err := stopClusterProcesses(managementTestCommand(t), managementFixtureName, []int{owner.Pid, later.command.Process.Pid}); !errors.Is(err, windows.ERROR_ACCESS_DENIED) {
				t.Fatalf("native partial stop lost permission error: %v", err)
			}
			waitStopTestChild(t, later)
			assertManagedChildAlive(t, protected)
			t.Logf("native deny mask=%#x owner PID=%d error=%v; later PID=%d stopped", permission, owner.Pid, denial, later.command.Process.Pid)
		})
	}
}
