# Copyright (c) 2026 Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at
# https://oss.oracle.com/licenses/upl.

"""Native PID and TCP inspection for management integration tests."""

import json
import os
import platform
from pathlib import Path
import re
import shutil
import signal
import subprocess


def powershell(script, query):
    environment = dict(os.environ, COHCTL_PROCESS_QUERY=json.dumps(query))
    result = subprocess.run(["powershell.exe", "-NoProfile", "-NonInteractive", "-Command",
                             "$ErrorActionPreference='Stop'; $q=ConvertFrom-Json $env:COHCTL_PROCESS_QUERY; " + script],
                            env=environment, capture_output=True, text=True, timeout=20, check=True)
    return json.loads(result.stdout.lstrip("\ufeff"))


def prerequisites():
    required = ("powershell.exe",) if os.name == "nt" else ("ps", "lsof")
    missing = [name for name in required if not shutil.which(name)]
    if missing:
        raise RuntimeError("MISSING COVERAGE: native inspection requires " + ", ".join(missing))


def identity():
    record = {"system": platform.system(), "release": platform.release(), "machine": platform.machine()}
    if platform.system() == "Linux":
        for name in ("/proc/self/ns/net", "/proc/self/ns/pid"):
            record[name] = os.readlink(name)
    return record


def assert_listener_bind(addresses, port, bind_address):
    """A network rejection never substitutes for the requested socket bind."""
    selected = [address for address in addresses if address.rsplit(":", 1)[-1] == str(port)]
    allowed = {f"{bind_address}:{port}"}
    if bind_address == "0.0.0.0":
        allowed.add(f"*:{port}")  # Unix lsof's IPv4 wildcard representation.
    assert len(selected) == 1 and selected[0] in allowed, ("incorrect listener bind", bind_address, port, selected)


def processes(marker):
    pattern = r"(?:^|\s)-Dcoherence.cluster=" + re.escape(marker) + r"(?:\s|$)"
    if os.name == "nt":
        rows = powershell("ConvertTo-Json -Compress -InputObject @(Get-CimInstance Win32_Process | Where-Object { $_.CommandLine -match "
                          "('(?:^|\\s)-Dcoherence.cluster=' + [regex]::Escape($q.marker) + '(?:\\s|$)') } | "
                          "Select-Object ProcessId,CommandLine,CreationDate)", {"marker": marker})
        if isinstance(rows, dict):
            rows = [rows]
        return {int(row["ProcessId"]): row["CommandLine"] for row in rows or []}
    result = subprocess.run(["ps", "-axo", "pid=,command="], capture_output=True, text=True, timeout=10, check=True)
    found = {}
    for line in result.stdout.splitlines():
        fields = line.strip().split(None, 1)
        if len(fields) == 2 and re.search(pattern, fields[1]):
            found[int(fields[0])] = fields[1]
    return found


def snapshot(pid):
    if os.name == "nt":
        row = powershell("$p=Get-CimInstance Win32_Process -Filter ('ProcessId=' + $q.pid); "
                         "if (!$p) {throw 'owned PID absent'}; "
                         "$s=@(Get-NetTCPConnection -State Listen | Where-Object OwningProcess -eq $q.pid | "
                         "Select-Object LocalAddress,LocalPort,OwningProcess); "
                         "@{pid=[int]$p.ProcessId;command=$p.CommandLine;started=$p.CreationDate;tcp=$s} | ConvertTo-Json -Depth 5 -Compress",
                         {"pid": pid})
        assert row["pid"] == pid
        row["listeners"] = [str(s["LocalAddress"]) + ":" + str(s["LocalPort"]) for s in row.pop("tcp")]
        return row
    command = subprocess.check_output(["ps", "-p", str(pid), "-o", "command="], text=True, timeout=10).strip()
    raw = subprocess.check_output(["lsof", "-nP", "-a", "-p", str(pid), "-iTCP", "-sTCP:LISTEN", "-Fpn"],
                                  stderr=subprocess.DEVNULL, text=True, timeout=10)
    assert f"p{pid}\n" in raw, "socket PID identity missing"
    result = {"pid": pid, "command": command,
            "started": subprocess.check_output(["ps", "-p", str(pid), "-o", "lstart="], text=True, timeout=10).strip(),
            "listeners": [line[1:] for line in raw.splitlines() if line.startswith("n")], "raw_lsof": raw}
    if platform.system() == "Linux":
        result["linux_process_identity"] = linux_process_state(pid)
        assert result["linux_process_identity"] is not None, "owned PID exited during inspection"
    return result


def linux_process_state(pid):
    """Kernel birth identity survives exit-to-zombie and cannot follow PID reuse."""
    if platform.system() != "Linux":
        return None
    try:
        raw = (Path("/proc") / str(pid) / "stat").read_text()
    except (FileNotFoundError, ProcessLookupError):
        return None
    fields = raw[raw.rindex(") ") + 2:].split()
    return {"pid": pid, "state": fields[0], "ppid": int(fields[1]), "start_ticks": int(fields[19])}


def stop_owned(marker):
    """Recheck the unique marker immediately before each task-owned forced rescue."""
    stopped = []
    for pid in processes(marker):
        if pid in processes(marker):
            try:
                if os.name == "nt":
                    import ctypes
                    from ctypes import wintypes
                    kernel = ctypes.WinDLL("kernel32", use_last_error=True)
                    kernel.OpenProcess.argtypes = [wintypes.DWORD, wintypes.BOOL, wintypes.DWORD]
                    kernel.OpenProcess.restype = wintypes.HANDLE
                    kernel.TerminateProcess.argtypes = [wintypes.HANDLE, wintypes.UINT]
                    kernel.TerminateProcess.restype = wintypes.BOOL
                    kernel.CloseHandle.argtypes = [wintypes.HANDLE]
                    handle = kernel.OpenProcess(0x100001, False, pid)  # SYNCHRONIZE | PROCESS_TERMINATE
                    if not handle:
                        error = ctypes.get_last_error()
                        if error == 87:  # departed positive PID: ERROR_INVALID_PARAMETER
                            continue
                        raise ctypes.WinError(error)
                    try:
                        # Recheck AFTER retaining the handle. PID reuse cannot redirect termination.
                        if pid in processes(marker) and not kernel.TerminateProcess(handle, 1):
                            raise ctypes.WinError(ctypes.get_last_error())
                    finally:
                        kernel.CloseHandle(handle)
                else:
                    os.kill(pid, signal.SIGKILL)
                stopped.append(pid)
            except ProcessLookupError:
                pass
    return stopped
