#!/usr/bin/env bash

set -euo pipefail

# --- Config ---
protoVer="0.14.0"
protoImageName="ghcr.io/cosmos/proto-builder:${protoVer}"

PROJECT_NAME="cosmos-sdk"
# Determine project name from git remote, trimming ".git"
# PROJECT_NAME="$(git remote get-url origin | xargs basename -s .git)"

containerProtoGen="${PROJECT_NAME}-proto-gen-${protoVer}"
echo "Generating Protobuf files"

# Does the container already exist?
if docker ps -a --format '{{.Names}}' | grep -Eq "^${containerProtoGen}$"; then
  echo "Found existing container: ${containerProtoGen}"
  docker start -a "${containerProtoGen}"
else
  echo "Running new proto-builder container"
  docker run --name "${containerProtoGen}" \
    -v "$(pwd)":/workspace \
    --workdir /workspace \
    "${protoImageName}" \
    bash ./contrib/scripts/protocgen.sh
fi
