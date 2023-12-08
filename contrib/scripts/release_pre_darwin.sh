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

# libwasmvm
WASMVM_VERSION=$(go list -m github.com/CosmWasm/wasmvm | awk '{sub(/^v/, "", $2); print $2}')
wget https://github.com/CosmWasm/wasmvm/releases/download/v${WASMVM_VERSION}/libwasmvmstatic_darwin.a -O /usr/local/osxcross/SDK/MacOSX12.0.sdk/usr/lib/libwasmvmstatic_darwin.a

# librocksdb
ROCKSDB_VERSION=8.8.1
wget https://github.com/NibiruChain/gorocksdb/releases/download/v${ROCKSDB_VERSION}/include.${ROCKSDB_VERSION}.tar.gz -O /root/include.${ROCKSDB_VERSION}.tar.gz
tar -xf /root/include.${ROCKSDB_VERSION}.tar.gz -C /usr/local/osxcross/SDK/MacOSX12.0.sdk/usr/include/
wget https://github.com/NibiruChain/gorocksdb/releases/download/v${ROCKSDB_VERSION}/librocksdb_${ROCKSDB_VERSION}_darwin_all.tar.gz -O /root/librocksdb_${ROCKSDB_VERSION}_darwin_all.tar.gz
tar -xf /root/librocksdb_${ROCKSDB_VERSION}_darwin_all.tar.gz -C /usr/local/osxcross/SDK/MacOSX12.0.sdk/usr/lib/
