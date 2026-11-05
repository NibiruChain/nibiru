#!/usr/bin/env bash
# Check stale-source and corrupt-library cache handling without compiling Rust.
set -Eeuo pipefail
script_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd -P)"
fixture_dir="$(mktemp -d)"
trap 'rm -rf -- "$fixture_dir"' EXIT
mkdir -p "$fixture_dir/repo/contrib/scripts" "$fixture_dir/bin" "$fixture_dir/output"
cp "$script_dir/build-wasmvm-source.sh" "$fixture_dir/repo/contrib/scripts/"
printf 'fixture source\n' >"$fixture_dir/repo/Cargo.toml"
git -C "$fixture_dir/repo" init -q
git -C "$fixture_dir/repo" add .
fixture_tree="$(git -C "$fixture_dir/repo" write-tree)"
fixture_commit="$(printf 'fixture\n' | git -C "$fixture_dir/repo" -c user.name=Fixture -c user.email=fixture@example.invalid commit-tree "$fixture_tree")"
git -C "$fixture_dir/repo" update-ref HEAD "$fixture_commit"
export BUILD_CALLS="$fixture_dir/calls"
cat >"$fixture_dir/bin/docker" <<'STUB'
#!/usr/bin/env bash
set -Eeuo pipefail
printf 'build\n' >> "$BUILD_CALLS"
for argument in "$@"; do
  if [[ "$argument" == type=local,dest=* ]]; then
    destination="${argument#type=local,dest=}"
    printf 'rebuilt fixture archive\n' > "$destination/libwasmvm_muslc.a"
    exit 0
  fi
done
exit 1
STUB
chmod +x "$fixture_dir/bin/docker"
export PATH="$fixture_dir/bin:$PATH"
unset NIBIRU_WASMVM_NATIVE
builder="$fixture_dir/repo/contrib/scripts/build-wasmvm-source.sh"
output="$fixture_dir/output"
# A previously downloaded archive cannot qualify as a source-built cache.
printf 'old downloaded archive\n' >"$output/libwasmvm_muslc.a"
"$builder" linux arm64 "$output"
[[ "$(wc -l <"$BUILD_CALLS")" == 1 ]]
"$builder" linux arm64 "$output"
[[ "$(wc -l <"$BUILD_CALLS")" == 1 ]]
# Corruption forces a rebuild even with a matching source stamp.
printf 'damaged archive\n' >"$output/libwasmvm_muslc.a"
"$builder" linux arm64 "$output"
[[ "$(wc -l <"$BUILD_CALLS")" == 2 ]]
# A source edit invalidates the cache without requiring a new commit.
printf 'edited source\n' >>"$fixture_dir/repo/Cargo.toml"
"$builder" linux arm64 "$output"
[[ "$(wc -l <"$BUILD_CALLS")" == 3 ]]
(cd "$output" && sha256sum -c checksums.txt)
printf 'source library cache checks passed\n'
