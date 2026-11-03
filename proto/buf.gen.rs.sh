#!/usr/bin/env bash
# Regenerate Rust types from this checkout's Nibiru protobuf definitions.
# Pass Buf input filters, e.g. --path proto/nibiru/sudo/v1, to limit generation.
set -euo pipefail

repo_root="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"
source contrib/bashlib.sh
for required_command in buf cargo mktemp cp; do
  which_ok "$required_command"
done

# Remote generation writes only to a disposable output directory. Copy the
# selected packages into nibiru-std only after Buf succeeds.
proto_output="$(mktemp -d "${TMPDIR:-/tmp}/nibiru-rust-proto.XXXXXX")"
trap 'rm -rf -- "$proto_output"' EXIT
buf generate proto --template proto/buf.gen.rs.yaml --output "$proto_output" "$@"
shopt -s nullglob
proto_files=("$proto_output"/proto-rs/*.rs)
if [ "${#proto_files[@]}" -eq 0 ]; then
  echo "Buf produced no Rust protobuf files." >&2
  exit 1
fi
cp -- "${proto_files[@]}" lib/nibiru-std/src/proto/buf/
cargo run --locked -p scripts --bin proto_clean
