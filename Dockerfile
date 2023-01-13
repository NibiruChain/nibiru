FROM golang:1.19 AS builder

WORKDIR /nibiru
ARG ARCH=aarch64
ENV BUILD_TAGS=muslc \
  CGO_ENABLED=1 \
  LDFLAGS='-s -w -linkmode=external -extldflags "-Wl,-z,muldefs -static -lm"'

RUN apt-get update && \
  apt-get install -y \
  musl-dev && \
  rm -rf /var/lib/apt/lists/*

RUN wget https://github.com/CosmWasm/wasmvm/releases/download/v1.1.1/libwasmvm_muslc.${ARCH}.a -O /lib/libwasmvm_muslc.a

COPY go.* ./
RUN go mod download

COPY . ./
RUN --mount=type=cache,target=/root/.cache/go-build make build

FROM alpine:latest
WORKDIR /root
RUN apk --no-cache add ca-certificates
COPY --from=builder /nibiru/build/nibid /usr/local/bin/nibid
ENTRYPOINT ["nibid"]
CMD [ "start" ]
