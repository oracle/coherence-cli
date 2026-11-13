# Copyright (c) 2026 Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at
# https://oss.oracle.com/licenses/upl.
"""Local release asset definitions and validation; no CLI or GitHub operations."""
import hashlib
import os
import re
import stat

# Source-relative path, uploaded filename, historical Content-Type. The last
# asset is the manifest; it lists only the six preceding payloads, never itself.
ASSETS = (
    ("bin/linux/amd64/cohctl", "cohctl-{version}-linux-amd64", "application/x-mach-binary"),
    ("bin/linux/arm64/cohctl", "cohctl-{version}-linux-arm64", "application/x-executable"),
    ("bin/linux/386/cohctl", "cohctl-{version}-linux-386", "application/x-executable"),
    ("bin/windows/amd64/cohctl.exe", "cohctl-{version}-windows-amd64.exe", "application/x-dosexec"),
    ("bin/windows/arm64/cohctl.exe", "cohctl-{version}-windows-arm64.exe", "application/x-dosexec"),
    ("build/_output/docs.zip", "docs.zip", "application/zip"),
    ("cohctl-{version}-checksums.txt", "cohctl-{version}-checksums.txt", "text/plain"),
)


class ReleaseError(Exception):
    """Release inputs or state could not be verified."""


def require(condition, message):
    if not condition:
        raise ReleaseError(message)


def validate_version(version):
    # Match the shell generator/verifier's literal filename convention.
    require(re.fullmatch(r"[0-9A-Za-z][0-9A-Za-z._-]*", version) is not None, "Invalid release version.")


def release_assets(version):
    return tuple((source.format(version=version), name.format(version=version), mime)
                 for source, name, mime in ASSETS)


def required_env(name):
    value = os.environ.get(name, "")
    require(bool(value) and "\n" not in value and "\r" not in value,
            f"{name} must be set to a single-line value.")
    return value


def regular_file(path):
    require(stat.S_ISREG(path.lstat().st_mode), f"Expected a regular non-symlink file: {path}")


def parse_manifest(data, names):
    # Validate every line and the complete allowlist before using any name as a
    # path. Accept precisely the six-entry format from generate-checksums.sh.
    lines = data.split(b"\n")
    require(len(lines) == 7 and lines[-1] == b"", "Expected six newline-terminated checksum entries.")
    checksums = {}
    for line in lines[:-1]:
        match = re.fullmatch(rb"([0-9a-f]{64})[ ]{2}([A-Za-z0-9._-]+)", line)
        require(match is not None, "Malformed checksum entry.")
        digest, name = (part.decode("ascii") for part in match.groups())
        require(name in names and name not in checksums, "Unexpected or duplicate checksum name.")
        checksums[name] = digest
    require(checksums.keys() == names, "Missing release checksum entries.")
    return checksums


def sha256(path):
    digest = hashlib.sha256()
    with path.open("rb") as payload:
        for block in iter(lambda: payload.read(1024 * 1024), b""):
            digest.update(block)
    return digest.hexdigest()
