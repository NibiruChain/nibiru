#!/usr/bin/env bash
# this script is for generating protobuf files for the new google.golang.org/protobuf API

set -Eeuo pipefail

protoc_install_gopulsar() {
  if command -v go >/dev/null 2>&1; then
    # Installation with `go install ...@latest` can work but is less stable than
    # pinned versions.
    go install github.com/cosmos/cosmos-proto/cmd/protoc-gen-go-pulsar@v1.0.0-beta.5
    go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.2.0
    return
  fi

  command -v protoc-gen-go-pulsar >/dev/null
  command -v protoc-gen-go-grpc >/dev/null
}

protoc_install_gopulsar

echo "Cleaning API directory"
(
  cd api
  find ./ -type f \( -iname \*.pulsar.go -o -iname \*.pb.go -o -iname \*.cosmos_orm.go -o -iname \*.pb.gw.go \) -delete
  find . -empty -type d -delete
  cd ..
)

echo "Generating API module"
(
  cd proto
  buf generate --template buf.gen.pulsar.yaml
)

echo "Pruning unused generated code in API directory"
dirs_to_rm=(
  "api/eth/evm"
  "api/eth/types"
  "api/nibiru/epochs"
  "api/nibiru/devgas"
  "api/nibiru/genmsg"
  "api/nibiru/inflation"
  "api/nibiru/oracle"
  "api/nibiru/sudo"
  "api/nibiru/tokenfactory"
)
rm -rf "${dirs_to_rm[@]}"
