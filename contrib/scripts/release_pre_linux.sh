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

flock -x /tmp/apt-lock -c "[ \"$(ls -A /var/lib/apt/lists)\" ] || apt-get update"
flock -x /tmp/rocksdb-linux-headers-lock -c "wget -c https://github.com/NibiruChain/gorocksdb/releases/download/v${ROCKSDB_VERSION}/include.${ROCKSDB_VERSION}.tar.gz -O /tmp/include.${ROCKSDB_VERSION}.tar.gz && [ ! -d /usr/include/rocksdb ] && tar -xvf /tmp/include.${ROCKSDB_VERSION}.tar.gz -C /usr/include/; echo 'rocksdb headers installed'"

if [ "$TARGET" == "linux_amd64_v1" ]; then
  apt-get -o DPkg::Lock::Timeout=60 install --no-install-recommends -y libzstd-dev:amd64 libsnappy-dev:amd64 liblz4-dev:amd64 libbz2-dev:amd64 zlib1g-dev:amd64

  wget -c https://github.com/CosmWasm/wasmvm/releases/download/v${WASMVM_VERSION}/libwasmvm_muslc.x86_64.a -O /tmp/libwasmvm_muslc.x86_64.a
  cp /tmp/libwasmvm_muslc.x86_64.a /usr/lib/x86_64-linux-gnu/libwasmvm_muslc.a

  wget -c https://github.com/NibiruChain/gorocksdb/releases/download/v${ROCKSDB_VERSION}/librocksdb_${ROCKSDB_VERSION}_linux_amd64.tar.gz -O /tmp/librocksdb_${ROCKSDB_VERSION}_linux_amd64.tar.gz
  tar -xvf /tmp/librocksdb_${ROCKSDB_VERSION}_linux_amd64.tar.gz -C /usr/lib/x86_64-linux-gnu/
else
  apt-get -o DPkg::Lock::Timeout=60 install --no-install-recommends -y libzstd-dev:arm64 libsnappy-dev:arm64 liblz4-dev:arm64 libbz2-dev:arm64 zlib1g-dev:arm64

  wget -c https://github.com/CosmWasm/wasmvm/releases/download/v${WASMVM_VERSION}/libwasmvm_muslc.aarch64.a -O /tmp/libwasmvm_muslc.aarch64.a
  cp /tmp/libwasmvm_muslc.aarch64.a /usr/lib/aarch64-linux-gnu/libwasmvm_muslc.a

  wget -c https://github.com/NibiruChain/gorocksdb/releases/download/v${ROCKSDB_VERSION}/librocksdb_${ROCKSDB_VERSION}_linux_arm64.tar.gz -O /tmp/librocksdb_${ROCKSDB_VERSION}_linux_arm64.tar.gz
  tar -xvf /tmp/librocksdb_${ROCKSDB_VERSION}_linux_arm64.tar.gz -C /usr/lib/aarch64-linux-gnu/
fi
