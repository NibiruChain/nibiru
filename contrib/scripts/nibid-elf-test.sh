#!/usr/bin/env bash
#
# nibid-elf-test.sh - Assert that a Linux nibid binary is a static PIE.
#
# Usage:
#   contrib/scripts/nibid-elf-test.sh --bin build/nibid
#   contrib/scripts/nibid-elf-test.sh --require-checksec --bin build/nibid

set -Eeuo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd -P)"
SCRIPT_NAME="$(basename -- "${BASH_SOURCE[0]}")"
REPO_ROOT="$(cd -- "$SCRIPT_DIR/../.." && pwd -P)"

NIBID_BIN=""
REQUIRE_CHECKSEC=false

trap 'log_error "failed on line $LINENO"; exit 1' ERR

log_info() {
  printf '[nibid-elf-test] %s\n' "$*" >&2
}

log_error() {
  printf '[nibid-elf-test] ERROR: %s\n' "$*" >&2
}

log_success() {
  printf '[nibid-elf-test] OK: %s\n' "$*" >&2
}

show_help() {
  cat <<EOF
Usage: $SCRIPT_NAME [OPTIONS] [BIN]

Assert that a Linux nibid binary is an ELF DYN static PIE.

Binary selection:
  --bin PATH   Path to nibid binary.
  BIN          Positional path to nibid binary (same as --bin).
  (default)    Prefer \$REPO_ROOT/build/nibid if executable, else nibid on PATH.

Options:
  --require-checksec  Require checksec and assert that it reports PIE enabled.
  -h, --help          Show this help message and exit.
EOF
}

parse_args() {
  while [[ $# -gt 0 ]]; do
    case "$1" in
    -h | --help)
      show_help
      exit 0
      ;;
    --require-checksec)
      REQUIRE_CHECKSEC=true
      shift
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

require_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    log_error "required command not found: $1"
    return 1
  fi
}

assert_elf_type() {
  local bin="$1"
  local type_line
  type_line="$(LC_ALL=C readelf -h "$bin" | awk '/^[[:space:]]*Type:/{print; exit}')"
  log_info "$type_line"

  if [[ ! "$type_line" =~ [[:space:]]DYN[[:space:]] ]]; then
    log_error "expected ELF Type DYN (static PIE), got: $type_line"
    return 1
  fi
}

assert_no_interpreter() {
  local bin="$1"
  if LC_ALL=C readelf -lW "$bin" | grep -q 'Requesting program interpreter'; then
    log_error "static PIE must not have a program interpreter"
    return 1
  fi
}

assert_checksec_pie() {
  local bin="$1"
  local result
  require_cmd checksec
  result="$(LC_ALL=C checksec --file="$bin")"
  printf '%s\n' "$result"

  if ! printf '%s\n' "$result" | grep -Eqi 'PIE[[:space:]]+enabled'; then
    log_error "checksec did not report PIE enabled"
    return 1
  fi
}

main() {
  parse_args "$@"
  require_cmd readelf

  local bin
  bin="$(resolve_nibid_bin)"
  if [[ ! -x "$bin" ]]; then
    log_error "not an executable binary: $bin"
    return 1
  fi

  log_info "checking $bin"
  assert_elf_type "$bin"
  assert_no_interpreter "$bin"
  if [[ "$REQUIRE_CHECKSEC" == true ]]; then
    assert_checksec_pie "$bin"
  fi
  log_success "ELF is a static PIE"
}

main "$@"
