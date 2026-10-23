use anyhow::Result;
use clap::{Parser, Subcommand};
use sai_wasm_optimizer::{build, validation};
use std::path::PathBuf;

#[derive(Parser)]
#[command(
    version,
    about = "Build CosmWasm contracts with pinned native container tools"
)]
struct Cli {
    #[command(subcommand)]
    command: Command,
}

#[derive(Subcommand)]
enum Command {
    /// Report Docker architecture and pinned compiler versions.
    Doctor {
        #[arg(long, value_enum, default_value_t = build::Mode::Dev)]
        mode: build::Mode,
    },
    /// Compile, optimize, and validate workspace contracts.
    Build(build::BuildArgs),
    /// Validate artifacts with Nibiru's maintained CosmWasm VM.
    Check {
        #[arg(required = true)]
        artifacts: Vec<PathBuf>,
    },
}

fn main() -> Result<()> {
    match Cli::parse().command {
        Command::Doctor { mode } => build::doctor(mode),
        Command::Build(args) => build::run(args),
        Command::Check { artifacts } => {
            for artifact in artifacts {
                validation::check_file(&artifact)?;
                println!("{}: valid", artifact.display());
            }
            Ok(())
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn clap_accepts_build_options_and_rejects_bad_mode() {
        let cli = Cli::try_parse_from([
            "sai-wasm-optimizer",
            "build",
            "-p",
            "foo",
            "--package",
            "bar",
            "--mode",
            "release",
            "--dry-run",
        ])
        .unwrap();
        let Command::Build(args) = cli.command else {
            panic!("expected build");
        };
        assert_eq!(args.packages, ["foo", "bar"]);
        assert!(matches!(args.mode, build::Mode::Release));
        assert!(args.dry_run);
        assert!(Cli::try_parse_from([
            "sai-wasm-optimizer",
            "build",
            "--mode",
            "unknown"
        ])
        .is_err());
    }
    #[test]
    fn check_requires_artifact() {
        assert!(Cli::try_parse_from(["sai-wasm-optimizer", "check"]).is_err());
    }
}
