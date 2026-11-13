#!/usr/bin/env python3
# Copyright (c) 2026 Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at
# https://oss.oracle.com/licenses/upl.
"""Execute the actual manual Linux snippets with local download fixtures."""
import hashlib
import os
from pathlib import Path
import re
import shutil
import subprocess
import tempfile
import unittest

ROOT = Path(__file__).resolve().parent.parent
VERSION = re.search(r'^VERSION \?= (.+)$', (ROOT / 'Makefile').read_text(), re.M)[1]


class LinuxDocs(unittest.TestCase):
    def test_documented_procedures(self):
        readme = (ROOT / 'README.md').read_text()
        guide = (ROOT / 'docs/installation/installation.adoc').read_text()
        snippets = [next(s for s in re.findall(r'```bash\n(.*?)\n```', readme, re.S) if 'ASSET=' in s),
                    next(s for s in re.findall(r'\[source,bash[^\n]*\]\n----\n(.*?)\n----', guide, re.S)
                         if 'ASSET=' in s)]
        for doc, snippet in enumerate(snippets):
            for arch in ('amd64', 'arm64'):
                for mode in ('valid', 'missing', 'duplicate', 'malformed', 'extra-field', 'mismatch',
                             'manifest-download', 'binary-download', 'invalid-version'):
                    with self.subTest(doc=doc, arch=arch, mode=mode), tempfile.TemporaryDirectory() as directory:
                        root = Path(directory)
                        mock, temp = root / 'mock', root / 'temporary'
                        mock.mkdir()
                        temp.mkdir()
                        log = root / 'operations'
                        log.touch()
                        artifact = root / 'fixture'
                        artifact.write_bytes(b'manual download fixture\n')
                        name = f'cohctl-{VERSION}-linux-{arch}'
                        digest = hashlib.sha256(artifact.read_bytes()).hexdigest()
                        entry = f'{digest}  {name}\n'
                        manifest = root / 'manifest'
                        manifest.write_text({'missing': f'{digest}  unrelated\n', 'duplicate': entry * 2,
                                             'malformed': f'{digest.upper()}  {name}\n',
                                             'extra-field': entry.rstrip() + ' extra\n',
                                             'mismatch': f'{"0" * 64}  {name}\n'}.get(mode, entry))
                        for command in ('sha256sum', 'chmod', 'mv'):
                            real = shutil.which(command)
                            (mock / command).write_text(f'#!/bin/bash\nprintf "{command}\\n" >> "$LOG"\nexec "{real}" "$@"\n')
                            (mock / command).chmod(0o700)
                        (mock / 'curl').write_text('''#!/usr/bin/env python3
import os, shutil, sys
args = sys.argv[1:]; url = args[-1]; mode = os.environ['MODE']
expected = ['-fsSL', '--proto', '=https', '--proto-redir', '=https']
if not url.endswith('stable.txt'): expected += ['-o', args[-2]]
if args != expected + [url]: raise RuntimeError('Unexpected curl arguments: ' + repr(args))
if url.endswith('stable.txt'):
 print('bad/version' if mode == 'invalid-version' else os.environ['MOCK_VERSION']); sys.exit(0)
manifest = url.endswith('checksums.txt')
if mode == ('manifest-download' if manifest else 'binary-download'): sys.exit(22)
shutil.copyfile(os.environ['MOCK_MANIFEST' if manifest else 'MOCK_ARTIFACT'], args[args.index('-o') + 1])
''')
                        (mock / 'curl').chmod(0o700)
                        code = snippet.replace('{cli-version}', 'bad/version' if mode == 'invalid-version' else VERSION)
                        code = code.replace('ARCH=amd64 #', f'ARCH={arch} #')
                        # A later ordinary shell command must not cause a failed subshell to install.
                        code += '\nstatus=$?\nprintf "caller continued\\n"\nexit "$status"\n'
                        env = dict(os.environ, PATH=f'{mock}:{os.environ["PATH"]}', LOG=str(log), MODE=mode,
                                   MOCK_ARTIFACT=str(artifact), MOCK_MANIFEST=str(manifest), MOCK_VERSION=VERSION, TMPDIR=str(temp))
                        result = subprocess.run(['bash', '-c', code], cwd=root, env=env,
                                                capture_output=True, text=True)
                        self.assertEqual(list(temp.iterdir()), [], result.stderr)
                        operations = log.read_text().splitlines()
                        if mode == 'valid':
                            self.assertEqual(result.returncode, 0, result.stderr)
                            self.assertEqual(operations, ['sha256sum', 'chmod', 'mv'])
                            self.assertEqual((root / name).read_bytes(), artifact.read_bytes())
                            self.assertTrue(os.access(root / name, os.X_OK))
                        else:
                            self.assertNotEqual(result.returncode, 0)
                            self.assertNotIn('chmod', operations)
                            self.assertNotIn('mv', operations)
                            self.assertFalse((root / name).exists())


if __name__ == '__main__':
    unittest.main(verbosity=2)
