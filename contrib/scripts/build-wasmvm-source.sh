#!/usr/bin/env bash
# Build a native static WasmVM library from this checkout, never a release download.
set -Eeuo pipefail

repo_root="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/../.." && pwd -P)"
if [[ $# != 3 ]]; then
  echo "usage: $0 <linux|darwin> <amd64|arm64> <output-directory>" >&2
  exit 2
fi
os_name="$1"
arch_name="$2"
output_dir="$3"
case "$os_name/$arch_name" in
linux/amd64)
  rust_target=x86_64-unknown-linux-musl
  library=libwasmvm_muslc.a
  ;;
linux/arm64)
  rust_target=aarch64-unknown-linux-musl
  library=libwasmvm_muslc.a
  ;;
darwin/amd64)
  rust_target=x86_64-apple-darwin
  library=libwasmvmstatic_darwin.a
  ;;
darwin/arm64)
  rust_target=aarch64-apple-darwin
  library=libwasmvmstatic_darwin.a
  ;;
*)
  echo "unsupported WasmVM target: $os_name/$arch_name" >&2
  exit 2
  ;;
esac
mkdir -p "$output_dir"
output_dir="$(cd "$output_dir" && pwd -P)"

if command -v sha256sum >/dev/null 2>&1; then
  sha256=(sha256sum)
else
  sha256=(shasum -a 256)
fi

# Include untracked imported sources, and ignore removed files and build outputs.
source_hash="$(
  cd "$repo_root"
  git ls-files -z --cached --others --exclude-standard -- \
    Cargo.toml Cargo.lock rust-toolchain.toml lib/wasmer lib/cosmwasm-vm \
    lib/wasmvm/libwasmvm lib/wasmvm/builders contrib/scripts/build-wasmvm-source.sh \
    contrib/docker/Dockerfile.wasmvm-source |
    sort -zu |
    while IFS= read -r -d '' source_file; do
      if [[ -f "$source_file" ]]; then
        "${sha256[@]}" "$source_file"
      fi
    done |
    "${sha256[@]}" | awk '{print $1}'
)"
source_commit="$(git -C "$repo_root" rev-parse HEAD)"
stamp="$source_hash/$rust_target/rust-1.95.0/$source_commit"
if [[ -f "$output_dir/source-id" && -f "$output_dir/checksums.txt" &&
  "$(cat "$output_dir/source-id")" == "$stamp" ]]; then
  if (cd "$output_dir" && "${sha256[@]}" -c checksums.txt); then
    exit 0
  fi
fi

if [[ "${NIBIRU_WASMVM_NATIVE:-false}" == true || "$os_name" == darwin && "$(uname -s)" == Darwin ]]; then
  export RUSTFLAGS="${RUSTFLAGS:-} -C relocation-model=pic"
  cargo +1.95.0 build --locked --release \
    --manifest-path "$repo_root/lib/wasmvm/libwasmvm/Cargo.toml" \
    --target "$rust_target" --example wasmvmstatic
  cp "$repo_root/lib/wasmvm/libwasmvm/target/$rust_target/release/examples/libwasmvmstatic.a" "$output_dir/$library"
elif [[ "$os_name" == linux ]]; then
  docker buildx build --platform "linux/$arch_name" \
    --file "$repo_root/contrib/docker/Dockerfile.wasmvm-source" \
    --output "type=local,dest=$output_dir" "$repo_root"
else
  echo "Darwin source builds require a native macOS host; use just wasmvm artifact-darwin for cross builds" >&2
  exit 1
fi
(
  cd "$output_dir"
  "${sha256[@]}" "$library" >checksums.txt
  printf '%s\n' "$stamp" >source-id
  printf 'commit=%s\nsource_sha256=%s\ntarget=%s\nrust=1.95.0\nwasmer=7.4.2\nwasmvm=1.5.10-nibiru.1\n' \
    "$source_commit" "$source_hash" "$rust_target" >provenance.txt
)
