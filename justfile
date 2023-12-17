# Displays available recipes by running `just -l`.
setup:
  #!/usr/bin/env bash
  just -l

# Locally install the `nibid` binary and build if needed.
install: 
  make install

# Build the `nibid` binary.
build: 
  make build

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

# Runs golang formatter (gofumpt)
fmt:
  gofumpt -w x app

# Format and lint
tidy: 
  #!/usr/bin/env bash
  go mod tidy
  just proto-gen
  just lint
  just fmt

release-test:
  make release-snapshot

release-publish:
  make release
