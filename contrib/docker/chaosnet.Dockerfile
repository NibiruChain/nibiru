FROM golang:1.19-alpine AS builder

WORKDIR /nibiru

ENV BUILD_TAGS=muslc \
  LINK_STATICALLY=true \
  CGO_ENABLED=1 \
  LDFLAGS='-s -w -linkmode=external -extldflags "-Wl,-z,muldefs -static -lm"'

RUN apk update && \
  apk add --no-cache \
  make \
  git \
  ca-certificates \
  build-base

ADD https://github.com/CosmWasm/wasmvm/releases/download/v1.1.1/libwasmvm_muslc.aarch64.a /lib/libwasmvm_muslc.aarch64.a
ADD https://github.com/CosmWasm/wasmvm/releases/download/v1.1.1/libwasmvm_muslc.x86_64.a /lib/libwasmvm_muslc.x86_64.a
RUN sha256sum /lib/libwasmvm_muslc.aarch64.a | grep 9ecb037336bd56076573dc18c26631a9d2099a7f2b40dc04b6cae31ffb4c8f9a
RUN sha256sum /lib/libwasmvm_muslc.x86_64.a | grep 6e4de7ba9bad4ae9679c7f9ecf7e283dd0160e71567c6a7be6ae47c81ebe7f32
RUN cp /lib/libwasmvm_muslc.$(uname -m).a /lib/libwasmvm_muslc.a

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
RUN MNEMONIC=${MNEMONIC} CHAIN_ID=${CHAIN_ID} ./chaosnet.sh

ENTRYPOINT ["nibid"]
CMD [ "start" ]
