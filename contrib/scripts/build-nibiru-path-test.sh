#!/usr/bin/env bash
#
# build-nibiru-path-test.sh - Test the PATH checks of build-nibiru.sh.
#
# Sources build-nibiru.sh and runs warn_if_shadowed against temp dirs that hold
# real files and symlinks, with a constructed PATH.
#
# Usage: contrib/scripts/build-nibiru-path-test.sh

set -Eeuo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd -P)"
# shellcheck source=build-nibiru.sh
source "$SCRIPT_DIR/build-nibiru.sh"
trap - ERR

fails=0
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

# expect: Compare stdout of warn_if_shadowed with the wanted text.
expect() {
  local label="$1" want="$2" got
  got="$(warn_if_shadowed "$tmp/inst" 2>/dev/null)"
  if [[ "$got" == "$want" ]]; then
    printf 'ok   %s\n' "$label"
  else
    printf 'FAIL %s\n  want: %s\n  got:  %s\n' "$label" "$want" "$got"
    fails=$((fails + 1))
  fi
}

mkdir -p "$tmp/inst" "$tmp/other" "$tmp/linkdir"
printf '#!/bin/sh\n' >"$tmp/inst/nibiru"
chmod +x "$tmp/inst/nibiru"
ln -sfn nibiru "$tmp/inst/nibid"
printf '#!/bin/sh\n' >"$tmp/other/nibiru"
chmod +x "$tmp/other/nibiru"
ln -sfn "$tmp/inst/nibiru" "$tmp/linkdir/nibiru"
ln -sfn "$tmp/inst/nibiru" "$tmp/linkdir/nibid"

# Single copy, and the nibid symlink resolves to the installed file.
PATH="$tmp/inst:/usr/bin:/bin"
expect "single copy prints nothing" ""

# Another directory holds a symlink to the installed file: not another copy.
PATH="$tmp/inst:$tmp/linkdir:/usr/bin:/bin"
expect "symlink to self is not another copy" ""
PATH="$tmp/linkdir:$tmp/inst:/usr/bin:/bin"
expect "symlink to self wins without a marker" ""

# Installed build wins and another real copy exists.
PATH="$tmp/inst:$tmp/other:/usr/bin:/bin"
expect "ok with other copy" \
  "nibiru-path-ok name=nibiru runs=$tmp/inst/nibiru other=$tmp/other/nibiru"

# Another copy wins.
PATH="$tmp/other:$tmp/inst:/usr/bin:/bin"
expect "shadowed" \
  "nibiru-path-shadowed name=nibiru runs=$tmp/other/nibiru installed=$tmp/inst/nibiru"

# Both names shadowed gives one marker per name.
ln -sfn nibiru "$tmp/other/nibid"
PATH="$tmp/other:$tmp/inst:/usr/bin:/bin"
expect "one marker per shadowed name" \
  "nibiru-path-shadowed name=nibiru runs=$tmp/other/nibiru installed=$tmp/inst/nibiru
nibiru-path-shadowed name=nibid runs=$tmp/other/nibid installed=$tmp/inst/nibid"

if ((fails > 0)); then
  printf '%d failure(s)\n' "$fails"
  exit 1
fi
printf 'all passed\n'
