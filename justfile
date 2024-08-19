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

alias b := build

# Generate protobuf code (Golang) for Nibiru
proto-gen: 
  #!/usr/bin/env bash
  make proto-gen

alias proto := proto-gen

# Build protobuf types (Rust)
proto-rs:
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
