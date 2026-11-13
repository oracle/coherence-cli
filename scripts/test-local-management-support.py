#!/usr/bin/env python3
# Copyright (c) 2026 Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at
# https://oss.oracle.com/licenses/upl.

"""Regression tests for local management readiness and listener inspection."""

import importlib.util
import json
from pathlib import Path
import tempfile
import threading
import types
import unittest
from unittest.mock import patch

import management_platform as native

spec = importlib.util.spec_from_file_location("local_management", Path(__file__).with_name("test-local-management.py"))
management = importlib.util.module_from_spec(spec)
spec.loader.exec_module(management)


class MembershipReadinessTests(unittest.TestCase):
    def test_inspection_waits_for_partial_secondary_view(self):
        self.inspect_views({58081})

    def test_inspection_waits_for_partial_primary_view(self):
        self.inspect_views({58080})

    def test_inspection_still_rejects_identity_change_after_readiness(self):
        self.inspect_views(set(), regress=True)

    def inspect_views(self, partial_ports, regress=False):
        marker = "management-membership-control"
        expected = {"storage-0": 11050, "storage-1": 11051}
        ports = {"storage-0": 58080, "storage-1": 58081}
        calls = []
        synchronized = False

        def members(port=58080, timeout=3):
            calls.append(port)
            if regress and synchronized:
                return {"storage-0": 99999, "storage-1": 11051}
            if port in partial_ports and calls.count(port) == 1:
                member = next(name for name, selected in ports.items() if selected == port)
                return {member: expected[member]}
            return dict(expected)

        wait_for_rest_membership = management.wait_for_rest_membership

        def wait(*args):
            nonlocal synchronized
            result = wait_for_rest_membership(*args)
            synchronized = True
            return result

        class Control:
            def __enter__(self):
                return self
            def __exit__(self, *args):
                pass
            def bind(self, address):
                pass
            def listen(self, count):
                pass
            def getsockname(self):
                return ("192.0.2.10", 59999)

        with tempfile.TemporaryDirectory(prefix="management-membership-control-") as temp:
            directory = Path(temp)
            logs = directory / "logs" / marker
            logs.mkdir(parents=True)
            snapshots = {}
            for member, pid in expected.items():
                (logs / (member + ".log")).write_text(f"HttpAcceptor now listening for connections on 127.0.0.1:{ports[member]}\n")
                snapshots[pid] = {"pid": pid, "command": f"java -cp fixture-classpath -Dcoherence.cluster={marker} -Dcoherence.member={member}",
                                  "listeners": [f"127.0.0.1:{ports[member]}"]}
            record = {"checks": []}
            inputs = dict(marker=marker, config=directory, interface="192.0.2.10", jmx=58082,
                          probe_classpath=directory, members=members, marked_processes=lambda: set(expected.values()),
                          platform=types.SimpleNamespace(processes=lambda selected: {pid: row["command"] for pid, row in snapshots.items()},
                                                         snapshot=lambda pid: dict(snapshots[pid]), assert_listener_bind=native.assert_listener_bind),
                          request=lambda selected: {"clusterName": marker}, connect=lambda host, port: port == 59999,
                          observer_lock=threading.Lock(), linux_identities={}, evidence=record)
            with patch.object(management, "wait_for_rest_membership", side_effect=wait), patch.object(management.socket, "socket", Control):
                if regress:
                    with self.assertRaisesRegex(AssertionError, "REST endpoint member/PID identity mismatch"):
                        management.inspect("dynamic create", 2, False, **inputs)
                    self.assertEqual(record["checks"], [])
                    return
                result = management.inspect("dynamic create", 2, False, **inputs)
            self.assertEqual(result["members"], expected)
            self.assertEqual(result["rest"], ports)
            for port in partial_ports:
                self.assertGreaterEqual(calls.count(port), 2)
            self.assertEqual(len(record["checks"]), 1)

    def clock(self):
        clock = types.SimpleNamespace(now=0)
        def sleep(seconds):
            self.assertGreater(seconds, 0)
            clock.now += seconds
        for name, effect in (("monotonic", lambda: clock.now), ("sleep", sleep)):
            control = patch.object(management.time, name, side_effect=effect)
            control.start()
            self.addCleanup(control.stop)
        return clock

    def test_transient_request_errors_are_retried(self):
        self.clock()
        expected = {"storage-0": 11050}
        outcomes = [OSError("listener not ready"), ValueError("incomplete JSON"), expected]
        def read(port, timeout):
            item = outcomes.pop(0)
            if isinstance(item, Exception):
                raise item
            return item
        result = management.wait_for_rest_membership(expected, {"storage-0": 58080}, read, timeout=1)
        self.assertEqual(result, {"storage-0": {"port": 58080, "members": expected}})
        self.assertEqual(outcomes, [])

    def test_timeout_retains_each_endpoint_map_and_exact_expected_identities(self):
        clock = self.clock()
        expected = {"storage-0": 11050, "storage-1": 11051}
        primary = dict(expected)
        for observed in ({"storage-1": 11051}, {"storage-0": 99999, "storage-1": 11051},
                         dict(expected, unexpected=11052)):
            with self.subTest(observed=observed):
                clock.now = 0
                def read(port, timeout):
                    return primary if port == 58080 else observed
                with self.assertRaises(TimeoutError) as raised:
                    management.wait_for_rest_membership(expected, {"storage-0": 58080, "storage-1": 58081}, read, timeout=.25)
                diagnostic = json.loads(str(raised.exception).split(": ", 1)[1])
                self.assertEqual(diagnostic["expected"], expected)
                self.assertEqual(diagnostic["endpoints"]["storage-0"]["members"], primary)
                self.assertEqual(diagnostic["endpoints"]["storage-1"]["members"], observed)
                self.assertAlmostEqual(clock.now, .25)

    def test_timeout_preserves_last_map_and_request_error(self):
        self.clock()
        reads = []
        def read(port, timeout):
            reads.append(port)
            if len(reads) == 1:
                return {"storage-1": 11051}
            raise OSError("endpoint unavailable")
        with self.assertRaises(TimeoutError) as raised:
            management.wait_for_rest_membership({"storage-0": 11050, "storage-1": 11051}, {"storage-1": 58081}, read, timeout=.25)
        diagnostic = json.loads(str(raised.exception).split(": ", 1)[1])
        self.assertEqual(diagnostic["endpoints"]["storage-1"]["members"], {"storage-1": 11051})
        self.assertIn("endpoint unavailable", diagnostic["endpoints"]["storage-1"]["error"])

    def test_requests_share_one_deadline(self):
        clock = self.clock()
        calls = []
        def read(port, timeout):
            calls.append((port, timeout))
            clock.now += timeout
            raise TimeoutError("read timed out")
        with self.assertRaises(TimeoutError) as raised:
            management.wait_for_rest_membership({"storage-0": 11050}, {"primary": 58080, "secondary": 58081}, read, timeout=.25)
        self.assertEqual(calls, [(58080, .25)])
        self.assertAlmostEqual(clock.now, .25)
        diagnostic = json.loads(str(raised.exception).split(": ", 1)[1])
        self.assertIsNone(diagnostic["endpoints"]["secondary"]["members"])

    def test_ambiguous_identity_is_not_retried(self):
        self.clock()
        def read(port, timeout):
            raise AssertionError("ambiguous member/PID identities")
        with self.assertRaisesRegex(AssertionError, "ambiguous member/PID identities"):
            management.wait_for_rest_membership({"storage-0": 11050}, {"storage-0": 58080}, read)


class ListenerBindTests(unittest.TestCase):
    def test_complete_selected_socket_set(self):
        for bind in ("127.0.0.1", "192.0.2.10", "0.0.0.0"):
            with self.subTest(bind=bind):
                native.assert_listener_bind([f"{bind}:58080", "127.0.0.1:59999"], 58080, bind)
                if bind == "0.0.0.0":
                    native.assert_listener_bind(["*:58080"], 58080, bind)
                wrong = "192.0.2.11:58080" if bind != "192.0.2.11" else "127.0.0.1:58080"
                invalid = [[], ["[::]:58080"], [wrong], [f"{bind}:58080", "*:58080"]]
                if bind != "0.0.0.0":
                    invalid.extend([["*:58080"], ["0.0.0.0:58080"]])
                for addresses in invalid:
                    with self.assertRaisesRegex(AssertionError, "incorrect listener bind"):
                        native.assert_listener_bind(addresses, 58080, bind)

    def test_actual_rest_and_jmx_inspection_rejects_wrong_wildcard(self):
        marker, member, pid, port, jmx = "management-bind-control", "storage-0", 12345, 58080, 58081

        class Control:
            def __enter__(self):
                return self
            def __exit__(self, *args):
                pass
            def bind(self, address):
                pass
            def listen(self, count):
                pass
            def getsockname(self):
                return ("192.0.2.10", 59999)

        with tempfile.TemporaryDirectory(prefix="management-bind-control-") as temp:
            directory = Path(temp)
            logs = directory / "logs" / marker
            logs.mkdir(parents=True)
            (logs / (member + ".log")).write_text(f"HttpAcceptor now listening for connections on 127.0.0.1:{port}\n")
            command = f"java -cp fixture-classpath -Dcoherence.cluster={marker} -Dcoherence.member={member}"
            # JMX-only negatives retain a valid REST socket so the JMX assertion
            # itself must reject the wildcard; network observations still match.
            for bind in ("127.0.0.1", "192.0.2.10", "0.0.0.0"):
                for wrong in ("rest", "jmx"):
                    with self.subTest(bind=bind, wrong=wrong):
                        addresses = [f"{'*' if wrong == 'rest' else bind}:{port}",
                                     f"{'*' if wrong == 'jmx' else bind}:{jmx}"]
                        record = {"checks": []}
                        inputs = dict(marker=marker, config=directory, interface="192.0.2.10", jmx=jmx,
                                      probe_classpath=directory, members=lambda selected=port, timeout=3: {member: pid},
                                      marked_processes=lambda: {pid},
                                      platform=types.SimpleNamespace(processes=lambda selected: {pid: command},
                                                                     snapshot=lambda selected: {"pid": pid, "command": command, "listeners": addresses},
                                                                     assert_listener_bind=native.assert_listener_bind),
                                      request=lambda selected: {"clusterName": marker},
                                      connect=lambda host, selected: selected == 59999 or (selected in (port, jmx) and bind != "127.0.0.1"),
                                      observer_lock=threading.Lock(), linux_identities={}, evidence=record)
                        with patch.object(management.socket, "socket", Control), patch.object(management.subprocess, "check_output", return_value=f"endpoint:[192.0.2.10:{jmx}] runtime={pid}@control"):
                            if bind == "0.0.0.0":
                                management.inspect("valid wildcard", 1, True, bind, **inputs)
                                self.assertEqual(len(record["checks"]), 1)
                            else:
                                with self.assertRaisesRegex(AssertionError, "incorrect listener bind"):
                                    management.inspect("incorrect wildcard", 1, True, bind, **inputs)
                                self.assertEqual(record["checks"], [])



if __name__ == "__main__":
    unittest.main(verbosity=2)
