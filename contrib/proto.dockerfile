FROM golang:1.17-alpine3.15 AS builder

RUN apk add git

# add orm codegenerator
WORKDIR /sdk
RUN git clone https://github.com/cosmos/cosmos-sdk .
WORKDIR /sdk/orm
RUN go build -o protoc-gen-go-cosmos-orm ./cmd/protoc-gen-go-cosmos-orm
RUN ls -al

# add pulsar codegenerator
WORKDIR /pulsar
RUN git clone https://github.com/cosmos/cosmos-proto .
RUN go build -o protoc-gen-go-pulsar ./cmd/protoc-gen-go-pulsar

# add go-grpc
RUN GO111MODULE=on GOBIN=/usr/bin go install \
    google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.1

# add gogo for msgs

# add protoc
RUN apk add "protoc"
RUN protoc --version

# add buf
WORKDIR /buf
RUN GO111MODULE=on GOBIN=/usr/bin go install \
      github.com/bufbuild/buf/cmd/buf@v1.0.0

RUN buf --help

FROM golang:1.17-alpine3.15
COPY --from=builder /sdk/orm/protoc-gen-go-cosmos-orm /usr/bin/protoc-gen-go-cosmos-orm
COPY --from=builder /pulsar/protoc-gen-go-pulsar /usr/bin/protoc-gen-go-pulsar
COPY --from=builder /usr/bin/buf /usr/bin/buf
COPY --from=builder /usr/bin/protoc-gen-go-grpc /usr/bin/protoc-gen-go-grpc

WORKDIR /work
COPY . ./

