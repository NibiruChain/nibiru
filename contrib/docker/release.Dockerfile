FROM busybox AS temp

WORKDIR /root
COPY ./dist/ /root/

ARG TARGETARCH
RUN if [ "${TARGETARCH}" = "arm64" ]; then \
  cp arm64_bin/nibid /root/nibid; \
  else \
  cp amd64_bin/nibid /root/nibid; \
  fi

FROM alpine:latest

WORKDIR /root
RUN apk --no-cache add ca-certificates
COPY --from=temp /root/nibid /usr/local/bin/nibid

ENTRYPOINT ["nibid"]
CMD [ "start" ]
