# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Nibiru Chain is a breakthrough L1 blockchain and smart contract ecosystem providing superior throughput, reduced latency, and improved security. It operates two virtual machines in parallel:
- **NibiruEVM**: Ethereum Virtual Machine for Ethereum compatibility
- **Wasm**: For CosmWasm smart contracts

## Repository Structure

### Core Directories
- `/x/` - Custom blockchain modules (oracle, evm, devgas, epochs, inflation, tokenfactory, sudo)
- `/app/` - Core blockchain application logic and module integration
- `/eth/` - Ethereum-specific implementations (accounts, crypto, RPC)
- `/proto/` - Protocol Buffer definitions for blockchain messages
- `/cmd/nibid/` - CLI binary for interacting with the blockchain
- `/evm-core-ts/` - TypeScript library for EVM interactions
- `/gosdk/` - Go SDK for programmatic blockchain interactions

### Key Modules
- **evm**: Ethereum Virtual Machine implementation
- **oracle**: Decentralized price oracle system  
- **devgas**: Revenue sharing for smart contract developers
- **epochs**: Time-based hook system for periodic tasks
- **inflation**: Tokenomics implementation
- **tokenfactory**: Token creation and management

## Essential Commands

### Build & Install
```bash
# Install the nibid binary
make install

# Build the project
make build

# Alternative with just
just build
just install
```

### Local Development
```bash
# Run a local blockchain node
make localnet

# Alternative single-node testnet
just init-local-testnet [moniker]
just start
```

### Testing
```bash
# Run all unit tests
make test-unit

# Run specific package tests
go test ./x/evm/...

# Run with coverage
make test-coverage

# Run integration tests
make test-coverage-integration

# Run EVM E2E tests
cd evm-e2e && npm test
```

### Code Quality
```bash
# Format code
make format

# Run linter
make lint

# Fix linting issues
make lint-fix

# Generate protobuf code
make proto-gen
```

### Development Utilities
```bash
# Install development tools
make tools-clean

# Run simulation tests
make sim-full

# Generate documentation
make docs-generate

# Check for breaking changes
make proto-break-check
```

## System Requirements

- Go version 1.22.2 or higher
- Node.js 20+ and npm 10+ (for TypeScript/EVM tests)
- Make and Git
- For validators: 8-core x64 CPU, 64GB RAM, 1TB+ SSD

## Development Workflow

1. **Setup Environment**
   ```bash
   git clone https://github.com/NibiruChain/nibiru
   cd nibiru
   make install
   ```

2. **Run Local Network**
   ```bash
   make localnet
   # or
   just init-local-testnet test-moniker
   just start
   ```

3. **Make Changes**
   - Core logic: `/x/` modules
   - Application: `/app/`
   - Tests: `*_test.go` files

4. **Test Changes**
   ```bash
   # Test specific module
   go test ./x/oracle/...
   
   # Run integration tests
   make test-coverage-integration
   ```

5. **Lint & Format**
   ```bash
   make lint-fix
   make format
   ```

## Technical Details

### Build System
- Primary: Makefile with includes from `/contrib/make/`
- Alternative: justfile for common tasks
- Protocol Buffers: buf for proto compilation

### Dependencies
- Cosmos SDK v0.47.11
- CometBFT/Tendermint consensus
- go-ethereum fork for EVM compatibility
- CosmWasm for WASM smart contracts

### Testing Framework
- Go: Built-in testing with testify assertions
- TypeScript: Jest for EVM E2E tests
- Simulation: Custom simulation framework

## Troubleshooting

### Common Issues
1. **Import errors**: Run `go mod tidy` and `go mod download`
2. **Proto errors**: Run `make proto-gen`
3. **Build failures**: Ensure Go 1.22.2+ is installed
4. **Test failures**: Check if local node is running for integration tests

### Local Network Ports
- RPC: http://localhost:26657
- gRPC: localhost:9090
- EVM RPC: http://localhost:8545
- API: http://localhost:1317

## Documentation Resources

- Main docs: https://docs.nibiru.fi
- Contracts: https://github.com/NibiruChain/contracts
- TypeScript SDK: https://www.npmjs.com/package/@nibiruchain/ts-sdk
- Oracle docs: See `/x/oracle/README.md`

## Community

- Discord: https://discord.gg/nibiru
- Twitter: https://twitter.com/NibiruChain
- Telegram: https://t.me/nibiruchainofficial