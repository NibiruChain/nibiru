#!/usr/bin/env bash

set -Eeuo pipefail

echo "Generating gogo proto code (contrib/scripts/protocgen.sh)"
buf generate proto --template proto/buf.gen.gogo.yaml

# move proto files to the right places
cp -r github.com/NibiruChain/nibiru/v2/* ./
rm -rf github.com

# DEPRECATED: file `protocgen-pulsar.sh` generated local `api/` packages for
# depinject module config protos. Repo policy is manual module wiring, with no
# local depinject config protos and no generated directory `api/`.
# ./contrib/scripts/protocgen-pulsar.sh
