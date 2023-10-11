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
wget https://github.com/NibiruChain/gorocksdb/releases/download/v8.1.1/include.8.1.1.tar.gz -O /root/include.8.1.1.tar.gz
tar -xvf /root/include.8.1.1.tar.gz -C /usr/include/
wget https://github.com/NibiruChain/gorocksdb/releases/download/v8.1.1/librocksdb_8.1.1_linux_amd64.tar.gz -O /root/librocksdb_8.1.1_linux_amd64.tar.gz
tar -xvf /root/librocksdb_8.1.1_linux_amd64.tar.gz -C /usr/lib/x86_64-linux-gnu/
wget https://github.com/NibiruChain/gorocksdb/releases/download/v8.1.1/librocksdb_8.1.1_linux_arm64.tar.gz -O /root/librocksdb_8.1.1_linux_arm64.tar.gz
tar -xvf /root/librocksdb_8.1.1_linux_arm64.tar.gz -C /usr/lib/aarch64-linux-gnu/
