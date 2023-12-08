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

# libwasmvm_muslc
WASMVM_VERSION=$(go list -m github.com/CosmWasm/wasmvm | awk '{sub(/^v/, "", $2); print $2}')

wget https://github.com/CosmWasm/wasmvm/releases/download/v${WASMVM_VERSION}/libwasmvm_muslc.x86_64.a -O /usr/lib/x86_64-linux-gnu/libwasmvm_muslc.a
wget https://github.com/CosmWasm/wasmvm/releases/download/v${WASMVM_VERSION}/libwasmvm_muslc.aarch64.a -O /usr/lib/aarch64-linux-gnu/libwasmvm_muslc.a

# librocksdb
ROCKSDB_VERSION=8.8.1
wget https://github.com/NibiruChain/gorocksdb/releases/download/v${ROCKSDB_VERSION}/include.${ROCKSDB_VERSION}.tar.gz -O /root/include.${ROCKSDB_VERSION}.tar.gz
tar -xvf /root/include.${ROCKSDB_VERSION}.tar.gz -C /usr/include/

wget https://github.com/NibiruChain/gorocksdb/releases/download/v${ROCKSDB_VERSION}/librocksdb_${ROCKSDB_VERSION}_linux_amd64.tar.gz -O /root/librocksdb_${ROCKSDB_VERSION}_linux_amd64.tar.gz
tar -xvf /root/librocksdb_${ROCKSDB_VERSION}_linux_amd64.tar.gz -C /usr/lib/x86_64-linux-gnu/

wget https://github.com/NibiruChain/gorocksdb/releases/download/v${ROCKSDB_VERSION}/librocksdb_${ROCKSDB_VERSION}_linux_arm64.tar.gz -O /root/librocksdb_${ROCKSDB_VERSION}_linux_arm64.tar.gz
tar -xvf /root/librocksdb_${ROCKSDB_VERSION}_linux_arm64.tar.gz -C /usr/lib/aarch64-linux-gnu/

# compression libraries
apt-get update
apt-get install -y git make build-essential libgflags-dev libsnappy-dev zlib1g-dev libbz2-dev liblz4-dev libzstd-dev