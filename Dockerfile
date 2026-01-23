# Nibiru/Dockerfile
# 
# ## Build Targets
# 
# Example: docker build --target release .  
# 
# - build-base    : Build nibid from source using Go
# - build-external: Pull precompiled binary from dist/. Used by release workflow
#   (release.yml) with build-args: src=external to create official binaries for
#   different hardware (dist/arm64, dist/amd64).
# - build-source  : Unified binary source (selects base or external)
# - chaosnet      : Chaosnet development/test IMAGE.
# - release       : Minimal production runtime IMAGE.

ARG src=base

# ----- Stage "build-base" ----------
FROM golang:1.24 AS build-base

WORKDIR /nibiru

RUN apt-get update && apt-get install -y --no-install-recommends \
    liblz4-dev libsnappy-dev zlib1g-dev libbz2-dev libzstd-dev

# COPY go.mod go.sum ./
COPY ["go.mod", "go.sum", "./"]
COPY ["internal/", "./internal/"]

# Configure git for private Go modules using BuildKit secret
# The secret is mounted at /run/secrets/gh_pat during build
RUN --mount=type=secret,id=gh_pat \
    if [ -f /run/secrets/gh_pat ]; then \
      git config --global url."https://$(cat /run/secrets/gh_pat)@github.com/".insteadOf "https://github.com/"; \
    fi && \
    GOPRIVATE=github.com/cometbft/* go mod download

COPY . .

RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg \
    --mount=type=cache,target=/nibiru/temp \
    make build && cp build/nibid /root/

# ----- Stage "build-external" 
# Binary Copy (External Build)
FROM busybox AS build-external

WORKDIR /root
COPY ./dist/ /root/

ARG TARGETARCH
RUN if [ "${TARGETARCH}" = "arm64" ]; then \
      cp arm64/nibid /root/nibid; \
    else \
      cp amd64/nibid /root/nibid; \
    fi

# ----- Stage "build-source"
FROM build-${src} AS build-source

# ----- Stage "chaosnet": Creates IMAGE
FROM alpine:latest AS chaosnet

WORKDIR /root

RUN apk --no-cache add \
  ca-certificates \
  build-base \
  bash \
  curl \
  jq

COPY --from=build-source /root/nibid /usr/local/bin/nibid
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

# ----- Stage "release": Creates IMAGE
FROM alpine:latest AS release

WORKDIR /root
RUN apk --no-cache add ca-certificates
COPY --from=build-source /root/nibid /usr/local/bin/nibid

ENTRYPOINT ["nibid"]
CMD ["start"]
