#!/usr/bin/env python3
"""Run the real installer against an offline release server and disposable PATH."""
import hashlib
import os
from pathlib import Path
import shutil
import subprocess
import tempfile
import unittest


INSTALLER = Path(__file__).resolve().with_name("install.sh")


class InstallerTests(unittest.TestCase):
    def install(self, scenario, *, version="0.15.0"):
        with tempfile.TemporaryDirectory(prefix="tdx-installer-") as tmp:
            root = Path(tmp)
            commands = root / "commands"
            commands.mkdir()
            destination = root / "bin"
            destination.mkdir()
            previous = destination / "tdx"
            previous.write_bytes(b"previous installation")
            artifact = root / "artifact"
            artifact.write_bytes(b"#!/bin/sh\necho new installation\n")
            digest = hashlib.sha256(artifact.read_bytes()).hexdigest()
            manifest = f"{digest}  tdx-linux-amd64\n"
            if scenario == "corrupt":
                manifest = f"{'0' * 64}  tdx-linux-amd64\n"
            if scenario == "missing":
                manifest = f"{digest}  tdx-darwin-amd64\n"
            if scenario == "duplicate":
                manifest *= 2
            if scenario == "malformed":
                manifest = "not-a-hash  tdx-linux-amd64\n"
            (root / "manifest").write_text(manifest)
            for name, script in {
                "uname": '#!/bin/sh\ncase "$1" in -s) echo Linux;; -m) echo x86_64;; esac\n',
                "curl": '''#!/bin/sh
url=""
output=""
while [ "$#" -gt 0 ]; do
    case "$1" in
        https:*) url="$1" ;;
        -o) shift; output="$1" ;;
    esac
    shift
done
echo "$url" >> "$FIXTURE/requests"
case "$url" in
    */SHA256SUMS)
        [ "$SCENARIO" != "manifest-download" ] || exit 22
        cp "$FIXTURE/manifest" "$output" ;;
    */tdx-linux-amd64)
        [ "$SCENARIO" != "binary-download" ] || exit 22
        cp "$FIXTURE/artifact" "$output" ;;
    *) exit 1 ;;
esac
''',
            }.items():
                path = commands / name
                path.write_text(script)
                path.chmod(0o755)
            if scenario == "staging-failure":
                path = commands / "install"
                path.write_text('#!/bin/sh\nprintf partial > "$4"\nexit 1\n')
                path.chmod(0o755)
            env = dict(os.environ, PATH=str(commands)+os.pathsep+os.environ["PATH"],
                       FIXTURE=str(root), SCENARIO=scenario,
                       TDX_INSTALL_DIR=str(destination), TDX_VERSION=version)
            result = subprocess.run([shutil.which("bash"), str(INSTALLER)], env=env,
                                    capture_output=True, text=True, timeout=10)
            if scenario == "success":
                self.assertEqual(result.returncode, 0, result.stderr)
                self.assertEqual(previous.read_bytes(), artifact.read_bytes())
                self.assertTrue(os.access(previous, os.X_OK))
            else:
                self.assertNotEqual(result.returncode, 0, result.stdout)
                self.assertEqual(previous.read_bytes(), b"previous installation")
            self.assertEqual(sorted(p.name for p in destination.iterdir()), ["tdx"])
            if version == "0.15.0":
                self.assertIn("/download/v0.15.0/", (root / "requests").read_text())

    def test_verified_install(self):
        self.install("success")

    def test_latest_install(self):
        self.install("success", version="")

    def test_failures_preserve_previous_installation(self):
        for scenario in ("corrupt", "missing", "duplicate", "malformed",
                         "binary-download", "manifest-download", "staging-failure"):
            with self.subTest(scenario=scenario):
                self.install(scenario)


if __name__ == "__main__":
    unittest.main()
