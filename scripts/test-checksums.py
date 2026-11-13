#!/usr/bin/env python3
# Copyright (c) 2026 Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at
# https://oss.oracle.com/licenses/upl.
"""Hermetic checksum generator and public-release gate regression tests."""
import hashlib
import json
import os
from pathlib import Path
import re
import subprocess
import tempfile
import unittest

SCRIPTS = Path(__file__).resolve().parent
VERSION = "1.10.15"
REPOSITORIES = (None, "oracle/coherence-cli", "fixture-owner/fixture-repo")
PLATFORMS = ("linux/amd64/cohctl", "linux/arm64/cohctl", "linux/386/cohctl",
             "windows/amd64/cohctl.exe", "windows/arm64/cohctl.exe")
NAMES = [f"cohctl-{VERSION}-{p.rsplit('/', 1)[0].replace('/', '-')}{'.exe' if p.endswith('.exe') else ''}"
         for p in PLATFORMS] + ["docs.zip"]
MANIFEST = f"cohctl-{VERSION}-checksums.txt"
ASSETS = [MANIFEST] + NAMES
WORKFLOW = (SCRIPTS.parent / ".github/workflows/release.yml").read_text()
VERIFY_STEP = WORKFLOW.split("    - name: Verify public release downloads\n", 1)[1].split("    - name: ", 1)[0]
VERIFY_COMMAND = VERIFY_STEP.split("      run: ", 1)[1].strip()
PAGES_STEP = WORKFLOW.split("    - name: Release GitHub Pages Stable.txt\n", 1)[1].split("    - ", 1)[0]
PAGES_CONDITION = re.search(r"^      if: (.+)$", PAGES_STEP, re.MULTILINE)[1]
WORKFLOW_REPOSITORIES = ("oracle/coherence-cli", "fixture-owner/fixture-repo",
                         "another-fixture-owner/another-fixture-repo")


def public_urls(repository):
    repository = "oracle/coherence-cli" if repository is None else repository
    return [f"https://github.com/{repository}/releases/download/{VERSION}/{name}" for name in ASSETS]


class Checksums(unittest.TestCase):
    def setUp(self):
        self.directory = tempfile.TemporaryDirectory()
        self.addCleanup(self.directory.cleanup)
        self.root = Path(self.directory.name)
        self.bin = self.root / "bin"
        self.inputs = [self.bin / p for p in PLATFORMS] + [self.root / "docs.zip"]
        self.public = self.root / "public"
        self.public.mkdir()
        self.expected = ""
        for index, (path, name) in enumerate(zip(self.inputs, NAMES)):
            path.parent.mkdir(parents=True, exist_ok=True)
            data = f"distinct known fixture {index}\n".encode()
            path.write_bytes(data)
            (self.public / name).write_bytes(data)
            self.expected += f"{hashlib.sha256(data).hexdigest()}  {name}\n"
        self.output = self.root / "manifest.txt"
        self.temp = self.root / "temporary"
        self.temp.mkdir()
        self.env = dict(os.environ, TMPDIR=str(self.temp))
        self.env.pop("RELEASE_REPOSITORY", None)
        self.curl_log = self.root / "curl.jsonl"
        self.pages = self.root / "Pages marker"
        (self.root / "scripts").symlink_to(SCRIPTS, target_is_directory=True)

    def generate(self):
        return subprocess.run(["bash", str(SCRIPTS / "generate-checksums.sh"), VERSION,
                               str(self.bin), str(self.inputs[-1]), str(self.output)],
                              env=self.env, capture_output=True, text=True)

    def verify(self, manifest=None, public_manifest=None, fail_download=None, repository=None, workflow=False):
        manifest = self.expected if manifest is None else manifest
        self.output.write_text(manifest)
        (self.public / MANIFEST).write_text(
            manifest if public_manifest is None else public_manifest)
        self.curl_log.write_text("")
        mock = self.root / "mock"
        mock.mkdir(exist_ok=True)
        (mock / "curl").write_text('''#!/usr/bin/env python3
import json, os, pathlib, shutil, sys
args = sys.argv[1:]
with pathlib.Path(os.environ['CURL_LOG']).open('a') as log:
    log.write(json.dumps(args) + '\\n')
name = args[-1].rsplit('/', 1)[1]
url = 'https://github.com/' + os.environ['EXPECTED_REPOSITORY'] + '/releases/download/' + os.environ['EXPECTED_VERSION'] + '/' + name
# Exact arguments also prohibit authentication flags, headers or config files.
assert args == ['-fsSL', '--proto', '=https', '--proto-redir', '=https',
                '--retry', '5', '--retry-all-errors', '--retry-delay', '2',
                '--retry-max-time', '600', '--connect-timeout', '10', '--max-time', '180',
                '-o', args[-2], url], args
if os.environ['FAIL_DOWNLOAD'] == name: sys.exit(22)
shutil.copyfile(pathlib.Path(os.environ['PUBLIC']) / name, args[-2])
''')
        (mock / "curl").chmod(0o700)
        env = dict(self.env, PATH=f"{mock}:{os.environ['PATH']}", PUBLIC=str(self.public),
                   FAIL_DOWNLOAD=fail_download or "", CURL_LOG=str(self.curl_log), EXPECTED_VERSION=VERSION,
                   EXPECTED_REPOSITORY="oracle/coherence-cli" if repository is None else repository)
        if repository is not None and not workflow:
            env["RELEASE_REPOSITORY"] = repository
        command = ["bash", str(SCRIPTS / "verify-release.sh"), VERSION, str(self.output)]
        if workflow:
            self.pages.unlink(missing_ok=True)
            context = {"steps.version.outputs.version": VERSION, "github.repository": repository}
            for key, expression in re.findall(r"^        (\w+): \$\{\{ (.+?) \}\}$", VERIFY_STEP, re.MULTILINE):
                env[key] = context[expression]
            (self.root / MANIFEST).write_text(manifest)
            # The workflow's success-only Pages condition and adjacent ordering
            # are checked below. Model that step with a marker after the real
            # verifier under Actions' Bash failure flags. Never run Pages itself.
            self.assertEqual(PAGES_CONDITION, "${{ success() }}")
            env["PAGES_MARKER"] = str(self.pages)
            command = ["bash", "--noprofile", "--norc", "-e", "-o", "pipefail", "-c",
                       VERIFY_COMMAND + '\nprintf pages > "$PAGES_MARKER"']
        result = subprocess.run(command, cwd=self.root, env=env, capture_output=True, text=True)
        self.requests = [json.loads(line)[-1] for line in self.curl_log.read_text().splitlines()]
        self.assertEqual(list(self.temp.iterdir()), [], result.stderr)
        return result

    def test_known_values_exact_names_and_format(self):
        result = self.generate()
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(self.output.read_text(), self.expected)
        self.assertEqual(len(set(NAMES)), 6)
        self.assertEqual(list(self.root.glob("manifest.txt.tmp.*")), [])

    def test_each_missing_input_never_publishes_partial_manifest(self):
        for missing in self.inputs:
            data = missing.read_bytes()
            missing.unlink()
            for existing in (False, True):
                with self.subTest(missing=missing, existing=existing):
                    if existing:
                        self.output.write_text("preserve existing manifest\n")
                    result = self.generate()
                    self.assertNotEqual(result.returncode, 0)
                    if existing:
                        self.assertEqual(self.output.read_text(), "preserve existing manifest\n")
                        self.output.unlink()
                    else:
                        self.assertFalse(self.output.exists())
                    self.assertEqual(list(self.root.glob("manifest.txt.tmp.*")), [])
            missing.write_bytes(data)

    def test_hashing_failure_does_not_publish(self):
        mock = self.root / "mock"
        mock.mkdir()
        (mock / "sha256sum").write_text("#!/bin/bash\nexit 1\n")
        (mock / "sha256sum").chmod(0o700)
        self.env["PATH"] = f"{mock}:{os.environ['PATH']}"
        self.assertNotEqual(self.generate().returncode, 0)
        self.assertFalse(self.output.exists())
        self.assertEqual(list(self.root.glob("manifest.txt.tmp.*")), [])

    def test_public_gate_checks_all_six_assets(self):
        for repository in REPOSITORIES:
            with self.subTest(repository=repository):
                result = self.verify(repository=repository)
                self.assertEqual(result.returncode, 0, result.stderr)
                self.assertEqual(result.stdout.count(": OK"), 6)
                self.assertEqual(self.requests, public_urls(repository))
                if repository is not None and repository != "oracle/coherence-cli":
                    self.assertFalse(any("/oracle/" in url for url in self.requests))
            for name in NAMES:
                with self.subTest(repository=repository, corrupt=name):
                    path = self.public / name
                    data = path.read_bytes()
                    path.write_bytes(b"corrupted download\n")
                    result = self.verify(repository=repository)
                    self.assertNotEqual(result.returncode, 0)
                    self.assertIn(f"{name}: FAILED", result.stdout)
                    self.assertEqual(self.requests, public_urls(repository))
                    path.write_bytes(data)

    def test_public_gate_rejects_metadata_failures(self):
        lines = self.expected.splitlines(keepends=True)
        invalid = ["", "".join(lines[1:]), lines[0] + self.expected,
                   self.expected.upper(), self.expected.replace("  ", " ", 1),
                   self.expected + "0" * 64 + "  ../outside\n"]
        for repository in REPOSITORIES:
            for manifest in invalid:
                with self.subTest(repository=repository, manifest=manifest):
                    self.assertNotEqual(self.verify(manifest=manifest, repository=repository).returncode, 0)
                    self.assertEqual(self.requests, public_urls(repository)[:1])
            for public_manifest in (self.expected + "\n", self.expected.replace("\n", "\r\n"),
                                    "".join(reversed(lines))):
                with self.subTest(repository=repository, public_manifest=public_manifest):
                    self.assertNotEqual(self.verify(public_manifest=public_manifest, repository=repository).returncode, 0)
                    self.assertEqual(self.requests, public_urls(repository)[:1])

    def test_public_gate_rejects_each_download_failure(self):
        for repository in REPOSITORIES:
            for index, name in enumerate(ASSETS):
                with self.subTest(repository=repository, download=name):
                    result = self.verify(fail_download=name, repository=repository)
                    self.assertEqual(result.returncode, 22, result.stderr)
                    self.assertEqual(self.requests, public_urls(repository)[:index + 1])

    def test_invalid_repository_fails_before_network_access(self):
        invalid = ("", "oracle", "oracle/", "/coherence-cli", "oracle/coherence-cli/extra",
                   "https://github.com/oracle/coherence-cli", "http://github.com/oracle/coherence-cli",
                   "git@github.com:oracle/coherence-cli", " oracle/coherence-cli", "oracle/coherence-cli ",
                   "oracle/coherence cli", "oracle/coherence-cli\n", "oracle/\tcoherence-cli",
                   "oracle/coherence-cli\r", "../coherence-cli", "./coherence-cli", "oracle/.", "oracle/..",
                   "oracle/../coherence-cli", "oracle/%2e%2e", "oracle/repo?query", "oracle/repo#fragment",
                   "oracle/repo\\name", "oracle/repo;command", "oracle/$(command)", "oracle/`command`",
                   "oracle//coherence-cli", "or_acle/coherence-cli", "orácle/coherence-cli", "oracle/répo")
        for repository in invalid:
            with self.subTest(repository=repository):
                result = self.verify(repository=repository)
                self.assertNotEqual(result.returncode, 0)
                self.assertIn("Invalid RELEASE_REPOSITORY", result.stderr)
                self.assertEqual(self.requests, [])

    def test_repository_name_characters_are_preserved(self):
        repository = "Owner-1/Repo_2.3-rc"
        result = self.verify(repository=repository)
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(self.requests, public_urls(repository))

    def test_workflow_verification_success_allows_pages_for_each_repository(self):
        for repository in WORKFLOW_REPOSITORIES:
            with self.subTest(repository=repository):
                result = self.verify(repository=repository, workflow=True)
                self.assertEqual(result.returncode, 0, result.stderr)
                self.assertEqual(result.stdout.count(": OK"), 6)
                self.assertEqual(self.requests, public_urls(repository))
                self.assertEqual(self.pages.read_text(), "pages")

    def test_workflow_each_download_failure_blocks_pages(self):
        for repository in WORKFLOW_REPOSITORIES:
            for index, name in enumerate(ASSETS):
                with self.subTest(repository=repository, download=name):
                    result = self.verify(repository=repository, workflow=True, fail_download=name)
                    self.assertEqual(result.returncode, 22, result.stderr)
                    self.assertEqual(self.requests, public_urls(repository)[:index + 1])
                    self.assertFalse(self.pages.exists())

    def test_workflow_each_checksum_failure_blocks_pages(self):
        for repository in WORKFLOW_REPOSITORIES:
            for name in NAMES:
                with self.subTest(repository=repository, corrupt=name):
                    path = self.public / name
                    data = path.read_bytes()
                    path.write_bytes(b"corrupted download\n")
                    result = self.verify(repository=repository, workflow=True)
                    self.assertNotEqual(result.returncode, 0)
                    self.assertIn(f"{name}: FAILED", result.stdout)
                    self.assertEqual(self.requests, public_urls(repository))
                    self.assertFalse(self.pages.exists())
                    path.write_bytes(data)

    def test_workflow_manifest_mismatch_blocks_pages(self):
        for repository in WORKFLOW_REPOSITORIES:
            with self.subTest(repository=repository):
                result = self.verify(repository=repository, workflow=True,
                                     public_manifest=self.expected + "\n")
                self.assertNotEqual(result.returncode, 0)
                self.assertEqual(self.requests, public_urls(repository)[:1])
                self.assertFalse(self.pages.exists())


class Workflow(unittest.TestCase):
    def test_public_gate_uses_repository_context_before_pages(self):
        self.assertIn("        RELEASE_REPOSITORY: ${{ github.repository }}\n", VERIFY_STEP)
        self.assertIn("        RELEASE_VERSION: ${{ steps.version.outputs.version }}\n", VERIFY_STEP)
        self.assertNotIn("${{", VERIFY_COMMAND)
        self.assertNotIn("TOKEN", VERIFY_STEP)
        self.assertNotRegex(VERIFY_STEP, r"(?m)^\s+(if|continue-on-error):")
        steps = re.findall(r"^    - name: (.+)$", WORKFLOW, re.MULTILINE)
        start = steps.index("Create, validate and publish release")
        self.assertEqual(steps[start:start + 3], ["Create, validate and publish release", "Verify public release downloads",
                                                "Release GitHub Pages Stable.txt"])
        self.assertIn("make test-install test-checksums test-install-docs\n", WORKFLOW)

    def test_pages_requires_success(self):
        self.assertEqual(PAGES_CONDITION, "${{ success() }}")
        self.assertIn("        make release-ghpages\n", PAGES_STEP)
        self.assertNotRegex(WORKFLOW, r"(?m)^\s+continue-on-error:")


if __name__ == "__main__":
    unittest.main(verbosity=2)
