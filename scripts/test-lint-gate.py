#!/usr/bin/env python3
# Copyright (c) 2026 Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at
# https://oss.oracle.com/licenses/upl.

"""Exercise the real Make recipes with controlled local tool failures."""

import json
import os
from pathlib import Path
import subprocess
import tempfile
import unittest


MAKEFILE = Path(__file__).resolve().parents[1] / "Makefile"
FAKE_LINTER = r'''#!/usr/bin/env python3
import json, os, pathlib, sys
args = sys.argv[1:]
with open(os.environ["CALLS"], "a") as log:
    log.write(json.dumps(["linter"] + args) + "\n")
mode = os.environ.get("CONTROL", "")
if args[0] == "version":
    print("wrong-version" if mode == "wrong-version" else "2.15.0")
    sys.exit(19 if mode == "version-failure" else 0)
if args[0] == "fmt":
    sys.exit(17 if mode == "formatting-failure" else 0)
assert args[0] == "run"
assert os.environ.get("CGO_ENABLED") == "0"
assert os.environ.get("GOFLAGS") == ""
assert os.environ.get("GOEXPERIMENT") == ""
assert os.environ.get("GOWORK") == "off"
if mode != "missing-report":
    for arg in args:
        if arg.startswith("--output.json.path="):
            pathlib.Path(arg.split("=", 1)[1]).write_text('{"Issues":null}')
sys.exit(23 if mode == "linter-failure" else 0)
'''
FAKE_GO = r'''#!/usr/bin/env python3
import json, os, sys
args = sys.argv[1:]
if args == ["env", "GOBIN"]:  # Unrelated Makefile initialization.
    print("/fixture/gobin")
    sys.exit(0)
with open(os.environ["CALLS"], "a") as log:
    log.write(json.dumps(["go"] + args) + "\n")
mode = os.environ.get("CONTROL", "")
if args[:1] == ["env"]:
    print("darwin" if args[1] == "GOOS" else "arm64")
    sys.exit(29 if mode == args[1] + "-failure" else 0)
assert args[:1] == ["run"]
assert os.environ.get("GOOS") == ""
assert os.environ.get("GOARCH") == ""
assert os.environ.get("CGO_ENABLED") == "0"
assert os.environ.get("GOFLAGS") == ""
assert os.environ.get("GOEXPERIMENT") == ""
assert os.environ.get("GOWORK") == "off"
assert sys.stdin.read() == '{"Issues":null}'
sys.exit(31 if mode == "checker-failure" else 0)
'''
FAKE_CURL = r'''#!/usr/bin/env python3
import os, pathlib, sys
with open(os.environ["CALLS"], "a") as log:
    log.write('["curl"]\n')
pathlib.Path(sys.argv[sys.argv.index("-o") + 1]).write_text("exit 37\n")
sys.exit(41 if os.environ["CONTROL"] == "download-failure" else 0)
'''


class MakeGateTests(unittest.TestCase):
    def setUp(self):
        self.temporary = tempfile.TemporaryDirectory(prefix="lint-gate-")
        self.addCleanup(self.temporary.cleanup)
        self.root = Path(self.temporary.name)
        self.bin = self.root / "bin"
        self.bin.mkdir()
        self.calls = self.root / "calls.jsonl"
        for name, source in [("golangci-lint", FAKE_LINTER), ("go", FAKE_GO), ("curl", FAKE_CURL)]:
            path = self.bin / name
            path.write_text(source)
            path.chmod(0o700)

    def invoke(self, control="", target="golangci", install=False):
        env = os.environ.copy()
        env.update(PATH=str(self.bin) + os.pathsep + env["PATH"],
                   CALLS=str(self.calls), CONTROL=control, TMPDIR=str(self.root))
        args = ["make", "--no-print-directory", "-f", str(MAKEFILE), target,
                "TOOLS_DIRECTORY=" + str(self.root / "tools"),
                "SOURCE_DATE_EPOCH=1", "GITCOMMIT=fixture"]
        if not install:
            args.append("GOLANGCI_LINT=" + str(self.bin / "golangci-lint"))
        result = subprocess.run(args, cwd=self.root, env=env, capture_output=True,
                                text=True, timeout=20)
        calls = [json.loads(line) for line in self.calls.read_text().splitlines()]
        return result, calls

    def test_collection_success_and_explicit_flags(self):
        result, calls = self.invoke()
        self.assertEqual(result.returncode, 0, result.stderr)
        run = next(call for call in calls if call[:2] == ["linter", "run"])
        self.assertIn("--issues-exit-code=0", run)
        self.assertEqual(run[-1], "./...")
        self.assertEqual(calls[-1][-1], "darwin/arm64")
        self.assertEqual(calls[-1][:2], ["go", "run"])
        self.assertEqual(list(self.root.glob("tmp.*")), [])

    def test_formatting_failure_stops_analysis(self):
        result, calls = self.invoke("formatting-failure")
        self.assertNotEqual(result.returncode, 0)
        self.assertFalse(any(call[:2] == ["linter", "run"] for call in calls))

    def test_tool_failure_after_valid_json_stops_checker(self):
        result, calls = self.invoke("linter-failure")
        self.assertNotEqual(result.returncode, 0)
        self.assertFalse(any(call[:2] == ["go", "run"] for call in calls))
        self.assertEqual(list(self.root.glob("tmp.*")), [])

    def test_missing_report_stops_checker(self):
        result, calls = self.invoke("missing-report")
        self.assertNotEqual(result.returncode, 0)
        self.assertFalse(any(call[:2] == ["go", "run"] for call in calls))

    def test_version_failure_and_mismatch_stop_formatting(self):
        for control in ["version-failure", "wrong-version"]:
            with self.subTest(control=control):
                self.calls.unlink(missing_ok=True)
                result, calls = self.invoke(control)
                self.assertNotEqual(result.returncode, 0)
                self.assertEqual(len(calls), 1)

    def test_go_environment_failures_stop_analysis(self):
        for control in ["GOOS-failure", "GOARCH-failure"]:
            with self.subTest(control=control):
                self.calls.unlink(missing_ok=True)
                result, calls = self.invoke(control)
                self.assertNotEqual(result.returncode, 0)
                self.assertFalse(any(call[:2] == ["linter", "run"] for call in calls))

    def test_checker_failure_is_enforced(self):
        result, _ = self.invoke("checker-failure")
        self.assertNotEqual(result.returncode, 0)

    def test_full_route_retains_normal_issue_exit(self):
        result, calls = self.invoke(target="golangci-full")
        self.assertEqual(result.returncode, 0, result.stderr)
        run = next(call for call in calls if call[:2] == ["linter", "run"])
        self.assertNotIn("--issues-exit-code=0", run)
        self.assertFalse(any(call[:2] == ["go", "run"] for call in calls))

    def test_download_and_installer_failures_are_enforced(self):
        for control in ["download-failure", "installer-failure"]:
            with self.subTest(control=control):
                self.calls.unlink(missing_ok=True)
                result, calls = self.invoke(control, install=True)
                self.assertNotEqual(result.returncode, 0)
                self.assertEqual(calls, [["curl"]])
                self.assertEqual(list(self.root.glob("tmp.*")), [])


if __name__ == "__main__":
    unittest.main(verbosity=2)
