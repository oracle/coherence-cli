#!/usr/bin/env python3
# Copyright (c) 2026 Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at
# https://oss.oracle.com/licenses/upl.
"""Host-side driver fault injection; does not qualify native Windows signals."""

import ast
import importlib.util
import io
import json
import os
from pathlib import Path
import subprocess
import sys
import tempfile
import time
from types import SimpleNamespace
import unittest
from unittest.mock import patch


def load_driver():
    spec = importlib.util.spec_from_file_location("console_driver", Path(__file__).with_name("test-windows-console.py"))
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


class Fixture:
    """Supply deterministic process/fixture outcomes to the actual driver."""

    def __init__(self, fault):
        self.fault = fault
        self.case = 0
        self.original = None
        self.owned = {}
        self.work = None
        self.waits = 0
        self.cleanups = 0

    def processes(self, marker):
        return dict(self.owned)

    def stop_owned(self, marker):
        self.cleanups += 1
        if self.fault == "cleanup" and self.case == 1:
            raise OSError("injected cleanup failure")
        stopped = list(self.owned)
        self.owned.clear()
        if self.original:
            self.original.done = True
        return stopped

    def popen(self, argv, **kwargs):
        controller = "--control" in argv
        work = Path(argv[0]).parent.parent
        process = SimpleNamespace(pid=100 + self.case, done=False, stdin=io.StringIO())
        process.poll = lambda: 0 if process.done else None
        process.kill = lambda: setattr(process, "done", True)
        if not controller:
            self.original = process
            self.owned = {process.pid: "original"}
            (work / "old" / f"child-{process.pid}").touch()
            process.wait = lambda **kw: 0
            return process

        self.case += 1
        self.work = work
        request = json.loads((work / "request.json").read_text())
        operation, phase = request["Operation"], request["Phase"]
        (work / "child-200").touch()
        (work / "readiness").touch()
        (work / "old" / "readiness").touch()
        text = "Are you sure you want to stop" if phase == "confirmation" else "context canceled"
        if self.fault == "cancel-output" and self.case == 1:
            text = "wrong cancellation output"
        (work / "cli.log").write_text(text)
        if self.fault == "controller-alive" and self.case == 1:
            process.done = True

        def wait(timeout):
            self.waits += 1
            if self.fault == "controller-status" and self.case == 1:
                (work / "controller.err").write_text("injected controller exit")
            if self.fault == "rescue" and self.case == 1 and timeout != 5:
                if timeout == 48:
                    raise subprocess.TimeoutExpired(argv, timeout)
                return 1
            process.done = True
            command = [str(work / "bin" / "cohctl.exe"), "--config-dir", str(work), "--config", str(work / "cohctl.json"),
                       operation, "cluster", request["Cluster"], "-r", "2" if operation == "scale" or phase == "delay" else "1"]
            if operation == "create":
                command += ["-y", "-I", "-H", str(request["Port"]), "-p", str(request["ClusterPort"])]
            if operation == "restart" and phase != "confirmation":
                command += ["-y"]
            if phase == "delay":
                command += ["-D", "30s"]
            result = {"argv": command, "exit": 0 if phase == "success" else 1, "sent": phase != "success", "rescue": False}
            if self.fault == "argv" and self.case == 1:
                result["argv"] = []
            if self.fault == "success-result" and phase == "success":
                result["exit"] = 1
            if self.fault == "cancel-result" and self.case == 1:
                result["rescue"] = True
            (work / "controller.log").write_text(json.dumps(result))
            config = work / "cohctl.json"
            if (phase == "success" and self.fault != "success-save") or (self.fault == "cancel-save" and self.case == 1):
                config.write_text(config.read_text() + "\n")
            survives = self.original and (operation == "scale" or phase == "confirmation")
            self.owned = {self.original.pid: "original"} if survives else ({200: "accepted"} if phase == "success" else {})
            if self.original and not survives and self.fault != "original-owner":
                self.original.done = True
            return 1 if self.fault == "controller-status" and self.case == 1 else 0

        process.wait = wait
        return process

    def sleep(self, delay):
        # The scale case deliberately removes and waits for renewed readiness.
        (self.work / "old" / "readiness").touch()


def probe(fault, propagate=False):
    driver = load_driver()
    fixture = Fixture(fault)
    with tempfile.TemporaryDirectory(prefix="console-driver-support-") as temporary:
        root = Path(temporary)
        cli, controller, output = (root / name for name in ("cli", "fixture", "evidence.json"))
        cli.touch()
        controller.touch()
        arguments = ["driver", "--cli", str(cli), "--fixture", str(controller), "--scratch-dir", str(root), "--output", str(output)]
        driver.os = SimpleNamespace(name="nt", environ=os.environ, pathsep=os.pathsep)
        driver.native = SimpleNamespace(prerequisites=lambda: None, identity=lambda: {"scope": "host-side fake"},
                                        processes=fixture.processes, stop_owned=fixture.stop_owned)
        driver.subprocess = SimpleNamespace(Popen=fixture.popen, run=lambda *a, **kw: SimpleNamespace(stdout="initialized"),
                                            DEVNULL=subprocess.DEVNULL, PIPE=subprocess.PIPE, TimeoutExpired=subprocess.TimeoutExpired)
        driver.time = SimpleNamespace(monotonic=time.monotonic, sleep=fixture.sleep)
        ports = iter(range(40000, 40100))
        driver.free_port = lambda: next(ports)
        error = None
        failure = None
        with patch.object(sys, "argv", arguments):
            try:
                driver.main()
            except Exception as caught:
                error = repr(caught)
                failure = caught
        row = {"evidence": json.loads(output.read_text()), "error": error, "waits": fixture.waits, "cleanups": fixture.cleanups,
               "leftovers": sorted(p.name for p in root.iterdir() if p.name not in ("cli", "fixture", "evidence.json"))}
        if propagate:
            print(json.dumps(row), flush=True)
            if failure:
                raise failure
        return row


FAULTS = ("none", "controller-alive", "controller-status", "argv", "success-result", "success-save",
          "cancel-result", "cancel-save", "cancel-output", "original-owner", "cleanup", "rescue")
FAILURES = {"controller-alive": "'controller':", "controller-status": "injected controller exit",
            "argv": "changed approved CLI arguments", "success-result": "success control failed",
            "success-save": "accepted success did not save endpoint", "cancel-result": "cancellation or rescue check failed",
            "cancel-save": "cancellation changed saved bytes", "cancel-output": "console event bypassed lifecycle cancellation",
            "original-owner": "original process ownership changed"}


class ConsoleDriverTests(unittest.TestCase):
    def test_aggregate_rejects_failed_and_missing_evidence(self):
        tree = ast.parse(Path(__file__).with_name("test-windows-console.py").read_text())
        aggregate = next(node for node in ast.walk(tree) if isinstance(node, ast.If) and
                         any(isinstance(child, ast.Constant) and child.value == "one or more native console cases failed"
                             for child in ast.walk(node)))
        for optimize in (0, 1, 2):
            code = compile(ast.fix_missing_locations(ast.Module(body=[aggregate], type_ignores=[])), "actual-aggregate", "exec", optimize=optimize)
            for checks in ([{"result": "PASS"}] * 15, [{"result": "PASS"}] * 14,
                           [{"result": "PASS"}] * 14 + [{}], [{"result": "PASS"}] * 14 + [{"result": "FAIL"}], []):
                with self.subTest(optimize=optimize, checks=checks):
                    env = {"evidence": {"checks": checks}, "phases": [None] * 7, "QualificationError": RuntimeError}
                    if len(checks) == 15 and all(row.get("result") == "PASS" for row in checks):
                        exec(code, env)
                    else:
                        with self.assertRaisesRegex(RuntimeError, "one or more native console cases failed"):
                            exec(code, env)

    def test_matrix_failures_and_cleanup_in_all_optimization_modes(self):
        for flags, optimize in (([], ""), (["-O"], ""), (["-OO"], ""), ([], "1"), ([], "2")):
            with self.subTest(flags=flags, PYTHONOPTIMIZE=optimize):
                env = dict(os.environ, PYTHONOPTIMIZE=optimize, PYTHONDONTWRITEBYTECODE="1")
                result = subprocess.run([sys.executable, *flags, __file__, "--probe"], env=env, capture_output=True, text=True, timeout=30)
                self.assertEqual(result.returncode, 0, result.stderr)
                rows = json.loads(result.stdout)
                failed_run = subprocess.run([sys.executable, *flags, __file__, "--case", "cancel-save"],
                                            env=env, capture_output=True, text=True, timeout=10)
                self.assertNotEqual(failed_run.returncode, 0)
                failed_evidence = json.loads(failed_run.stdout)["evidence"]
                self.assertEqual(failed_evidence["status"], "FAIL")
                self.assertEqual(len(failed_evidence["checks"]), 15)
                self.assertEqual(failed_evidence["checks"][-1]["result"], "PASS")
                for fault, row in rows.items():
                    with self.subTest(fault=fault):
                        evidence = row["evidence"]
                        self.assertEqual(row["leftovers"], [])
                        if fault == "none":
                            self.assertIsNone(row["error"])
                            self.assertEqual(evidence["status"], "PASS")
                            self.assertEqual(len(evidence["checks"]), 15)
                            self.assertEqual(row["waits"], 15)
                        else:
                            self.assertIsNotNone(row["error"])
                            self.assertEqual(evidence["status"], "FAIL")
                            if fault in ("cleanup", "rescue"):
                                self.assertEqual(len(evidence["checks"]), 1)
                                self.assertIn(fault, evidence["failure"])
                            else:
                                self.assertEqual(len(evidence["checks"]), 15)
                                self.assertTrue(any(r["result"] == "FAIL" for r in evidence["checks"]))
                                self.assertTrue(any(r["result"] == "PASS" for r in evidence["checks"]))
                                self.assertTrue(any(FAILURES[fault] in r.get("failure", "") for r in evidence["checks"]), evidence)
                                self.assertEqual(row["cleanups"], 15)
                        if fault == "rescue":
                            self.assertTrue(evidence["checks"][0]["controller_rescue"])
                            self.assertTrue(evidence["checks"][0]["unresolved_cli_ownership"])

    def test_local_management_rejects_optimization_before_resources(self):
        for flags, optimize in ((["-O"], ""), (["-OO"], ""), ([], "1"), ([], "2")):
            with self.subTest(flags=flags, PYTHONOPTIMIZE=optimize), tempfile.TemporaryDirectory() as temporary:
                scratch = Path(temporary) / "must-not-exist"
                result = subprocess.run([sys.executable, *flags, str(Path(__file__).with_name("test-local-management.py")),
                                         "test-version", "--scratch-dir", str(scratch)],
                                        env=dict(os.environ, PYTHONOPTIMIZE=optimize), capture_output=True, text=True, timeout=10)
                self.assertNotEqual(result.returncode, 0)
                self.assertIn("requires assertions", result.stderr)
                self.assertFalse(scratch.exists())


if __name__ == "__main__":
    if sys.argv[1:] == ["--probe"]:
        print(json.dumps({fault: probe(fault) for fault in FAULTS}))
    elif len(sys.argv) == 3 and sys.argv[1] == "--case":
        probe(sys.argv[2], propagate=True)
    else:
        unittest.main(verbosity=2)
