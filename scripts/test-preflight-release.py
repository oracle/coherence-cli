#!/usr/bin/env python3
# Copyright (c) 2026 Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at
# https://oss.oracle.com/licenses/upl.
"""Direct tests of publication's internal release gate with fixture API/CLI.

Set RELEASE_PREFLIGHT_TEST_GH to an absolute gh 2.103.3 executable to repeat
the API cases with that CLI and a loopback HTTP fixture (no GitHub writes).
"""
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
import json
import os
from pathlib import Path
import subprocess
import sys
import tempfile
import threading
import types
import unittest
from unittest import mock

SCRIPTS = Path(__file__).resolve().parent
HELPER = types.ModuleType("publication")
exec(compile((SCRIPTS / "publish-release.py").read_text(), str(SCRIPTS / "publish-release.py"), "exec"),
     HELPER.__dict__)
REPOSITORY = "fixture-owner/fixture-repo"
PREFIX = f"repos/{REPOSITORY}"
VERSION = "1.10.15"
TAG_SHA = "a" * 40
OTHER_SHA = "b" * 40
REAL_GH = os.environ.get("RELEASE_PREFLIGHT_TEST_GH")
NOTES = f"{PREFIX}/releases/generate-notes"
# Reduced contract fixtures, not captured responses from a hosted run.
# GitHub's full-repository schema makes permissions optional and its role
# fields boolean. The notes response uses
# the documented name/body contract and requires Contents: write:
# https://github.com/github/rest-api-description/blob/main/descriptions/api.github.com/api.github.com.json
# https://docs.github.com/en/rest/releases/releases#generate-release-notes-content-for-a-release
ACTIONS_REPOSITORY = {"full_name": REPOSITORY, "permissions": {
    "admin": False, "maintain": False, "push": False, "triage": False, "pull": False}}
GENERATED_NOTES = {"name": "Release v1.0.0 is now available!",
                   "body": "##Changes in Release v1.0.0 ... ##Contributors @monalisa"}


def release(tag, draft=False, identity=1):
    return {"id": identity, "tag_name": tag, "draft": draft}


class Preflight(unittest.TestCase):
    def setUp(self):
        self.directory = tempfile.TemporaryDirectory(prefix="release-preflight-test-")
        self.addCleanup(self.directory.cleanup)
        self.root = Path(self.directory.name)
        self.sha = "c" * 40
        self.config = self.root / "config"
        self.config.mkdir()
        self.responses = {
            PREFIX: {"full_name": REPOSITORY, "permissions": {"push": True}},
            f"{PREFIX}/git/ref/tags/{VERSION}": {
                "ref": f"refs/tags/{VERSION}", "object": {"type": "commit", "sha": self.sha}},
            NOTES: GENERATED_NOTES,
            f"{PREFIX}/releases?per_page=100": [[]],
        }
        self.events = self.root / "events.jsonl"
        self.fixture = self.root / "responses.json"
        self.gh = self.root / "pinned cli" / "gh"
        self.gh.parent.mkdir()
        self.gh.write_text(f'''#!{sys.executable}
import json, os, pathlib, sys
args = sys.argv[1:]
endpoint = args[-1]
notes = endpoint == {NOTES!r}
assert args[:7] == ['api', '--hostname', 'github.com', '--method', 'POST' if notes else 'GET', '--header', 'Accept: application/vnd.github+json'], args
assert args[7:9] == ['--header', 'X-GitHub-Api-Version: 2022-11-28'], args
assert endpoint.startswith({PREFIX!r}), endpoint
expected = ['--raw-field', 'tag_name=' + os.environ['RELEASE_VERSION'], '--raw-field', 'target_commitish=' + os.environ['RELEASE_SHA']] if notes else (['--paginate', '--slurp'] if '/releases?' in endpoint else [])
assert args[9:-1] == expected, args
assert os.environ['GH_TOKEN'] == 'fixture-token'
assert not any(k in os.environ for k in ('GITHUB_TOKEN', 'GH_ENTERPRISE_TOKEN', 'GITHUB_ENTERPRISE_TOKEN', 'GH_DEBUG'))
assert os.environ['GH_PROMPT_DISABLED'] == '1'
assert list(pathlib.Path(os.environ['GH_CONFIG_DIR']).iterdir()) == []
with open(os.environ['EVENTS'], 'a') as events: events.write(json.dumps(args) + '\\n')
if os.environ.get('REAL_GH'):
    # Only redirect the endpoint to loopback; real gh handles HTTP, errors,
    # Link pagination and --slurp. Only a fake fixture token is in this env.
    args[-1] = os.environ['FIXTURE_URL'] + '/' + endpoint
    os.execv(os.environ['REAL_GH'], [os.environ['REAL_GH']] + args)
responses = json.loads(pathlib.Path(os.environ['FIXTURE']).read_text())
response = responses[endpoint]
if isinstance(response, dict) and response.get('_disconnect'):
    sys.exit(7)
if isinstance(response, dict) and '_exit' in response:
    print(response.get('_stdout', '[]'))
    sys.exit(response['_exit'])
if isinstance(response, dict) and '_raw' in response:
    print(response['_raw'])
else:
    print(json.dumps(response))
''')
        self.gh.chmod(0o700)
        self.env = {k: v for k, v in os.environ.items() if not k.startswith(("GH_", "GITHUB_"))}
        self.env.update(RELEASE_VERSION=VERSION, RELEASE_SHA=self.sha, GH_TOKEN="fixture-token",
                        GH_CONFIG_DIR=str(self.config), GH_PROMPT_DISABLED="1",
                        FIXTURE=str(self.fixture), EVENTS=str(self.events))
        self.publisher = HELPER.Publisher(str(self.gh), REPOSITORY, VERSION, self.sha,
                                          self.env, self.root, "unused notes")
        self.http_events = []
        self.http_posts = []
        if REAL_GH:
            self.start_api_fixture()

    def start_api_fixture(self):
        case = self

        class Handler(BaseHTTPRequestHandler):
            def do_GET(self):
                self.respond()

            def do_POST(self):
                body = json.loads(self.rfile.read(int(self.headers["Content-Length"])))
                case.http_posts.append((self.path, body))
                if self.path != "/" + NOTES or body != {
                        "tag_name": case.env["RELEASE_VERSION"], "target_commitish": case.env["RELEASE_SHA"]}:
                    self.send_error(400)
                    return
                self.respond()

            def respond(self):
                endpoint = self.path.lstrip("/")
                case.http_events.append(endpoint)
                response = case.responses.get(endpoint, {"_exit": 1})
                paginated = "/releases?" in endpoint and isinstance(response, list) and bool(response)
                payload = response[0] if paginated else response
                if isinstance(payload, dict) and payload.get("_disconnect"):
                    self.close_connection = True
                    return
                status = 200
                if isinstance(payload, dict) and "_exit" in payload:
                    status = payload.get("_status", 503)
                    body = payload.get("_stdout", '{"message":"fixture failure"}')
                elif isinstance(payload, dict) and "_raw" in payload:
                    body = payload["_raw"]
                else:
                    body = json.dumps(payload)
                self.send_response(status)
                self.send_header("Content-Type", "application/json")
                # This header describes endpoint requirements, not grants. It
                # must not turn a denied or malformed response into authority.
                self.send_header("X-Accepted-GitHub-Permissions", "contents=write")
                if status == 200 and paginated and len(response) > 1:
                    next_endpoint = endpoint + "&page=2"
                    case.responses[next_endpoint] = response[1:]
                    self.send_header("Link", f'<{case.env["FIXTURE_URL"]}/{next_endpoint}>; rel="next"')
                self.end_headers()
                self.wfile.write(body.encode())

            def log_message(self, *_args):
                pass

        server = ThreadingHTTPServer(("127.0.0.1", 0), Handler)
        self.env.update(REAL_GH=REAL_GH, FIXTURE_URL=f"http://127.0.0.1:{server.server_port}")
        thread = threading.Thread(target=server.serve_forever, daemon=True)
        thread.start()

        def stop():
            server.shutdown()
            server.server_close()
            thread.join()

        self.addCleanup(stop)

    def invoke(self):
        self.fixture.write_text(json.dumps(self.responses))
        self.publisher.check_release(f"refs/tags/{VERSION}")

    def failed(self, message=None):
        with self.assertRaises(HELPER.ReleaseError) as caught:
            self.invoke()
        if message:
            self.assertIn(message, str(caught.exception))

    def test_lightweight_tag_success_with_explicit_destination_and_cli(self):
        with mock.patch.object(HELPER.subprocess, "run", wraps=subprocess.run) as run:
            self.invoke()
        self.assertEqual([call.kwargs["timeout"] for call in run.call_args_list], [300] * 4)
        events = [json.loads(line) for line in self.events.read_text().splitlines()]
        self.assertEqual([args[-1] for args in events], list(self.responses)[:4])

    def test_actions_token_repository_response_with_server_authority_succeeds(self):
        for repo in (ACTIONS_REPOSITORY, {"full_name": REPOSITORY}):
            with self.subTest(repo=repo):
                self.responses[PREFIX] = repo
                self.invoke()
        if REAL_GH:
            self.assertEqual(self.http_posts, [("/" + NOTES, {
                "tag_name": VERSION, "target_commitish": self.sha})] * 2)

    def test_repository_push_role_cannot_replace_token_authority(self):
        for repo in (self.responses[PREFIX], ACTIONS_REPOSITORY, {"full_name": REPOSITORY}):
            with self.subTest(repo=repo):
                self.responses[PREFIX] = repo
                self.responses[NOTES] = {"_exit": 1, "_status": 403,
                                         "_stdout": json.dumps(GENERATED_NOTES)}
                self.failed("CLI failed")
        events = [json.loads(line)[-1] for line in self.events.read_text().splitlines()]
        self.assertNotIn(f"{PREFIX}/releases?per_page=100", events)

    def test_malformed_authority_response_stops_before_release_listing(self):
        self.responses[PREFIX] = ACTIONS_REPOSITORY
        for response in (None, [], {}, {"message": "not authorized"}, {"name": "notes"}, {"body": "notes"},
                         {"name": True, "body": "notes"}, {"name": "notes", "body": []}):
            with self.subTest(response=response):
                self.responses[NOTES] = response
                self.failed("Cannot verify Contents: write authority")
        events = [json.loads(line)[-1] for line in self.events.read_text().splitlines()]
        self.assertNotIn(f"{PREFIX}/releases?per_page=100", events)

    def test_annotated_and_nested_tag_success(self):
        target_sha, kind = self.sha, "commit"
        for annotated in (TAG_SHA, OTHER_SHA):
            self.responses[f"{PREFIX}/git/ref/tags/{VERSION}"]["object"] = {"type": "tag", "sha": annotated}
            self.responses[f"{PREFIX}/git/tags/{annotated}"] = {
                "sha": annotated, "object": {"type": kind, "sha": target_sha}}
            self.invoke()
            target_sha, kind = annotated, "tag"

    def test_remote_commit_mismatch(self):
        self.responses[f"{PREFIX}/git/ref/tags/{VERSION}"]["object"]["sha"] = OTHER_SHA
        self.failed("Remote tag commit")

    def test_annotated_tag_depth_is_bounded(self):
        for depth in (32, 33):
            with self.subTest(depth=depth):
                target_sha, kind = self.sha, "commit"
                for index in range(depth):
                    annotated = f"{index:040x}"
                    self.responses[f"{PREFIX}/git/tags/{annotated}"] = {
                        "sha": annotated, "object": {"type": kind, "sha": target_sha}}
                    target_sha, kind = annotated, "tag"
                self.responses[f"{PREFIX}/git/ref/tags/{VERSION}"]["object"] = {
                    "type": kind, "sha": target_sha}
                if depth == 32:
                    self.invoke()
                else:
                    self.failed("cyclic or too deep")

    def test_missing_tag_and_lookup_errors_never_allow_publication(self):
        for endpoint in list(self.responses):
            original = self.responses[endpoint]
            for status in (401, 403, 404, 429, 500):
                with self.subTest(endpoint=endpoint, status=status):
                    self.responses[endpoint] = {"_exit": 1, "_status": status}
                    self.failed("CLI failed")
            self.responses[endpoint] = original

    def test_malformed_or_wrong_repository_stops_before_permission_probe(self):
        for response in ([], None, {}, {"full_name": "other/repo", "permissions": {"push": True}},
                         {"full_name": True}):
            with self.subTest(response=response):
                self.responses[PREFIX] = response
                self.failed()
        events = [json.loads(line)[-1] for line in self.events.read_text().splitlines()]
        self.assertNotIn(NOTES, events)

    def test_malformed_ref_and_tag_objects_fail(self):
        endpoint = f"{PREFIX}/git/ref/tags/{VERSION}"
        for response in ([], {}, {"ref": "refs/tags/wrong"},
                         {"ref": f"refs/tags/{VERSION}", "object": {"type": "blob", "sha": self.sha}},
                         {"ref": f"refs/tags/{VERSION}", "object": {"type": "commit", "sha": "bad"}}):
            with self.subTest(response=response):
                self.responses[endpoint] = response
                self.failed()
        self.responses[endpoint] = {"ref": f"refs/tags/{VERSION}", "object": {"type": "tag", "sha": TAG_SHA}}
        for response in ({}, {"sha": OTHER_SHA, "object": {"type": "commit", "sha": self.sha}},
                         {"sha": TAG_SHA, "object": {"type": "tree", "sha": self.sha}},
                         {"sha": TAG_SHA, "object": {"type": "tag", "sha": TAG_SHA}},
                         {"_exit": 1, "_status": 404}):
            with self.subTest(tag=response):
                self.responses[f"{PREFIX}/git/tags/{TAG_SHA}"] = response
                self.failed()

    def test_existing_drafts_and_published_releases_on_later_pages(self):
        self.responses[PREFIX] = ACTIONS_REPOSITORY
        for draft in (False, True):
            for page in (0, 2):
                with self.subTest(draft=draft, page=page):
                    pages = [[release(f"unrelated-{i}", identity=i + 1)] for i in range(page)]
                    pages.append([release(VERSION, draft)])
                    self.responses[f"{PREFIX}/releases?per_page=100"] = pages
                    self.failed("A release already exists")
        if REAL_GH:
            self.assertTrue(any("page=2" in event for event in self.http_events))

    def test_absence_requires_all_pages_and_exact_tag_equality(self):
        self.responses[f"{PREFIX}/releases?per_page=100"] = [
            [release("v" + VERSION)], [release(VERSION + "-rc1", True)], []]
        self.invoke()

    def test_failed_later_page_never_means_absence(self):
        if REAL_GH:
            self.responses[f"{PREFIX}/releases?per_page=100"] = [[release("unrelated")], []]
            # Fixture handler overwrites its next page; this object becomes a
            # failing later-page response rather than a release array.
            self.responses[f"{PREFIX}/releases?per_page=100"][-1] = {"_exit": 1}
        else:
            self.responses[f"{PREFIX}/releases?per_page=100"] = {"_exit": 1, "_stdout": "[[ ]]"}
        self.failed()

    def test_transport_failure_never_means_absence(self):
        self.responses[f"{PREFIX}/releases?per_page=100"] = {"_disconnect": True}
        self.failed("CLI failed")

    def test_malformed_json_and_release_pages_never_mean_absence(self):
        for endpoint in list(self.responses):
            original = self.responses[endpoint]
            for raw in ("", "{truncated", "null"):
                with self.subTest(endpoint=endpoint, raw=raw):
                    self.responses[endpoint] = {"_raw": raw}
                    self.failed()
            self.responses[endpoint] = original
        # A real gh --slurp always wraps an empty API page as [[]]. An empty
        # outer array is additionally checked with the command fixture.
        invalid = ([[]] if not REAL_GH else []) + [{}, [None], [[{}]], [[release(VERSION, draft="false")]],
                   [[release("unrelated", identity=True)]], [[release("")]]]
        for response in invalid:
            with self.subTest(pages=response):
                self.responses[f"{PREFIX}/releases?per_page=100"] = response
                self.failed()


if __name__ == "__main__":
    if REAL_GH:
        binary = Path(REAL_GH)
        if not binary.is_absolute():
            raise SystemExit("RELEASE_PREFLIGHT_TEST_GH must be an absolute path")
        version = subprocess.run([REAL_GH, "--version"], capture_output=True, text=True, check=True).stdout
        if version.splitlines()[0].split()[2] != "2.103.3":
            raise SystemExit("The API behavior check requires the pinned gh 2.103.3")
    unittest.main(verbosity=2)
