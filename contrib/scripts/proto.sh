#!/usr/bin/env bash
set -Eeuo pipefail

BUF_VERSION="1.55.1"

REPO_ROOT="$(git -C "$(dirname "${BASH_SOURCE[0]}")/../.." rev-parse --show-toplevel)"
cd "${REPO_ROOT}"
# shellcheck source=contrib/bashlib.sh
source contrib/bashlib.sh

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

check_clang_format() {
  if which_ok clang-format; then
    return 0
  fi

  log_error "clang-format is required for protobuf formatting. Install clang-format and ensure it is on PATH."
  return 1
}

check_buf() {
  if which_ok buf >/dev/null 2>&1; then
    return 0
  fi

  log_info "buf is not present; installing buf ${BUF_VERSION}"
  if ! which_ok go; then
    log_error "buf is required for protobuf linting and generation. Install Go, then run: go install github.com/bufbuild/buf/cmd/buf@v${BUF_VERSION}"
    return 1
  fi

  run_cmd go install "github.com/bufbuild/buf/cmd/buf@v${BUF_VERSION}"
  if ! which_ok buf; then
    log_error "buf was installed but is not on PATH. Add \$(go env GOPATH)/bin to PATH and retry."
    return 1
  fi
}

proto_gen() {
  log_info "Generating Protobuf files"
  if [[ "${PRINT_CMD}" != "true" ]]; then
    check_buf
  fi
  run_cmd bash contrib/scripts/protocgen.sh
}

proto_fmt() {
  log_info "Formatting Protobuf files"
  if [[ "${PRINT_CMD}" != "true" ]]; then
    check_clang_format
  fi
  run_cmd find ./proto -name '*.proto' -exec clang-format -i {} \;
}

proto_lint() {
  log_info "Linting Protobuf files"
  if [[ "${PRINT_CMD}" != "true" ]]; then
    check_buf
  fi
  (
    cd proto
    run_cmd buf lint --error-format=json "$@"
  )
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
