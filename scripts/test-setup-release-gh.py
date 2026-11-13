#!/usr/bin/env python3
# Copyright (c) 2026 Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at
# https://oss.oracle.com/licenses/upl.
"""Hermetic release CLI setup tests; real hashing/extraction, fixture gh/curl/uname."""
import hashlib
import io
import os
from pathlib import Path
import re
import shutil
import subprocess
import tarfile
import tempfile
import unittest

SOURCE = (Path(__file__).resolve().parent / "setup-release-gh.sh").read_text()
VERSION = re.search(r"^version='([^']+)'$", SOURCE, re.MULTILINE)[1]
PIN = re.search(r"^sha256='([0-9a-f]{64})'$", SOURCE, re.MULTILINE)[1]
TOKENS = ("GH_TOKEN", "GITHUB_TOKEN", "GH_ENTERPRISE_TOKEN", "GITHUB_ENTERPRISE_TOKEN")


class ReleaseGhSetup(unittest.TestCase):
    def setUp(self):
        self.directory = tempfile.TemporaryDirectory(prefix="release-gh-test-")
        self.addCleanup(self.directory.cleanup)
        self.root = Path(self.directory.name)
        self.runner = self.root / "runner temp"
        self.runner.mkdir()
        self.mock = self.root / "mock"
        self.mock.mkdir()
        self.path_file = self.root / "path"
        self.output_file = self.root / "output"
        self.path_file.touch()
        self.output_file.touch()
        self.archive = self.root / f"gh_{VERSION}_linux_amd64.tar.gz"
        self.env = dict(os.environ, RUNNER_TEMP=str(self.runner),
                        GITHUB_PATH=str(self.path_file), GITHUB_OUTPUT=str(self.output_file),
                        PATH=f"{self.mock}:{os.environ['PATH']}",
                        FIXTURE=str(self.archive), EVENTS=str(self.root / "events"),
                        EXECUTED=str(self.root / "executed"))
        self.env.update({token: "test-secret-must-not-be-inherited" for token in TOKENS})
        self.write_mock("uname", '#!/bin/bash\nif [ "$1" = -s ]; then echo Linux; else echo x86_64; fi\n')
        self.write_mock("curl", f'''#!{shutil.which("python3")}
import os, pathlib, shutil, sys
args = sys.argv[1:]
assert not any(key in os.environ for key in {TOKENS!r})
assert args[-1] == 'https://github.com/cli/cli/releases/download/v{VERSION}/gh_{VERSION}_linux_amd64.tar.gz'
for option, value in {{'--proto': '=https', '--proto-redir': '=https',
                      '--retry': '3', '--connect-timeout': '10', '--max-time': '180'}}.items():
    assert args[args.index(option) + 1] == value
assert '--fail' in args and '--tlsv1.2' in args
pathlib.Path(os.environ['EVENTS']).write_text('curl\\n')
if os.environ.get('FAIL_DOWNLOAD'): sys.exit(22)
shutil.copyfile(os.environ['FIXTURE'], args[args.index('--output') + 1])
''')
        real_tar = shutil.which("tar")
        self.write_mock("tar", f'''#!{shutil.which("python3")}
import os, pathlib, sys
with pathlib.Path(os.environ['EVENTS']).open('a') as events: events.write('tar\\n')
os.execv({real_tar!r}, [{real_tar!r}] + sys.argv[1:])
''')
        self.make_archive(VERSION)

    def write_mock(self, name, content):
        path = self.mock / name
        path.write_text(content)
        path.chmod(0o700)

    def make_archive(self, reported_version):
        binary = f'''#!/bin/bash
set -eu
for token in {' '.join(TOKENS)}; do
    if [ -n "${{!token+x}}" ]; then exit 90; fi
done
test "$#" = 1 && test "$1" = --version
test "$GH_NO_UPDATE_NOTIFIER" = 1
test "$GH_NO_EXTENSION_UPDATE_NOTIFIER" = 1
printf 'executed\\n' > "$EXECUTED"
printf 'gh version {reported_version} (fixture)\\nhttps://github.com/cli/cli/releases/tag/v{reported_version}\\n'
'''.encode()
        with tarfile.open(self.archive, "w:gz") as archive:
            for name, data, mode in (("bin/gh", binary, 0o755), ("LICENSE", b"fixture license\n", 0o644)):
                info = tarfile.TarInfo(f"gh_{VERSION}_linux_amd64/{name}")
                info.size, info.mode = len(data), mode
                archive.addfile(info, io.BytesIO(data))
        # Only the temporary test copy gets a fixture pin. Production is unchanged.
        digest = hashlib.sha256(self.archive.read_bytes()).hexdigest()
        self.helper = self.root / "setup-release-gh.sh"
        self.helper.write_text(SOURCE.replace(f"sha256='{PIN}'", f"sha256='{digest}'", 1))

    def run_helper(self):
        return subprocess.run(["bash", str(self.helper)], env=self.env, capture_output=True, text=True)

    def assert_failed_cleanly(self, result, extracted=False, executed=False):
        self.assertNotEqual(result.returncode, 0, result.stdout)
        self.assertEqual(self.path_file.read_text(), "")
        self.assertEqual(self.output_file.read_text(), "")
        self.assertEqual(list(self.runner.iterdir()), [])
        events = self.root / "events"
        self.assertEqual(events.exists() and "tar\n" in events.read_text(), extracted)
        self.assertEqual((self.root / "executed").exists(), executed)

    def test_verified_install_reports_version_and_publishes_local_path(self):
        result = self.run_helper()
        self.assertEqual(result.returncode, 0, result.stderr)
        install, = self.runner.iterdir()
        self.assertEqual(self.path_file.read_text(), f"{install}/bin\n")
        self.assertEqual(self.output_file.read_text(), f"version={VERSION}\npath={install}/bin/gh\n")
        self.assertIn(f"gh version {VERSION} (fixture)", result.stdout)
        self.assertTrue((self.root / "executed").exists())
        self.assertEqual(sorted(p.name for p in install.iterdir()), ["LICENSE", "bin"])

    def test_corrupt_download_fails_before_extraction_or_execution(self):
        with self.archive.open("ab") as archive:
            archive.write(b"tampered")
        result = self.run_helper()
        self.assertIn("FAILED", result.stdout)
        self.assert_failed_cleanly(result)

    def test_production_pin_rejects_unreviewed_archive_before_execution(self):
        self.helper.write_text(SOURCE)
        self.assert_failed_cleanly(self.run_helper())

    def test_download_failure_never_extracts_or_executes(self):
        self.env["FAIL_DOWNLOAD"] = "1"
        self.assert_failed_cleanly(self.run_helper())

    def test_hash_tool_failure_never_extracts_or_executes(self):
        self.write_mock("sha256sum", "#!/bin/bash\nexit 1\n")
        self.assert_failed_cleanly(self.run_helper())

    def test_verified_wrong_version_is_not_published(self):
        self.make_archive("0.0.0")
        result = self.run_helper()
        self.assertIn(f"Expected GitHub CLI {VERSION}, got 0.0.0", result.stderr)
        self.assert_failed_cleanly(result, extracted=True, executed=True)

    def test_verified_invalid_archive_is_not_published(self):
        self.archive.write_bytes(b"not a tar archive")
        digest = hashlib.sha256(self.archive.read_bytes()).hexdigest()
        self.helper.write_text(SOURCE.replace(f"sha256='{PIN}'", f"sha256='{digest}'", 1))
        self.assert_failed_cleanly(self.run_helper(), extracted=True)

    def test_unsupported_runner_and_missing_environment_fail_before_download(self):
        self.write_mock("uname", "#!/bin/bash\necho unsupported\n")
        self.assert_failed_cleanly(self.run_helper())
        self.assertFalse((self.root / "events").exists())
        self.write_mock("uname", '#!/bin/bash\nif [ "$1" = -s ]; then echo Linux; else echo x86_64; fi\n')
        del self.env["RUNNER_TEMP"]
        self.assert_failed_cleanly(self.run_helper())
        self.assertFalse((self.root / "events").exists())


if __name__ == "__main__":
    unittest.main(verbosity=2)
