# ---------- Build Stage ----------
FROM golang:1.24 AS build-base

WORKDIR /nibiru

RUN apt-get update && apt-get install -y --no-install-recommends \
    liblz4-dev libsnappy-dev zlib1g-dev libbz2-dev libzstd-dev

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg \
    --mount=type=cache,target=/nibiru/temp \
    make build


# ---------- Chaosnet Image ----------
FROM alpine:latest AS chaosnet

WORKDIR /root

RUN apk --no-cache add \
  ca-certificates \
  build-base \
  bash \
  curl \
  jq

COPY --from=build-base /nibiru/build/nibid /usr/local/bin/nibid
COPY ./contrib/scripts/chaosnet.sh ./

ARG MNEMONIC
ARG CHAIN_ID
ARG RPC_PORT
ARG GRPC_PORT
ARG LCD_PORT

RUN chmod +x ./chaosnet.sh && \
    MNEMONIC=${MNEMONIC} \
    CHAIN_ID=${CHAIN_ID} \
    RPC_PORT=${RPC_PORT} \
    GRPC_PORT=${GRPC_PORT} \
    LCD_PORT=${LCD_PORT} \
    ./chaosnet.sh

ENTRYPOINT ["nibid"]
CMD ["start"]


# ---------- Binary Copy (External Build) ----------
FROM busybox AS external

WORKDIR /root
COPY ./dist/ /root/

ARG TARGETARCH
RUN if [ "${TARGETARCH}" = "arm64" ]; then \
      cp arm64_bin/nibid /root/nibid; \
    else \
      cp amd64_bin/nibid /root/nibid; \
    fi

FROM alpine:latest AS release

WORKDIR /root
RUN apk --no-cache add ca-certificates
COPY --from=external /root/nibid /usr/local/bin/nibid

ENTRYPOINT ["nibid"]
CMD ["start"]


# ---------- Regular Image ----------
FROM alpine:latest AS regular

WORKDIR /root
RUN apk --no-cache add ca-certificates

COPY --from=build-base /nibiru/build/nibid /usr/local/bin/nibid

ENTRYPOINT ["nibid"]
CMD ["start"]