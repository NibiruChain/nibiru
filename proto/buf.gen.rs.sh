#!/usr/bin/env bash

# buf.gen.rs.sh: Generates Rust protobuf types based on NibiruChain/nibiru/proto.

set -eo pipefail

# move_to_dir_with_protos: Make sure the script is runnning from inside the
# NibiruChain/Nibiru repo and has protobuf files. 
move_to_dir_with_protos() {
  # Get the name of the current directory
  local start_path
  local start_dir_name
  start_path="$(pwd)"
  start_dir_name=$(basename "$start_path")

  echo "start_path: ${start_path}"
  echo "start_dir_name: ${start_dir_name}"

  echo "Check if 'start_dir_name' is (\"nibiru\", \"nibi-chain\") (the repo)."
  echo "Or if the immediate parent is one of those directories, move to it."
  if [ "$start_dir_name" != "nibiru" ] && [ "$start_dir_name" != "nibi-chain" ]; then
    if [ -d ../nibiru ]; then
      cd ../nibiru
    elif [ -d ../nibi-chain ]; then
      cd ../nibi-chain
    else
      echo "Not in 'nibiru' or 'nibi-chain' directory, or an immediate child. Exiting."
      return 1
    fi
  fi

  # Check if nibiru/go.mod exists
  if [ ! -f "go.mod" ]; then
    echo "start_path: $start_path"
    echo "go.mod not found in 'nibiru' directory. Exiting."
    return 1
  fi

  go mod tidy

  # Check if the nibiru/proto directory exists
  if [ ! -d "proto" ]; then
    echo "start_path: $start_path"
    echo "'proto' directory not found in 'nibiru'. Exiting."
    return 1
  fi

  NIBIRU_REPO_PATH=$(pwd)
  echo "NIBIRU_REPO_PATH: $NIBIRU_REPO_PATH"
}

init_globals() {
  proto_dir="proto" # nibiru/proto
  out_dir="dist"    # nibiru/dist

  cosmos_sdk_gh_path=$(go list -f '{{ .Dir }}' -m github.com/cosmos/cosmos-sdk)
  # cosmos_sdk_gh_path is parsed from the go.mod and will be a string like:
  # $HOME/go/pkg/mod/github.com/cosmos/cosmos-sdk@v0.47.4

  nibiru_cosmos_sdk_version=$(echo "$cosmos_sdk_gh_path" | awk -F'@' '{print $2}') 
  # Example: v0.47.4
  
  # OUT_PATH: Path to the generated protobuf types.
  OUT_PATH="${NIBIRU_REPO_PATH:-.}/$out_dir"
  echo "OUT_PATH: $OUT_PATH"
}


ensure_nibiru_cosmos_sdk_version() {
  # Ensure nibiru_cosmos_sdk_version has a value before proceeding
  if [ -z "$nibiru_cosmos_sdk_version" ]; then
    echo "nibiru_cosmos_sdk_version is empty. Exiting."
    return 1
  fi
}

go_get_cosmos_protos() {
  echo -e "\nGrabbing cosmos-sdk proto file locations from disk"
  if ! grep "github.com/gogo/protobuf => github.com/regen-network/protobuf" go.mod &>/dev/null; then
    echo -e "\tPlease run this command from somewhere inside the cosmos-sdk folder."
    return 1
  fi

  ensure_nibiru_cosmos_sdk_version

  # get protos for: cosmos-sdk, cosmos-proto
  go get "github.com/cosmos/cosmos-sdk@$nibiru_cosmos_sdk_version"
  go get github.com/cosmos/cosmos-proto
}

buf_gen() {
  # Called by proto_gen
  local proto_dir="$1"

  if ! command -v buf &> /dev/null; then
    echo "Please install buf to generate protos. Try using:"
    echo "go install github.com/bufbuild/buf/cmd/buf@latest"
    exit 1
  fi 

  local buf_dir="$NIBIRU_REPO_PATH/proto"

  buf generate "$proto_dir" \
    --template "$buf_dir/buf.gen.rs.yaml" \
    -o "$OUT_PATH" \
    --config "$proto_dir/buf.yaml"
}

proto_clone_sdk() {
  move_to_dir_with_protos
  local start_dir
  start_dir=$(pwd) # nibiru
  cd .. 

  # degit copies a GitHub repo without downloading the entire git history, 
  # which is much faster than a full git clone.
  # TODO: OLD:  npx degit "cosmos/cosmos-sdk#$nibiru_cosmos_sdk_version" cosmos-sdk --force
  npx degit "NibiruChain/cosmos-sdk#$nibiru_cosmos_sdk_version" cosmos-sdk --force
  DIR_COSMOS_SDK="$(pwd)/cosmos-sdk"
  cd "$start_dir"
}

proto_gen() {

  go_get_cosmos_protos
  proto_clone_sdk

  printf "\nProto Directories: \n"
  cd "$NIBIRU_REPO_PATH/.."
  proto_dirs=()
  proto_dirs+=("$DIR_COSMOS_SDK/proto")
  proto_dirs+=("$NIBIRU_REPO_PATH/proto")
  
  echo "Generating protobuf types for each proto_dir"
  for proto_dir in "${proto_dirs[@]}"; do
    string=$proto_dir
    # Remove prefix
    prefix=$HOME/go/pkg/mod/github.com/
    prefix_removed_string=${string/#$prefix/}
    # Remove prefix
    # prefix=$(dirname $(pwd))
    prefix=$(pwd)/
    prefix_removed_string=${prefix_removed_string/#$prefix/}
    echo "------------ generating $prefix_removed_string ------------"
    mkdir -p "$OUT_PATH/${proto_dir}"

    echo "proto_dir: $proto_dir"
    buf_gen "$proto_dir"
  done

  echo "Completed - proto_gen() to path: $OUT_PATH"
}

# clean: the Rust buf generator we're using leaves some garbage.
# This function clears that way after a successful build.
clean() {

  # Guarantee that OUT_PATH is defined
  if [ -z "$OUT_PATH" ]; then
    echo "skipped clean() since NIBIRU_REPO_PATH is not defined."
    return 0
  fi

  # Guarantee that OUT_PATH is defined again just to be extra safe.
  OUT_PATH="${OUT_PATH:-.}" 
  rm -rf "${OUT_PATH:?}/home" # use ${var:?} to ensure this never expands to /home.
  rm -rf "${OUT_PATH:?}/nibiru"
}

die() {
  echo "⚠️ $*"
  exit 1
}

# ------------------------------------------------
# main : Start of script execution
# ------------------------------------------------

main () {
  move_to_dir_with_protos || die "failed move_to_dir_with_protos"
  init_globals || die "failed init_globals"
  proto_gen || die "failed proto_gen"
  clean || die "failed clean"
}


if main; then 
  echo "🔥 Generated Rust proto types successfully. "
else 
  echo "❌ Generation failed."
fi
