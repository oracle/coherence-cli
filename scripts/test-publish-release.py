#!/usr/bin/env python3
# Copyright (c) 2026 Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at
# https://oss.oracle.com/licenses/upl.
"""Hermetic publication fixtures. No live CLI, credentials or GitHub writes."""
import hashlib
import io
import json
import os
from pathlib import Path
import re
import shutil
import subprocess
import sys
import tempfile
import types
import unittest
from unittest import mock

SCRIPTS = Path(__file__).resolve().parent
HELPER = types.ModuleType("publication")
exec(compile((SCRIPTS / "publish-release.py").read_text(), str(SCRIPTS / "publish-release.py"), "exec"),
     HELPER.__dict__)
WORKFLOW = (SCRIPTS.parent / ".github/workflows/release.yml").read_text()
NAME = "Create, validate and publish release"
STEP = WORKFLOW.split(f"    - name: {NAME}\n", 1)[1].split("    - name: ", 1)[0]
COMMAND = STEP.split("      run: ", 1)[1].strip()
VERSION = "1.11.0"
NOTES = (f"# Coherence CLI {VERSION}\n\n"
         "## Upgrade actions\n\n"
         "* Review TLS exceptions — keep `false` when certificates are trusted.\n"
         '* Preserve literal $(touch injected), `commands`, and "quoted" text.\n')
REPOSITORY = "fixture-owner/fixture-repo"
NAMES = [f"cohctl-{VERSION}-linux-amd64", f"cohctl-{VERSION}-linux-arm64", f"cohctl-{VERSION}-linux-386",
         f"cohctl-{VERSION}-windows-amd64.exe", f"cohctl-{VERSION}-windows-arm64.exe",
         "docs.zip", f"cohctl-{VERSION}-checksums.txt"]
MIMES = ["application/x-mach-binary", "application/x-executable", "application/x-executable",
         "application/x-dosexec", "application/x-dosexec", "application/zip", "text/plain"]
WRITES = ["create", *["upload:" + name for name in NAMES], "publish"]

# The executable accepts only the intended commands. It stores remote state and
# exact request bytes in local files. It has no networking code or CLI fallback.
CLI = r'''
import hashlib, json, os, pathlib, sys
args = sys.argv[1:]
state_path = pathlib.Path(os.environ['FIXTURE_STATE'])
s = json.loads(state_path.read_text())
f = s['fault']
repo, version, sha = s['repo'], s['version'], s['sha']
prefix = 'repos/' + repo
assert os.environ['GH_TOKEN'] == 'fixture-token'
assert os.environ['GH_HOST'] == 'github.com'
assert os.environ['GH_PROMPT_DISABLED'] == '1'
assert os.environ['GH_NO_UPDATE_NOTIFIER'] == '1'
assert os.environ['GH_NO_EXTENSION_UPDATE_NOTIFIER'] == '1'
assert os.environ['GH_PAGER'] == os.environ['PAGER'] == 'cat'
assert {k for k in os.environ if k.startswith(('GH_', 'GITHUB_'))} == {
    'GH_TOKEN', 'GH_HOST', 'GH_PROMPT_DISABLED', 'GH_NO_UPDATE_NOTIFIER',
    'GH_NO_EXTENSION_UPDATE_NOTIFIER', 'GH_PAGER', 'GH_CONFIG_DIR'}
config = pathlib.Path(os.environ['GH_CONFIG_DIR'])
assert config.is_absolute() and config.is_dir() and list(config.iterdir()) == []
assert config != pathlib.Path(os.environ['BAD_CONFIG'])
s['configs'].append(str(config))
url = 'https://api.github.com/' + prefix + '/releases/42'
upload = 'https://uploads.github.com/' + prefix + '/releases/42/assets'
html = 'https://github.com/' + repo + '/releases/tag/untagged-fixture'
extra = []
if args[:2] == ['release', 'create']:
    assert args == ['release', 'create', version, '--repo', 'github.com/' + repo,
                    '--verify-tag', '--draft', '--prerelease=false', '--target', sha,
                    '--title', 'Coherence CLI - ' + version, '--notes-file', args[-1]], args
    notes_file = pathlib.Path(args[-1])
    assert notes_file.is_absolute() and notes_file.parent == config.parent
    assert notes_file.read_text(encoding='utf-8') == s['notes']
    op = 'create'
else:
    assert args[:5] == ['api', '--hostname', 'github.com', '--method', args[4]], args
    method = args[4]
    assert args[5] == '--header' and args[6].startswith('Accept: '), args
    accept = args[6][8:]
    assert args[7:9] == ['--header', 'X-GitHub-Api-Version: 2022-11-28'], args
    extra, endpoint = args[9:-1], args[-1]
    if method == 'POST' and endpoint == prefix + '/releases/generate-notes':
        assert extra == ['--raw-field', 'tag_name=' + version,
                         '--raw-field', 'target_commitish=' + sha], args
        assert accept == 'application/vnd.github+json'
        op = 'notes'
    elif method == 'POST':
        name = endpoint.split('?name=')[1]
        assert endpoint == upload + '?name=' + name, args
        assert name in s['names'], args
        assert extra[:3] == ['--header', 'Content-Type: ' + s['mimes'][s['names'].index(name)], '--input'], args
        assert len(extra) == 4 and pathlib.Path(extra[3]) == pathlib.Path(s['assets']) / name, args
        assert accept == 'application/vnd.github+json'
        op = 'upload:' + name
    elif method == 'PATCH':
        assert endpoint == prefix + '/releases/42' and extra == ['--field', 'draft=false'], args
        assert accept == 'application/vnd.github+json'
        op = 'publish'
    else:
        assert method == 'GET'
        if endpoint == prefix:
            op = 'repository'
        elif endpoint == prefix + '/git/ref/tags/' + version:
            op = 'tag'
        elif endpoint == prefix + '/releases?per_page=100':
            op = 'releases'
        elif endpoint == prefix + '/releases/42/assets?per_page=100':
            op = 'assets'
        elif endpoint == prefix + '/releases/42':
            op = 'release'
        elif endpoint.startswith(prefix + '/releases/assets/'):
            identity = int(endpoint.rsplit('/', 1)[1])
            name = next(a['name'] for a in s['remote_assets'] if a['id'] == identity)
            op = 'download:' + name
        else:
            raise AssertionError(args)
        if op in ('releases', 'assets'):
            assert extra == ['--paginate', '--slurp'], args
        elif op.startswith('download:'):
            assert extra == ['--allow-escape-sequences'] and accept == 'application/octet-stream', args
        else:
            assert extra == [], args
        if not op.startswith('download:'):
            assert accept == 'application/vnd.github+json', args
s['events'].append(op)
occurrence = s['events'].count(op)
def save():
    state_path.write_text(json.dumps(s))
def stop():
    save()
    # Deliberate mock secret in stderr and stdout must never reach user logs.
    print('fixture-token response body')
    print('fixture-token debug stderr', file=sys.stderr)
    sys.exit(23)
if f.get('fail_before') == op and f.get('at', occurrence) == occurrence:
    stop()
result = None
raw = None
if op == 'create':
    if f.get('tag_disappeared'):
        stop()
    s['release'] = dict(id=42, url=url, upload_url=upload + '{?name,label}', html_url=html,
                        tag_name=version, target_commitish=sha, name='Coherence CLI - ' + version,
                        body=notes_file.read_text(encoding='utf-8'), draft=True, prerelease=False)
    if f.get('change_staged'):
        pathlib.Path(s['assets'], f['change_staged']).write_bytes(b'changed after validation')
    if f.get('seed_asset'):
        s['remote_assets'] = [dict(id=777, name='unexpected')]
    raw = html.encode() + b'\n'
elif op == 'repository':
    result = f.get('repository_response', dict(full_name=repo, permissions=dict(push=True)))
elif op == 'tag':
    result = dict(ref='refs/tags/' + version, object=dict(type='commit', sha=sha))
elif op == 'notes':
    # Documented, non-persisting contents-write probe. These notes must never
    # replace the release's existing explicit title and body.
    result = f.get('notes_response', dict(name='Discarded generated title', body='Discarded generated body'))
elif op == 'releases':
    old = dict(id=1, tag_name='old-version', draft=False,
               html_url='https://github.com/' + repo + '/releases/tag/old-version')
    if f.get('existing') and s['release'] is None:
        result = [[old], [dict(id=40, tag_name=version, draft=f['existing'] == 'draft')]]
    else:
        result = [[old], [s['release']] if s['release'] else []]
    if s['release'] and f.get('wrong_created_url'):
        result[-1][0]['html_url'] += '-different'
    if s['release'] and f.get('duplicate_release'):
        result[-1].append(dict(s['release'], id=43))
    if s['release'] and occurrence <= 1 + f.get('missing_created_reads', 0):
        # Model a release list that has not exposed the newly created draft.
        # This is a regression scenario, not a captured hosted API response.
        result = [[old], []]
elif op.startswith('upload:'):
    data = pathlib.Path(extra[3]).read_bytes()
    a = dict(id=100 + len(s['remote_assets']), name=name, label=None, state='uploaded', size=len(data),
             content_type=s['mimes'][s['names'].index(name)])
    a['url'] = 'https://api.github.com/' + prefix + '/releases/assets/' + str(a['id'])
    if not f.get('omit_digest'):
        a['digest'] = 'sha256:' + hashlib.sha256(data).hexdigest()
    s['bytes'][name] = data.hex()
    s['remote_assets'].append(a)
    result = dict(a)
    if f.get('bad_upload'):
        result.update(f['bad_upload'])
elif op == 'assets':
    result = [s['remote_assets'][:3], s['remote_assets'][3:]]
    if len(s['remote_assets']) == 7 and f.get('at', occurrence) == occurrence:
        result = json.loads(json.dumps(result))
        if f.get('asset_list') == 'extra':
            result[-1].append(dict(result[0][0], id=999, name='unexpected'))
        elif f.get('asset_list') == 'missing':
            result[-1].pop()
        elif f.get('asset_list') == 'duplicate':
            result[-1][-1] = dict(result[0][0])
        elif f.get('asset_list') == 'changed-id':
            result[0][0]['id'] = 999
        elif f.get('asset_list') == 'malformed':
            result[0][0] = None
elif op == 'release':
    result = dict(s['release'])
    if f.get('bad_release') and f.get('at', occurrence) == occurrence:
        result.update(f['bad_release'])
elif op.startswith('download:'):
    raw = bytes.fromhex(s['bytes'][name])
    if f.get('corrupt_download') == name:
        raw = b'corrupt'
elif op == 'publish':
    assert s['release']['draft'] and len(s['remote_assets']) == 7
    assert all('download:' + n in s['events'] for n in s['names'])
    s['release']['draft'] = False
    result = s['release']
if f.get('fail_after') == op and f.get('at', occurrence) == occurrence:
    stop()
save()
if f.get('malformed') == op and f.get('at', occurrence) == occurrence:
    sys.stdout.buffer.write(b'{malformed')
elif raw is not None:
    sys.stdout.buffer.write(raw)
else:
    print(json.dumps(result))
'''


class Publication(unittest.TestCase):
    def setUp(self):
        self.directory = tempfile.TemporaryDirectory(prefix="release-publication-test-")
        self.addCleanup(self.directory.cleanup)
        self.root = Path(self.directory.name)
        self.repo = self.root / "checkout space $(touch injected)"
        self.repo.mkdir()
        subprocess.run(["git", "init", "--quiet", str(self.repo)], check=True)
        self.notes = self.repo / "release-notes" / f"{VERSION}.md"
        self.notes.parent.mkdir()
        self.notes.write_text(NOTES, encoding="utf-8")
        self.expected_notes = NOTES.strip()
        subprocess.run(["git", "add", "release-notes"], cwd=self.repo, check=True)
        subprocess.run(["git", "-c", "user.name=Fixture", "-c", "user.email=fixture@example.invalid",
                        "-c", "commit.gpgsign=false", "commit", "--quiet", "--allow-empty", "-m", "fixture"],
                       cwd=self.repo, check=True)
        self.sha = subprocess.check_output(["git", "rev-parse", "HEAD"], cwd=self.repo, text=True).strip()
        scripts = self.repo / "scripts"
        scripts.mkdir()
        for name in ("publish-release.py", "release_support.py"):
            shutil.copyfile(SCRIPTS / name, scripts / name)
        self.runner = self.root / "runner space #literal"
        self.runner.mkdir()
        (self.runner / "unrelated").write_bytes(b"keep")
        self.assets = self.root / "staged space #literal"
        self.assets.mkdir()
        for index, name in enumerate(NAMES[:-1]):
            (self.assets / name).write_bytes(bytes(range(256)) + bytes([index]) + b'\x1b[31m\x00\xff')
        (self.assets / NAMES[-1]).write_text("".join(
            hashlib.sha256((self.assets / name).read_bytes()).hexdigest() + "  " + name + "\n"
            for name in NAMES[:-1]))
        self.gh = self.root / "pinned gh"
        self.gh.write_text(f"#!{sys.executable}\n" + CLI)
        self.gh.chmod(0o700)
        self.bad_config = self.root / "stored config"
        self.bad_config.mkdir()
        (self.bad_config / "hosts.yml").write_text("wrong.invalid: {oauth_token: wrong-token}\n")
        no_gh = self.root / "path override"
        no_gh.mkdir()
        (no_gh / "gh").write_text("#!/bin/sh\nexit 99\n")
        (no_gh / "gh").chmod(0o700)
        self.local_git = self.root / "local Git isolation marker"
        (no_gh / "git").write_text(f'''#!{sys.executable}
import os, pathlib, re, sys
assert (sys.argv[1:] == ['rev-parse', '--verify', 'HEAD^{{commit}}'] or
        len(sys.argv) == 3 and sys.argv[1] == 'check-ref-format' or
        len(sys.argv) == 3 and sys.argv[1] == 'show' and
        re.fullmatch(r'[0-9a-f]{{40}}:release-notes/[0-9A-Za-z][0-9A-Za-z._-]*\\.md', sys.argv[2]))
assert not any(k.startswith(('GH_', 'GITHUB_')) for k in os.environ)
pathlib.Path({str(self.local_git)!r}).touch()
os.execv({shutil.which('git')!r}, [{shutil.which('git')!r}] + sys.argv[1:])
''')
        (no_gh / "git").chmod(0o700)
        self.state_path = self.root / "state.json"
        self.public = self.root / "public verification marker"
        # Resolve all workflow values from the declared environment, never from
        # interpolation in the command. These are fixture identities and tokens.
        context = {"steps.release_gh.outputs.path": str(self.gh),
                   "steps.release_assets.outputs.path": str(self.assets),
                   "steps.version.outputs.version": VERSION, "github.repository": REPOSITORY,
                   "github.ref": "refs/tags/" + VERSION, "github.ref_type": "tag",
                   "github.sha": self.sha, "github.token": "fixture-token"}
        self.env = dict(os.environ, FIXTURE_STATE=str(self.state_path), BAD_CONFIG=str(self.bad_config),
                        RUNNER_TEMP=str(self.runner), GH_CONFIG_DIR=str(self.bad_config), GH_DEBUG="api",
                        GH_HOST="wrong.invalid", GH_REPO="wrong/repo", GH_API_HOST="wrong.invalid",
                        GITHUB_TOKEN="wrong-token", GH_ENTERPRISE_TOKEN="wrong-token",
                        GITHUB_ENTERPRISE_TOKEN="wrong-token", GH_PAGER="bad-pager", PAGER="bad-pager",
                        GITHUB_RUN_ID="12345", PATH=str(no_gh) + ":" + os.environ["PATH"],
                        PUBLIC_MARKER=str(self.public))
        for key, expression in re.findall(r"^        (\w+): \$\{\{ (.+?) \}\}$", STEP, re.MULTILINE):
            self.env[key] = context[expression]

    def invoke(self, fault=None):
        self.public.unlink(missing_ok=True)
        initial = dict(repo=REPOSITORY, version=VERSION, sha=self.sha, names=NAMES, mimes=MIMES,
                       assets=str(self.assets), fault=fault or {}, release=None, remote_assets=[],
                       bytes={}, events=[], configs=[], notes=self.expected_notes)
        self.state_path.write_text(json.dumps(initial))
        # A harmless marker models the subsequent public verification step.
        # Separate checksum tests run the actual unauthenticated verifier command
        # and test the Pages expression, including verification failures.
        command = COMMAND + '\nprintf public > "$PUBLIC_MARKER"'
        result = subprocess.run(["bash", "--noprofile", "--norc", "-e", "-o", "pipefail", "-c", command],
                                cwd=self.repo, env=self.env, capture_output=True, text=True, timeout=60)
        self.state = json.loads(self.state_path.read_text())
        self.assertEqual({p.name for p in self.runner.iterdir()}, {"unrelated"})
        self.assertEqual((self.runner / "unrelated").read_bytes(), b"keep")
        self.assertFalse((self.repo / "injected").exists())
        self.assertNotIn("fixture-token", result.stdout + result.stderr)
        self.assertNotIn("wrong-token", result.stdout + result.stderr)
        for config in self.state["configs"]:
            self.assertFalse(Path(config).exists())
        self.assertEqual((self.bad_config / "hosts.yml").read_text(),
                         "wrong.invalid: {oauth_token: wrong-token}\n")
        return result

    def failed(self, fault=None):
        result = self.invoke(fault)
        self.assertNotEqual(result.returncode, 0, result.stdout)
        self.assertFalse(self.public.exists())
        self.assertNotIn("Release published:", result.stdout)
        return result

    def commit_notes(self, data):
        self.notes.write_bytes(data)
        subprocess.run(["git", "add", "release-notes"], cwd=self.repo, check=True)
        subprocess.run(["git", "-c", "user.name=Fixture", "-c", "user.email=fixture@example.invalid",
                        "-c", "commit.gpgsign=false", "commit", "--quiet", "-m", "update notes"],
                       cwd=self.repo, check=True)
        self.sha = subprocess.check_output(["git", "rev-parse", "HEAD"], cwd=self.repo, text=True).strip()
        self.env["RELEASE_SHA"] = self.sha

    def test_repository_release_notes_are_published(self):
        data = (SCRIPTS.parent / "release-notes" / f"{VERSION}.md").read_bytes()
        self.commit_notes(data)
        self.expected_notes = data.decode("utf-8").strip()
        result = self.invoke()
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(self.state["release"]["body"], self.expected_notes)

    def test_notes_come_from_release_commit_not_working_tree(self):
        self.notes.write_text("Uncommitted changes must not be published.")
        result = self.invoke()
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(self.state["release"]["body"], NOTES.strip())
        self.assertNotIn("Uncommitted", self.state["release"]["body"])

    def test_missing_notes_stop_before_cli(self):
        self.env.update(RELEASE_VERSION="1.10.16", RELEASE_REF="refs/tags/1.10.16")
        result = self.failed()
        self.assertIn("Cannot read release-notes/1.10.16.md", result.stderr)
        self.assertEqual(self.state["events"], [])

    def test_invalid_committed_notes_stop_before_cli(self):
        for data in (b"", b" \n", b"# Coherence CLI 1.10.16\n\nWrong version\n",
                     f"# Coherence CLI {VERSION}\n\n".encode(), b"\xffinvalid UTF-8",
                     f"Coherence CLI Release {VERSION}".encode()):
            with self.subTest(data=data):
                self.commit_notes(data)
                self.failed()
                self.assertEqual(self.state["events"], [])

    def test_notes_mutation_after_downloads_prevents_publication(self):
        self.failed({"bad_release": {"body": self.expected_notes + "\nChanged"}, "at": 2})
        self.assertTrue(all("download:" + n in self.state["events"] for n in NAMES))
        self.assertNotIn("publish", self.state["events"])

    def test_success_preserves_exact_names_bytes_mime_metadata_and_sequence(self):
        before = {p.name: p.read_bytes() for p in self.assets.iterdir()}
        result = self.invoke()
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual({k: bytes.fromhex(v) for k, v in self.state["bytes"].items()}, before)
        self.assertEqual([a["name"] for a in self.state["remote_assets"]], NAMES)
        self.assertEqual([a["content_type"] for a in self.state["remote_assets"]], MIMES)
        self.assertEqual({p.name: p.read_bytes() for p in self.assets.iterdir()}, before)
        self.assertFalse(self.state["release"]["draft"])
        self.assertFalse(self.state["release"]["prerelease"])
        self.assertEqual(self.state["release"]["body"], self.expected_notes)
        self.assertTrue(self.public.exists())
        self.assertTrue(self.local_git.exists())
        events = self.state["events"]
        self.assertEqual(events, ["repository", "tag", "notes", "releases", "create", "releases", "assets",
                                 *WRITES[1:-1], "release", "assets", *["download:" + n for n in NAMES],
                                 "release", "assets", "publish", "release", "assets"])
        self.assertIn("release_id=42", result.stdout)
        self.assertIn("asset_id=106", result.stdout)
        self.assertIn(f"Draft creation returned: https://github.com/{REPOSITORY}/releases/tag/untagged-fixture",
                      result.stdout)

    def test_actions_token_authority_path_preserves_publication(self):
        for repo in ({"full_name": REPOSITORY, "permissions": {"push": False}},
                     {"full_name": REPOSITORY}):
            with self.subTest(repo=repo):
                result = self.invoke({"repository_response": repo})
                self.assertEqual(result.returncode, 0, result.stderr)
                self.assertLess(self.state["events"].index("notes"), self.state["events"].index("releases"))
                self.assertLess(self.state["events"].index("releases"), self.state["events"].index("create"))
                self.assertEqual([e for e in self.state["events"] if e in WRITES], WRITES)
                self.assertEqual(self.state["release"]["name"], "Coherence CLI - " + VERSION)
                self.assertEqual(self.state["release"]["body"], self.expected_notes)
                self.assertNotIn("Discarded generated", result.stdout + result.stderr)

    def test_unverified_authority_stops_publication_before_any_write(self):
        for fault in ({"fail_before": "notes"}, {"malformed": "notes"},
                      *[{"notes_response": response} for response in (
                          None, [], {}, {"name": "only name"}, {"name": "notes", "body": False})],
                      {"repository_response": {"full_name": "wrong/repo"}}):
            with self.subTest(fault=fault):
                result = self.failed(fault)
                self.assertIsNone(self.state["release"])
                self.assertFalse(any(e in WRITES for e in self.state["events"]))
                self.assertNotIn("releases", self.state["events"])
                self.assertNotIn("A write may have succeeded", result.stderr)

    def test_api_digest_is_optional_but_draft_bytes_are_always_verified(self):
        result = self.invoke({"omit_digest": True})
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(sum(e.startswith("download:") for e in self.state["events"]), 7)

    def test_each_write_failure_before_or_after_server_change_stops_without_retry_or_cleanup(self):
        for operation in WRITES:
            for timing in ("fail_before", "fail_after"):
                with self.subTest(operation=operation, timing=timing):
                    result = self.failed({timing: operation})
                    self.assertEqual(result.returncode, 23, result.stderr)
                    writes = [e for e in self.state["events"] if e in WRITES]
                    self.assertEqual(writes, WRITES[:WRITES.index(operation) + 1])
                    self.assertIn("A write may have succeeded", result.stderr)
                    self.assertIn(f"repository={REPOSITORY} tag={VERSION}", result.stderr)
                    if operation != "create":
                        self.assertIn("release_id=42", result.stderr)
                    if timing == "fail_after" and operation == "publish":
                        self.assertFalse(self.state["release"]["draft"])
                    elif self.state["release"]:
                        self.assertTrue(self.state["release"]["draft"])

    def test_existing_published_release_or_draft_is_rejected(self):
        for state in ("draft", "published"):
            with self.subTest(state=state):
                self.failed({"existing": state, "repository_response": {
                    "full_name": REPOSITORY, "permissions": {"push": False}}})
                self.assertNotIn("create", self.state["events"])

    def test_tag_disappearing_at_creation_does_not_create_release(self):
        self.failed({"tag_disappeared": True})
        self.assertIsNone(self.state["release"])
        self.assertEqual(self.state["events"][-1], "create")

    def test_delayed_created_draft_visibility_repeats_only_reads(self):
        result = self.invoke({"missing_created_reads": 1})
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(self.state["events"].count("releases"), 3)
        self.assertEqual([e for e in self.state["events"] if e in WRITES], WRITES)
        self.assertTrue(self.public.exists())
        self.assertIn("attempt=1/5 pages=2 releases=1 tag_matches=0 url_matches=0", result.stdout)
        self.assertIn("attempt=2/5 pages=2 releases=2 tag_matches=1 url_matches=1", result.stdout)

    def test_created_draft_lookup_is_bounded_and_can_succeed_on_final_read(self):
        url = f"https://github.com/{REPOSITORY}/releases/tag/untagged-fixture"
        created = {"id": 42, "tag_name": VERSION, "html_url": url, "draft": True}
        for visible in (True, False):
            with self.subTest(visible=visible):
                publisher = HELPER.Publisher(str(self.gh), REPOSITORY, VERSION, self.sha, {}, self.root,
                                             self.expected_notes)
                responses = [[[]]] * 4 + [[[created]] if visible else [[]]]
                with mock.patch.object(publisher, "api", side_effect=responses) as api, \
                        mock.patch.object(HELPER.time, "sleep") as sleep, \
                        mock.patch("sys.stdout", new_callable=io.StringIO) as logs:
                    if visible:
                        self.assertIs(publisher.identify_created_draft(url), created)
                    else:
                        with self.assertRaisesRegex(HELPER.ReleaseError, "not visible after five"):
                            publisher.identify_created_draft(url)
                self.assertEqual(api.call_count, 5)
                self.assertTrue(all(call.args == ("GET", f"repos/{REPOSITORY}/releases?per_page=100",
                                                 ("--paginate", "--slurp")) for call in api.call_args_list))
                self.assertEqual(sleep.call_args_list, [mock.call(1), mock.call(2), mock.call(4), mock.call(8)])
                self.assertIn("attempt=5/5", logs.getvalue())
                self.assertIsNone(publisher.identity)

    def test_created_draft_conflicts_malformed_data_and_api_errors_are_not_retried(self):
        url = f"https://github.com/{REPOSITORY}/releases/tag/untagged-fixture"
        created = {"id": 42, "tag_name": VERSION, "html_url": url, "draft": True}
        responses = [None, [], {}, [None], [[None]], [[{}]],
                     [[dict(created, id=True)]], [[dict(created, id=0)]],
                     [[dict(created, tag_name=None)]], [[dict(created, html_url=[])]],
                     [[dict(created, draft="true")]],
                     [[dict(created, html_url=url + "-other")]],
                     [[dict(created, tag_name="other")]],
                     [[created], [dict(created, html_url=url + "-other")]],
                     [[created], [dict(created, tag_name="other")]],
                     HELPER.PublicationError("fixture API failure", 23),
                     subprocess.TimeoutExpired("fixture", 600)]
        for response in responses:
            with self.subTest(response=response):
                publisher = HELPER.Publisher(str(self.gh), REPOSITORY, VERSION, self.sha, {}, self.root,
                                             self.expected_notes)
                with mock.patch.object(publisher, "api", side_effect=[response]) as api, \
                        mock.patch.object(HELPER.time, "sleep") as sleep, \
                        mock.patch("sys.stdout", new_callable=io.StringIO):
                    with self.assertRaises((HELPER.ReleaseError, subprocess.TimeoutExpired)):
                        publisher.identify_created_draft(url)
                api.assert_called_once()
                sleep.assert_not_called()
                self.assertIsNone(publisher.identity)

    def test_created_draft_must_be_unambiguous_owned_and_empty(self):
        for fault in ("wrong_created_url", "duplicate_release", "seed_asset"):
            with self.subTest(fault=fault):
                result = self.failed({fault: True})
                self.assertIn(f"Draft creation returned: https://github.com/{REPOSITORY}/releases/tag/untagged-fixture",
                              result.stdout)
                self.assertEqual([e for e in self.state["events"] if e in WRITES], ["create"])

    def test_upload_response_metadata_must_match(self):
        for key, value in (("id", True), ("name", "other"), ("label", "label"), ("state", "starter"),
                           ("size", True), ("size", 0), ("content_type", "application/octet-stream"),
                           ("url", "https://other.invalid/asset"), ("digest", "sha256:" + "0" * 64)):
            with self.subTest(key=key):
                self.failed({"bad_upload": {key: value}})
                self.assertEqual([e for e in self.state["events"] if e in WRITES], WRITES[:2])

    def test_complete_remote_asset_allowlist_and_identity_is_checked(self):
        for mode in ("extra", "missing", "duplicate", "changed-id", "malformed"):
            with self.subTest(mode=mode):
                self.failed({"asset_list": mode})
                self.assertNotIn("publish", self.state["events"])

    def test_release_metadata_changes_prevent_publication(self):
        for key, value in (("id", 43), ("tag_name", "other"), ("target_commitish", "b" * 40),
                           ("name", "other"), ("body", "other"), ("draft", False), ("prerelease", True),
                           ("url", "https://other.invalid/release"), ("upload_url", "https://other.invalid/assets")):
            with self.subTest(key=key):
                self.failed({"bad_release": {key: value}})
                self.assertNotIn("publish", self.state["events"])

    def test_each_corrupt_draft_download_prevents_publication(self):
        for name in NAMES:
            with self.subTest(name=name):
                result = self.failed({"corrupt_download": name})
                self.assertIn("Draft download mismatch", result.stderr)
                self.assertNotIn("publish", self.state["events"])
                self.assertTrue(self.state["release"]["draft"])

    def test_lookup_download_and_malformed_response_failures_prevent_later_steps(self):
        for operation in ("repository", "tag", "notes", "releases", "assets", "release", *["download:" + n for n in NAMES]):
            with self.subTest(operation=operation):
                self.failed({"fail_before": operation})
                self.assertNotIn("publish", self.state["events"])
        for operation in ("create", "releases", "assets", "release", "upload:" + NAMES[0], "publish"):
            with self.subTest(malformed=operation):
                self.failed({"malformed": operation})
                if operation != "publish":
                    self.assertNotIn("publish", self.state["events"])

    def test_final_draft_and_published_checks_keep_the_downstream_gate_closed_on_failure(self):
        cases = (({"fail_before": "release", "at": 2}, False), ({"fail_before": "assets", "at": 3}, False),
                 ({"bad_release": {"body": "changed during download"}, "at": 2}, False),
                 ({"asset_list": "extra", "at": 3}, False),
                 ({"fail_before": "release", "at": 3}, True), ({"fail_before": "assets", "at": 4}, True),
                 ({"bad_release": {"prerelease": True}, "at": 3}, True), ({"asset_list": "missing", "at": 4}, True))
        for fault, published in cases:
            with self.subTest(fault=fault):
                self.failed(fault)
                self.assertTrue(all("download:" + name in self.state["events"] for name in NAMES))
                self.assertEqual("publish" in self.state["events"], published)
                self.assertEqual(self.state["release"]["draft"], not published)

    def test_missing_or_nonregular_staged_asset_fails_before_cli(self):
        for name in NAMES:
            original = (self.assets / name).read_bytes()
            for kind in ("missing", "directory", "symlink"):
                with self.subTest(name=name, kind=kind):
                    path = self.assets / name
                    path.unlink()
                    if kind == "directory":
                        path.mkdir()
                    elif kind == "symlink":
                        path.symlink_to(self.runner / "unrelated")
                    self.failed()
                    self.assertEqual(self.state["events"], [])
                    if kind == "directory":
                        path.rmdir()
                    elif kind == "symlink":
                        path.unlink()
                    path.write_bytes(original)

    def test_unexpected_asset_and_malformed_manifest_fail_before_cli(self):
        extra = self.assets / "unexpected"
        extra.write_bytes(b"extra")
        self.failed()
        self.assertEqual(self.state["events"], [])
        extra.unlink()
        path = self.assets / NAMES[-1]
        original = path.read_bytes()
        for data in (b"", original.replace(NAMES[0].encode(), b"../other"),
                     b"0" * 64 + original[64:]):
            with self.subTest(data=data[:80]):
                path.write_bytes(data)
                self.failed()
                self.assertEqual(self.state["events"], [])
        path.write_bytes(original)
        for name in NAMES[:-1]:
            data = (self.assets / name).read_bytes()
            (self.assets / name).write_bytes(b"corrupt")
            self.failed()
            self.assertEqual(self.state["events"], [])
            (self.assets / name).write_bytes(data)

    def test_invalid_identity_paths_and_tokens_fail_before_cli(self):
        cases = {"RELEASE_REPOSITORY": ("", "../repo", "owner/..", "https://github.com/owner/repo", "a/b/c", "a/b\n"),
                 "RELEASE_VERSION": ("", "v;touch injected", "../version", VERSION + "\n"),
                 "RELEASE_REF": ("refs/heads/" + VERSION, "refs/tags/other"), "RELEASE_REF_TYPE": ("branch", ""),
                 "RELEASE_SHA": ("b" * 40, "invalid", ""), "RELEASE_GH": ("gh", "/missing", ""),
                 "RELEASE_ASSETS": ("relative", str(self.root / "missing"), ""),
                 "RUNNER_TEMP": ("relative", str(self.root / "missing"), ""), "GH_TOKEN": ("", "invalid\n")}
        for key, values in cases.items():
            original = self.env[key]
            for value in values:
                with self.subTest(key=key, value=value):
                    self.env[key] = value
                    self.failed()
                    self.assertEqual(self.state["events"], [])
            self.env[key] = original

    def test_missing_context_fails_before_cli(self):
        for key in ("RELEASE_GH", "RELEASE_REPOSITORY", "RELEASE_VERSION", "RELEASE_REF",
                    "RELEASE_REF_TYPE", "RELEASE_SHA", "RELEASE_ASSETS", "RUNNER_TEMP", "GH_TOKEN"):
            with self.subTest(missing=key):
                original = self.env.pop(key)
                self.failed()
                self.assertEqual(self.state["events"], [])
                self.env[key] = original

    def test_git_ref_format_failures_with_filename_safe_versions_stop_before_cli(self):
        for version in ("1..2", VERSION + ".", VERSION + ".lock"):
            with self.subTest(version=version):
                self.env.update(RELEASE_VERSION=version, RELEASE_REF="refs/tags/" + version)
                result = self.failed()
                self.assertIn("Local Git command failed", result.stderr)
                self.assertEqual(self.state["events"], [])

    def test_slash_and_metacharacter_versions_are_rejected_as_literal_data(self):
        for version in ("release/" + VERSION, "$(touch injected)", "`touch injected`"):
            with self.subTest(version=version):
                self.env.update(RELEASE_VERSION=version, RELEASE_REF="refs/tags/" + version)
                result = self.failed()
                self.assertIn("Invalid release version", result.stderr)
                self.assertEqual(self.state["events"], [])

    def test_unresolvable_checkout_stops_before_cli(self):
        subprocess.run(["git", "symbolic-ref", "HEAD", "refs/heads/nonexistent"], cwd=self.repo, check=True)
        result = self.failed()
        self.assertIn("Local Git command failed", result.stderr)
        self.assertEqual(self.state["events"], [])

    def test_staged_bytes_changed_after_validation_stop_before_upload(self):
        path = self.assets / NAMES[0]
        original = path.read_bytes()
        result = self.failed({"change_staged": NAMES[0]})
        self.assertIn("Staged bytes changed", result.stderr)
        self.assertEqual([e for e in self.state["events"] if e in WRITES], ["create"])
        path.write_bytes(original)

    def test_cli_timeout_and_signal_are_failures(self):
        publisher = HELPER.Publisher(str(self.gh), REPOSITORY, VERSION, self.sha, {}, self.root,
                                     self.expected_notes)
        with mock.patch.object(HELPER.subprocess, "run", side_effect=subprocess.TimeoutExpired("fixture", 600)) as run:
            for arguments, timeout in (({}, 600), ({"timeout": 300}, 300)):
                with self.assertRaises(subprocess.TimeoutExpired):
                    publisher.command(["fixture"], **arguments)
                self.assertEqual(run.call_args.kwargs["timeout"], timeout)
        with mock.patch.object(HELPER.subprocess, "run", return_value=subprocess.CompletedProcess([], -9)):
            with self.assertRaisesRegex(HELPER.ReleaseError, "signal"):
                publisher.command(["fixture"])


class Workflow(unittest.TestCase):
    def test_exact_context_cli_assets_and_success_only_order(self):
        self.assertEqual(COMMAND, "python3 ./scripts/publish-release.py")
        self.assertNotIn("${{", COMMAND)
        expected = {"RELEASE_GH": "steps.release_gh.outputs.path", "RELEASE_ASSETS": "steps.release_assets.outputs.path",
                    "RELEASE_VERSION": "steps.version.outputs.version", "RELEASE_REPOSITORY": "github.repository",
                    "RELEASE_REF": "github.ref", "RELEASE_REF_TYPE": "github.ref_type", "RELEASE_SHA": "github.sha",
                    "GH_TOKEN": "github.token"}
        self.assertEqual(dict(re.findall(r"^        (\w+): \$\{\{ (.+?) \}\}$", STEP, re.MULTILINE)), expected)
        names = re.findall(r"^    - name: (.+)$", WORKFLOW, re.MULTILINE)
        start = names.index("Generate SHA-256 checksums")
        self.assertEqual(names[start:start + 5], ["Generate SHA-256 checksums", "Stage and verify release assets",
                         NAME, "Verify public release downloads", "Release GitHub Pages Stable.txt"])
        for name in names[start:start + 4]:
            step = WORKFLOW.split(f"    - name: {name}\n", 1)[1].split("    - name: ", 1)[0]
            self.assertNotRegex(step, r"(?m)^\s+(if|continue-on-error):")
        staging = WORKFLOW.split("    - name: Stage and verify release assets\n", 1)[1].split("    - name: ", 1)[0]
        self.assertIn("      id: release_assets\n", staging)
        self.assertIn("        RELEASE_VERSION: ${{ steps.version.outputs.version }}\n", staging)
        self.assertEqual(staging.split("      run: ", 1)[1].strip(), "python3 ./scripts/stage-release-assets.py")
        self.assertNotIn("TOKEN", staging)
        self.assertNotIn("Validate release identity and absence", names)
        self.assertNotIn("actions/create-release@", WORKFLOW)
        self.assertNotIn("actions/upload-release-asset@", WORKFLOW)
        self.assertEqual(len(re.findall(r"^\s+GH_TOKEN:", WORKFLOW, re.MULTILINE)), 1)
        self.assertNotIn("        GITHUB_TOKEN:", WORKFLOW)
        self.assertIn("permissions:\n  contents: write\n", WORKFLOW)
        self.assertIn("make test-release-gh test-release-preflight test-release-staging test-release-publication\n", WORKFLOW)

    def test_same_repository_tag_is_serialized_without_cancelling_running_release(self):
        self.assertIn("concurrency:\n  group: release-${{ github.repository }}-${{ github.ref }}\n"
                      "  cancel-in-progress: false\n", WORKFLOW)


if __name__ == "__main__":
    unittest.main(verbosity=2)
