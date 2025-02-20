FROM golang:1.21 AS builder

WORKDIR /nibiru

# install OS dependencies
RUN apt-get update && apt-get install -y --no-install-recommends \
    liblz4-dev libsnappy-dev zlib1g-dev libbz2-dev libzstd-dev

# install Go dependencies
COPY go.sum go.mod ./
RUN go mod download

# build nibid
COPY . .
RUN --mount=type=cache,target=/root/.cache/go-build \
  --mount=type=cache,target=/go/pkg \
  --mount=type=cache,target=/nibiru/temp \
  make build

FROM alpine:latest
WORKDIR /root

RUN apk --no-cache add \
  ca-certificates \
  build-base \
  bash \
  curl \
  jq

COPY --from=builder /nibiru/build/nibid /usr/local/bin/nibid

COPY ./contrib/scripts/chaosnet.sh ./

RUN chmod +x ./chaosnet.sh
ARG MNEMONIC
ARG CHAIN_ID
ARG RPC_PORT
ARG GRPC_PORT
ARG LCD_PORT
RUN MNEMONIC=${MNEMONIC} CHAIN_ID=${CHAIN_ID} RPC_PORT=${RPC_PORT} GRPC_PORT=${GRPC_PORT} LCD_PORT=${LCD_PORT} ./chaosnet.sh

ENTRYPOINT ["nibid"]
CMD [ "start" ]
