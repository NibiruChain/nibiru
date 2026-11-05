# Compiler versions

Nibiru pins Rust 1.95.0 for the vendored Wasmer 7.4.2 runtime and the
CosmWasm v1 VM/FFI integration. The root `rust-toolchain.toml`, Wasmer's
`rust-toolchain`, native library builders, and CI use this version.

The chain's Go version is defined in the root `go.mod`. Linux release
candidates use the pinned Go musl builder in
`contrib/docker/Dockerfile.release-musl` and rebuild the Rust static library
from the same checkout before linking `nibid` with static PIE.

Linux builds need a C/C++ compiler and libclang/LLVM development libraries.
Alpine additionally needs their static libraries for the Rust build tools.
`llvm-config` must be available on PATH. Shared Linux and Darwin cross builds
use the builders in `lib/wasmvm/builders`. Darwin artifacts target macOS 14.5
or newer. The cross builder pins the macOS 14.5 SDK archive and osxcross commit;
Wasmer 7's ARM64 compact-unwind registration uses symbols absent from SDK 11.3.

`contrib/scripts/build-wasmvm-source.sh` records the source commit and digest,
Rust version, runtime/FFI versions, target, and library checksum. It verifies
cached libraries before reuse. See `lib/wasmer/NIBIRU.md` for source provenance
and retained local fixes.
