# Displays available recipes by running `just -l`.
setup:
  #!/usr/bin/env bash
  just -l

# Locally install the `nibid` binary and build if needed.
install: 
  go mod tidy
  make install

install-clean:
  rm -rf temp
  just install

# Build the `nibid` binary.
build: 
  make build

# Cleans the Go cache, modcache, and testcashe
clean-cache:
  go clean -cache -testcache -modcache

# Generate protobuf-based types in Golang
proto-gen: 
  #!/usr/bin/env bash
  make proto-gen

# Generate Solidity artifacts for x/evm/embeds
gen-embeds:
  #!/usr/bin/env bash
  source contrib/bashlib.sh

  embeds_dir="x/evm/embeds"
  log_info "Begin to compile Solidity in $embeds_dir"
  which_ok npm
  log_info "Using system node version: $(npm exec -- node -v)"

  cd "$embeds_dir" || (log_error "path $embeds_dir not found" && exit)
  npx hardhat compile
  log_success "Compiled Solidity in $embeds_dir"

  go run "gen-abi/main.go"
  log_success "Saved ABI JSON files to $embeds_dir/abi for npm publishing"

alias gen-proto := proto-gen

# Generate the Nibiru Token Registry files
gen-token-registry:
  go run token-registry/main/main.go

# Generate protobuf-based types in Rust
gen-proto-rs:
  bash proto/buf.gen.rs.sh

lint: 
  #!/usr/bin/env bash
  source contrib/bashlib.sh
  if ! which_ok golangci-lint; then
    log_info "Installing golangci-lint"
    curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v1.55.2
  fi

  golangci-lint run --allow-parallel-runners --fix


# Runs a Nibiru local network. Ex: "just localnet", "just localnet --features featureA"
localnet *PASS_FLAGS:
  make localnet FLAGS="{{PASS_FLAGS}}"

# Runs a Nibiru local network without building and installing. "just localnet --no-build"
localnet-fast:
  make localnet FLAGS="--no-build"

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
  just test-e2e 2>&1 | tee -a logs/e2e.txt

# Runs the EVM E2E tests
test-e2e:
  #!/usr/bin/env bash
  source contrib/bashlib.sh
  log_info "Make sure the localnet is running! (just localnet)"

  cd evm-e2e
  just test


# Test: "localnet.sh" script
test-localnet:
  #!/usr/bin/env bash
  source contrib/bashlib.sh
  just install
  bash contrib/scripts/localnet.sh &
  log_info "Sleeping for 6 seconds to give network time to spin up and run a few blocks."
  sleep 6 
  kill $(pgrep -x nibid) # Stops network running as background process.
  log_success "Spun up localnet"

# Test: "chaosnet.sh" script
test-chaosnet:
  #!/usr/bin/env bash
  source contrib/bashlib.sh
  which_ok nibid
  bash contrib/scripts/chaosnet.sh 

# Stops any `nibid` processes, even if they're running in the background.
stop: 
  kill $(pgrep -x nibid) || true

# Runs golang formatter (gofumpt)
fmt:
  gofumpt -w x app gosdk eth

# Format and lint
tidy: 
  #!/usr/bin/env bash
  go mod tidy
  just proto-gen
  just lint
  just fmt

test-release:
  make release-snapshot

release-publish:
  make release

# Run Go tests (short mode)
test-unit:
  go test ./... -short

# Run Go tests (short mode) + coverage
test-coverage-unit:
  make test-coverage-unit

# Run Go tests, including live network tests + coverage
test-coverage-integration:
  make test-coverage-integration
