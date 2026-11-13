#!/usr/bin/env python3
# Copyright (c) 2026 Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at
# https://oss.oracle.com/licenses/upl.
"""Hermetic exact-name/byte staging tests; no release or upload operations."""
import hashlib
import os
from pathlib import Path
import shutil
import subprocess
import tempfile
import types
import unittest
from unittest import mock

SCRIPTS = Path(__file__).resolve().parent
SOURCE = (SCRIPTS / "stage-release-assets.py").read_text()
HELPER = types.ModuleType("staging")
exec(compile(SOURCE, str(SCRIPTS / "stage-release-assets.py"), "exec"), HELPER.__dict__)
VERSION = "1.10.15"
SOURCES = ("bin/linux/amd64/cohctl", "bin/linux/arm64/cohctl", "bin/linux/386/cohctl",
           "bin/windows/amd64/cohctl.exe", "bin/windows/arm64/cohctl.exe", "build/_output/docs.zip")


def names(version):
    return [f"cohctl-{version}-linux-amd64", f"cohctl-{version}-linux-arm64",
            f"cohctl-{version}-linux-386", f"cohctl-{version}-windows-amd64.exe",
            f"cohctl-{version}-windows-arm64.exe", "docs.zip", f"cohctl-{version}-checksums.txt"]


def snapshot(root):
    result = {}
    for path in root.rglob("*"):
        value = os.readlink(path) if path.is_symlink() else path.read_bytes() if path.is_file() else None
        result[str(path.relative_to(root))] = (path.lstat().st_mode, value)
    return result


class Staging(unittest.TestCase):
    def setUp(self):
        self.directory = tempfile.TemporaryDirectory(prefix="release-staging-test-")
        self.addCleanup(self.directory.cleanup)
        self.root = Path(self.directory.name)
        # Shell metacharacters and spaces in checkout/runner paths remain data.
        self.repo = self.root / "checkout space $(touch injected)"
        self.repo.mkdir()
        (self.repo / "scripts").mkdir()
        (self.repo / "scripts/stage-release-assets.py").write_text(SOURCE)
        shutil.copyfile(SCRIPTS / "release_support.py", self.repo / "scripts/release_support.py")
        self.runner = self.root / "runner space #literal"
        self.runner.mkdir()
        (self.runner / "unrelated").mkdir()
        (self.runner / "unrelated/keep").write_bytes(b"existing runner content")
        self.output = self.root / "step output"
        self.output.touch()
        self.env = dict(os.environ, RELEASE_VERSION=VERSION, RUNNER_TEMP=str(self.runner),
                        GITHUB_OUTPUT=str(self.output), PYTHONDONTWRITEBYTECODE="1")
        self.inputs = [self.repo / path for path in SOURCES]
        for index, path in enumerate(self.inputs):
            path.parent.mkdir(parents=True, exist_ok=True)
            path.write_bytes(bytes(range(256)) + f"distinct payload {index}\n".encode())
            path.chmod(0o755 if index < 5 else 0o644)
        for extra in ("bin/cohctl", "bin/darwin/amd64/cohctl", "bin/darwin/arm64/cohctl",
                      "build/_output/docs/index.html", "unrelated.txt"):
            path = self.repo / extra
            path.parent.mkdir(parents=True, exist_ok=True)
            path.write_bytes(b"unselected build output\n")
        self.manifest = self.repo / names(VERSION)[-1]
        self.expected = self.manifest_bytes(VERSION)
        self.manifest.write_bytes(self.expected)

    def manifest_bytes(self, version):
        return b"".join(f"{hashlib.sha256(path.read_bytes()).hexdigest()}  {name}\n".encode()
                        for path, name in zip(self.inputs, names(version)))

    def invoke(self):
        before = snapshot(self.repo)
        result = subprocess.run(["python3", "./scripts/stage-release-assets.py"], cwd=self.repo,
                                env=self.env, capture_output=True, text=True, timeout=20)
        self.assertEqual(snapshot(self.repo), before, "Staging changed source/build files")
        self.assertFalse((self.repo / "injected").exists())
        return result

    def failed(self):
        before = snapshot(self.runner)
        output_before = self.output.read_bytes()
        result = self.invoke()
        self.assertNotEqual(result.returncode, 0, result.stdout)
        self.assertEqual(self.output.read_bytes(), output_before)
        self.assertEqual(snapshot(self.runner), before, "Failed staging left partial output")
        self.assertNotIn("staged and verified", result.stdout)
        return result

    def assert_staged(self, version=VERSION):
        result = self.invoke()
        self.assertEqual(result.returncode, 0, result.stderr)
        record = self.output.read_text().splitlines()[-1]
        self.assertTrue(record.startswith("path="))
        staged = Path(record[len("path="):])
        self.assertTrue(staged.is_absolute())
        self.assertEqual(staged.parent, self.runner)
        self.assertEqual(sorted(path.name for path in staged.iterdir()), sorted(names(version)))
        for source, name in zip(self.inputs + [self.repo / names(version)[-1]], names(version)):
            self.assertEqual((staged / name).read_bytes(), source.read_bytes())
            self.assertFalse((staged / name).is_symlink())
        return staged

    def test_generator_manifest_exact_seven_names_and_bytes(self):
        result = subprocess.run(["bash", str(SCRIPTS / "generate-checksums.sh"), VERSION,
                                 str(self.repo / "bin"), str(self.inputs[-1]), str(self.manifest)],
                                capture_output=True, text=True, timeout=20)
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(self.manifest.read_bytes(), self.expected)
        self.assertEqual(len(self.expected.splitlines()), 6)
        self.assertEqual(self.output.read_text(), "")
        self.assert_staged()
        self.assertEqual(len(self.output.read_text().splitlines()), 1)

    def test_unique_successful_staging_survives_later_runs_and_failures(self):
        first = self.assert_staged()
        first_snapshot = snapshot(first)
        second = self.assert_staged()
        self.assertNotEqual(first, second)
        self.assertEqual(snapshot(first), first_snapshot)
        self.manifest.write_bytes(b"invalid\n")
        self.failed()

    def test_valid_reordered_manifest_is_copied_without_rewriting(self):
        self.manifest.write_bytes(b"".join(reversed(self.expected.splitlines(keepends=True))))
        self.assert_staged()

    def test_each_missing_input_including_manifest_fails_without_output(self):
        for source in self.inputs + [self.manifest]:
            with self.subTest(missing=source.relative_to(self.repo)):
                data, mode = source.read_bytes(), source.stat().st_mode
                source.unlink()
                self.failed()
                source.write_bytes(data)
                source.chmod(mode)

    def test_nonregular_and_symlink_inputs_are_rejected(self):
        for source in self.inputs + [self.manifest]:
            data = source.read_bytes()
            source.unlink()
            with self.subTest(source=source, kind="directory"):
                source.mkdir()
                self.failed()
                source.rmdir()
            with self.subTest(source=source, kind="symlink"):
                source.symlink_to(self.repo / "unrelated.txt")
                self.failed()
                source.unlink()
            with self.subTest(source=source, kind="fifo"):
                os.mkfifo(source)
                self.failed()
                source.unlink()
            source.write_bytes(data)

    def test_each_corrupt_payload_is_rejected(self):
        for source in self.inputs:
            with self.subTest(source=source):
                data = source.read_bytes()
                source.write_bytes(data + b"corruption")
                self.failed()
                source.write_bytes(data)

    def test_missing_duplicate_unexpected_and_self_checksum_entries(self):
        lines = self.expected.splitlines(keepends=True)
        invalid = [b"", b"".join(lines[:-1]), self.expected + lines[0],
                   b"".join(lines[:-1]) + lines[0],
                   self.expected.replace(b"docs.zip", b"extra.zip"),
                   self.expected.replace(b"docs.zip", self.manifest.name.encode())]
        for data in invalid:
            with self.subTest(manifest=data):
                self.manifest.write_bytes(data)
                self.failed()

    def test_path_like_and_label_names_are_rejected_before_any_copy(self):
        for name in (b"../outside", b"/tmp/outside", b"nested/docs.zip", b"./docs.zip",
                     b"..\\outside", b"C:\\docs.zip", b"docs.zip#label", b"docs.zip\x00"):
            with self.subTest(name=name):
                self.manifest.write_bytes(self.expected.replace(b"docs.zip", name))
                with mock.patch.object(HELPER.shutil, "copyfile") as copy:
                    with self.assertRaises(HELPER.ReleaseError):
                        HELPER.stage_assets(VERSION, self.repo, self.runner, self.output)
                    copy.assert_not_called()
                self.failed()

    def test_malformed_manifest_is_rejected(self):
        first_digest = self.expected[:64]
        invalid = [self.expected[:-1], self.expected + b"\n", self.expected.replace(b"\n", b"\r\n"),
                   self.expected.replace(b"  ", b" ", 1), self.expected.replace(b"  ", b"\t", 1),
                   self.expected.replace(b"  ", b" *", 1), self.expected.replace(b"docs.zip", b"docs.zip extra"),
                   self.expected.replace(first_digest, first_digest.upper(), 1),
                   self.expected.replace(first_digest, b"g" * 64, 1),
                   self.expected.replace(first_digest, b"0" * 63, 1), b"\xef\xbb\xbf" + self.expected,
                   self.expected.replace(b"docs.zip", b"docs.zip\xff")]
        for data in invalid:
            with self.subTest(manifest=data):
                self.manifest.write_bytes(data)
                self.failed()

    def test_wrong_but_well_formed_digest_is_rejected(self):
        self.manifest.write_bytes(b"0" * 64 + self.expected[64:])
        self.failed()

    def test_compatible_version_characters_remain_literal_filenames(self):
        for version in ("1.10.15-rc.1", "1.10.15_rc-2", "release_1.10.15", "1"):
            with self.subTest(version=version):
                self.env["RELEASE_VERSION"] = version
                (self.repo / names(version)[-1]).write_bytes(self.manifest_bytes(version))
                self.assert_staged(version)

    def test_invalid_version_and_missing_environment(self):
        for version in ("", ".", "..", "-1.2.3", "1/2", "../1.2.3", "1\\2", "1.2.3#label",
                        "1.2.3+meta", "1.2.3\n", "1.2.3\r", "1.2.3 ", "é1", "$(touch injected)", "`touch injected`"):
            with self.subTest(version=version):
                self.env["RELEASE_VERSION"] = version
                self.failed()
        self.env["RELEASE_VERSION"] = VERSION
        for key in ("RELEASE_VERSION", "RUNNER_TEMP", "GITHUB_OUTPUT"):
            with self.subTest(missing=key):
                value = self.env.pop(key)
                self.failed()
                self.env[key] = value

    def test_invalid_runner_and_output_paths(self):
        for key, value in (("RUNNER_TEMP", "relative"), ("RUNNER_TEMP", str(self.root / "missing")),
                           ("RUNNER_TEMP", str(self.output)), ("RUNNER_TEMP", str(self.runner) + "\n"),
                           ("GITHUB_OUTPUT", "relative"), ("GITHUB_OUTPUT", str(self.root / "missing")),
                           ("GITHUB_OUTPUT", str(self.runner)), ("GITHUB_OUTPUT", str(self.output) + "\n"),
                           ("GITHUB_OUTPUT", str(self.inputs[0])), ("GITHUB_OUTPUT", str(self.manifest))):
            with self.subTest(key=key, value=value):
                original = self.env[key]
                self.env[key] = value
                self.failed()
                self.env[key] = original

    def test_partial_copy_failure_and_staged_corruption_remove_only_owned_output(self):
        copyfile = shutil.copyfile
        for failure in ("copy", "payload", "manifest"):
            with self.subTest(failure=failure):
                before = snapshot(self.runner)
                sources = snapshot(self.repo)

                def faulty_copy(source, destination):
                    if failure == "copy" and source == self.inputs[2]:
                        destination.write_bytes(b"partial write")
                        raise OSError("injected copy failure")
                    copyfile(source, destination)
                    if ((failure == "payload" and source == self.inputs[3])
                            or (failure == "manifest" and source == self.manifest)):
                        destination.write_bytes(b"corrupt staged copy")

                with mock.patch.object(HELPER.shutil, "copyfile", side_effect=faulty_copy):
                    with self.assertRaises((OSError, HELPER.ReleaseError)):
                        HELPER.stage_assets(VERSION, self.repo, self.runner, self.output)
                self.assertEqual(snapshot(self.runner), before)
                self.assertEqual(snapshot(self.repo), sources)
                self.assertEqual(self.output.read_bytes(), b"")

    def test_output_failure_removes_verified_staging(self):
        before = snapshot(self.runner)
        real_open = Path.open

        def failing_output(path, *args, **kwargs):
            if path == self.output:
                # All seven files already exist and verify before publication.
                staged, = self.runner.glob("release-assets-*")
                self.assertEqual(len(list(staged.iterdir())), 7)
                raise OSError("injected output failure")
            return real_open(path, *args, **kwargs)

        with mock.patch.object(Path, "open", failing_output):
            with self.assertRaisesRegex(OSError, "injected output failure"):
                HELPER.stage_assets(VERSION, self.repo, self.runner, self.output)
        self.assertEqual(snapshot(self.runner), before)
        self.assertEqual(self.output.read_bytes(), b"")


if __name__ == "__main__":
    unittest.main(verbosity=2)
