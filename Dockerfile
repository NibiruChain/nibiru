FROM golang:1.19 AS builder

WORKDIR /nibiru

ENV BUILD_TAGS=muslc \
  CGO_ENABLED=1 \
  LDFLAGS='-s -w -linkmode=external -extldflags "-Wl,-z,muldefs -static -lm"'

RUN apt-get update && \
  apt-get install -y \
  musl-dev && \
  rm -rf /var/lib/apt/lists/*

ARG TARGETARCH
RUN if [ "${TARGETARCH}" = "arm64" ]; then \
  wget https://github.com/CosmWasm/wasmvm/releases/download/v1.2.0/libwasmvm_muslc.aarch64.a -O /lib/libwasmvm_muslc.a; \
  else \
  wget https://github.com/CosmWasm/wasmvm/releases/download/v1.2.0/libwasmvm_muslc.x86_64.a -O /lib/libwasmvm_muslc.a; \
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
