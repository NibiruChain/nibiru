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
wget https://github.com/CosmWasm/wasmvm/releases/download/v${WASMVM_VERSION}/libwasmvm_muslc.x86_64.a -O /usr/lib/x86_64-linux-gnu/libwasmvm_muslc.a
wget https://github.com/CosmWasm/wasmvm/releases/download/v${WASMVM_VERSION}/libwasmvm_muslc.aarch64.a -O /usr/lib/aarch64-linux-gnu/libwasmvm_muslc.a
wget https://github.com/NibiruChain/gorocksdb/releases/download/v8.5.3/include.8.5.3.tar.gz -O /root/include.8.5.3.tar.gz
tar -xvf /root/include.8.5.3.tar.gz -C /usr/include/
wget https://github.com/NibiruChain/gorocksdb/releases/download/v8.5.3/librocksdb_8.5.3_linux_amd64.tar.gz -O /root/librocksdb_8.5.3_linux_amd64.tar.gz
tar -xvf /root/librocksdb_8.5.3_linux_amd64.tar.gz -C /usr/lib/x86_64-linux-gnu/
wget https://github.com/NibiruChain/gorocksdb/releases/download/v8.5.3/librocksdb_8.5.3_linux_arm64.tar.gz -O /root/librocksdb_8.5.3_linux_arm64.tar.gz
tar -xvf /root/librocksdb_8.5.3_linux_arm64.tar.gz -C /usr/lib/aarch64-linux-gnu/
