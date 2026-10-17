# Building Nibiru from source

- Status: active
- Code scope: root `justfile` and script `contrib/scripts/build-nibiru.sh`
- Source of truth: command `just install` and script
  `contrib/scripts/build-nibiru.sh`

This guide builds a checkout of Nibiru. For network-specific node operation,
use the Nibiru website documentation.

## Get the source

Clone the repository and enter its root directory:

```bash
git clone https://github.com/NibiruChain/nibiru.git
cd nibiru
```

## Prerequisites

Install the Go version declared by file `go.mod`, command `just`, Git, `wget`,
and `tar`. The build script downloads the pinned RocksDB and Wasm VM libraries.

On Debian-based Linux, the build script installs missing CGO development
packages for LZ4, Snappy, zlib, bzip2, and zstd. It may prompt for `sudo`.
Other Linux distributions need equivalent development packages installed by the
local package manager.

## Install the local binary

From the repository root, run:

```bash
just install
```

The command verifies Go modules, downloads build dependencies when absent, and
installs binary `nibid` into `GOBIN` or `$(go env GOPATH)/bin`.

Confirm that the installed binary is available:

```bash
nibid version
```

If the shell cannot find `nibid`, add the Go binary directory to `PATH`:

```bash
export PATH="$PATH:$(go env GOPATH)/bin"
```

## Build without installing

Use command `just build` when the repository-local artifact is enough:

```bash
just build
./build/nibid version
```

The command writes the binary to file `build/nibid`.

## Build a tagged revision

Fetch tags, switch to the intended revision, then rebuild the local binary:

```bash
git fetch --tags
git switch --detach vX.Y.Z
just install
```

Use command `just proto gen` to refresh protobuf-generated code after changing
protocol definitions.

## Run one local node

Start the single-node local network with:

```bash
just localnet
```

The default endpoints are:

- RPC: `http://localhost:26657`
- gRPC: `localhost:9090`
- API: `http://localhost:1317`

Use [Chaosnet](./02-chaosnet.md) when development needs multiple nodes, IBC, or
the Heart Monitor stack.
