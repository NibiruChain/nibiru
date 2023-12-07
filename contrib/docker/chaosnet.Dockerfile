FROM golang:1.21 AS builder

WORKDIR /nibiru

COPY go.sum go.mod ./
RUN go mod download
COPY . .

RUN --mount=type=cache,target=/root/.cache/go-build \
  --mount=type=cache,target=/go/pkg \
  make build

FROM alpine:latest
WORKDIR /root

RUN apk --no-cache add \
  ca-certificates \
  build-base \
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
