#!/usr/env/bin bash
#
# This runs in the "hooks.pre" block of .goreleaser.yml.
# We do this because it enables us to dynamically set the $WASMVM_VERSION based
# on go.mod.
# 
# It's intended to be used with:
# ```bash
# make release-snapshot
# ```
set -e

WASMVM_VERSION=$(go list -m github.com/CosmWasm/wasmvm | awk '{sub(/^v/, "", $2); print $2}')
ROCKSDB_VERSION=8.9.1

flock -x /tmp/wasmvm-lock -c "wget -c https://github.com/CosmWasm/wasmvm/releases/download/v${WASMVM_VERSION}/libwasmvmstatic_darwin.a -O /tmp/libwasmvmstatic_darwin.a && [ ! -f /usr/local/osxcross/SDK/MacOSX12.0.sdk/usr/lib/libwasmvmstatic_darwin.a ] && cp /tmp/libwasmvmstatic_darwin.a /usr/local/osxcross/SDK/MacOSX12.0.sdk/usr/lib/libwasmvmstatic_darwin.a; echo 'libwasmvm installed'"
flock -x /tmp/rocksdb-darwin-headers-lock -c "wget -c https://github.com/NibiruChain/gorocksdb/releases/download/v${ROCKSDB_VERSION}/include.${ROCKSDB_VERSION}.tar.gz -O /tmp/include.${ROCKSDB_VERSION}.tar.gz && [ ! -d /usr/local/osxcross/SDK/MacOSX12.0.sdk/usr/include/rocksdb ] && tar -xvf /tmp/include.${ROCKSDB_VERSION}.tar.gz -C /usr/local/osxcross/SDK/MacOSX12.0.sdk/usr/include/; echo 'rocksdb headers installed'"
flock -x /tmp/rocksdb-lib-lock -c "wget -c https://github.com/NibiruChain/gorocksdb/releases/download/v${ROCKSDB_VERSION}/librocksdb_${ROCKSDB_VERSION}_darwin_all.tar.gz -O /tmp/librocksdb_${ROCKSDB_VERSION}_darwin_all.tar.gz && [ ! -f /usr/local/osxcross/SDK/MacOSX12.0.sdk/usr/lib/librocksdb.a ] && tar -xvf /tmp/librocksdb_${ROCKSDB_VERSION}_darwin_all.tar.gz -C /usr/local/osxcross/SDK/MacOSX12.0.sdk/usr/lib/; echo 'librocksdb installed'"
