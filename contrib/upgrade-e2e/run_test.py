"""Safety checks for the isolated runner, independent of chain binaries."""

import argparse
import hashlib
import io
from pathlib import Path
import socket
import tarfile
import tempfile
import unittest

from run import Runner, RunError, extract_verified_binary, require_unused_ports, set_toml_value


class IsolationTests(unittest.TestCase):
    def test_existing_run_directory_is_never_reused(self):
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory)
            marker = path / "operator-data"
            marker.write_text("keep me")
            with self.assertRaises(FileExistsError):
                Runner(argparse.Namespace(run_dir=path))
            self.assertEqual(marker.read_text(), "keep me")

    def test_busy_port_is_rejected(self):
        with socket.socket() as occupied:
            occupied.bind(("127.0.0.1", 0))
            with self.assertRaisesRegex(RunError, "already in use"):
                require_unused_ports([occupied.getsockname()[1]])

    def test_toml_section_boundary(self):
        source = '[consensus]\ntimeout_commit = "5s"\n[other]\ntimeout_commit = "keep"\n'
        actual = set_toml_value(source, "consensus", "timeout_commit", '"500ms"')
        self.assertIn('[consensus]\ntimeout_commit = "500ms"', actual)
        self.assertIn('[other]\ntimeout_commit = "keep"', actual)
        with self.assertRaisesRegex(RunError, "found 0"):
            set_toml_value(source, "missing", "timeout_commit", '"500ms"')

    def test_bad_release_checksum_writes_no_executable(self):
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory)
            archive = path / "release.tar.gz"
            archive.write_bytes(b"untrusted bytes")
            manifest = path / "checksums.txt"
            manifest.write_text(f"{'0' * 64}  {archive.name}\n")
            output = path / "nibid"
            with self.assertRaisesRegex(RunError, "Checksum"):
                extract_verified_binary(archive, manifest, output)
            self.assertFalse(output.exists())

    def test_archive_paths_are_not_extracted(self):
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory)
            archive = path / "release.tar.gz"
            with tarfile.open(archive, "w:gz") as bundle:
                member = tarfile.TarInfo("../../nibid")
                member.size = 4
                bundle.addfile(member, io.BytesIO(b"test"))
                ignored = tarfile.TarInfo("../outside")
                ignored.size = 3
                bundle.addfile(ignored, io.BytesIO(b"bad"))
            manifest = path / "checksums.txt"
            manifest.write_text(f"{hashlib.sha256(archive.read_bytes()).hexdigest()}  {archive.name}\n")
            output = path / "safe-nibid"
            extract_verified_binary(archive, manifest, output)
            self.assertEqual(output.read_bytes(), b"test")
            self.assertEqual({p.name for p in path.iterdir()}, {"release.tar.gz", "checksums.txt", "safe-nibid"})


if __name__ == "__main__":
    unittest.main()
