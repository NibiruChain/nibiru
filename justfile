# Displays available recipes by running `just -l`.
setup:
    #!/usr/bin/env bash
    just -l

# Locally install the `nibid` binary and build if needed.
install:
    go mod tidy
    contrib/scripts/build-nibiru.sh --run

alias i := install

install-clean:
    rm -rf temp
    just install

install-covtool:
    go install github.com/Unique-Divine/jiyuu/gocovmerge@v0.0.2

# Build the `nibid` binary.
build:
    contrib/scripts/build-nibiru.sh --run --just-build

alias b := build

# Cleans the Go cache, modcache, and testcashe
clean-cache:
    go clean -cache -testcache -modcache

# Protobuf command dispatcher. Ex: `just proto gen`, `just proto fmt`, `just proto lint`, `just proto all`
proto *ARGS:
    bash contrib/scripts/proto.sh {{ ARGS }}

# Generate Solidity artifacts for evm/embeds
gen-embeds:
    #!/usr/bin/env bash
    source contrib/bashlib.sh

    embeds_dir="evm/embeds"
    log_info "Begin to compile Solidity in $embeds_dir"
    which_ok yarn
    log_info "Using system node version: $(yarn exec -- node -v)"

    cd "$embeds_dir" || (log_error "path $embeds_dir not found" && exit 1)
    yarn --check-files
    yarn hardhat compile && echo "SUCCESS: yarn hardhat compile succeeded" || echo "Run failed"
    log_success "Compiled Solidity in $embeds_dir"

    go run "gen-abi/main.go"
    log_success "Saved ABI JSON files to $embeds_dir/abi for npm publishing"

# Generates CHANGELOG-UNRELEASED.md based on commits and pull requests.
gen-changelog:
    #!/usr/bin/env bash
    source contrib/bashlib.sh
    which_ok cargo
    if ! which_ok git-cliff; then 
      echo "Installing git-cliff with cargo"
      cargo install git-cliff
    fi 

    which_ok git-cliff

    LAST_VER="v2.9.0"
    start_branch="$(git branch --show-current)"

    origdir="$(pwd)"
    tmpdir="$(mktemp -d)"
    cleanup() { rm -rf "$tmpdir"; }
    trap cleanup EXIT

    # Create a detached worktree at main so we don’t touch the current branch
    git fetch -q origin main
    git worktree add --detach --quiet "$tmpdir" origin/main

    # Run git-cliff in the main worktree but write the file into the original repo
    ( cd "$tmpdir" && git-cliff "$LAST_VER.." --config="$origdir/cliff.toml" ) > CHANGELOG-UNRELEASED.md
    last_exit_code="$?"
    if [ "$last_exit_code" -ne 0 ]; then
      log_error "changelog generation failed"
      exit 1
    fi

    log_success "Created CHANGELOG-UNRELEASED.md with changes since $LAST_VER"

# Generate the Nibiru Token Registry files
gen-token-registry:
    go run token-registry/main/main.go

# Generate protobuf-based types in Rust
gen-proto-rs:
    bash proto/buf.gen.rs.sh

# Generate OpenAPI and Swagger files
gen-proto-openapi:
    bun run proto/buf-gen-swagger.ts 2>&1 | tee out.txt

lint:
    #!/usr/bin/env bash
    set -euo pipefail
    source contrib/bashlib.sh

    image_version="v2.6.1"
    # Cap golangci-lint parallelism so lint stays responsive on developer machines.
    # On a 12-CPU WSL host, -j 4 benchmarked faster than -j 6 while using much
    # less CPU, and was materially faster than -j 2 or -j 3.
    lint_cmd=(golangci-lint run -v --fix -j 4)

    if which_ok golangci-lint >/dev/null 2>&1; then
      local_version="$(golangci-lint version --short 2>/dev/null || true)"
      image_major="${image_version#v}"
      image_major="${image_major%%.*}"
      local_major="${local_version#v}"
      local_major="${local_major%%.*}"

      if [ "$local_version" = "$image_version" ] || [ "v$local_version" = "$image_version" ]; then
        log_info "Running local golangci-lint $local_version"
        GOLANGCI_LINT_CACHE="${GOLANGCI_LINT_CACHE:-/tmp/nibi-golangci-lint-cache}" "${lint_cmd[@]}"
        exit 0
      fi

      if [ -n "$local_major" ] && [ "$local_major" = "$image_major" ]; then
        log_warning "Running local golangci-lint ${local_version:-unknown}; repo pins $image_version"
        GOLANGCI_LINT_CACHE="${GOLANGCI_LINT_CACHE:-/tmp/nibi-golangci-lint-cache}" "${lint_cmd[@]}"
        exit 0
      fi

      log_warning "Local golangci-lint version ${local_version:-unknown} does not match major version $image_major; using Docker"
    else
      log_info "golangci-lint not found locally; using Docker"
    fi

    which_ok docker
    log_info "Running golangci-lint with Docker image golangci/golangci-lint:$image_version"
    docker run --rm \
      -v "$(pwd)":/app \
      -v ~/.cache/golangci-lint/$image_version:/root/.cache \
      -w /app \
      golangci/golangci-lint:$image_version \
      "${lint_cmd[@]}" 2>&1

# Runs a Nibiru local network. Ex: "just localnet --run --help". Optional flags: --no-build --log-level [debug|info]
localnet *PASS_FLAGS:
    bash cmd/nibid/localnet.sh --run {{ PASS_FLAGS }}

# Clears the logs directory
log-clear:
    #!/usr/bin/env bash
    if [ -d "logs" ] && [ "$(ls -A logs)" ]; then
      rm logs/* && echo "Logs cleared successfully."
    elif [ ! -d "logs" ]; then
      echo "Logs directory does not exist. Nothing to clear."
    else
      echo "Logs directory is already empty."
    fi

# Runs "just localnet" with logging (logs/localnet.txt)
log-localnet:
    #!/usr/bin/env bash
    mkdir -p logs
    just localnet 2>&1 | tee -a logs/localnet.txt

# Runs the EVM E2E test with logging (logs/e2e.txt)
log-e2e:
    #!/usr/bin/env bash
    set -euo pipefail
    just test-e2e 2>&1 | tee -a logs/e2e.txt

# Runs the EVM E2E tests
test-e2e:
    #!/usr/bin/env bash
    set -euo pipefail
    source contrib/bashlib.sh
    log_info "Make sure the localnet is running! (just localnet)"

    cd evm/e2e
    just test

# Test: "localnet.sh" script
test-localnet:
    #!/usr/bin/env bash
    set -euo pipefail
    source contrib/bashlib.sh
    log_info "Sleeping for 8 seconds to give network time to spin up and run a few blocks."
    set -x
    just install
    just localnet --no-build &
    sleep 8
    kill $(pgrep -x nibid) # Stops network running as background process.
    set +x
    log_success "Spun up localnet"

# Test: "chaosnet.sh" script
test-chaosnet:
    #!/usr/bin/env bash
    set -euo pipefail
    source contrib/bashlib.sh
    which_ok nibid
    bash contrib/scripts/chaosnet.sh 

# Stops any `nibid` processes, even if they're running in the background.
stop:
    kill $(pgrep -x nibid) || true

passkey-demo:
    #!/usr/bin/env bash
    contrib/scripts/passkey-demo.sh

# Runs golang formatter (gofumpt)
fmt:
    gofumpt -w evm x app gosdk eth

# Format and lint
tidy:
    #!/usr/bin/env bash
    set -euo pipefail

    find . -type f \( -name "go.mod" \) \
      | sed 's|/[^/]*$||' \
      | sort \
      | while IFS= read -r go_mod; do
          go_mod_dir=$(realpath "$go_mod")
          go_mod_dir_short="${go_mod_dir/#$HOME/\~}"
          printf "[%s/go.mod] go mod tidy\n" "$go_mod_dir_short"
          (cd "$go_mod_dir" && go mod tidy)
        done

    just proto gen
    just lint
    just fmt

test-release:
    #!/usr/bin/env bash
    set -euo pipefail
    make release-snapshot

release-publish:
    make release

[private]
_go-test-pkgs:
    #!/usr/bin/env bash
    set -euo pipefail
    go list ./... | grep -Ev '^github.com/NibiruChain/nibiru/v2/(api|lib)/'

# Run Go tests without cached test results
test:
    #!/usr/bin/env bash
    set -euo pipefail
    echo "Running: just test"
    just localnet-check
    GO_TEST_PKGS="$(just _go-test-pkgs)"
    echo "RUN: go test -count=1 \$GO_TEST_PKGS"
    go test -count=1 $GO_TEST_PKGS

# Run Go tests and allow cached test results
test-fast:
    #!/usr/bin/env bash
    set -euo pipefail
    echo "Running: just test-fast"
    GO_TEST_PKGS="$(just _go-test-pkgs)"
    echo "RUN: go test \$GO_TEST_PKGS # includes cache, skips localnet"
    go test $GO_TEST_PKGS

# Run the explicit IBC test suite and allow cached test results
test-ibc:
    #!/usr/bin/env bash
    set -euo pipefail
    echo "Running: just test-ibc"
    echo "RUN: go test ./lib/ibc-go/... # includes cache, skips localnet"
    go test ./lib/ibc-go/...

# Run Go tests without cached test results and generate coverage.out
test-cover:
    #!/usr/bin/env bash
    set -euo pipefail
    echo "Running: just test-cover"
    just localnet-check
    GO_TEST_PKGS="$(just _go-test-pkgs)"
    printf '%s\n' 'RUN: go test -tags=pebbledb -coverprofile=coverage.out -count=1 $GO_TEST_PKGS'
    go test \
      -tags=pebbledb \
      -coverprofile=coverage.out \
      -count=1 \
      $GO_TEST_PKGS
    go tool cover -func=coverage.out

# Alias for "test"
[private]
test-unit:
    #!/usr/bin/env bash
    set -euo pipefail
    just test-fast

# Report whether localnet-backed tests can reach a running nibid process
localnet-check:
    #!/usr/bin/env bash
    set -euo pipefail
    if pgrep -x nibid >/dev/null; then
      echo "✅ Localnet (nibid) is running. Tests with live chain can run"
    else
      echo "ERROR: not running nibid process. Start localnet before running Go tests." >&2
      exit 1
    fi

# Run commands from the wasmvm FFI subtree. Ex: `just wasmvm --list`.
wasmvm *args:
    cd lib/wasmvm && just {{ args }}

# Run commands from the sai-trading subtree. Ex: `just sai-trading test`.
sai-trading *args:
    cd lib/sai-trading && just {{ args }}

# Run commands from the lib/cosmos-sdk subtree.
csdk *args:
    cd lib/cosmos-sdk && just {{ args }}
