FROM golang:1.21 AS builder

WORKDIR /nibiru

# copy go.mod, go.sum to WORKDIR
COPY go.sum go.mod ./  
RUN go mod download
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
  curl \
  jq

COPY --from=builder /nibiru/build/nibid /usr/local/bin/nibid

ENTRYPOINT ["nibid"]
CMD [ "start" ]