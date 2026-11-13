#!/usr/bin/env python3
# Copyright (c) 2026 Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at
# https://oss.oracle.com/licenses/upl.

"""Native private-console CLI qualification. Requires Windows and built fixture.

The fixture supplies Java-shaped process/REST controls, not Coherence runtime
evidence. Ctrl-C and Ctrl-Break are real GenerateConsoleCtrlEvent deliveries.
Close/logoff/shutdown and hard OS deadlines are outside this procedure.

Each fresh private task owns bin/controller.exe, bin/cohctl.exe and Java/Maven
shims. The Go controller derives the task root from its executable, reads only
request.json and emits only cli.log; signal and child/readiness markers stay
rooted in that task. JSON selects a finite operation/phase, never paths or argv.
The task directory and copied binaries are trusted against concurrent writers.
"""

import argparse
import json
import os
from pathlib import Path
import shutil
import socket
import subprocess
import tempfile
import time
import uuid

import management_platform as native


class QualificationError(RuntimeError):
    """A required native-console check failed."""


def eventually(check, description, timeout=15):
    deadline = time.monotonic() + timeout
    while time.monotonic() < deadline:
        if check():
            return
        time.sleep(.03)
    raise QualificationError("timeout: " + description)


def free_port():
    with socket.socket() as listener:
        listener.bind(("127.0.0.1", 0))
        return listener.getsockname()[1]


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--cli", type=Path, required=True)
    parser.add_argument("--fixture", type=Path, required=True)
    parser.add_argument("--scratch-dir", type=Path, required=True)
    parser.add_argument("--output", type=Path, required=True)
    args = parser.parse_args()
    if os.name != "nt":
        raise RuntimeError("MISSING COVERAGE: native Windows private-console session required")
    native.prerequisites()
    cli, fixture = str(args.cli.resolve()), str(args.fixture.resolve())
    evidence = {"platform": native.identity(), "checks": [], "status": "RUNNING",
                "scope": "actual private-console Ctrl-C/Ctrl-Break; Java-shaped process/REST fixture; no close/logoff/shutdown claim"}
    phases = [("start", "delay"), ("start", "readiness"), ("create", "delay"), ("create", "readiness"),
              ("restart", "readiness"), ("restart", "confirmation"), ("scale", "readiness")]
    try:
        for event, operation, phase in [(event, operation, phase) for event in (0, 1) for operation, phase in phases] + [(0, "start", "success")]:
            record = {"event": "CTRL_C_EVENT" if event == 0 else "CTRL_BREAK_EVENT", "operation": operation, "phase": phase}
            evidence["checks"].append(record)
            marker = "console-test-" + uuid.uuid4().hex[:12]
            old, controller = None, None
            with tempfile.TemporaryDirectory(prefix="console-test-", dir=args.scratch_dir) as directory:
                work = Path(directory)
                shim = work / "bin"
                shim.mkdir()
                shutil.copyfile(cli, shim / "cohctl.exe")
                task_fixture = str(shim / "controller.exe")
                shutil.copyfile(fixture, task_fixture)
                for name in ("java.exe", "mvn.exe"):
                    shutil.copyfile(fixture, shim / name)
                env = dict(os.environ, PATH=str(shim) + os.pathsep + os.environ["PATH"],
                           COHCTL_WINDOWS_READY="1" if phase == "success" else "0",
                           JAVA_TOOL_OPTIONS="", _JAVA_OPTIONS="", JDK_JAVA_OPTIONS="")
                port = free_port()
                olddir = work / "old"
                olddir.mkdir()
                try:
                    if operation in ("restart", "scale"):
                        old = subprocess.Popen([task_fixture, f"-Dcoherence.cluster={marker}", "-Dcoherence.member=storage-0",
                                                f"-Dcoherence.management.http.port={port}"],
                                               env=dict(env, COHCTL_WINDOWS_MEMBER_DIR="old", COHCTL_WINDOWS_READY="1"),
                                               stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
                        eventually(lambda: (olddir / ("child-" + str(old.pid))).exists(), "original member starts")  # NOSONAR(S1515) Synchronous polling finishes before old/olddir can be rebound.
                    conn = {"name": marker, "clusterName": marker, "manuallyCreated": True, "connectionType": "http",
                            "connectionurl": f"http://127.0.0.1:{port}/old", "managementPort": port, "clusterversion": "14.1.2-0-5",
                            "baseClasspath": "fixture-classpath", "arguments": f"-Dcoherence.cluster={marker}",
                            "management": {"restAddress": "127.0.0.1"}}
                    if operation == "create":
                        conn["name"] = conn["clusterName"] = "existing-" + marker
                    config = work / "cohctl.json"
                    config.write_text(json.dumps({"clusters": [conn], "requestTimeout": 2, "color": "off"}))
                    initialized = subprocess.run([cli, "--config-dir", str(work), "--config", str(config), "version"],
                                                 env=env, capture_output=True, text=True, timeout=15, check=True)
                    record["config_initialization"] = initialized.stdout
                    before = config.read_bytes()
                    command = [cli, "--config-dir", str(work), "--config", str(config), operation, "cluster", marker,
                               "-r", "2" if operation == "scale" or phase == "delay" else "1"]
                    if operation == "create":
                        cluster_port = free_port()
                        while cluster_port == port:
                            cluster_port = free_port()
                        command += ["-y", "-I", "-H", str(port), "-p", str(cluster_port)]
                    if operation == "restart" and phase != "confirmation":
                        command += ["-y"]
                    if phase == "delay":
                        command += ["-D", "30s"]
                    request = {"Operation": operation, "Phase": phase, "Cluster": marker, "Port": port,
                               "ClusterPort": cluster_port if operation == "create" else 0, "Event": event}
                    request_path = work / "request.json"
                    request_path.write_text(json.dumps(request))
                    with (work / "controller.log").open("w") as out, (work / "controller.err").open("w") as err:
                        controller = subprocess.Popen([task_fixture, "--control", "request.json"], env=env, stdin=subprocess.PIPE,
                                                      stdout=out, stderr=err)

                        def output():
                            return (work / "cli.log").read_text(errors="replace") if (work / "cli.log").exists() else ""

                        scale_armed = False

                        def reached():
                            nonlocal scale_armed
                            if controller.poll() is not None and phase != "success":  # NOSONAR(S1515) eventually completes before controller/phase can be rebound.
                                raise QualificationError({
                                    "cli": output(), "controller": (work / "controller.err").read_text(errors="replace")})  # NOSONAR(S1515) output/work are consumed synchronously within this case.
                            if phase == "confirmation":
                                return "Are you sure you want to stop" in output()
                            if phase == "readiness":
                                if operation == "scale" and any(work.glob("child-*")):  # NOSONAR(S1515) operation is read before synchronous polling returns.
                                    if not scale_armed:
                                        (olddir / "readiness").unlink(missing_ok=True)  # NOSONAR(S1515) olddir belongs to the current synchronous polling case.
                                        scale_armed = True
                                    return (olddir / "readiness").exists()
                                return (work / "readiness").exists()
                            return any(work.glob("child-*"))

                        eventually(reached, operation + " " + phase)
                        if phase != "success":
                            (work / "signal").touch()
                        controller_status = controller.wait(timeout=15)
                        if controller_status != 0:
                            raise QualificationError((work / "controller.err").read_text())
                        controller.stdin.close()
                    result = json.loads((work / "controller.log").read_text())
                    if result["argv"] != [str(shim / "cohctl.exe")] + command[1:]:
                        raise QualificationError("controller changed approved CLI arguments")
                    record.update(controller=result, command=command, cli_output=output())
                    owned = native.processes(marker)
                    record["owned_after_operation"] = owned
                    if phase == "success":
                        if result["exit"] != 0 or result["sent"] or len(owned) != 1:
                            raise QualificationError("success control failed: " + repr(result))
                        if config.read_bytes() == before:
                            raise QualificationError("accepted success did not save endpoint")
                    else:
                        if not result["sent"] or result["exit"] == 0 or result["rescue"]:
                            raise QualificationError("cancellation or rescue check failed: " + repr(result))
                        if config.read_bytes() != before:
                            raise QualificationError("cancellation changed saved bytes")
                        if phase != "confirmation":
                            if "context canceled" not in output():
                                raise QualificationError("console event bypassed lifecycle cancellation")
                        expected = {old.pid} if old and (operation == "scale" or phase == "confirmation") else set()
                        eventually(lambda: set(native.processes(marker)) == expected, "only original members survive cancellation")  # NOSONAR(S1515) marker/expected cannot be rebound until synchronous polling returns.
                        if old:
                            if (old.poll() is None) != bool(expected):
                                raise QualificationError("original process ownership changed")
                    record["result"] = "PASS"
                except Exception as error:
                    # Retain the failed case and run the remaining planned cases.
                    # Every explicit check above remains a qualification requirement.
                    record["result"] = "FAIL"
                    record["failure"] = repr(error)
                finally:
                    for filename in ("cli.log", "controller.log", "controller.err"):
                        if (work / filename).exists():
                            record[filename] = (work / filename).read_text(errors="replace")
                    if controller and controller.poll() is None:
                        # Allow the independent native controller deadline to stop
                        # and reap its CLI before the driver tears down the owner.
                        try:
                            controller.wait(timeout=48)
                        except subprocess.TimeoutExpired:
                            controller.kill()
                            controller.wait(timeout=5)
                            record["controller_rescue"] = True
                            record["unresolved_cli_ownership"] = True
                    record["cleanup_requests"] = native.stop_owned(marker)
                    if old:
                        old.wait(timeout=5)  # Sole owner of original fixture's Wait.
                    eventually(lambda: not native.processes(marker), "owned Windows cleanup")  # NOSONAR(S1515) Cleanup polling completes before the next case can rebind marker.
                    record["cleanup_survivors"] = native.processes(marker)
                    if record.get("controller_rescue"):
                        raise QualificationError("controller rescue leaves CLI ownership unresolved; qualification failed")
        if len(evidence["checks"]) != 2 * len(phases) + 1 or not all(record.get("result") == "PASS" for record in evidence["checks"]):
            raise QualificationError("one or more native console cases failed")
        evidence["status"] = "PASS"
    except BaseException as error:
        evidence["status"] = "FAIL"
        evidence["failure"] = repr(error)
        raise
    finally:
        args.output.write_text(json.dumps(evidence, indent=2) + "\n")  # NOSONAR(S8707) Evidence destination is explicitly granted by the trusted test operator.


if __name__ == "__main__":
    main()
