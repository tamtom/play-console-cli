"""Exercise the installer and Makefile through their public process interfaces."""
import hashlib
import os
from pathlib import Path
import shutil
import subprocess
import sys
import tempfile
import unittest

ROOT = Path(__file__).resolve().parents[2]


@unittest.skipIf(os.name == "nt", "Bash installer and Makefile are POSIX entry points")
class ReleaseToolsTest(unittest.TestCase):
    def test_installer_requires_an_exact_verified_asset(self):
        for mode in ("valid", "unavailable", "missing", "malformed", "duplicate", "mismatch", "no_hash_tool"):
            with self.subTest(mode=mode), tempfile.TemporaryDirectory() as directory:
                root = Path(directory)
                commands = root / "bin"
                commands.mkdir()
                for name in ("uname", "mktemp", "rm", "awk", "grep", "mkdir", "install", "mv", "chmod"):
                    path = shutil.which(name)
                    if path:
                        (commands / name).symlink_to(path)
                if mode != "no_hash_tool":
                    for name in ("shasum", "sha256sum"):
                        path = shutil.which(name)
                        if path:
                            (commands / name).symlink_to(path)
                curl = commands / "curl"
                curl.write_text(f"#!{sys.executable}\n" + '''
import hashlib, os, pathlib, platform, sys
args=sys.argv[1:]; output=pathlib.Path(args[args.index('-o')+1])
url=next(arg for arg in args if arg.startswith('https://'))
body=b'#!/bin/sh\\necho candidate-1.0.0\\n'
arch={'x86_64':'amd64','aarch64':'arm64'}.get(platform.machine(),platform.machine())
asset='gplay-'+platform.system().lower()+'-'+arch
mode=os.environ['TEST_CHECKSUM_MODE']
if url.endswith('checksums.txt'):
 if mode=='unavailable': sys.exit(22)
 checksum=hashlib.sha256(body).hexdigest()
 if mode=='mismatch': checksum='0'*64
 if mode=='malformed': checksum='invalid'
 if mode=='missing': asset='another-asset'
 text=checksum+'  '+asset+'\\n'
 if mode=='duplicate': text+=text
 output.write_text(text)
else: output.write_bytes(body)
''')
                curl.chmod(0o755)
                install = root / "installed"
                install.mkdir()
                binary = install / "gplay"
                binary.write_bytes(b"previous executable")
                env = dict(os.environ, PATH=str(commands), HOME=str(root), GPLAY_INSTALL_DIR=str(install),
                           GPLAY_VERSION="v1.0.0", TEST_CHECKSUM_MODE=mode)
                result = subprocess.run([shutil.which("bash"), str(ROOT / "install.sh")], env=env,
                                        capture_output=True, text=True, timeout=20)
                if mode == "valid":
                    self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
                    self.assertIn(b"candidate-1.0.0", binary.read_bytes())
                    self.assertTrue(os.access(binary, os.X_OK))
                else:
                    self.assertNotEqual(result.returncode, 0, result.stdout + result.stderr)
                    self.assertEqual(binary.read_bytes(), b"previous executable")

    def test_make_build_rebuilds_changed_source_and_version(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            shutil.copy(ROOT / "Makefile", root / "Makefile")
            (root / "go.mod").write_text("module fixture\n")
            (root / "main.go").write_text("first source")
            go = root / "fixture-go"
            go.write_text(f"#!{sys.executable}\n" + '''
import pathlib,sys
if sys.argv[1]=='env': print(pathlib.Path.cwd());sys.exit()
pathlib.Path('gplay').write_text(pathlib.Path('main.go').read_text()+' '+repr(sys.argv))
''')
            go.chmod(0o755)
            for version, source in (("0.9.0", "first source"), ("1.0.0", "changed source")):
                (root / "main.go").write_text(source)
                result = subprocess.run(["make", "build", "GO=" + str(go), "VERSION=" + version], cwd=root,
                                        capture_output=True, text=True, timeout=20)
                self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
                built = (root / "gplay").read_text()
                self.assertIn(source, built)
                self.assertIn("Version=" + version, built)

class PowerShellInstallerTest(unittest.TestCase):
    def test_powershell_requires_an_exact_verified_asset(self):
        shell = os.environ.get("GPLAY_TEST_PWSH") or shutil.which("pwsh")
        if not shell:
            self.skipTest("PowerShell contract runs on native Windows release runners")
        for mode in ("valid", "unavailable", "missing", "substring", "malformed", "duplicate", "mismatch"):
            with self.subTest(mode=mode), tempfile.TemporaryDirectory() as work:
                result = subprocess.run([shell, "-NoProfile", "-NonInteractive", "-File",
                                         str(ROOT / "scripts/tests/install_contract.ps1"),
                                         "-Installer", str(ROOT / "install.ps1"), "-Work", work, "-Mode", mode],
                                        capture_output=True, text=True, timeout=30)
                self.assertEqual(result.returncode, 0, result.stdout + result.stderr)


if __name__ == "__main__":
    unittest.main()
