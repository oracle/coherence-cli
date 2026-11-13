#!/usr/bin/env python3
# Copyright (c) 2026 Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at
# https://oss.oracle.com/licenses/upl.
"""Stage the seven release assets locally; publish a path only after verification."""
from pathlib import Path
import shutil
import sys
import tempfile

from release_support import (ReleaseError, parse_manifest, regular_file, release_assets,
                             require, required_env, sha256, validate_version)


def stage_assets(version, root, runner, output):
    validate_version(version)
    require(runner.is_absolute() and runner.is_dir(), "RUNNER_TEMP must be an absolute existing directory.")
    require(output.is_absolute(), "GITHUB_OUTPUT must be an absolute regular file.")
    regular_file(output)
    inventory = release_assets(version)
    inputs = {name: root / source for source, name, _mime in inventory}
    manifest_name = inventory[-1][1]
    manifest = inputs[manifest_name]
    regular_file(manifest)
    manifest_bytes = manifest.read_bytes()
    checksums = parse_manifest(manifest_bytes, inputs.keys() - {manifest_name})
    for source in inputs.values():
        regular_file(source)
        require(not output.samefile(source), "GITHUB_OUTPUT must not alias a release input.")

    staged = Path(tempfile.mkdtemp(prefix="release-assets-", dir=runner))
    try:
        require("\n" not in str(staged) and "\r" not in str(staged), "Invalid staging output path.")
        for name, source in inputs.items():
            shutil.copyfile(source, staged / name)
        require((staged / manifest_name).read_bytes() == manifest_bytes, "Staged manifest bytes changed.")
        for name, digest in checksums.items():
            require(sha256(staged / name) == digest, f"Staged SHA-256 mismatch: {name}")
        # The manifest is the seventh asset, never an entry in its own list.
        with output.open("a", encoding="utf-8") as stream:
            stream.write(f"path={staged}\n")
    except BaseException:
        # Only this invocation's unique directory is owned by the helper.
        shutil.rmtree(staged)
        raise
    return staged


def main():
    stage_assets(required_env("RELEASE_VERSION"), Path.cwd(),
                 Path(required_env("RUNNER_TEMP")), Path(required_env("GITHUB_OUTPUT")))
    print("Release assets staged and verified.")


if __name__ == "__main__":
    try:
        main()
    except (ReleaseError, OSError) as error:
        print(f"Release asset staging failed: {error}", file=sys.stderr)
        sys.exit(1)
