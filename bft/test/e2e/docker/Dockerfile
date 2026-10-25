# Nibiru BFT e2e node image: cometbft/e2e-node:local-version.
#
# Build from the repo root, since bft/ belongs to the root Go module:
#
#   just bft e2e-image
#   docker build -f bft/test/e2e/docker/Dockerfile \
#     -t cometbft/e2e-node:local-version .
#
# Dockerfile.dockerignore next to this file replaces the root .dockerignore
# for this build and sends only go.mod, go.sum, and bft/.
#
# The image holds two binaries:
#   /usr/bin/app         bft/test/e2e/node, Nibiru BFT and the e2e key/value
#                        app in one process (abci_protocol = "builtin")
#   /usr/bin/nibiru-bft  bft/cmd/nibiru-bft, used by the entrypoint when the
#                        app runs as a separate process (tcp, unix, grpc)
#
# Both stages use Debian bookworm, so the binaries built against the
# RocksDB and LevelDB C libraries in the build stage find the same library
# versions at runtime. The runtime stage drops the Go toolchain and module
# cache (about 2 GB).

# Keep in sync with the go directive in the root go.mod.
ARG GO_VERSION=1.27

# ----- Stage "build" ----------
FROM golang:${GO_VERSION}-bookworm AS build

SHELL ["/bin/bash", "-o", "pipefail", "-c"]

# LevelDB comes from Debian (cleveldb backend). The compression libraries are
# RocksDB dependencies.
RUN apt-get update -qq \
  && apt-get install -y -qq --no-install-recommends \
    build-essential libbz2-dev libgflags-dev liblz4-dev libsnappy-dev \
    libzstd-dev zlib1g-dev libleveldb-dev \
  && rm -rf /var/lib/apt/lists/*

# The root go.mod replaces github.com/linxGnu/grocksdb with v1.8.12, which
# binds RocksDB 8.9.1 (see its build.sh). Debian ships no 8.x package, so we
# build the shared library here. Change ROCKSDB_VERSION only together with
# that replace directive.
ARG ROCKSDB_VERSION=8.9.1
RUN cd /tmp \
  && curl -fsSL -o rocksdb.tar.gz \
    "https://github.com/facebook/rocksdb/archive/refs/tags/v${ROCKSDB_VERSION}.tar.gz" \
  && tar -xzf rocksdb.tar.gz \
  && cd "rocksdb-${ROCKSDB_VERSION}" \
  && DEBUG_LEVEL=0 PORTABLE=1 make -j"$(nproc)" shared_lib \
  && make install-shared \
  && strip --strip-unneeded /usr/local/lib/librocksdb.so.* \
  && ldconfig \
  && cd /tmp \
  && rm -rf rocksdb.tar.gz "rocksdb-${ROCKSDB_VERSION}"

WORKDIR /src/nibiru

# Fetch dependencies separately (for layer caching).
COPY go.mod go.sum ./
RUN go mod download

COPY bft ./bft

# Build tags for the cometbft-db backends that ci.toml mixes. goleveldb and
# memdb need no tag.
ARG BUILD_TAGS=badgerdb,boltdb,cleveldb,rocksdb
# Passed by `just bft e2e-image`. The build context has no .git directory.
ARG GIT_COMMIT=""
RUN ldflags="-X github.com/NibiruChain/nibiru/v2/bft/version.TMGitCommitHash=${GIT_COMMIT}" \
  && go build -mod=readonly -trimpath -tags "${BUILD_TAGS}" \
    -ldflags "${ldflags}" -o /usr/bin/nibiru-bft ./bft/cmd/nibiru-bft \
  && go build -mod=readonly -trimpath -tags "${BUILD_TAGS}" \
    -ldflags "${ldflags}" -o /usr/bin/app ./bft/test/e2e/node

# ----- Stage "runtime" ----------
FROM debian:bookworm-slim

SHELL ["/bin/bash", "-o", "pipefail", "-c"]

RUN apt-get update -qq \
  && apt-get install -y -qq --no-install-recommends \
    ca-certificates libbz2-1.0 libgflags2.2 libleveldb1d liblz4-1 \
    libsnappy1v5 libstdc++6 libzstd1 zlib1g \
  && rm -rf /var/lib/apt/lists/*

# Copy the real file only. ldconfig creates the librocksdb.so.8.9 soname
# link that the binaries load.
ARG ROCKSDB_VERSION=8.9.1
COPY --from=build /usr/local/lib/librocksdb.so.${ROCKSDB_VERSION} /usr/local/lib/
RUN ldconfig

COPY --from=build /usr/bin/nibiru-bft /usr/bin/app /usr/bin/
COPY bft/test/e2e/docker/entrypoint* /usr/bin/

# Set up runtime directory.
WORKDIR /cometbft
VOLUME /cometbft
ENV CMTHOME=/cometbft

EXPOSE 26656 26657 26660 6060
ENTRYPOINT ["/usr/bin/entrypoint"]
CMD ["node"]
STOPSIGNAL SIGTERM
