workspaces := "./packages"
# workspaces := "./packages ./core"

# Displays available recipes by running `just -l`.
setup:
  #!/usr/bin/env bash
  just -l

install:
  # https://crates.io/crates/clippy
  rustup component add clippy
  # https://crates.io/crates/cargo-llvm-cov
  cargo install cargo-llvm-cov
  # https://crates.io/crates/cosmwasm-check
  cargo install cosmwasm-check

wasm-all:
  bash scripts/wasm-out.sh

# Check if a Wasm smart contract binary is ready for the blockchain
wasm-check:
  cosmwasm-check artifacts/*.wasm

# Compiles a single CW contract to wasm bytecode.
# wasm-single:
#   bash scripts/wasm-out.sh --single

# Runs rustfmt
fmt:
  cargo fmt --all

# Runs rustfmt without updating
fmt-check:
  cargo fmt --all -- --check

# Compiles Rust code
build:
  cargo build

build-update:
  cargo update
  cargo build

# Clean target files and temp files
clean:
  cargo clean

# Run linter + fix
clippy:
  cargo clippy --fix --allow-dirty --allow-staged

# Run linter + check only
clippy-check:
  cargo clippy

# Test a specific package or contract
test *pkg:
  #!/usr/bin/env bash
  set -e;
  if [ -z "{{pkg}}" ]; then
    just test-all
  else
    RUST_BACKGTRACE="1" cargo test --package "{{pkg}}"
  fi

# Test everything in the workspace.
test-all:
  cargo test

# Run vendored Wasmer lib unit tests + compilers/WAST integration (CI job `wasmer`).
test-wasmer:
  #!/usr/bin/env bash
  # TODO: Wire Wasmer validation into `just test`, `just test-all`, and/or `just tidy`
  # once we decide how it should interact with root-workspace `cargo test` and
  # package `cosmwasm-vm` (separate Cargo workspace, longer runtime, cache paths).
  set -euo pipefail
  cd packages/wasmer
  cargo test -p wasmer --lib --no-default-features --features cranelift,singlepass,wat
  cargo test --lib \
    -p wasmer-vm \
    -p wasmer-types \
    -p wasmer-middlewares \
    -p wasmer-compiler \
    -p wasmer-compiler-singlepass \
    -p wasmer-compiler-cranelift
  # WAST spectests + hand-written compiler integration (~452 pass, ~119 ignored).
  cargo test --test compilers --features 'cranelift,singlepass'

# Wasmer coverage: same three test passes as test-wasmer, merged lcov at repo root.
test-wasmer-cov:
  #!/usr/bin/env bash
  set -euo pipefail
  root="$(git rev-parse --show-toplevel)"
  output="${root}/lcov-wasmer.info"
  cd "${root}/packages/wasmer"
  rustup component add llvm-tools-preview --toolchain 1.81
  cargo llvm-cov clean --workspace
  cargo llvm-cov --no-report -p wasmer --lib \
    --no-default-features --features cranelift,singlepass,wat
  cargo llvm-cov --no-report --lib \
    -p wasmer-vm \
    -p wasmer-types \
    -p wasmer-middlewares \
    -p wasmer-compiler \
    -p wasmer-compiler-singlepass \
    -p wasmer-compiler-cranelift
  cargo llvm-cov --no-report --test compilers --features 'cranelift,singlepass'
  cargo llvm-cov report --lcov --output-path "${output}" --no-default-ignore-filename-regex
  echo "Wrote ${output}"

# Test everything and output coverage report.
test-coverage:
  cargo llvm-cov --lcov --output-path lcov.info \
    --ignore-filename-regex .*buf\/[^\/]+\.rs$
  # TODO: Include wasmer workspace in coverage reporting.

alias t := tidy

# Format, lint, and test
tidy:
  just fmt
  just clippy
  just wasm-check
  just test

# Format, lint, update dependencies, and test
tidy-update: build-update
  just tidy

gen-schema:
  #!/usr/bin/env bash
  for dir in contracts/*/; do
    dir_name=$(basename $dir)

    echo "Generating schema for $dir"
    cd $dir
    cargo schema
    mv ./schema ../../schema/$dir_name
  done

# (Safe) Dry run for publishing coupled packages (default behavior)
publish:
  bash scripts/publish-coupled.sh

# Publish coupled packages that share "workspace.version" to crates.io
publish-run:
  bash scripts/publish-coupled.sh --run