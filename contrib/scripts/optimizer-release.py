#!/usr/bin/env python3
"""Package and publish native optimizer binaries independently of chain releases."""

from __future__ import annotations

import argparse
import gzip
import hashlib
import io
import json
import os
from pathlib import Path
import re
import subprocess
import sys
import tarfile
import tempfile
import tomllib


ROOT = Path(__file__).resolve().parents[2]
CRATE = ROOT / "lib/cosmwasm-optimizer"
NAME = "sai-wasm-optimizer"
REPOSITORY = "NibiruChain/nibiru"
VERSION_PATTERN = r"(?:0|[1-9][0-9]*)\.(?:0|[1-9][0-9]*)\.(?:0|[1-9][0-9]*)(?:-rc\.[1-9][0-9]*)?"
TARGETS = (
    "x86_64-unknown-linux-gnu",
    "aarch64-unknown-linux-gnu",
    "x86_64-apple-darwin",
    "aarch64-apple-darwin",
)
CONTENTS = {
    NAME, "LICENSE", "LICENSE-WASMER", "LICENSE-WASMER-SINGLEPASS",
    "README.md", "build-info.json",
}


def capture(*command: str) -> str:
    return subprocess.check_output(command, cwd=ROOT, text=True).strip()


def configuration() -> dict:
    with (CRATE / "Cargo.toml").open("rb") as source:
        return tomllib.load(source)["package"]


def version() -> str:
    value = configuration()["version"]
    if not re.fullmatch(VERSION_PATTERN, value):
        raise ValueError("expected MAJOR.MINOR.PATCH or MAJOR.MINOR.PATCH-rc.N, with N >= 1")
    return value


def validate_tag(tag: str) -> None:
    expected = f"{NAME}/v{version()}"
    if tag != expected:
        raise ValueError(f"expected tag {expected!r}, got {tag!r}")


def filename(target: str) -> str:
    return f"{NAME}-{version()}-{target}.tar.gz"


def checksum(path: Path) -> str:
    return hashlib.sha256(path.read_bytes()).hexdigest()


def install_manifest() -> str:
    package = configuration()
    # No source dependencies: Binstall only needs metadata to locate an archive.
    lines = ["[package]"]
    for field in ("name", "version", "edition", "license", "description", "repository"):
        lines.append(f"{field} = {json.dumps(package[field])}")
    lines.extend([
        "", "[[bin]]", f"name = {json.dumps(NAME)}", 'path = "src/main.rs"',
        "", "[package.metadata.binstall]",
    ])
    for key, value in package["metadata"]["binstall"].items():
        lines.append(f"{key} = {json.dumps(value)}")
    return "\n".join(lines) + "\n"


def inspect_archive(path: Path, target: str) -> dict:
    with tarfile.open(path, "r:gz") as archive:
        members = archive.getmembers()
        if len(members) != len(CONTENTS) or {m.name for m in members} != CONTENTS:
            raise ValueError(f"unexpected archive contents: {path}")
        if not all(m.isfile() for m in members):
            raise ValueError(f"archive must contain only regular files: {path}")
        if archive.getmember(NAME).mode & 0o111 == 0:
            raise ValueError(f"archive executable has no execute permission: {path}")
        source = archive.extractfile("build-info.json")
        assert source is not None
        info = json.load(source)
        binary = archive.extractfile(NAME)
        assert binary is not None
        if hashlib.sha256(binary.read()).hexdigest() != info.get("binary_sha256"):
            raise ValueError(f"binary checksum does not match provenance: {path}")
    expected = {"name": NAME, "version": version(), "target": target}
    if any(info.get(key) != value for key, value in expected.items()):
        raise ValueError(f"archive provenance does not match release: {path}")
    return info


def package(args: argparse.Namespace) -> None:
    host = next(
        line.removeprefix("host: ")
        for line in capture("rustc", "-vV").splitlines()
        if line.startswith("host: ")
    )
    if args.target != host:
        raise ValueError(f"package on a native {args.target} runner; this host is {host}")
    compiler = capture("rustc", "--version")
    if not compiler.startswith("rustc 1.95.0 "):
        raise ValueError(f"expected Rust 1.95.0, got {compiler}")
    if args.target.endswith("linux-gnu"):
        libc = capture("getconf", "GNU_LIBC_VERSION")
        baseline = (2, 35) if args.target.startswith("x86_64") else (2, 39)
        if tuple(map(int, libc.removeprefix("glibc ").split("."))) > baseline:
            raise ValueError(f"build {args.target} on glibc {baseline} or earlier; got {libc}")
    else:
        libc = None
    revision = capture("git", "rev-parse", "HEAD")
    lock_hash = checksum(ROOT / "Cargo.lock")
    dirty = bool(capture("git", "status", "--porcelain"))
    env = os.environ.copy()
    if args.target.endswith("apple-darwin"):
        env["MACOSX_DEPLOYMENT_TARGET"] = "13.0"
    target_dir = Path(env.get("CARGO_TARGET_DIR", ROOT / "target/optimizer-dist"))
    if not target_dir.is_absolute():
        target_dir = ROOT / target_dir
    env["CARGO_TARGET_DIR"] = str(target_dir)
    # Avoid the chain workspace's expensive whole-program LTO for this host CLI.
    env["CARGO_PROFILE_RELEASE_LTO"] = "false"
    env["CARGO_PROFILE_RELEASE_CODEGEN_UNITS"] = "16"
    env["CARGO_PROFILE_RELEASE_STRIP"] = "debuginfo"
    env.setdefault("CARGO_BUILD_JOBS", "2")
    subprocess.run([
        "cargo", "build", "--locked", "--release", "--package", NAME,
        "--bin", NAME, "--target", args.target,
    ], cwd=ROOT, env=env, check=True)
    if capture("git", "rev-parse", "HEAD") != revision or checksum(ROOT / "Cargo.lock") != lock_hash:
        raise ValueError("source revision or lockfile changed during packaging")
    binary = target_dir / args.target / "release" / NAME
    info = {
        "name": NAME, "version": version(), "target": args.target,
        "source_revision": revision,
        "source_dirty": dirty or bool(capture("git", "status", "--porcelain")),
        "source_lockfile_sha256": lock_hash,
        "rustc": compiler, "binary_sha256": checksum(binary),
        "glibc": libc,
        "macos_deployment_target": env.get("MACOSX_DEPLOYMENT_TARGET"),
        "profile": {"name": "release", "lto": False, "codegen_units": 16, "strip": "debuginfo"},
    }
    files = {
        NAME: binary.read_bytes(),
        "LICENSE": (CRATE / "LICENSE").read_bytes(),
        "LICENSE-WASMER": (ROOT / "lib/wasmer/LICENSE").read_bytes(),
        "LICENSE-WASMER-SINGLEPASS": (ROOT / "lib/wasmer/lib/compiler-singlepass/LICENSE").read_bytes(),
        "README.md": (CRATE / "README.md").read_bytes(),
        "build-info.json": (json.dumps(info, indent=2) + "\n").encode(),
    }
    args.output_dir.mkdir(parents=True, exist_ok=True)
    destination = args.output_dir / filename(args.target)
    epoch = int(capture("git", "show", "-s", "--format=%ct", "HEAD"))
    with destination.open("wb") as output:
        with gzip.GzipFile(filename="", fileobj=output, mode="wb", mtime=epoch) as compressed:
            with tarfile.open(fileobj=compressed, mode="w") as archive:
                for name, data in files.items():
                    entry = tarfile.TarInfo(name)
                    entry.size = len(data)
                    entry.mode = 0o755 if name == NAME else 0o644
                    entry.mtime = epoch
                    archive.addfile(entry, io.BytesIO(data))
    inspect_archive(destination, args.target)
    # Run the executable from the archive, not the compiler's output directory.
    with tempfile.TemporaryDirectory(prefix="optimizer-package-") as temp:
        installed = Path(temp) / NAME
        with tarfile.open(destination, "r:gz") as archive:
            source = archive.extractfile(NAME)
            assert source is not None
            installed.write_bytes(source.read())
        installed.chmod(0o755)
        if capture(str(installed), "--version") != f"{NAME} {version()}":
            raise ValueError("packaged executable reports an unexpected version")
        subprocess.run([
            str(installed), "check", str(ROOT / "lib/cosmwasm-vm/testdata/hackatom_1.0.wasm"),
        ], check=True)
    destination.with_name(destination.name + ".sha256").write_text(
        f"{checksum(destination)}  {destination.name}\n"
    )
    print(destination)


def verify(directory: Path) -> list[Path]:
    expected = {filename(target) for target in TARGETS}
    found = {path.name for path in directory.glob("*.tar.gz")}
    if found != expected:
        raise ValueError(f"expected four platform archives; missing={expected-found}, extra={found-expected}")
    revision = capture("git", "rev-parse", "HEAD")
    archives = []
    for target in TARGETS:
        path = directory / filename(target)
        recorded = path.with_name(path.name + ".sha256").read_text()
        if recorded != f"{checksum(path)}  {path.name}\n":
            raise ValueError(f"checksum mismatch for {path}")
        info = inspect_archive(path, target)
        if (
            info.get("source_revision") != revision
            or info.get("source_lockfile_sha256") != checksum(ROOT / "Cargo.lock")
        ):
            raise ValueError(f"archive source does not match this checkout: {path}")
        if info.get("source_dirty"):
            raise ValueError(f"refusing to release an archive built from dirty sources: {path}")
        archives.append(path)
    return archives


def publish(args: argparse.Namespace) -> None:
    validate_tag(args.tag)
    archives = verify(args.artifacts_dir)
    manifest = args.artifacts_dir / f"{NAME}.toml"
    manifest.write_text(install_manifest())
    sums = args.artifacts_dir / "SHA256SUMS"
    sums.write_text("".join(f"{checksum(path)}  {path.name}\n" for path in [*archives, manifest]))
    notes = args.artifacts_dir / "release-notes.md"
    notes.write_text(
        f"# {NAME} {version()}\n\n"
        "Native optimizer CLI binaries with Nibiru VM validation. Docker is required for contract builds.\n\n"
        "Download the versioned sai-wasm-optimizer.toml manifest and install with "
        "cargo binstall --manifest-path <manifest> --strategies crate-meta-data "
        "--no-confirm sai-wasm-optimizer.\n\n"
        "Linux AMD64 requires glibc 2.35 or later; Linux ARM64 requires glibc 2.39 or later. "
        "macOS binaries require macOS 13 or later.\n"
    )
    command = [
        "gh", "release", "create", args.tag, "--repo", REPOSITORY,
        "--verify-tag", "--latest=false", "--title", f"{NAME} {version()}",
        "--notes-file", str(notes), *map(str, archives), str(manifest), str(sums),
    ]
    if "-rc." in version():
        command.append("--prerelease")
    if not args.run:
        print("Release verified; no publication requested.")
        print(" ".join(command))
        return
    if capture("git", "status", "--porcelain"):
        raise ValueError("release publication requires a clean checkout")
    tag_revision = capture("git", "rev-parse", f"refs/tags/{args.tag}^{{commit}}")
    if tag_revision != capture("git", "rev-parse", "HEAD"):
        raise ValueError("release tag does not point to the packaged checkout")
    subprocess.run(command, cwd=ROOT, check=True)


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    sub = parser.add_subparsers(dest="command", required=True)
    sub.add_parser("version")
    tag = sub.add_parser("validate-tag")
    tag.add_argument("--tag", required=True)
    manifest = sub.add_parser("manifest")
    manifest.add_argument("--output-dir", type=Path, required=True)
    build = sub.add_parser("package")
    build.add_argument("--target", choices=TARGETS, required=True)
    build.add_argument("--output-dir", type=Path, default=ROOT / "dist/optimizer")
    check = sub.add_parser("verify")
    check.add_argument("--artifacts-dir", type=Path, required=True)
    release = sub.add_parser("publish")
    release.add_argument("--artifacts-dir", type=Path, required=True)
    release.add_argument("--tag", required=True)
    release.add_argument("--run", action="store_true")
    args = parser.parse_args()
    for attribute in ("output_dir", "artifacts_dir"):
        if hasattr(args, attribute):
            setattr(args, attribute, getattr(args, attribute).resolve())
    if args.command == "version":
        print(version())
    elif args.command == "validate-tag":
        validate_tag(args.tag)
        print(args.tag)
    elif args.command == "manifest":
        args.output_dir.mkdir(parents=True, exist_ok=True)
        path = args.output_dir / f"{NAME}.toml"
        path.write_text(install_manifest())
        print(path)
    elif args.command == "package":
        package(args)
    elif args.command == "verify":
        print(f"Verified {len(verify(args.artifacts_dir))} platform archives")
    else:
        publish(args)


if __name__ == "__main__":
    try:
        main()
    except (ValueError, OSError, subprocess.CalledProcessError, StopIteration) as error:
        print(f"optimizer release: {error}", file=sys.stderr)
        sys.exit(1)
