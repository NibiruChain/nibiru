#!/usr/bin/env bash
# Build on homebox from a pushed source revision. Output stays outside testdata.
set -Eeuo pipefail

script_dir=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd -P)
repo_dir=$(git -C "$script_dir" rev-parse --show-toplevel)
out_dir=${1:-"$repo_dir/wasm-contracts/artifacts/usdc-holder-set"}

if [[ $(uname -m) != x86_64 ]]; then
  printf '%s\n' 'Build the fixture on the x86_64 homebox, then copy it back for Keeper tests.' >&2
  exit 1
fi
for tool in cargo rustup bun python3; do
  command -v "$tool" >/dev/null || { printf 'Missing build tool: %s\n' "$tool" >&2; exit 1; }
done
if ! git -C "$repo_dir" diff --quiet HEAD -- Cargo.lock wasm-contracts/usdc-holder-set; then
  printf '%s\n' 'Commit and push the contract changes before building the fixture.' >&2
  exit 1
fi
cd -- "$repo_dir"
mkdir -p -- "$out_dir"
out_dir=$(cd -- "$out_dir" && pwd -P)

rustup target add --toolchain 1.86.0 wasm32-unknown-unknown
RUSTFLAGS='-C target-cpu=mvp' cargo +1.86.0 build --locked --release \
  --target wasm32-unknown-unknown -p usdc-holder-set
bun x --bun --package binaryen@116.0.0 wasm-opt \
  target/wasm32-unknown-unknown/release/usdc_holder_set.wasm \
  -o "$out_dir/usdc_holder_set.wasm" -Os --signext-lowering --mvp-features

python3 - "$repo_dir" "$out_dir" <<'PY'
import datetime
import gzip
import hashlib
import json
import platform
import subprocess
import sys
from pathlib import Path

repo, out = map(Path, sys.argv[1:])
wasm = (out / 'usdc_holder_set.wasm').read_bytes()
if not wasm.startswith(b'\x00asm\x01\x00\x00\x00'):
    raise SystemExit('Unexpected Wasm header')
compressed = gzip.compress(wasm, compresslevel=9, mtime=0)
(out / 'usdc_holder_set.wasm.gz').write_bytes(compressed)
metadata = {
    'source_revision': subprocess.check_output(['git', 'rev-parse', 'HEAD'], cwd=repo, text=True).strip(),
    'built_at_utc': datetime.datetime.now(datetime.timezone.utc).isoformat(),
    'host': platform.node(),
    'architecture': platform.machine(),
    'rustc': subprocess.check_output(['rustup', 'run', '1.86.0', 'rustc', '--version'], text=True).strip(),
    'rustflags': '-C target-cpu=mvp',
    'binaryen': '116.0.0',
    'wasm_opt_flags': ['-Os', '--signext-lowering', '--mvp-features'],
    'cargo_lock_sha256': hashlib.sha256((repo / 'Cargo.lock').read_bytes()).hexdigest(),
    'wasm_sha256': hashlib.sha256(wasm).hexdigest(),
    'wasm_bytes': len(wasm),
    'gzip_sha256': hashlib.sha256(compressed).hexdigest(),
    'gzip_bytes': len(compressed),
    'status': 'development artifact; requires Keeper tests before deployment',
}
(out / 'build-info.json').write_text(json.dumps(metadata, indent=2) + '\n')
print(json.dumps(metadata, indent=2))
PY
