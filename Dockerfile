FROM golang:1.19 AS builder

WORKDIR /nibiru

ENV BUILD_TAGS="muslc ledger" \
  CGO_ENABLED=1 \
  LDFLAGS='-linkmode=external -extldflags "-Wl,-z,muldefs -static -lrocksdb -lstdc++ -lm -lz -lbz2 -lsnappy -llz4 -lzstd -lpthread"'

RUN apt-get update && \
  apt-get install -y \
  musl-dev build-essential libgflags-dev libsnappy-dev zlib1g-dev libbz2-dev liblz4-dev libzstd-dev && \
  rm -rf /var/lib/apt/lists/*

ARG TARGETARCH
RUN if [ "${TARGETARCH}" = "arm64" ]; then \
  wget https://github.com/CosmWasm/wasmvm/releases/download/v1.2.0/libwasmvm_muslc.aarch64.a -O /usr/lib/aarch64-linux-gnu/libwasmvm_muslc.a; \
  wget https://github.com/NibiruChain/gorocksdb/releases/download/v8.1.1/librocksdb_8.1.1_linux_arm64.tar.gz -O /usr/lib/aarch64-linux-gnu/librocksdb_8.1.1_linux_arm64.tar.gz; \
  tar -xf /usr/lib/aarch64-linux-gnu/librocksdb_8.1.1_linux_arm64.tar.gz -C /usr/lib/aarch64-linux-gnu/; \
  wget https://github.com/NibiruChain/gorocksdb/releases/download/v8.1.1/include.8.1.1.tar.gz -O /usr/include/rocksdb.tar.gz; \
  tar -xf /usr/include/rocksdb.tar.gz -C /usr/include/; \
  else \
  wget https://github.com/CosmWasm/wasmvm/releases/download/v1.2.0/libwasmvm_muslc.x86_64.a -O /usr/lib/x86_64-linux-gnu/libwasmvm_muslc.a; \
  wget https://github.com/NibiruChain/gorocksdb/releases/download/v8.1.1/librocksdb_8.1.1_linux_amd64.tar.gz -O /usr/lib/x86_64-linux-gnu/librocksdb_8.1.1_linux_amd64.tar.gz; \
  tar -xf /usr/lib/x86_64-linux-gnu/librocksdb_8.1.1_linux_amd64.tar.gz -C /usr/lib/x86_64-linux-gnu/; \
  wget https://github.com/NibiruChain/gorocksdb/releases/download/v8.1.1/include.8.1.1.tar.gz -O /usr/include/rocksdb.tar.gz; \
  tar -xf /usr/include/rocksdb.tar.gz -C /usr/include/; \
  fi

COPY go.sum go.mod ./
RUN go mod download
COPY . .

RUN --mount=type=cache,target=/root/.cache/go-build \
  --mount=type=cache,target=/go/pkg \
  make build

FROM alpine:latest
WORKDIR /root
RUN apk --no-cache add ca-certificates
COPY --from=builder /nibiru/build/nibid /usr/local/bin/nibid

ENTRYPOINT ["nibid"]
CMD [ "start" ]
