FROM golang:1.18 AS builder

RUN apt-get update && apt-get install -y \
  jq \
  && rm -rf /var/lib/apt/lists/*

WORKDIR /nibid
COPY . ./
RUN CGO_ENABLED=0 make build

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /nibid/build/nibid /usr/local/bin/nibid
ENTRYPOINT ["nibid"]
