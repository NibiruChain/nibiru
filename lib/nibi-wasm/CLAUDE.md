# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

This is a Rust workspace containing CosmWasm smart contracts for the Nibiru blockchain. The codebase uses Cargo workspaces to manage multiple contracts and utility packages.

## Essential Commands

### Building and Testing
```bash
# Install development tools
just install

# Compile all contracts to WASM
just wasm-all

# Run tests for all packages
just test

# Run tests for a specific package
just test <package-name>

# Generate test coverage report
just test-coverage

# Run a single test function
cargo test --package <package-name> <test-function-name>
```

### Code Quality
```bash
# Format code
just fmt

# Run linter with fixes
just clippy

# Full code quality check (format, lint, wasm-check, test)
just tidy

# Validate WASM binaries for blockchain deployment
just wasm-check
```

### Development Workflow
```bash
# Clean build artifacts
just clean

# Build all Rust code
just build

# Generate JSON schemas for contract interfaces
just gen-schema
```

## Architecture

### Project Structure
- **contracts/** - Smart contracts implementing specific functionality (incentives, lockup, vesting, etc.)
- **nibiru-std/** - Nibiru standard library providing proto types and bindings for Stargate messages
- **packages/** - Utility packages including:
  - `cw-address-like` - Address-like helper traits for CosmWasm types
  - `easy-addr` - Address construction helpers for tests
  - `nibiru-ownable` - Ownership patterns for contracts
- **artifacts/** - Compiled WASM binaries
- **schema/** - JSON schemas for contract interfaces

### Key Patterns
1. **CosmWasm Standards**: All contracts follow CosmWasm patterns with InstantiateMsg, ExecuteMsg, and QueryMsg
2. **Stargate Messages**: Use `nibiru-std` for native Nibiru chain interactions via Stargate messages
3. **Testing**: Integration tests use `cw-multi-test` framework, unit tests embedded in source files
4. **Workspace Dependencies**: Shared dependencies defined in root Cargo.toml

### Contract Development
When creating new contracts:
1. Follow existing contract structure (see contracts/lockup or contracts/incentives)
2. Use `nibiru-std` for Nibiru-specific functionality
3. Include comprehensive integration tests using `cw-multi-test`
4. Generate schemas with `cargo schema` in the contract directory

### Testing Approach
- Unit tests: In-file with `#[cfg(test)]` modules
- Integration tests: Separate `testing.rs` or `integration_tests.rs` files
- Use `cw-multi-test::App` for simulating blockchain environment
- Test coverage tracked with `cargo-llvm-cov`