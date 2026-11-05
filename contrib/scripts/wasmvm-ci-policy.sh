#!/usr/bin/env bash
# Select Wasm VM tests, platform builds, and release publication.
set -euo pipefail

if [[ $# -ne 3 || ( "$3" != true && "$3" != false ) ]]; then
  echo "Usage: $0 EVENT REF HAS_VM_CHANGES(true|false)" >&2
  exit 2
fi

event="$1"
ref="$2"
has_changes="$3"
do_run=false
do_package=false
do_release=false
reason="No Wasm VM FFI changes detected."

case "$event" in
  schedule|workflow_dispatch)
    do_run=true
    do_package=true
    reason="Scheduled or manual full Wasm VM validation."
    ;;
  push)
    if [[ "$ref" =~ ^refs/tags/lib/wasmvm/v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
      do_run=true
      do_package=true
      do_release=true
      reason="Wasm VM version tag runs full validation and release publication."
    elif [[ "$ref" == refs/heads/main && "$has_changes" == true ]]; then
      do_run=true
      reason="Wasm VM FFI changes detected on main."
    fi
    ;;
  pull_request)
    if [[ "$has_changes" == true ]]; then
      do_run=true
      reason="Wasm VM FFI changes detected in pull request."
    fi
    ;;
esac

printf 'do-run=%s\ndo-package=%s\ndo-release=%s\nreason=%s\n' \
  "$do_run" "$do_package" "$do_release" "$reason"
