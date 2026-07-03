#!/usr/bin/env bash
set -Eeuo pipefail

PROTO_VERSION="0.14.0"
PROTO_IMAGE="ghcr.io/cosmos/proto-builder:${PROTO_VERSION}"
PROTO_FMT_IMAGE="tendermintdev/docker-build-proto"

usage() {
  cat <<EOF
NAME:
   proto.sh - protobuf command dispatcher

USAGE:
   contrib/scripts/proto.sh [global options] command

COMMANDS:
   all             Run fmt, lint, and gen
   gen             Generate protobuf Go and API files
   fmt, format     Format protobuf files
   lint            Lint protobuf files
   help, h         Show this help text

GLOBAL OPTIONS:
   --cmd           Print the underlying command instead of executing it
   --help, -h      Show this help text
EOF
}

log_info() {
  printf 'INFO: %s\n' "$*" >&2
}

log_error() {
  printf 'ERROR: %s\n' "$*" >&2
}

shell_join() {
  printf '%q ' "$@"
  printf '\n'
}

run_cmd() {
  if [[ "${PRINT_CMD}" == "true" ]]; then
    shell_join "$@"
    return 0
  fi

  shell_join "$@"
  "$@"
}

repo_root() {
  git rev-parse --show-toplevel
}

docker_run() {
  local -r image="$1"
  shift

  run_cmd docker run \
    --rm \
    -v "$(repo_root):/workspace" \
    --workdir /workspace \
    "${image}" \
    "$@"
}

proto_gen() {
  log_info "Generating Protobuf files"
  run_cmd docker run \
    --rm \
    --user root \
    -e "HOST_UID=$(id -u)" \
    -e "HOST_GID=$(id -g)" \
    -v "$(repo_root):/workspace" \
    --workdir /workspace \
    "${PROTO_IMAGE}" \
    sh -lc 'apk add --no-cache bash go git su-exec >/dev/null && export PATH="/go/bin:$PATH" && su-exec "$HOST_UID:$HOST_GID" env PATH="/go/bin:$PATH" bash ./contrib/scripts/protocgen.sh'
}

# FIXME: TODO https://github.com/NibiruChain/nibiru/issues/2636
proto_fmt() {
  log_info "Formatting Protobuf files"
  docker_run \
    "${PROTO_FMT_IMAGE}" \
    find ./ -not -path "./third_party/*" -name "*.proto" -exec clang-format -i {} ";"
}

proto_lint() {
  log_info "Linting Protobuf files"
  run_cmd docker run --rm \
    -v "$(repo_root):/workspace" \
    --workdir /workspace/proto \
    "${PROTO_IMAGE}" \
    buf lint --error-format=json "$@"
}

proto_all() {
  proto_fmt
  proto_lint
  proto_gen
}

PRINT_CMD=false
args=()
for arg in "$@"; do
  case "${arg}" in
  --cmd)
    PRINT_CMD=true
    ;;
  --help | -h)
    usage
    exit 0
    ;;
  *)
    args+=("${arg}")
    ;;
  esac
done

set -- "${args[@]}"
cmd="${1:-help}"
shift || true

case "${cmd}" in
all)
  proto_all "$@"
  ;;
gen)
  proto_gen "$@"
  ;;
fmt | format)
  proto_fmt "$@"
  ;;
lint)
  proto_lint "$@"
  ;;
help | h | "")
  usage
  ;;
*)
  log_error "Unknown proto command: ${cmd}"
  usage >&2
  exit 1
  ;;
esac
