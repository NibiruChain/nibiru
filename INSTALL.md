# Nibiru Installation Guide

This guide explains how to install the Nibiru binary, `nibid`, from this repository.

## 1) Prerequisites

- Go (see `go.mod` for the required version): [Install Go](https://go.dev/doc/install)
- command `just`: [Install just](https://just.systems/man/en/packages.html)
- Common build tooling (`git`, C compiler toolchain, `curl`, `jq`)

Example packages on Ubuntu:

```bash
sudo apt update
sudo apt install --yes git build-essential curl jq
```

## 2) Clone the Repository

```bash
git clone https://github.com/NibiruChain/nibiru
cd nibiru
```

## 3) Build and Install `nibid`

From the repository root, run:

```bash
just install
```

This builds and installs the `nibid` binary.

## 4) Start a Local Network

```bash
just localnet
```

Open another terminal if you want to query the node while the localnet is running.

## Upgrade or Switch Versions

To switch to a specific tagged release:

```bash
git fetch --tags
git checkout <tag>
just install
```

For network-specific instructions:

- Testnet: [Nibiru Networks - Testnet](https://github.com/NibiruChain/Networks/tree/main/Testnet)
- Mainnet: [Nibiru Networks - Mainnet](https://github.com/NibiruChain/Networks/tree/main/Mainnet)

## Troubleshooting

### `nibid` Command Not Found

If `nibid` is not on your `PATH`, add your Go bin directory:

```bash
export PATH="$PATH:$(go env GOPATH)/bin"
```

Then reload your shell config.

### Binary Does Not Reflect Local Code Changes

Rebuild and reinstall from the repository root:

```bash
just install
```

### Updating Protobuf-Generated Code

Use command `just proto gen` from the repository root.

### macOS Notes

If a local command used by your shell scripts is missing, install it with Homebrew (for example, command `brew install wget`).
