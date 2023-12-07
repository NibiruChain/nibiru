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
wget https://github.com/CosmWasm/wasmvm/releases/download/v${WASMVM_VERSION}/libwasmvmstatic_darwin.a -O /usr/local/osxcross/SDK/MacOSX12.0.sdk/usr/lib/libwasmvmstatic_darwin.a
wget https://github.com/NibiruChain/gorocksdb/releases/download/v8.5.3/include.8.5.3.tar.gz -O /root/include.8.5.3.tar.gz
tar -xf /root/include.8.5.3.tar.gz -C /usr/local/osxcross/SDK/MacOSX12.0.sdk/usr/include/
wget https://github.com/NibiruChain/gorocksdb/releases/download/v8.5.3/librocksdb_8.5.3_darwin_all.tar.gz -O /root/librocksdb_8.5.3_darwin_all.tar.gz
tar -xf /root/librocksdb_8.5.3_darwin_all.tar.gz -C /usr/local/osxcross/SDK/MacOSX12.0.sdk/usr/lib/
