// Copyright (c) 2026 Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at
// https://oss.oracle.com/licenses/upl.

package cmd

import (
	"context"
	"errors"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"go.uber.org/zap"
)

const (
	rootLogTestInitial    = "original CLI output\n"
	rootLogTestAppend     = "appended CLI output\n"
	rootLogTestOutside    = "outside log sentinel\n"
	rootLogTestHelper     = "COHCTL_ROOT_LOG_TEST_HELPER"
	rootLogTestMarker     = "root log integration marker"
	rootLogTestFragment   = "fragment"
	rootLogTestDangling   = "relative-dangling"
	rootLogTestUnwritable = "unwritable-log"
)

func rootLogTestWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
}

func rootLogTestRequire(t *testing.T, path, content string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil || string(data) != content {
		t.Fatalf("%s = %q, err=%v; want %q", path, data, err, content)
	}
}

func rootLogTestURL(t *testing.T) *url.URL {
	t.Helper()
	u, err := url.Parse(winFileSinkURL)
	if err != nil {
		t.Fatal(err)
	}
	return u
}

func rootLogTestAppendToSink(t *testing.T, factory func(*url.URL) (zap.Sink, error)) {
	t.Helper()
	sink, err := factory(rootLogTestURL(t))
	if err != nil {
		t.Fatal(err)
	}
	_, writeErr := sink.Write([]byte(rootLogTestAppend))
	syncErr := sink.Sync()
	closeErr := sink.Close()
	if err := errors.Join(writeErr, syncErr, closeErr); err != nil {
		t.Fatal(err)
	}
}

func rootLogTestFactory(t *testing.T, directory string) func(*url.URL) (zap.Sink, error) {
	t.Helper()
	factory, err := newWinFileSink(directory)
	if err != nil {
		t.Fatal(err)
	}
	return factory
}

func rootLogTestSymlink(t *testing.T, target, link string) {
	t.Helper()
	if err := os.Symlink(target, link); err != nil {
		if runtime.GOOS == managementTestWindows && os.IsPermission(err) {
			t.Skipf("symlink privilege unavailable: %v", err)
		}
		t.Fatal(err)
	}
}

func TestRootLogLiteralDirectories(t *testing.T) {
	for _, tt := range []struct{ name, component, decoded string }{
		{"ordinary", "chosen", ""},
		{"spaces", "chosen dir", ""},
		{"unicode", "chosen-ž", ""},
		{"percent-parent", "%2e%2e", ""},
		{"percent-space", "chosen%20dir", "chosen dir"},
		{rootLogTestFragment, "chosen#tail", ""},
		{"invalid-escape", "chosen%dir", ""},
	} {
		t.Run(tt.name, func(t *testing.T) {
			base := t.TempDir()
			holder := filepath.Join(base, "holder")
			directory := filepath.Join(holder, tt.component)
			if err := os.MkdirAll(directory, 0700); err != nil {
				t.Fatal(err)
			}
			outside := filepath.Join(base, logFile)
			if tt.name == rootLogTestFragment {
				outside = filepath.Join(holder, "chosen")
			}
			if tt.decoded != "" {
				decoded := filepath.Join(holder, tt.decoded)
				if err := os.Mkdir(decoded, 0700); err != nil {
					t.Fatal(err)
				}
				outside = filepath.Join(decoded, logFile)
			}
			rootLogTestWrite(t, outside, rootLogTestOutside)
			path := filepath.Join(directory, logFile)
			rootLogTestWrite(t, path, rootLogTestInitial)
			factory := rootLogTestFactory(t, directory)
			rootLogTestAppendToSink(t, factory)
			rootLogTestAppendToSink(t, factory)
			rootLogTestRequire(t, path, rootLogTestInitial+rootLogTestAppend+rootLogTestAppend)
			rootLogTestRequire(t, outside, rootLogTestOutside)
		})
	}
}

func TestRootLogRejectsMalformedURLs(t *testing.T) {
	for _, raw := range []string{
		"", "file:///cohctl.log", "winfile://server/cohctl.log", "winfile://user@/cohctl.log",
		"winfile:cohctl.log", "winfile:/cohctl.log", "winfile:///other.log", "winfile:///../cohctl.log",
		"winfile:///%2e%2e/cohctl.log", "winfile:///%63ohctl.log", "winfile:///cohctl.log?",
		"winfile:///cohctl.log?option=value", "winfile:///cohctl.log#fragment",
	} {
		t.Run(raw, func(t *testing.T) {
			// A missing root proves validation precedes even the root open.
			directory := filepath.Join(t.TempDir(), "absent")
			factory := rootLogTestFactory(t, directory)
			u, err := url.Parse(raw)
			if err != nil {
				t.Fatal(err)
			}
			sink, err := factory(u)
			if sink != nil {
				if closeErr := sink.Close(); closeErr != nil {
					t.Error(closeErr)
				}
				t.Fatal("malformed URL opened a sink")
			}
			if err == nil || !strings.Contains(err.Error(), "invalid Windows log sink URL") {
				t.Fatalf("unexpected validation result: %v", err)
			}
			if _, err := os.Stat(directory); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("validation created root: %v", err)
			}
		})
	}
	for _, tt := range []struct {
		name string
		u    *url.URL
	}{
		{"nil", nil},
		{"nul", &url.URL{Scheme: winFileSinkScheme, Path: "/cohctl.log\x00"}},
		{"raw-fragment", &url.URL{Scheme: winFileSinkScheme, Path: "/" + logFile, RawFragment: rootLogTestFragment}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			factory := rootLogTestFactory(t, t.TempDir())
			if sink, err := factory(tt.u); sink != nil || err == nil {
				t.Fatalf("invalid URL accepted: sink=%v err=%v", sink, err)
			}
		})
	}
}

func TestRootLogFactorySelections(t *testing.T) {
	base := t.TempDir()
	first, second := filepath.Join(base, "first"), filepath.Join(base, "second")
	for _, directory := range []string{first, second} {
		if err := os.Mkdir(directory, 0700); err != nil {
			t.Fatal(err)
		}
	}
	t.Chdir(base)
	firstFactory := rootLogTestFactory(t, filepath.Base(first))
	secondFactory := rootLogTestFactory(t, second)
	t.Chdir(second)
	old := logFilePath
	logFilePath = filepath.Join(second, logFile)
	t.Cleanup(func() { logFilePath = old })
	rootLogTestAppendToSink(t, firstFactory)
	rootLogTestAppendToSink(t, secondFactory)
	rootLogTestRequire(t, filepath.Join(first, logFile), rootLogTestAppend)
	rootLogTestRequire(t, filepath.Join(second, logFile), rootLogTestAppend)
	selected := filepath.Join(base, "selected-root")
	rootLogTestSymlink(t, first, selected)
	rootLogTestAppendToSink(t, rootLogTestFactory(t, selected))
	rootLogTestRequire(t, filepath.Join(first, logFile), rootLogTestAppend+rootLogTestAppend)
}

func TestRootLogFileModes(t *testing.T) {
	directory := t.TempDir()
	factory := rootLogTestFactory(t, directory)
	rootLogTestAppendToSink(t, factory)
	path := filepath.Join(directory, logFile)
	rootLogTestRequire(t, path, rootLogTestAppend)
	if runtime.GOOS != managementTestWindows {
		// Compare with the requested mode under the same process umask.
		reference := filepath.Join(directory, "mode-reference")
		f, err := os.OpenFile(reference, os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			t.Fatal(err)
		}
		if err := f.Close(); err != nil {
			t.Fatal(err)
		}
		want, err := os.Stat(reference)
		if err != nil {
			t.Fatal(err)
		}
		got, err := os.Stat(path)
		if err != nil || got.Mode().Perm() != want.Mode().Perm() {
			t.Fatalf("new sink mode: got=%v want=%v err=%v", got, want, err)
		}
		if err := os.Chmod(path, 0600); err != nil {
			t.Fatal(err)
		}
		rootLogTestAppendToSink(t, factory)
		got, err = os.Stat(path)
		if err != nil || got.Mode().Perm() != 0600 {
			t.Fatalf("existing mode changed: %v, %v", got, err)
		}
	}
}

func TestRootLogSymlinkBoundary(t *testing.T) {
	for _, mode := range []string{"relative-contained", rootLogTestDangling, "absolute-contained", "absolute-outside", "relative-outside", "loop"} {
		t.Run(mode, func(t *testing.T) {
			base := t.TempDir()
			directory, outside := filepath.Join(base, "selected"), filepath.Join(base, "outside")
			for _, path := range []string{directory, outside} {
				if err := os.Mkdir(path, 0700); err != nil {
					t.Fatal(err)
				}
			}
			insideFile := filepath.Join(directory, "actual.log")
			outsideFile := filepath.Join(outside, logFile)
			rootLogTestWrite(t, insideFile, rootLogTestInitial)
			rootLogTestWrite(t, outsideFile, rootLogTestOutside)
			target := filepath.Base(insideFile)
			allowed := mode == "relative-contained" || mode == rootLogTestDangling
			switch mode {
			case rootLogTestDangling:
				target = "new.log"
			case "absolute-contained":
				target = insideFile
			case "absolute-outside":
				target = outsideFile
			case "relative-outside":
				target = filepath.Join("..", filepath.Base(outside), logFile)
			case "loop":
				target = logFile
			}
			rootLogTestSymlink(t, target, filepath.Join(directory, logFile))
			factory := rootLogTestFactory(t, directory)
			if allowed {
				rootLogTestAppendToSink(t, factory)
				want := rootLogTestInitial + rootLogTestAppend
				if mode == rootLogTestDangling {
					insideFile, want = filepath.Join(directory, target), rootLogTestAppend
				}
				rootLogTestRequire(t, insideFile, want)
			} else if sink, err := factory(rootLogTestURL(t)); sink != nil || err == nil {
				t.Fatalf("escaping/looping link accepted: sink=%v err=%v", sink, err)
			} else {
				rootLogTestRequire(t, insideFile, rootLogTestInitial)
			}
			rootLogTestRequire(t, outsideFile, rootLogTestOutside)
		})
	}
}

func TestRootLogIOErrors(t *testing.T) {
	if factory, err := newWinFileSink(""); factory != nil || err == nil {
		t.Fatal("empty directory accepted")
	}
	for _, mode := range []string{"missing-root", "file-root", "directory-log", rootLogTestUnwritable} {
		t.Run(mode, func(t *testing.T) {
			directory := t.TempDir()
			switch mode {
			case "missing-root":
				directory = filepath.Join(directory, "absent")
			case "file-root":
				directory = filepath.Join(directory, "regular-file")
				rootLogTestWrite(t, directory, rootLogTestOutside)
			case "directory-log":
				if err := os.Mkdir(filepath.Join(directory, logFile), 0700); err != nil {
					t.Fatal(err)
				}
			case rootLogTestUnwritable:
				if runtime.GOOS == managementTestWindows || os.Geteuid() == 0 {
					t.Skip("Unix unprivileged permission control")
				}
				path := filepath.Join(directory, logFile)
				rootLogTestWrite(t, path, rootLogTestOutside)
				if err := os.Chmod(path, 0400); err != nil {
					t.Fatal(err)
				}
			}
			factory := rootLogTestFactory(t, directory)
			for range 2 {
				if sink, err := factory(rootLogTestURL(t)); sink != nil || err == nil {
					t.Fatalf("I/O error lost: sink=%v err=%v", sink, err)
				}
			}
			if mode == rootLogTestUnwritable {
				rootLogTestRequire(t, filepath.Join(directory, logFile), rootLogTestOutside)
			}
		})
	}
}

func TestRootLogOpenedSinkSurvivesDirectoryRename(t *testing.T) {
	base := t.TempDir()
	directory, moved := filepath.Join(base, "selected"), filepath.Join(base, "moved")
	if err := os.Mkdir(directory, 0700); err != nil {
		t.Fatal(err)
	}
	rootLogTestWrite(t, filepath.Join(directory, logFile), rootLogTestInitial)
	sink, err := rootLogTestFactory(t, directory)(rootLogTestURL(t))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if sink != nil {
			if err := sink.Close(); err != nil {
				t.Error(err)
			}
		}
	})
	if err := os.Rename(directory, moved); err != nil {
		if runtime.GOOS == managementTestWindows {
			t.Skipf("directory rename with an open log sink unavailable: %v", err)
		}
		t.Fatal(err)
	}
	if err := os.Mkdir(directory, 0700); err != nil {
		t.Fatal(err)
	}
	rootLogTestWrite(t, filepath.Join(directory, logFile), rootLogTestOutside)
	_, writeErr := sink.Write([]byte(rootLogTestAppend))
	closeErr := sink.Close()
	sink = nil
	if err := errors.Join(writeErr, closeErr); err != nil {
		t.Fatal(err)
	}
	rootLogTestRequire(t, filepath.Join(moved, logFile), rootLogTestInitial+rootLogTestAppend)
	rootLogTestRequire(t, filepath.Join(directory, logFile), rootLogTestOutside)
}

func TestRootLogZapIntegration(t *testing.T) {
	if directory := os.Getenv(rootLogTestHelper); directory != "" {
		var logger *zap.Logger
		var err error
		if isWindows() {
			logger, err = initLogging(directory)
		} else {
			factory := rootLogTestFactory(t, directory)
			if err := zap.RegisterSink(winFileSinkScheme, factory); err != nil {
				t.Fatal(err)
			}
			cfg := zap.NewDevelopmentConfig()
			cfg.OutputPaths = []string{winFileSinkURL}
			logger, err = cfg.Build()
		}
		if err != nil {
			t.Fatal(err)
		}
		logger.Info(rootLogTestMarker)
		if err := logger.Sync(); err != nil {
			t.Fatal(err)
		}
		return
	}
	// The subprocess owns Zap's one-time registration; no registry state leaks
	// into other command-package tests or repeated test runs.
	base := t.TempDir()
	directory := filepath.Join(base, "%2e%2e#chosen%dir")
	if err := os.Mkdir(directory, 0700); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(base, logFile)
	rootLogTestWrite(t, outside, rootLogTestOutside)
	binary, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, binary, "-test.run=^TestRootLogZapIntegration$", "-test.count=1")
	command.Env = append(os.Environ(), rootLogTestHelper+"="+directory)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("isolated Zap integration: %v\n%s", err, output)
	}
	data, err := os.ReadFile(filepath.Join(directory, logFile))
	if err != nil || strings.Count(string(data), rootLogTestMarker) != 1 {
		t.Fatalf("Zap output = %q, err=%v", data, err)
	}
	rootLogTestRequire(t, outside, rootLogTestOutside)
}
