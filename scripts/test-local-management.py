#!/usr/bin/env python3
# Copyright (c) 2026 Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at
# https://oss.oracle.com/licenses/upl.

"""Local management integration tests. Requires java/javac, cached Maven dependencies and lsof.

Uses task-owned config/JVMs, local REST/JMX success, PID-owned listener inspection,
and negative probes through a real host interface with a positive TCP control.
This is same-host evidence; second-host and insecure-remote qualification are separate.
"""

import argparse
import base64
from functools import partial
from datetime import datetime, timezone
import json
import os
from pathlib import Path
import re
import secrets
import shutil
import socket
import subprocess
import sys
import tempfile
import threading
import time
import urllib.request
import uuid
import shlex

import management_platform as native


def wait_for_rest_membership(expected, endpoints, read_members, timeout=30, poll_interval=.1):
    """Wait for every selected endpoint under one deadline; retain exact identities."""
    deadline = time.monotonic() + timeout
    last = {member: {"port": port, "members": None} for member, port in endpoints.items()}
    while time.monotonic() < deadline:
        complete = True
        for member, port in endpoints.items():
            remaining = deadline - time.monotonic()
            if remaining <= 0:
                complete = False
                break
            try:
                observed = read_members(port, timeout=min(3, remaining))
                last[member] = {"port": port, "members": observed}
                complete = complete and observed == expected
            except (OSError, ValueError) as error:
                last[member] = {"port": port, "members": last[member]["members"], "error": repr(error)}
                complete = False
        if complete and time.monotonic() <= deadline:
            return last
        remaining = deadline - time.monotonic()
        if remaining > 0:
            time.sleep(min(poll_interval, remaining))
    raise TimeoutError(f"REST membership did not converge within {timeout}s: " +
                       json.dumps({"expected": expected, "endpoints": last}, sort_keys=True))



def inspect(label, expected_http, with_jmx, bind_address="127.0.0.1", *,
            marker, config, interface, jmx, probe_classpath, members, request,
            connect, marked_processes, observer_lock, linux_identities, evidence,
            platform=native):
    """Inspect one run using exact membership and PID-owned listener evidence."""
    remote = bind_address != "127.0.0.1"
    current, snapshots, rest, rest_identity = {}, [], {}, {}
    # REST can accept requests before its membership view has converged.
    # Obtain expected identities from the marked JVMs rather than an
    # arbitrarily early REST response, including from the primary.
    for pid in platform.processes(marker):
        snapshot = platform.snapshot(pid)
        process_args = snapshot["command"]
        tokens = [word.strip('"') for word in shlex.split(process_args, posix=os.name != "nt")]
        assert f"-Dcoherence.cluster={marker}" in tokens, "cluster PID identity mismatch"
        names = [word.removeprefix("-Dcoherence.member=") for word in tokens if word.startswith("-Dcoherence.member=")]
        assert len(names) == 1 and names[0] and names[0] not in current, "member PID identity mismatch"
        member = names[0]
        current[member] = pid
        addresses = snapshot["listeners"]
        snapshot["member"] = member
        snapshots.append(snapshot)
        state = snapshot.get("linux_process_identity")
        if state:
            with observer_lock:
                linux_identities.setdefault(f"{pid}:{state['start_ticks']}", state)
        # The Coherence log names each selected management HTTP acceptor.
        log = (config / "logs" / marker / (member + ".log")).read_text()
        ports = set(map(int, re.findall(r"HttpAcceptor now listening for connections on [^\n]*?:(\d+)", log)))
        for port in ports:
            # Logs can retain entries from an earlier incarnation. Only a
            # socket owned by this current member can supply evidence.
            if not any(address.rsplit(":", 1)[-1] == str(port) for address in addresses):
                continue
            # Health endpoints may share HttpAcceptor logs; only count Management REST.
            try:
                data = request(port)
            except (OSError, ValueError):
                continue
            if data.get("clusterName") != marker:
                continue
            assert member not in rest, ("multiple Management REST listeners on one member", member)
            rest[member] = port
            rest_identity[member] = {"owner_pid": pid, "cluster_name": data["clusterName"]}
    assert current and set(current.values()) == marked_processes(), "REST snapshot differs from current test-owned processes"
    assert len(rest) == expected_http, (label, "REST listener count", rest, snapshots)
    wait_for_rest_membership(current, rest, members)
    for snapshot in snapshots:
        fresh = platform.snapshot(snapshot["pid"])
        assert fresh["command"] == snapshot["command"] and fresh.get("started") == snapshot.get("started"), "member process changed during REST readiness"
        snapshot.update(fresh)
        member = snapshot["member"]
        if member not in rest:
            continue
        port = rest[member]
        platform.assert_listener_bind(snapshot["listeners"], port, bind_address)
        observed = members(port)
        assert observed == current, ("REST endpoint member/PID identity mismatch", member, port, observed, current)
        assert connect(interface, port) == remote, ("unexpected interface REST reachability", port)
        rest_identity[member]["members"] = observed
    if with_jmx:
        owner = current["storage-0"]
        sockets = [s for s in snapshots if any(a.rsplit(":", 1)[-1] == str(jmx) for a in s["listeners"])]
        assert len(sockets) == 1 and sockets[0]["pid"] == owner, sockets
        platform.assert_listener_bind(sockets[0]["listeners"], jmx, bind_address)
        output = subprocess.check_output(["java", "-cp", str(probe_classpath), "ManagementProbe", "127.0.0.1", str(jmx), str(owner)], text=True, timeout=15)
        assert connect(interface, jmx) == remote, "unexpected interface JMX registry/RMI reachability"
        assert "endpoint:[" in output and f":{jmx}]" in output, "actual RMI endpoint not identified"
    else:
        output = "JMX disabled"
        for snapshot in snapshots:
            assert "com.sun.management.jmxremote" not in snapshot["command"]
            assert not any(address.rsplit(":", 1)[-1] == str(jmx) for address in snapshot["listeners"]), snapshot
        assert not connect("127.0.0.1", jmx), "disabled JMX port still accepts loopback connections"
        assert not connect(interface, jmx), "disabled JMX port still accepts interface connections"
    assert members() == current and marked_processes() == set(current.values()), "members changed during socket inspection"
    record = {"case": label, "result": "PASS", "members": current, "rest": rest, "rest_identity": rest_identity,
              "sockets": snapshots, "jmx": output.strip(), "jmx_port_closed": not with_jmx,
              "nonloopback_probes": "all selected REST and JMX reachable" if remote else "all selected REST and JMX failed"}
    with socket.socket() as control:
        control.bind((interface, 0)); control.listen(1)
        assert connect(interface, control.getsockname()[1]), "same-case positive interface control failed"
        record["positive_control"] = f"{interface}:{control.getsockname()[1]}"
    evidence["checks"].append(record)
    print(label, "PASS", rest, output.strip(), flush=True)
    return record


def main():
    if sys.flags.optimize:
        raise RuntimeError("Management qualification requires assertions; run without -O, -OO or PYTHONOPTIMIZE")
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("version")
    parser.add_argument("--cli", default="bin/cohctl")
    parser.add_argument("--download-deps", action="store_true", help="allow Maven to populate missing dependency cache entries")
    parser.add_argument("--commercial", action="store_true")
    parser.add_argument("--monitor", action="store_true", help="also exercise the existing terminal monitor fixture (requires expect)")
    parser.add_argument("--output", help="optional JSON evidence destination")
    parser.add_argument("--scratch-dir", type=Path, help="existing task-owned temporary parent")
    parser.add_argument("--interface", help="assigned local IPv4 interface; automatic route selection otherwise")
    parser.add_argument("--rest-port", type=int, help="assigned fixed Management REST test port")
    parser.add_argument("--cluster-port", type=int, help="assigned fixed NameService/cluster test port")
    parser.add_argument("--jmx-port", type=int, help="assigned fixed registry/RMI test port")
    parser.add_argument("--initialize-config", action="store_true", help="run version first as an explicit initialized-configuration control")
    args = parser.parse_args()
    assigned_ports = [port for port in (args.rest_port, args.cluster_port, args.jmx_port) if port is not None]
    if any(not 0 < port < 65536 for port in assigned_ports) or len(assigned_ports) != len(set(assigned_ports)):
        parser.error("assigned test ports must be distinct integers in 1..65535")
    native.prerequisites()
    if not all(shutil.which(tool) for tool in ("java", "javac")):
        raise RuntimeError("MISSING COVERAGE: java and javac are required")
    root = Path(__file__).resolve().parent.parent
    cli = str(Path(args.cli).resolve())
    base = args.scratch_dir or Path(tempfile.gettempdir())
    base.mkdir(parents=True, exist_ok=True)  # NOSONAR(S8707) Scratch parent is explicitly selected by the trusted test operator.
    evidence = {"version": args.version, "platform": native.identity(),
                "started_at": datetime.now(timezone.utc).isoformat(), "status": "RUNNING",
                "java": subprocess.check_output(["java", "-version"], stderr=subprocess.STDOUT, text=True).strip(),
                "topology": "same host, real non-loopback interface; no second-host test", "checks": [], "commands": []}
    evidence["requested_fixed_ports"] = {"rest": args.rest_port, "cluster": args.cluster_port, "jmx": args.jmx_port}
    opener = urllib.request.build_opener(urllib.request.ProxyHandler({}))
    marker = "management-test-" + uuid.uuid4().hex[:12]
    evidence["cluster_name"] = marker

    local_host = "127.0.0.1"

    # Both REST probes use credential-free GETs, no environment proxies and
    # per-request timeouts; callers correlate the test-owned JVM/cluster identity.
    # The owner accepts local test HTTP including default redirects off loopback.
    # These timeouts are not hard whole-response deadlines.
    def request(port):
        with opener.open(f"http://{local_host}:{port}/management/coherence/cluster", timeout=2) as response:  # NOSONAR(S5332) Approved local test HTTP; redirects may leave loopback.
            return json.load(response)

    def free_port():
        # Keep fixed Coherence discovery/management ports below the host's usual
        # ephemeral range. A socket assigned with bind(..., 0) can be reused by
        # an outgoing connection before the cluster's NameService binds it.
        for _ in range(100):
            candidate = 18000 + secrets.randbelow(10000)
            try:
                with socket.socket() as tcp, socket.socket(socket.AF_INET, socket.SOCK_DGRAM) as udp:
                    tcp.bind(("127.0.0.1", candidate))
                    udp.bind(("127.0.0.1", candidate))
                    return candidate
            except OSError:
                pass
        raise RuntimeError("no free fixed test port in 18000..27999")

    def connect(host, port):
        try:
            with socket.create_connection((host, port), timeout=1):
                return True
        except OSError:
            return False

    # UDP connect chooses an interface without sending a packet.
    with socket.socket(socket.AF_INET, socket.SOCK_DGRAM) as route:
        route.connect(("192.0.2.1", 9))
        interface = args.interface or route.getsockname()[0]
    if interface.startswith("127.") or interface == "0.0.0.0":
        raise RuntimeError("MISSING COVERAGE: no real non-loopback interface")
    with socket.socket() as control:
        control.bind((interface, 0)); control.listen(1)
        assert connect(interface, control.getsockname()[1]), "positive interface control failed"
        evidence["positive_tcp_control_address"] = f"{interface}:{control.getsockname()[1]}"
    evidence["interface"] = interface
    evidence["positive_tcp_control"] = True

    with tempfile.TemporaryDirectory(prefix="management-test-", dir=base) as temp:
        config = Path(temp) / "config"
        config.mkdir()
        reserved = set(assigned_ports)

        def select_port(assigned):
            if assigned is not None:
                return assigned
            while True:
                port = free_port()
                if port not in reserved:
                    reserved.add(port)
                    return port

        http, cluster, jmx = select_port(args.rest_port), select_port(args.cluster_port), select_port(args.jmx_port)
        evidence["selected_fixed_ports"] = {"rest": http, "cluster": cluster, "jmx": jmx}
        linux_identities, observer_errors = {}, []
        observer_stop, observer_lock = threading.Event(), threading.Lock()

        def remember_linux_pids():
            try:
                for pid in native.processes(marker):
                    state = native.linux_process_state(pid)
                    if state:
                        with observer_lock:
                            linux_identities.setdefault(f"{pid}:{state['start_ticks']}", state)
            except Exception as error:
                observer_errors.append(repr(error))

        def observe_linux():
            while not observer_stop.wait(.1):
                remember_linux_pids()

        observer = None
        if evidence["platform"]["system"] == "Linux":
            observer = threading.Thread(target=observe_linux, daemon=True)
            observer.start()

        def command(*words, success=True, env=None):
            started = time.monotonic()
            result = subprocess.run([cli, "--config-dir", str(config), *map(str, words)], text=True,
                                    stdout=subprocess.PIPE, stderr=subprocess.STDOUT, timeout=180, env=env)
            evidence["commands"].append({"argv": [cli, "--config-dir", str(config), *map(str, words)],
                                         "exit_code": result.returncode, "expected_success": success,
                                         "elapsed_seconds": time.monotonic() - started,
                                         "stdout": result.stdout})
            print("cohctl", *words, flush=True)
            if (result.returncode == 0) != success:
                raise AssertionError(result.stdout)
            return result.stdout

        def members(port=http, timeout=3):
            with opener.open(f"http://{local_host}:{port}/management/coherence/cluster/members", timeout=timeout) as response:  # NOSONAR(S5332) Approved local test HTTP; redirects may leave loopback.
                items = json.load(response)["items"]
            result = {item["memberName"]: int(item["processName"]) for item in items}
            assert len(result) == len(items) and len(set(result.values())) == len(items), "ambiguous member/PID identities"
            return result

        def marked_processes():
            return set(native.processes(marker))


        inspect_run = partial(inspect, marker=marker, config=config, interface=interface,
                              jmx=jmx, probe_classpath=temp, members=members, request=request,
                              connect=connect, marked_processes=marked_processes,
                              observer_lock=observer_lock, linux_identities=linux_identities,
                              evidence=evidence)

        try:
            subprocess.run(["javac", "-d", temp, str(root / "scripts" / "ManagementProbe.java")], check=True)
            create = ["create", "cluster", marker, "-y", "-I", "-v", args.version, "-r", 2, "-H", http, "-p", cluster]
            if args.download_deps:
                create.remove("-I")
            if args.commercial:
                create.append("-C")
            if args.initialize_config:
                command("version")
                evidence["configuration_preinitialized"] = True
            command(*create)
            inspect_run("default create", 1, False)
            assert marker in command("nslookup", f"127.0.0.1:{cluster}", "-q", "Cluster/name")
            assert "127.0.0.1" in command("nslookup", f"127.0.0.1:{cluster}", "-q", "NameService/string/management/HTTPManagementURL")
            command("get", "members", "-c", f"127.0.0.1:{cluster}")
            evidence["name_service_discovery"] = "PASS"
            if args.monitor:
                fixture = Path(temp) / "monitor.expect"
                fixture.write_text((root / "scripts" / "monitor.expect").read_text().replace("monitor cluster local", "monitor cluster $env(COHCTL_TEST_CLUSTER)"))
                environment = dict(os.environ, DIR=str(root), CONFIG_DIR=str(config), COHCTL_TEST_CLUSTER=marker)
                result = subprocess.run(["expect", "-f", str(fixture)], env=environment, text=True,
                                        stdout=subprocess.PIPE, stderr=subprocess.STDOUT, timeout=30)
                assert result.returncode == 0 and "Coherence CLI" in result.stdout, result.stdout
                evidence["terminal_monitor"] = "PASS"

            # Stop needs neither Java nor valid inherited management options.
            no_java = dict(os.environ, PATH="/nonexistent", JAVA_TOOL_OPTIONS="@invalid")
            command("stop", "cluster", marker, "-y", env=no_java)
            command("remove", "cluster", marker, "-y")
            command(*create, "-N")
            inspect_run("dynamic create", 2, False)
            command("stop", "cluster", marker, "-y")
            command("start", "cluster", marker, "-r", 3, "-J", jmx, "-N")
            inspect_run("dynamic start with local JMX", 3, True)
            for count, expected in ((4, 3), (6, 4), (9, 6)):
                command("scale", "cluster", marker, "-r", count, "-N")
                inspect_run(f"dynamic scale to {count}", expected, True)
            before = (config / "cohctl.yaml").read_bytes()
            old_members = members()
            command("restart", "cluster", marker, "-y", "--management-address=0.0.0.0", success=False)
            assert (config / "cohctl.yaml").read_bytes() == before
            assert members() == old_members and marked_processes() == set(old_members.values()), "invalid restart changed running members"
            evidence["invalid_restart"] = {"result": "PASS", "members_before": old_members, "members_after": members(),
                                           "no_replacement": True}
            command("restart", "cluster", marker, "-y", "-r", 2, "--management-address=localhost", "--jmx-port=0")
            # Restart without -N retains the ordinary single REST selection.
            record = inspect_run("restart override persistence / explicit JMX disable", 1, False)
            assert len(record["members"]) == 2 and not set(record["members"].values()).intersection(old_members.values()), "restart retained old members"
            record["old_members"] = old_members
            record["all_members_replaced"] = True
            # Successful endpoint override must round-trip through config for get/stop.
            output = command("get", "members")
            assert marker in command("get", "clusters") and output
            saved = (config / "cohctl.yaml").read_text()
            assert "http://127.0.0.1:" in saved and "jmxport: 0" in saved
            record["saved_loopback_url_and_jmx_zero"] = True
            if marked_processes():
                command("stop", "cluster", marker, "-y", env=no_java)
            deadline = time.monotonic() + 10
            while marked_processes() and time.monotonic() < deadline:
                time.sleep(.1)
            assert not marked_processes(), "Java-free stop left owned JVMs"
            evidence["status"] = "PASS"
        except Exception as error:
            evidence["status"] = "FAIL"
            evidence["failure"] = repr(error)
            selected = config / "cohctl.yaml"
            if selected.exists():
                body = selected.read_bytes()
                evidence["failed_configuration"] = {"encoding": "base64", "original_length": len(body),
                                                    "data": base64.b64encode(body).decode()}
            evidence["logs"] = {log.name: log.read_text() for log in sorted((config / "logs" / marker).glob("*.log"))}
            for name, content in evidence["logs"].items():
                print(name, content[-4000:], flush=True)
            raise
        finally:
            observer_stop.set()
            if observer:
                observer.join(timeout=12)
                if observer.is_alive():
                    observer_errors.append("owned PID observer did not finish")
                remember_linux_pids()
            before_cleanup = native.processes(marker)
            stopped = native.stop_owned(marker)
            def kernel_survivors():
                result = {}
                with observer_lock:
                    identities = list(linux_identities.items())
                for key, original in identities:
                    try:
                        state = native.linux_process_state(original["pid"])
                    except OSError as error:
                        result[key] = {"observation_error": repr(error)}
                        continue
                    if state and state["start_ticks"] == original["start_ticks"]:
                        result[key] = state
                return result
            deadline = time.monotonic() + 10
            while (marked_processes() or kernel_survivors()) and time.monotonic() < deadline:
                time.sleep(.1)
            remaining = sorted(marked_processes())
            kernel_remaining = kernel_survivors()
            evidence["cleanup"] = {"before": before_cleanup, "forced_owned_pids": stopped,
                                   "remaining_marked_pids": remaining, "linux_observed_identities": linux_identities,
                                   "remaining_owned_kernel_processes": kernel_remaining, "observer_errors": observer_errors,
                                   "result": "FAIL" if remaining or kernel_remaining or observer_errors else "PASS"}
            evidence["finished_at"] = datetime.now(timezone.utc).isoformat()
            if remaining or kernel_remaining or observer_errors:
                evidence["status"] = "FAIL"
            if args.output:
                Path(args.output).write_text(json.dumps(evidence, indent=2) + "\n")  # NOSONAR(S8707) Evidence destination is explicitly granted by the trusted test operator.
            assert not remaining and not kernel_remaining and not observer_errors, "test-owned JVM cleanup incomplete"
    if args.output:
        Path(args.output).write_text(json.dumps(evidence, indent=2) + "\n")  # NOSONAR(S8707) Evidence destination is explicitly granted by the trusted test operator.
    print(json.dumps(evidence, indent=2))


if __name__ == "__main__":
    main()
