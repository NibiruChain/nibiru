#!/usr/bin/env bash
# Check the release cache without downloading or compiling a runtime.
set -Eeuo pipefail
script_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd -P)"
# shellcheck source=contrib/scripts/build-nibiru.sh
source "$script_dir/build-nibiru.sh"
fixture_dir="$(mktemp -d)"
trap 'rm -rf -- "$fixture_dir"' EXIT
SCRIPT_DIR="$fixture_dir/scripts"
mkdir -p "$SCRIPT_DIR"
printf 'verified archive\n' > "$fixture_dir/archive"
digest="$(sha256sum "$fixture_dir/archive" | awk '{print $1}')"
printf '%s  libwasmvm_muslc.aarch64.a\n' "$digest" > "$SCRIPT_DIR/wasmvm-checksums.txt"
wget() {
  [[ "$4" == */v1.13.0/libwasmvm_muslc.aarch64.a ]]
  cp "$fixture_dir/archive" "$3"
  printf 'download\n' >> "$fixture_dir/downloads"
}
export -f wget
ensure_wasmvm_lib "$fixture_dir/cache" linux arm64 v1.13.0
library="$fixture_dir/cache/wasmvm/v1.13.0/lib/linux_arm64/libwasmvm_muslc.a"
cmp "$library" "$fixture_dir/archive"
ensure_wasmvm_lib "$fixture_dir/cache" linux arm64 v1.13.0
[[ "$(wc -l < "$fixture_dir/downloads")" -eq 1 ]]
printf 'corrupt cache\n' > "$library"
ensure_wasmvm_lib "$fixture_dir/cache" linux arm64 v1.13.0
cmp "$library" "$fixture_dir/archive"
[[ "$(wc -l < "$fixture_dir/downloads")" -eq 2 ]]
printf 'corrupt download\n' > "$fixture_dir/archive"
printf 'corrupt cache\n' > "$library"
if ensure_wasmvm_lib "$fixture_dir/cache" linux arm64 v1.13.0; then
  printf 'checksum mismatch was accepted\n' >&2
  exit 1
fi
[[ -z "$(find "$fixture_dir/cache" -name 'download.*' -print)" ]]
printf 'release cache tests passed\n'
