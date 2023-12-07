FROM golang:1.21 AS builder

WORKDIR /root
COPY ./dist/ /root/

ARG TARGETARCH
RUN if [ "${TARGETARCH}" = "arm64" ]; then \
  cp linux_linux_arm64/nibid /root/nibid; \
  else \
  cp linux_linux_amd64_v1/nibid /root/nibid; \
  fi

FROM alpine:latest

WORKDIR /root
RUN apk --no-cache add ca-certificates
COPY --from=builder /root/nibid /usr/local/bin/nibid

ENTRYPOINT ["nibid"]
CMD [ "start" ]