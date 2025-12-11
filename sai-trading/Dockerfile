FROM golang:1.24 AS builder

WORKDIR /sai-trading
ARG WASMVM_VERSION=1.5.8


RUN apt-get update && \
  apt-get install -y --no-install-recommends \
  musl-dev build-essential libgflags-dev libsnappy-dev zlib1g-dev libbz2-dev liblz4-dev libzstd-dev wget ca-certificates && \
  rm -rf /var/lib/apt/lists/*

ARG TARGETARCH
RUN mkdir -p /usr/lib && \
  if [ "${TARGETARCH}" = "arm64" ]; then \
    wget --progress=dot:giga --timeout=30 --tries=3 \
      https://github.com/CosmWasm/wasmvm/releases/download/v${WASMVM_VERSION}/libwasmvm_muslc.aarch64.a \
      -O /usr/lib/libwasmvm_muslc.a; \
  else \
    wget --progress=dot:giga --timeout=30 --tries=3 \
      https://github.com/CosmWasm/wasmvm/releases/download/v${WASMVM_VERSION}/libwasmvm_muslc.x86_64.a \
      -O /usr/lib/libwasmvm_muslc.a; \
  fi
  
COPY go.sum go.mod ./
RUN go mod download

COPY . .

RUN --mount=type=cache,target=/root/.cache/go-build \
  --mount=type=cache,target=/go/pkg \
  --mount=type=cache,target=/sai-trading/temp \
  CGO_ENABLED=1 \
  go build -tags "netgo muslc" \
  -ldflags '-linkmode=external -extldflags "-Wl,-z,muldefs -static -lstdc++ -lm -lz -lbz2 -lsnappy -llz4 -lzstd -lpthread"' \
  -o ./build/trader ./cmd/trader

FROM alpine:latest

WORKDIR /root

RUN apk --no-cache add ca-certificates

COPY --from=builder /sai-trading/build/trader /usr/local/bin/trader

COPY --from=builder /sai-trading/networks.toml /root/networks.toml
COPY --from=builder /sai-trading/auto-trader.localnet.json /root/
COPY --from=builder /sai-trading/auto-trader.testnet.json /root/

ENTRYPOINT ["trader"]