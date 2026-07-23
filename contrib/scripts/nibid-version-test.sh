#!/usr/bin/env bash
#
# nibid-version-test.sh - Assert nibid was built with non-empty version metadata.
#
# Checks short `nibid version` and long fields injected via ldflags. Used by CI
# after build-nibiru.sh and runnable locally against build/nibid or PATH nibid.
#
# Usage:
#   contrib/scripts/nibid-version-test.sh
#   contrib/scripts/nibid-version-test.sh --bin build/nibid
#   contrib/scripts/nibid-version-test.sh build/nibid
#   contrib/scripts/nibid-version-test.sh --help

set -Eeuo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd -P)"
SCRIPT_NAME="$(basename -- "${BASH_SOURCE[0]}")"
REPO_ROOT="$(cd -- "$SCRIPT_DIR/../.." && pwd -P)"

NIBID_BIN=""

trap 'log_error "failed on line $LINENO"; exit 1' ERR

# log_info: Print an informational message to stderr.
log_info() {
  printf '[nibid-version-test] %s\n' "$*" >&2
}

# log_error: Print an error message to stderr.
log_error() {
  printf '[nibid-version-test] ERROR: %s\n' "$*" >&2
}

# log_success: Print a success message to stderr.
log_success() {
  printf '[nibid-version-test] OK: %s\n' "$*" >&2
}

show_help() {
  cat <<EOF
Usage: $SCRIPT_NAME [OPTIONS] [BIN]

Assert that a nibid binary has non-empty version metadata from build ldflags.

Binary selection:
  --bin PATH   Path to nibid binary.
  BIN          Positional path to nibid binary (same as --bin).
  (default)    Prefer \$REPO_ROOT/build/nibid if executable, else nibid on PATH.

Options:
  -h, --help   Show this help message and exit.
EOF
}

parse_args() {
  while [[ $# -gt 0 ]]; do
    case "$1" in
    -h | --help)
      show_help
      exit 0
      ;;
    --bin)
      if [[ $# -lt 2 ]]; then
        log_error "--bin requires a path argument"
        show_help
        exit 1
      fi
      NIBID_BIN="$2"
      shift 2
      ;;
    --)
      shift
      break
      ;;
    -*)
      log_error "unknown argument: $1"
      show_help
      exit 1
      ;;
    *)
      if [[ -n "$NIBID_BIN" ]]; then
        log_error "binary path already set to $NIBID_BIN; unexpected: $1"
        show_help
        exit 1
      fi
      NIBID_BIN="$1"
      shift
      ;;
    esac
  done

  if [[ $# -gt 0 ]]; then
    log_error "unexpected arguments: $*"
    show_help
    exit 1
  fi
}

# resolve_nibid_bin: Choose binary path when --bin / positional was not given.
resolve_nibid_bin() {
  if [[ -n "$NIBID_BIN" ]]; then
    printf '%s' "$NIBID_BIN"
    return 0
  fi

  if [[ -x "$REPO_ROOT/build/nibid" ]]; then
    printf '%s' "$REPO_ROOT/build/nibid"
    return 0
  fi

  if command -v nibid >/dev/null 2>&1; then
    command -v nibid
    return 0
  fi

  log_error "no nibid binary found: expected $REPO_ROOT/build/nibid or nibid on PATH"
  return 1
}

# require_executable: Fail unless path exists and is executable.
require_executable() {
  local bin="$1"
  if [[ ! -x "$bin" ]]; then
    log_error "not an executable binary: $bin"
    return 1
  fi
}

# assert_short_version: Fail if short \`nibid version\` is empty/whitespace.
assert_short_version() {
  local bin="$1"
  local version
  version="$("$bin" version | tr -d '[:space:]')"
  if [[ -z "$version" ]]; then
    log_error "short version output is empty"
    return 1
  fi
  log_info "short version: $version"
}

# assert_long_fields: Fail if required long version fields are missing/empty,
# or if server_name is the default <appd> (empty-ldflags failure mode).
assert_long_fields() {
  local bin="$1"
  local long field
  long="$("$bin" version --long)"
  printf '%s\n' "$long"

  for field in name server_name version commit build_tags; do
    if ! printf '%s\n' "$long" | grep -Eq "^${field}: .+"; then
      log_error "long version missing non-empty field: $field"
      return 1
    fi
  done

  # Known empty-ldflags failure mode (Cosmos SDK default AppName)
  if printf '%s\n' "$long" | grep -Eq '^server_name: <appd>$'; then
    log_error "server_name is <appd> (version ldflags did not apply)"
    return 1
  fi
}

main() {
  parse_args "$@"

  local bin
  bin="$(resolve_nibid_bin)"
  require_executable "$bin"

  log_info "checking $bin"
  assert_short_version "$bin"
  assert_long_fields "$bin"
  log_success "version metadata looks populated"
}

main "$@"
