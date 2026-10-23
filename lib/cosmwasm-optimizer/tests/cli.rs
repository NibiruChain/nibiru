//! Exercise subprocess failures and dry runs without Docker or a compiler.
#![cfg(unix)]

use std::{fs, os::unix::fs::PermissionsExt, path::PathBuf, process::Command};

struct Fixture {
    _temp: tempfile::TempDir,
    root: PathBuf,
    output: PathBuf,
    docker: PathBuf,
    metadata: String,
}

impl Fixture {
    fn new() -> Self {
        let temp = tempfile::tempdir().unwrap();
        let root = temp.path().join("contract with spaces");
        fs::create_dir(&root).unwrap();
        fs::write(
            root.join("Cargo.toml"),
            "[package]\nname = 'counter'\nversion = '0.1.0'\n",
        )
        .unwrap();
        fs::write(root.join("Cargo.lock"), "version = 4\n").unwrap();
        let docker = temp.path().join("docker");
        fs::write(
            &docker,
            r#"#!/bin/sh
set -eu
if [ "$1" = info ]; then printf 'aarch64\n'; exit 0; fi
case "$*" in
  *'cargo metadata'*) printf '%s' "$FAKE_METADATA" ;;
  *)
    if [ "${FAKE_BUILD_INVALID_WASM:-0}" = 1 ]; then printf 'not Wasm'; exit 0; fi
    printf 'simulated build failure\n' >&2; exit 29 ;;
esac
"#,
        )
        .unwrap();
        fs::set_permissions(&docker, fs::Permissions::from_mode(0o755)).unwrap();
        let metadata = serde_json::json!({
            "workspace_members": ["counter"],
            "packages": [{"id":"counter", "name":"counter", "manifest_path":"/code/Cargo.toml", "targets":[{"name":"counter", "crate_types":["cdylib"]}], "metadata":{}}]
        }).to_string();
        let output = temp.path().join("output with spaces");
        Self {
            _temp: temp,
            root,
            output,
            docker,
            metadata,
        }
    }

    fn command(&self) -> Command {
        let mut command = Command::new(env!("CARGO_BIN_EXE_sai-wasm-optimizer"));
        let path = format!(
            "{}:{}",
            self.docker.parent().unwrap().display(),
            std::env::var("PATH").unwrap_or_default()
        );
        command
            .env("PATH", path)
            .env("FAKE_METADATA", &self.metadata)
            .args(["build", "--manifest-path"])
            .arg(self.root.join("Cargo.toml"))
            .arg("--output-dir")
            .arg(&self.output);
        command
    }
}

#[test]
fn build_failure_preserves_existing_artifacts_and_metadata() {
    let f = Fixture::new();
    fs::create_dir(&f.output).unwrap();
    fs::write(f.output.join("counter.wasm"), b"previous artifact").unwrap();
    fs::write(f.output.join("build-info.md"), b"previous metadata").unwrap();
    let result = f.command().output().unwrap();
    assert!(!result.status.success());
    assert!(String::from_utf8_lossy(&result.stderr)
        .contains("compile and optimize counter.wasm"));
    assert_eq!(
        fs::read(f.output.join("counter.wasm")).unwrap(),
        b"previous artifact"
    );
    assert_eq!(
        fs::read(f.output.join("build-info.md")).unwrap(),
        b"previous metadata"
    );
    assert_eq!(fs::read_dir(&f.output).unwrap().count(), 2);
}

#[test]
fn validation_failure_preserves_existing_artifacts_and_checksums() {
    let f = Fixture::new();
    fs::create_dir(&f.output).unwrap();
    fs::write(f.output.join("counter.wasm"), b"previous artifact").unwrap();
    fs::write(f.output.join("checksums.txt"), b"previous checksums").unwrap();
    let result = f
        .command()
        .env("FAKE_BUILD_INVALID_WASM", "1")
        .output()
        .unwrap();
    assert!(!result.status.success());
    assert!(String::from_utf8_lossy(&result.stderr)
        .contains("validate counter.wasm"));
    assert_eq!(
        fs::read(f.output.join("counter.wasm")).unwrap(),
        b"previous artifact"
    );
    assert_eq!(
        fs::read(f.output.join("checksums.txt")).unwrap(),
        b"previous checksums"
    );
    assert_eq!(fs::read_dir(&f.output).unwrap().count(), 2);
}

#[test]
fn dry_run_selects_pinned_native_tools_without_publishing() {
    let f = Fixture::new();
    let result = f.command().arg("--dry-run").output().unwrap();
    assert!(
        result.status.success(),
        "{}",
        String::from_utf8_lossy(&result.stderr)
    );
    let output = String::from_utf8(result.stdout).unwrap();
    assert!(output.contains("linux/arm64"));
    assert!(output.contains("optimizer-arm64:0.17.0@sha256:"));
    assert!(output.contains("RUSTUP_TOOLCHAIN=1.86.0"));
    assert!(!f.output.exists());
}

#[test]
fn release_dry_run_selects_amd64_on_an_arm_daemon() {
    let f = Fixture::new();
    let result = f
        .command()
        .args(["--mode", "release", "--dry-run"])
        .output()
        .unwrap();
    assert!(result.status.success());
    assert!(String::from_utf8(result.stdout)
        .unwrap()
        .contains("linux/amd64"));
}

#[test]
fn check_rejects_invalid_wasm() {
    let f = Fixture::new();
    let invalid = f.root.join("invalid.wasm");
    fs::write(&invalid, b"not Wasm").unwrap();
    let result = Command::new(env!("CARGO_BIN_EXE_sai-wasm-optimizer"))
        .arg("check")
        .arg(invalid)
        .output()
        .unwrap();
    assert!(!result.status.success());
    assert!(
        String::from_utf8_lossy(&result.stderr).contains("Nibiru VM rejected")
    );
}
