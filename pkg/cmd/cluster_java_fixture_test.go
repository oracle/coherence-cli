// Copyright (c) 2026 Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at
// https://oss.oracle.com/licenses/upl.

package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

const javaVersion17 = "17.0.21"
const windowsEndpointLaunchKey = "COHCTL_ENDPOINT_TEST_LAUNCH"

type javaFixture struct {
	Version                                string
	RawOutput, Query                       bool
	Exit                                   int
	Sleep                                  bool
	Marker                                 string
	Cleanup, QueryMarker, CaptureArguments bool
}

func fakeJava(t *testing.T, mode javaFixture) {
	t.Helper()
	if runtime.GOOS == managementTestWindows {
		installWindowsJavaFixture(t, mode)
		return
	}
	// Render the same constrained specification for POSIX. Fixture values are
	// quoted as literals; only the fixed, reviewed actions below are executable.
	quote := func(s string) string { return "'" + strings.ReplaceAll(s, "'", "'\"'\"'") + "'" }
	var script strings.Builder
	script.WriteString("#!/bin/sh\n")
	if mode.QueryMarker {
		script.WriteString("printf 'query\\n' >> \"$COHCTL_TEST_JAVA_VERSION_QUERIES\"\n")
	}
	if mode.Query {
		script.WriteString("if [ \"$1\" = \"-version\" ]; then\n")
	}
	if mode.Version != "" {
		output := mode.Version
		if !mode.RawOutput {
			output = fmt.Sprintf("openjdk version %q", mode.Version)
		}
		fmt.Fprintf(&script, "printf '%%s\\n' %s >&2\n", quote(output))
	}
	if mode.Query || mode.Version != "" {
		fmt.Fprintf(&script, "exit %d\n", mode.Exit)
	}
	if mode.Query {
		script.WriteString("fi\n")
	}
	if mode.Marker != "" {
		switch mode.Marker {
		case "COHCTL_RESTART_MARKER", "COHCTL_STOP_TEST_LAUNCH", windowsEndpointLaunchKey:
			fmt.Fprintf(&script, "printf launched > \"$%s\"\n", mode.Marker)
		default:
			t.Fatal("unknown Java fixture marker", mode.Marker)
		}
	}
	if mode.Cleanup {
		script.WriteString("exec \"$COHCTL_CLEANUP_EXE\" -test.run=^TestManagementCleanupChild$\n")
	}
	if mode.CaptureArguments {
		script.WriteString("printf '%s\\n' \"$@\"\n")
	}
	if mode.Sleep {
		script.WriteString("exec /bin/sleep 30\n")
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "java"), []byte(script.String()), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(filepath.Join(dir, "java"), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
}

func TestManagementJavaFixtureQueryMarker(t *testing.T) {
	marker := filepath.Join(t.TempDir(), "queries")
	t.Setenv("COHCTL_TEST_JAVA_VERSION_QUERIES", marker)
	fakeJava(t, javaFixture{Version: javaVersion17, Query: true, QueryMarker: true})
	for invocation := 1; invocation <= 2; invocation++ {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		child := exec.CommandContext(ctx, "java", "-version")
		output, err := child.CombinedOutput()
		if err != nil || ctx.Err() != nil {
			t.Fatalf("query %d failed or required rescue: error=%v context=%v output=%s", invocation, err, ctx.Err(), output)
		}
		if string(output) != "openjdk version \"17.0.21\"\n" {
			t.Fatalf("query %d: unexpected version output: %q", invocation, output)
		}
		body, err := os.ReadFile(marker)
		if err != nil || string(body) != strings.Repeat("query\n", invocation) {
			t.Fatalf("query %d: marker=%q error=%v", invocation, body, err)
		}
	}
}
