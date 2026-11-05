#!/usr/bin/env bash

set -Eeuo pipefail

# This script runs from the repository root (contrib/scripts/proto.sh cds
# there first). It regenerates the Go code for all four proto trees. gocosmos
# writes each file under its go_package path, so every tree except bft/proto
# produces a github.com/NibiruChain/nibiru/v2/... directory that gets copied
# into place and then deleted.

# ---------------------------------------------------------------------------
# 1. Nibiru protos: proto/
# ---------------------------------------------------------------------------
echo "Generating gogo proto code for proto/ (contrib/scripts/protocgen.sh)"
buf generate proto --template proto/buf.gen.gogo.yaml

# move proto files to the right places
cp -r github.com/NibiruChain/nibiru/v2/* ./
rm -rf github.com

# DEPRECATED: file `protocgen-pulsar.sh` generated local `api/` packages for
# depinject module config protos. Repo policy is manual module wiring, with no
# local depinject config protos and no generated directory `api/`.
# ./contrib/scripts/protocgen-pulsar.sh

# ---------------------------------------------------------------------------
# 2. Nibiru BFT protos: bft/proto
# ---------------------------------------------------------------------------
# Same steps as the header of bft/buf.gen.yaml. gogofaster uses
# paths=source_relative, so the output lands next to each .proto file under
# bft/proto. Two files belong in other packages and move out.
echo "Generating gogofaster proto code for bft/proto"
buf generate bft/proto --template bft/buf.gen.yaml -o bft
mv bft/proto/tendermint/abci/types.pb.go bft/abci/types/
cp bft/proto/tendermint/rpc/grpc/types.pb.go bft/rpc/grpc/

# ---------------------------------------------------------------------------
# 3. Cosmos SDK protos: lib/cosmos-sdk/proto
# ---------------------------------------------------------------------------
# Adapted from lib/cosmos-sdk/scripts/protocgen.sh. Upstream runs buf once per
# file under cosmos/ and amino/, and skips two kinds of file:
#   - files with no go_package (the depinject module config protos)
#   - files whose go_package is cosmossdk.io/api/..., which only the pulsar
#     generator builds
# Keep one buf call per file, as upstream does. One call for the whole tree
# would batch each directory into one plugin request, and gogo then embeds
# every file of the package in staking.pb.go's StakingDescription() set.
# The template writes to `..` relative to -o, which is lib/cosmos-sdk.
# Upstream also ran `go mod tidy` and protocgen-pulsar.sh. Neither applies
# here. lib/cosmos-sdk has no go.mod of its own and no local api/ module.
echo "Generating gogo proto code for lib/cosmos-sdk/proto"
while IFS= read -r file; do
  if grep -q "option go_package" "${file}" &&
    ! grep -q 'option go_package.*cosmossdk.io/api' "${file}"; then
    buf generate lib/cosmos-sdk/proto \
      --template lib/cosmos-sdk/proto/buf.gen.gogo.yaml \
      -o lib/cosmos-sdk/proto --path "${file}"
  fi
done < <(find lib/cosmos-sdk/proto/cosmos lib/cosmos-sdk/proto/amino \
  -name '*.proto' | sort)

# move proto files to the right places
cp -r lib/cosmos-sdk/github.com/NibiruChain/nibiru/v2/* ./
rm -rf lib/cosmos-sdk/github.com

# ---------------------------------------------------------------------------
# 4. IBC-Go protos: lib/ibc-go/proto
# ---------------------------------------------------------------------------
# Adapted from lib/ibc-go/scripts/protocgen.sh, which generates every file in
# the module. The template writes to `..` relative to -o, which is
# lib/ibc-go. Upstream copied github.com/cosmos/ibc-go/v*/modules/* into
# modules/. With the rewritten go_package the output path already starts at
# the repository root, so it copies to ./ like the other trees.
echo "Generating gogo proto code for lib/ibc-go/proto"
buf generate lib/ibc-go/proto \
  --template lib/ibc-go/proto/buf.gen.gogo.yaml \
  -o lib/ibc-go/proto

# move proto files to the right places
cp -r lib/ibc-go/github.com/NibiruChain/nibiru/v2/* ./
rm -rf lib/ibc-go/github.com
