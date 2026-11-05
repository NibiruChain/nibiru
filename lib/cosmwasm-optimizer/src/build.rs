use anyhow::{bail, ensure, Context, Result};
use clap::{Args, ValueEnum};
use serde::{Deserialize, Serialize};
use sha2::{Digest, Sha256};
use std::{
    collections::HashSet,
    fs,
    path::{Path, PathBuf},
    process::{Command, Stdio},
};

pub const TOOLCHAIN: &str = "1.86.0";
const ARM_IMAGE: &str = "cosmwasm/optimizer-arm64:0.17.0@sha256:7881e9cece93bb47a6cf4af620bfb9376431a229389f71845b7ed4d237631341";
const AMD_IMAGE: &str = "cosmwasm/optimizer:0.17.0@sha256:7e0b9229c1a4118d0c9a2af2e7f5d95a91f264c26a2ce5681c779926e74d7f85";

#[derive(Clone, Copy, Debug, Default, Serialize, ValueEnum)]
#[serde(rename_all = "lowercase")]
pub enum Mode {
    #[default]
    Dev,
    Release,
}

#[derive(Args, Debug)]
pub struct BuildArgs {
    #[arg(long, default_value = "Cargo.toml")]
    pub manifest_path: PathBuf,
    #[arg(short = 'p', long = "package")]
    pub packages: Vec<String>,
    #[arg(long, default_value = "artifacts")]
    pub output_dir: PathBuf,
    #[arg(long, value_enum, default_value_t = Mode::Dev)]
    pub mode: Mode,
    /// Discover contracts and print build commands without compiling.
    #[arg(long)]
    pub dry_run: bool,
}

#[derive(Clone, Debug, Serialize)]
pub struct Tools {
    pub architecture: String,
    pub platform: String,
    pub image: String,
}
impl Tools {
    pub fn select(daemon_arch: &str, mode: Mode) -> Result<Self> {
        let architecture = match mode {
            Mode::Release => "amd64",
            Mode::Dev => match daemon_arch.trim() {
                "aarch64" | "arm64" => "arm64",
                "x86_64" | "amd64" => "amd64",
                other => bail!("unsupported Docker daemon architecture {other:?}; expected arm64 or amd64"),
            },
        };
        Ok(Self {
            architecture: architecture.into(),
            platform: format!("linux/{architecture}"),
            image: if architecture == "arm64" {
                ARM_IMAGE
            } else {
                AMD_IMAGE
            }
            .into(),
        })
    }
    fn docker(&self) -> Command {
        let mut command = Command::new("docker");
        command.args([
            "run",
            "--rm",
            "--platform",
            &self.platform,
            "--env",
            "RUSTUP_TOOLCHAIN=1.86.0",
            "--entrypoint",
            "/bin/sh",
        ]);
        command
    }
}

fn output(command: &mut Command, stage: &str) -> Result<Vec<u8>> {
    let result = command
        .stderr(Stdio::inherit())
        .output()
        .with_context(|| format!("{stage}: could not start {command:?}"))?;
    ensure!(
        result.status.success(),
        "{stage} failed with {}",
        result.status
    );
    Ok(result.stdout)
}
fn tools(mode: Mode) -> Result<Tools> {
    let arch = output(
        Command::new("docker").args(["info", "--format", "{{.Architecture}}"]),
        "detect Docker daemon architecture",
    )?;
    Tools::select(std::str::from_utf8(&arch)?, mode)
}
pub fn doctor(mode: Mode) -> Result<()> {
    let tools = tools(mode)?;
    println!(
        "Docker platform: {}\nImage: {}",
        tools.platform, tools.image
    );
    let versions = output(
        tools.docker().args([
            &tools.image,
            "-c",
            "rustc --version && cargo --version && wasm-opt --version",
        ]),
        "inspect pinned tools",
    )?;
    print!("{}", String::from_utf8(versions)?);
    Ok(())
}

#[derive(Deserialize)]
struct Metadata {
    packages: Vec<Package>,
    workspace_members: Vec<String>,
}
#[derive(Deserialize)]
struct Package {
    id: String,
    name: String,
    manifest_path: PathBuf,
    targets: Vec<Target>,
    #[serde(default)]
    metadata: serde_json::Value,
}
#[derive(Deserialize)]
struct Target {
    name: String,
    crate_types: Vec<String>,
}
#[derive(Deserialize, Default)]
#[serde(rename_all = "kebab-case", deny_unknown_fields)]
struct Optimizer {
    #[serde(default = "yes")]
    standard_build: bool,
    #[serde(default)]
    builds: Vec<Variant>,
}
fn yes() -> bool {
    true
}
#[derive(Deserialize, Default, Clone)]
#[serde(rename_all = "kebab-case", deny_unknown_fields)]
struct Variant {
    name: String,
    #[serde(default)]
    features: Vec<String>,
    #[serde(default = "yes")]
    default_features: bool,
}
#[derive(Debug, Serialize)]
pub struct ContractBuild {
    pub package: String,
    pub manifest: PathBuf,
    pub target: String,
    pub artifact: String,
    pub cargo_args: Vec<String>,
}

fn safe_name(name: &str) -> Result<()> {
    ensure!(
        !name.is_empty()
            && name
                .bytes()
                .all(|c| c.is_ascii_alphanumeric() || c == b'-' || c == b'_'),
        "invalid artifact or variant name {name:?}"
    );
    Ok(())
}
fn plan(metadata: Metadata, requested: &[String]) -> Result<Vec<ContractBuild>> {
    let members: HashSet<_> = metadata.workspace_members.into_iter().collect();
    let mut found = HashSet::new();
    let mut outputs = HashSet::new();
    let mut builds = Vec::new();
    for package in metadata.packages {
        if !members.contains(&package.id)
            || (!requested.is_empty() && !requested.contains(&package.name))
        {
            continue;
        }
        let Some(target) = package
            .targets
            .iter()
            .find(|t| t.crate_types.iter().any(|k| k == "cdylib"))
        else {
            continue;
        };
        safe_name(&target.name)?;
        found.insert(package.name.clone());
        let optimizer = if let Some(value) = package.metadata.get("optimizer") {
            serde_json::from_value::<Optimizer>(value.clone()).with_context(
                || format!("optimizer metadata in {}", package.name),
            )?
        } else {
            Optimizer {
                standard_build: true,
                builds: vec![],
            }
        };
        let mut variants = optimizer.builds;
        if optimizer.standard_build {
            variants.insert(
                0,
                Variant {
                    default_features: true,
                    ..Variant::default()
                },
            );
        }
        for variant in variants {
            if !variant.name.is_empty() {
                safe_name(&variant.name)?;
            }
            let stem = if variant.name.is_empty() {
                target.name.clone()
            } else {
                format!("{}-{}", target.name, variant.name)
            };
            let artifact = format!("{stem}.wasm");
            ensure!(
                outputs.insert(artifact.clone()),
                "duplicate artifact {artifact}"
            );
            let mut cargo_args = vec![
                "build".into(),
                "--locked".into(),
                "--release".into(),
                "--lib".into(),
                "--target".into(),
                "wasm32-unknown-unknown".into(),
                "--manifest-path".into(),
                package.manifest_path.to_string_lossy().into(),
                "--package".into(),
                package.name.clone(),
            ];
            if !variant.default_features {
                cargo_args.push("--no-default-features".into());
            }
            if !variant.features.is_empty() {
                cargo_args
                    .extend(["--features".into(), variant.features.join(",")]);
            }
            builds.push(ContractBuild {
                package: package.name.clone(),
                manifest: package.manifest_path.clone(),
                target: target.name.clone(),
                artifact,
                cargo_args,
            });
        }
    }
    for name in requested {
        ensure!(
            found.contains(name),
            "package {name:?} is not a workspace cdylib contract"
        );
    }
    ensure!(!builds.is_empty(), "no workspace cdylib contracts selected");
    builds.sort_by(|a, b| a.artifact.cmp(&b.artifact));
    Ok(builds)
}

fn workspace_root(manifest: &Path) -> Result<PathBuf> {
    let parent = manifest.parent().context("manifest has no parent")?;
    for dir in parent.ancestors() {
        let file = dir.join("Cargo.toml");
        if file.exists() {
            let document: toml::Value =
                toml::from_str(&fs::read_to_string(&file)?)?;
            if document.get("workspace").is_some() {
                return Ok(dir.into());
            }
        }
    }
    Ok(parent.into())
}
fn mounted(tools: &Tools, root: &Path, cache: &str) -> Result<Command> {
    // Docker --mount uses commas as delimiters and cannot encode them in paths.
    ensure!(
        !root.to_string_lossy().contains(','),
        "workspace path contains a comma, unsupported by Docker --mount"
    );
    let mut command = tools.docker();
    command.args(["--mount", &format!("type=bind,source={},target=/code,readonly", root.display()), "--mount", &format!("type=volume,source={cache},target=/target"), "--mount", &format!("type=volume,source={cache}-registry,target=/usr/local/cargo/registry"), "--workdir", "/code", "--env", "CARGO_TARGET_DIR=/target", "--env", "RUSTFLAGS=-C link-arg=-s", &tools.image]);
    Ok(command)
}

// Only the artifact bytes go to stdout. Compiler diagnostics go to stderr.
const BUILD_SCRIPT: &str = r#"set -eu
# Cargo's own lock ends before optimization. Keep a volume lock for both stages.
exec 9>/target/.sai-wasm-build.lock
flock 9
wasm="$1"
cd "$2"
shift 2
rm -f "$wasm"
cargo "$@" >&2
wasm-opt -Os "$wasm" -o /tmp/sai-optimized.wasm >&2
cat /tmp/sai-optimized.wasm
"#;

pub fn run(args: BuildArgs) -> Result<()> {
    let tools = tools(args.mode)?;
    let manifest = fs::canonicalize(&args.manifest_path).with_context(|| {
        format!("open manifest {}", args.manifest_path.display())
    })?;
    let root = workspace_root(&manifest)?;
    let source_revision = git_text(&root, &["rev-parse", "HEAD"]);
    let source_dirty =
        git_text(&root, &["status", "--porcelain"]).map(|s| !s.is_empty());
    let lockfile_sha256 = fs::read(root.join("Cargo.lock"))
        .ok()
        .map(|bytes| hex::encode(Sha256::digest(bytes)));
    let container_manifest =
        Path::new("/code").join(manifest.strip_prefix(&root)?);
    let workspace_hash =
        hex::encode(Sha256::digest(root.to_string_lossy().as_bytes()));
    let cache = format!(
        "sai-wasm-{}-{}-{}",
        &workspace_hash[..16],
        tools.architecture,
        TOOLCHAIN.replace('.', "-")
    );
    let metadata_bytes = output(mounted(&tools, &root, &cache)?.args(["-c", "cargo metadata --locked --no-deps --format-version 1 --manifest-path \"$1\"", "metadata", container_manifest.to_str().context("manifest path must be UTF-8")?]), "discover workspace contracts")?;
    let builds = plan(
        serde_json::from_slice(&metadata_bytes)
            .context("parse Cargo metadata")?,
        &args.packages,
    )?;
    println!(
        "Building {} artifacts with {}",
        builds.len(),
        tools.platform
    );
    if args.dry_run {
        for build in &builds {
            let command = build_command(&tools, &root, &cache, build)?;
            println!("{}: {command:?}", build.artifact);
        }
        return Ok(());
    }
    fs::create_dir_all(&args.output_dir)?;
    let staging = tempfile::tempdir_in(&args.output_dir)?;
    let mut artifacts = Vec::new();
    let mut checksums = String::new();
    for build in &builds {
        eprintln!("Building {}", build.artifact);
        let bytes = output(
            &mut build_command(&tools, &root, &cache, build)?,
            &format!("compile and optimize {}", build.artifact),
        )?;
        let path = staging.path().join(&build.artifact);
        fs::write(&path, &bytes)?;
        crate::validation::check_file(&path)
            .with_context(|| format!("validate {}", build.artifact))?;
        let checksum = hex::encode(Sha256::digest(&bytes));
        checksums.push_str(&format!("{checksum}  {}\n", build.artifact));
        artifacts.push(serde_json::json!({"file": build.artifact, "sha256": checksum, "bytes": bytes.len(), "package": build.package, "cargo_args": build.cargo_args}));
    }
    fs::write(staging.path().join("checksums.txt"), checksums)?;
    let info = serde_json::json!({"optimizer_version": env!("CARGO_PKG_VERSION"), "tools": tools, "rust_toolchain": TOOLCHAIN, "binaryen_version": "116", "mode": args.mode, "source_revision": source_revision, "source_dirty": source_dirty, "source_lockfile_sha256": lockfile_sha256, "artifacts": artifacts});
    fs::write(
        staging.path().join("build-info.json"),
        serde_json::to_vec_pretty(&info)?,
    )?;
    fs::write(staging.path().join("build-info.md"), format!("# Wasm build information\n\n- Builder: sai-wasm-optimizer {}\n- Platform: {}\n- Image: `{}`\n- Rust: {}\n- Binaryen: 116\n- Source revision: {}\n- Source dirty: {}\n- Cargo.lock SHA256: {}\n\nSee `build-info.json` for package commands and artifact hashes.\n", env!("CARGO_PKG_VERSION"), tools.platform, tools.image, TOOLCHAIN, source_revision.as_deref().unwrap_or("unavailable"), source_dirty.map(|dirty| if dirty { "yes" } else { "no" }).unwrap_or("unavailable"), lockfile_sha256.as_deref().unwrap_or("unavailable")))?;
    publish(staging.path(), &args.output_dir)?;
    println!(
        "Wrote {} validated artifacts to {}",
        builds.len(),
        args.output_dir.display()
    );
    Ok(())
}
fn git_text(root: &Path, args: &[&str]) -> Option<String> {
    let output = Command::new("git")
        .arg("-C")
        .arg(root)
        .args(args)
        .output()
        .ok()?;
    output
        .status
        .success()
        .then(|| String::from_utf8_lossy(&output.stdout).trim().to_owned())
}

// Called only after all compiler and validation stages succeed.
fn publish(staging: &Path, destination: &Path) -> Result<()> {
    for entry in fs::read_dir(destination)? {
        let entry = entry?;
        if entry.file_type()?.is_file()
            && entry.path().extension().is_some_and(|ext| ext == "wasm")
        {
            fs::remove_file(entry.path())?;
        }
    }
    for entry in fs::read_dir(staging)? {
        let entry = entry?;
        fs::rename(entry.path(), destination.join(entry.file_name()))?;
    }
    Ok(())
}

fn build_command(
    tools: &Tools,
    root: &Path,
    cache: &str,
    build: &ContractBuild,
) -> Result<Command> {
    let mut command = mounted(tools, root, cache)?;
    command
        .args([
            "-c",
            BUILD_SCRIPT,
            "build",
            &format!(
                "/target/wasm32-unknown-unknown/release/{}.wasm",
                build.target
            ),
            build
                .manifest
                .parent()
                .context("contract manifest has no directory")?
                .to_str()
                .context("contract path must be UTF-8")?,
        ])
        .args(&build.cargo_args);
    Ok(command)
}

#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn selects_native_development_and_amd64_release() {
        assert_eq!(
            Tools::select("aarch64", Mode::Dev).unwrap().architecture,
            "arm64"
        );
        assert_eq!(
            Tools::select("x86_64", Mode::Dev).unwrap().architecture,
            "amd64"
        );
        assert_eq!(
            Tools::select("arm64", Mode::Release).unwrap().architecture,
            "amd64"
        );
        assert!(Tools::select("riscv64", Mode::Dev).is_err());
    }
    fn metadata(optimizer: serde_json::Value) -> Metadata {
        serde_json::from_value(serde_json::json!({"workspace_members":["contract"],"packages":[{"id":"contract","name":"my-contract","manifest_path":"/code/wasm-contracts/my-contract/Cargo.toml","targets":[{"name":"my_contract","crate_types":["cdylib","rlib"]}],"metadata":{"optimizer":optimizer}}]})).unwrap()
    }
    #[test]
    fn preserves_variants_and_workspace_paths() {
        let builds = plan(metadata(serde_json::json!({"standard-build":true,"builds":[{"name":"no-default","default-features":false,"features":["foo"]}]})), &[]).unwrap();
        assert_eq!(builds.len(), 2);
        let variant = builds
            .iter()
            .find(|b| b.artifact == "my_contract-no-default.wasm")
            .unwrap();
        assert!(variant.cargo_args.contains(&"--no-default-features".into()));
        assert!(variant
            .cargo_args
            .contains(&"/code/wasm-contracts/my-contract/Cargo.toml".into()));
        assert!(
            plan(metadata(serde_json::json!({})), &["absent".into()]).is_err()
        );
    }
    #[test]
    fn skips_cdylib_helpers_with_standard_build_disabled() {
        let mut workspace = metadata(serde_json::json!({}));
        let mut helper = metadata(serde_json::json!({"standard-build": false}))
            .packages
            .remove(0);
        helper.id = "utils".into();
        helper.name = "utils".into();
        helper.manifest_path = "/code/packages/utils/Cargo.toml".into();
        helper.targets[0].name = "utils".into();
        workspace.workspace_members.push(helper.id.clone());
        workspace.packages.push(helper);
        let builds = plan(workspace, &[]).unwrap();
        assert_eq!(builds.len(), 1);
        assert_eq!(builds[0].package, "my-contract");
        assert_eq!(builds[0].artifact, "my_contract.wasm");
    }
    #[test]
    fn rejects_duplicate_and_unsafe_outputs() {
        assert!(plan(
            metadata(serde_json::json!({"builds":[{"name":"../escape"}]})),
            &[]
        )
        .is_err());
        assert!(plan(
            metadata(
                serde_json::json!({"builds":[{"name":"a-b"},{"name":"a-b"}]})
            ),
            &[]
        )
        .is_err());
    }
    #[test]
    fn docker_command_pins_toolchain_and_uses_target_cache() {
        let tools = Tools::select("arm64", Mode::Dev).unwrap();
        let command =
            mounted(&tools, Path::new("/tmp/work space"), "test-cache").unwrap();
        let args: Vec<_> = command
            .get_args()
            .map(|a| a.to_string_lossy().into_owned())
            .collect();
        assert!(args.contains(&"RUSTUP_TOOLCHAIN=1.86.0".into()));
        assert!(args
            .contains(&"type=volume,source=test-cache,target=/target".into()));
        assert!(args.contains(&ARM_IMAGE.into()));
    }
    #[test]
    fn publishes_only_current_artifacts_and_preserves_other_files() {
        let output = tempfile::tempdir().unwrap();
        fs::write(output.path().join("old.wasm"), b"old").unwrap();
        fs::write(output.path().join("release-notes.md"), b"keep").unwrap();
        let staging = tempfile::tempdir_in(output.path()).unwrap();
        fs::write(staging.path().join("new.wasm"), b"new").unwrap();
        publish(staging.path(), output.path()).unwrap();
        assert!(!output.path().join("old.wasm").exists());
        assert_eq!(fs::read(output.path().join("new.wasm")).unwrap(), b"new");
        assert!(output.path().join("release-notes.md").exists());
    }
    #[test]
    fn subprocess_failure_propagates_stage() {
        let err =
            output(Command::new("sh").args(["-c", "exit 9"]), "test stage")
                .unwrap_err();
        assert!(err.to_string().contains("test stage failed"));
    }
}
