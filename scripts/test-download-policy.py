#!/usr/bin/env python3
# Copyright (c) 2026 Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at
# https://oss.oracle.com/licenses/upl.
"""Exercise the production curl policies against trusted local TLS redirects."""

import http.server
from pathlib import Path
import re
import shlex
import ssl
import subprocess
import tempfile
import threading
import unittest

ROOT = Path(__file__).resolve().parent.parent


def download_commands():
    for name in ("scripts/install.sh", "scripts/verify-release.sh", ".github/workflows/build-test-create-cluster.yaml",
                 "README.md", "docs/installation/installation.adoc"):
        text = (ROOT / name).read_text().replace("\\\n", " ")
        for line in text.splitlines():
            if name.endswith(".sh") and line.lstrip().startswith("#"):
                continue
            # Shell substitutions close after the URL. shlex retains the
            # actual option values and quoting used by these documented calls.
            match = re.search(r"\bcurl(?:\.exe)? (.*?)(?:\"?https://|\"\$\{(?:release_url|RELEASE_URL|URL|url)\})", line)
            if match:
                prefix = shlex.split(match[1])
                if "-o" in prefix:
                    prefix = prefix[:prefix.index("-o")]
                if "-fLo" in prefix:
                    prefix = prefix[:prefix.index("-fLo")] + ["-fL"]
                yield name, prefix


class DownloadPolicyTests(unittest.TestCase):
    def test_trusted_redirects_and_rejections(self):
        with tempfile.TemporaryDirectory(prefix="download-policy-") as temporary:
            directory = Path(temporary)
            cert, key = directory / "cert.pem", directory / "key.pem"
            subprocess.run(["openssl", "req", "-x509", "-newkey", "rsa:2048", "-nodes", "-keyout", str(key),
                            "-out", str(cert), "-days", "1", "-subj", "/CN=127.0.0.1",
                            "-addext", "subjectAltName=IP:127.0.0.1"], check=True, capture_output=True)
            hits = []

            class Handler(http.server.BaseHTTPRequestHandler):
                def do_GET(self):
                    hits.append((isinstance(self.connection, ssl.SSLSocket), self.path))
                    targets = {"/secure": f"https://127.0.0.1:{secure.server_port}/payload",
                               "/downgrade": f"http://127.0.0.1:{plain.server_port}/payload",
                               "/other": "file:///must-not-be-read"}
                    if self.path == "/failure":
                        self.send_response(404)
                        self.end_headers()
                    elif self.path in targets:
                        self.send_response(302)
                        self.send_header("Location", targets[self.path])
                        self.end_headers()
                    else:
                        self.send_response(200)
                        self.end_headers()
                        self.wfile.write(b"trusted fixture payload")

                def log_message(self, *args):
                    pass

            plain = http.server.ThreadingHTTPServer(("127.0.0.1", 0), Handler)
            secure = http.server.ThreadingHTTPServer(("127.0.0.1", 0), Handler)
            context = ssl.SSLContext(ssl.PROTOCOL_TLS_SERVER)
            context.load_cert_chain(str(cert), str(key))
            secure.socket = context.wrap_socket(secure.socket, server_side=True)
            servers = (plain, secure)
            threads = [threading.Thread(target=server.serve_forever, daemon=True) for server in servers]
            for thread in threads:
                thread.start()
            try:
                commands = list(download_commands())
                # Four installer transfers, the verifier, Gradle and all nine
                # README/guide examples (including the frozen Windows example).
                self.assertEqual(len(commands), 15)
                for name, policy in commands:
                    with self.subTest(source=name, policy=policy):
                        self.assertEqual(policy[policy.index("--proto") + 1], "=https")
                        self.assertEqual(policy[policy.index("--proto-redir") + 1], "=https")
                        # Availability retries are tested by the verifier's argv
                        # contract. Avoid retry delays for deliberate rejection.
                        if "--retry" in policy:
                            policy = policy[:policy.index("--retry")]
                        for route, trusted, success in (("/secure", True, True), ("/downgrade", True, False),
                                                        ("/other", True, False), ("/failure", True, False),
                                                        ("/payload", False, False)):
                            with self.subTest(route=route, trusted=trusted):
                                hits.clear()
                                command = ["curl", "-q", "--noproxy", "*", "--max-time", "5", *policy]
                                if trusted:
                                    command += ["--cacert", str(cert)]
                                command += [f"https://127.0.0.1:{secure.server_port}{route}"]
                                result = subprocess.run(command, capture_output=True, timeout=10)
                                self.assertEqual(result.returncode == 0, success, result.stderr)
                                self.assertTrue(all(tls for tls, path in hits), hits)
                                if success:
                                    self.assertEqual(result.stdout, b"trusted fixture payload")
                                    self.assertEqual(len(hits), 2)
                                else:
                                    self.assertEqual(result.stdout, b"")
                                if not trusted:
                                    self.assertEqual(hits, [])
                        hits.clear()
                        result = subprocess.run(["curl", "-q", "--noproxy", "*", *policy,
                                                 f"http://127.0.0.1:{plain.server_port}/payload"], capture_output=True, timeout=10)
                        self.assertNotEqual(result.returncode, 0)
                        self.assertEqual(hits, [])
            finally:
                for server in servers:
                    server.shutdown()
                    server.server_close()
                for thread in threads:
                    thread.join()


if __name__ == "__main__":
    unittest.main(verbosity=2)
