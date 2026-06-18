FROM golang:1.24 AS builder

WORKDIR /nibiru

RUN apt-get update && \
  apt-get install -y --no-install-recommends \
  musl-dev build-essential libgflags-dev libsnappy-dev zlib1g-dev libbz2-dev liblz4-dev libzstd-dev ca-certificates wget && \
  rm -rf /var/lib/apt/lists/*

ARG WASMVM_VERSION=1.5.9
ARG TARGETARCH

RUN mkdir -p /usr/lib && \
  if [ "${TARGETARCH}" = "arm64" ]; then \
    wget -q https://github.com/CosmWasm/wasmvm/releases/download/v${WASMVM_VERSION}/libwasmvm_muslc.aarch64.a -O /usr/lib/libwasmvm_muslc.a; \
  else \
    wget -q https://github.com/CosmWasm/wasmvm/releases/download/v${WASMVM_VERSION}/libwasmvm_muslc.x86_64.a -O /usr/lib/libwasmvm_muslc.a; \
  fi

COPY go.mod go.sum /nibiru/
COPY internal/ /nibiru/internal/
RUN go mod download

COPY sai-trading/go.mod sai-trading/go.sum /nibiru/sai-trading/
RUN cd sai-trading && go mod download

COPY . .

WORKDIR /nibiru/sai-trading

RUN --mount=type=cache,target=/root/.cache/go-build \
  --mount=type=cache,target=/go/pkg \
  --mount=type=cache,target=/nibiru/sai-trading/temp \
  CGO_ENABLED=1 \
  go build -tags "netgo muslc" \
  -ldflags '-linkmode=external -extldflags "-Wl,-z,muldefs -static -lstdc++ -lm -lz -lbz2 -lsnappy -llz4 -lzstd -lpthread"' \
  -o ./build/trader ./cmd/trader

FROM alpine:3.20

WORKDIR /root

RUN apk --no-cache add ca-certificates

COPY --from=builder /nibiru/sai-trading/build/trader /usr/local/bin/trader

COPY --from=builder /nibiru/sai-trading/networks.toml /root/networks.toml
COPY --from=builder /nibiru/sai-trading/auto-trader.localnet.json /root/
COPY --from=builder /nibiru/sai-trading/auto-trader.testnet.json /root/

ENTRYPOINT ["trader"]