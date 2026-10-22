"""Exercise the real installer with local release fixtures and a harmless deploy script."""
import hashlib
import io
import json
import os
from pathlib import Path
import subprocess
import tarfile
import tempfile
import unittest


INSTALLER = Path(__file__).resolve().parents[2] / 'install.sh'
VERSION = 'v0.1.0'


class InstallerTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory(prefix='gost-panel-installer-test-')
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        self.bin = self.root / 'bin'
        self.bin.mkdir()
        self.deploy_log = self.root / 'deploy.log'
        self.curl_log = self.root / 'curl.log'
        self.env = dict(os.environ, PATH=f'{self.bin}:/usr/bin:/bin',
                        INSTALL_TEST_ROOT=str(self.root), INSTALL_TEST_ARCH='x86_64',
                        INSTALL_TEST_LOG=str(self.deploy_log))
        self.write_tool('uname', '#!/bin/bash\nif [[ "$1" == -s ]]; then echo Linux; else echo "$INSTALL_TEST_ARCH"; fi\n')
        self.write_tool('curl', '''#!/usr/bin/env python3
import os, pathlib, shutil, sys
root = pathlib.Path(os.environ['INSTALL_TEST_ROOT'])
with (root / 'curl.log').open('a') as log:
    log.write(' '.join(sys.argv[1:]) + '\\n')
url = next(arg for arg in sys.argv[1:] if arg.startswith('https://'))
name = 'release.json' if url.endswith('/latest') else url.rsplit('/', 1)[1]
output = sys.argv[sys.argv.index('-o') + 1]
shutil.copyfile(root / name, output)
''')
        (self.root / 'release.json').write_text(json.dumps({'tag_name': VERSION}))
        self.make_package('amd64')
        self.make_package('arm64')
        self.write_checksums()

    def write_tool(self, name, content):
        target = self.bin / name
        target.write_text(content)
        target.chmod(0o755)

    def make_package(self, arch, unsafe=False, missing_frontend=False):
        prefix = f'gost-panel-{VERSION}-linux-{arch}'
        target = self.root / f'{prefix}.tar.gz'
        files = {'deploy.sh': '#!/bin/bash\nprintf "%s\\n" "$@" > "$INSTALL_TEST_LOG"\n',
                 'gost-panel': 'backend fixture'}
        if not missing_frontend:
            files['static/index.html'] = '<html>frontend fixture</html>'
        with tarfile.open(target, 'w:gz') as archive:
            for name, content in files.items():
                data = content.encode()
                entry = tarfile.TarInfo(f'{prefix}/{name}')
                entry.size = len(data)
                entry.mode = 0o755 if name == 'deploy.sh' else 0o644
                archive.addfile(entry, io.BytesIO(data))
            if unsafe:
                entry = tarfile.TarInfo('../escaped-file')
                archive.addfile(entry, io.BytesIO(b''))
        return target

    def write_checksums(self):
        lines = [f'{hashlib.sha256(p.read_bytes()).hexdigest()}  {p.name}\n'
                 for p in sorted(self.root.glob('*.tar.gz'))]
        (self.root / 'SHA256SUMS.txt').write_text(''.join(lines))

    def run_installer(self, *args):
        return subprocess.run(['bash', str(INSTALLER), *args], env=self.env,
                              capture_output=True, text=True, timeout=20)

    def test_latest_amd64_passes_deployment_arguments(self):
        result = self.run_installer('--host', '0.0.0.0', '--port', '19999')
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(self.deploy_log.read_text().splitlines(), ['--host', '0.0.0.0', '--port', '19999'])
        self.assertIn('/releases/latest', self.curl_log.read_text())
        self.assertIn(f'/download/{VERSION}/gost-panel-{VERSION}-linux-amd64.tar.gz', self.curl_log.read_text())

    def test_arm64_pinned_version_download_only(self):
        self.env['INSTALL_TEST_ARCH'] = 'aarch64'
        output = self.root / 'downloaded'
        result = self.run_installer('--version', VERSION, '--download-only', str(output))
        self.assertEqual(result.returncode, 0, result.stderr)
        name = f'gost-panel-{VERSION}-linux-arm64.tar.gz'
        self.assertEqual((output / name).read_bytes(), (self.root / name).read_bytes())
        self.assertIn(name, (output / 'SHA256SUMS.txt').read_text())
        self.assertNotIn('/releases/latest', self.curl_log.read_text())
        self.assertFalse(self.deploy_log.exists())

    def test_checksum_mismatch_never_deploys(self):
        with (self.root / f'gost-panel-{VERSION}-linux-amd64.tar.gz').open('ab') as package:
            package.write(b'tampered')
        result = self.run_installer('--version', VERSION)
        self.assertNotEqual(result.returncode, 0)
        self.assertIn('SHA256', result.stderr)
        self.assertFalse(self.deploy_log.exists())

    def test_missing_checksum_never_deploys(self):
        (self.root / 'SHA256SUMS.txt').write_text('')
        result = self.run_installer('--version', VERSION)
        self.assertNotEqual(result.returncode, 0)
        self.assertFalse(self.deploy_log.exists())

    def test_path_traversal_never_deploys(self):
        self.make_package('amd64', unsafe=True)
        self.write_checksums()
        result = self.run_installer('--version', VERSION)
        self.assertNotEqual(result.returncode, 0)
        self.assertFalse(self.deploy_log.exists())
        self.assertFalse((self.root / 'escaped-file').exists())

    def test_missing_frontend_never_deploys(self):
        self.make_package('amd64', missing_frontend=True)
        self.write_checksums()
        result = self.run_installer('--version', VERSION)
        self.assertNotEqual(result.returncode, 0)
        self.assertFalse(self.deploy_log.exists())

    def test_local_archive_works_without_network(self):
        result = self.run_installer('--archive', str(self.root / f'gost-panel-{VERSION}-linux-amd64.tar.gz'))
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertTrue(self.deploy_log.exists())
        self.assertFalse(self.curl_log.exists())

    def test_unsupported_architecture_stops_before_download(self):
        self.env['INSTALL_TEST_ARCH'] = 'riscv64'
        result = self.run_installer()
        self.assertNotEqual(result.returncode, 0)
        self.assertFalse(self.curl_log.exists())
        self.assertFalse(self.deploy_log.exists())

    def test_conflicting_modes_stop_before_download(self):
        result = self.run_installer('--source', '--archive', 'not-used.tar.gz')
        self.assertNotEqual(result.returncode, 0)
        self.assertFalse(self.curl_log.exists())
        self.assertFalse(self.deploy_log.exists())


if __name__ == '__main__':
    unittest.main(verbosity=2)
