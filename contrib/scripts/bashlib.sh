#!/usr/bin/env bash

set -e

# log_debug: Simple wrapper for `echo` with a DEBUG prefix.
log_debug() {
  echo "DEBUG" "$@"
}
log_error() {
  echo "❌ Error:" "$@"
}
log_success() {
  echo "✅ Success:" "$@"
}

# which_ok: Check if the given binary is in the $PATH.
# Returns code 0 on success and code 1 if the command fails.
which_ok() {
  if which "$1" >/dev/null 2>&1; then
    return 0
  else
    log_error "$1 is not present in \$PATH"
    return 1
  fi
}