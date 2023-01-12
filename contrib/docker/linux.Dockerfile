FROM golang:1.19

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

COPY . ./

ENTRYPOINT ["make"]
CMD [ "build" ]
