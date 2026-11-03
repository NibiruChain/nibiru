"""Runner isolation regressions that do not start nodes or download binaries."""

import argparse
import hashlib
import io
from pathlib import Path
import socket
import tarfile

import pytest

from run import Runner, RunError, extract_verified_binary, require_unused_ports, set_toml_value


def test_existing_run_directory_is_never_reused(tmp_path: Path):
    """Refuse an existing home before touching any operator data inside it."""
    marker = tmp_path / "operator-data"
    marker.write_text("keep me")
    with pytest.raises(FileExistsError):
        Runner(argparse.Namespace(run_dir=tmp_path))
    assert marker.read_text() == "keep me"


def test_busy_port_is_rejected():
    """Reject occupied ports instead of interfering with another local node."""
    with socket.socket() as occupied:
        occupied.bind(("127.0.0.1", 0))
        with pytest.raises(RunError, match="already in use"):
            require_unused_ports([occupied.getsockname()[1]])


def test_toml_section_boundary():
    """Change the intended section without overwriting a same-named setting."""
    source = '[consensus]\ntimeout_commit = "5s"\n[other]\ntimeout_commit = "keep"\n'
    actual = set_toml_value(source, "consensus", "timeout_commit", '"500ms"')
    assert '[consensus]\ntimeout_commit = "500ms"' in actual
    assert '[other]\ntimeout_commit = "keep"' in actual
    with pytest.raises(RunError, match="found 0"):
        set_toml_value(source, "missing", "timeout_commit", '"500ms"')


def test_bad_release_checksum_writes_no_executable(tmp_path: Path):
    """An unverified archive must not leave an executable for a later run."""
    archive = tmp_path / "release.tar.gz"
    archive.write_bytes(b"untrusted bytes")
    manifest = tmp_path / "checksums.txt"
    manifest.write_text(f"{'0' * 64}  {archive.name}\n")
    output = tmp_path / "nibid"
    with pytest.raises(RunError, match="Checksum"):
        extract_verified_binary(archive, manifest, output)
    assert not output.exists()


def test_archive_paths_are_not_extracted(tmp_path: Path):
    """Copy the verified binary bytes while ignoring archive traversal paths."""
    archive = tmp_path / "release.tar.gz"
    with tarfile.open(archive, "w:gz") as bundle:
        member = tarfile.TarInfo("../../nibid")
        member.size = 4
        bundle.addfile(member, io.BytesIO(b"test"))
        ignored = tarfile.TarInfo("../outside")
        ignored.size = 3
        bundle.addfile(ignored, io.BytesIO(b"bad"))
    manifest = tmp_path / "checksums.txt"
    manifest.write_text(f"{hashlib.sha256(archive.read_bytes()).hexdigest()}  {archive.name}\n")
    output = tmp_path / "safe-nibid"
    extract_verified_binary(archive, manifest, output)
    assert output.read_bytes() == b"test"
    assert {p.name for p in tmp_path.iterdir()} == {"release.tar.gz", "checksums.txt", "safe-nibid"}
