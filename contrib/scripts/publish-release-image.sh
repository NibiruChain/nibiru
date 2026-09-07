#!/usr/bin/env bash
# Build a multi-architecture GHCR image from binaries attached to a public release.
set -Eeuo pipefail

repo="NibiruChain/nibiru"
image="ghcr.io/nibiruchain/nibiru"
release_tag=""
version=""
push=false
smoke_test=false
dry_run=false
tmp_dir=""
smoke_tags=()

usage() {
  cat <<'EOF'
Usage:
  publish-release-image.sh --release-tag <tag> --version <version> [options]

Download and verify Linux release artifacts, then optionally publish them as a
multi-architecture image. Verification is the default and never changes GHCR.

Required:
  --release-tag <tag>   Published GitHub release tag, such as hotfix/v2.19.0
  --version <version>   Image version, such as 2.19.0

Options:
  --push                Publish <image>:<version> to GHCR after verification.
  --smoke-test          Build and run each Linux image locally without pushing.
  --repo <owner/repo>   GitHub release repository. Default: NibiruChain/nibiru
  --image <image>       Container image. Default: ghcr.io/nibiruchain/nibiru
  --dry-run             Print the planned operation and exit.
  -h, --help            Show this help.

Publishing requires GHCR_TOKEN with permission to write GitHub Packages. For a
local run, use a classic personal access token. In GitHub Actions, map the
workflow GITHUB_TOKEN from a job with packages: write to GHCR_TOKEN. Set
GHCR_USERNAME to override the authenticated GitHub username used for login.
EOF
}

log() {
  printf '%s\n' "$*" >&2
}

fail() {
  log "error: $*"
  exit 1
}

cleanup() {
  local tag
  for tag in "${smoke_tags[@]}"; do
    docker image rm --force "$tag" >/dev/null 2>&1 || true
  done
  if [[ -n "$tmp_dir" && -d "$tmp_dir" ]]; then
    rm -rf -- "$tmp_dir"
  fi
}
trap cleanup EXIT

require_command() {
  command -v "$1" >/dev/null 2>&1 || fail "missing required command: $1"
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --release-tag)
      [[ $# -ge 2 ]] || fail "--release-tag requires a value"
      release_tag="$2"
      shift 2
      ;;
    --version)
      [[ $# -ge 2 ]] || fail "--version requires a value"
      version="$2"
      shift 2
      ;;
    --repo)
      [[ $# -ge 2 ]] || fail "--repo requires a value"
      repo="$2"
      shift 2
      ;;
    --image)
      [[ $# -ge 2 ]] || fail "--image requires a value"
      image="$2"
      shift 2
      ;;
    --push)
      push=true
      shift
      ;;
    --smoke-test)
      smoke_test=true
      shift
      ;;
    --dry-run)
      dry_run=true
      shift
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      fail "unknown option: $1"
      ;;
  esac
done

[[ -n "$release_tag" ]] || fail "--release-tag is required"
[[ -n "$version" ]] || fail "--version is required"
[[ "$version" =~ ^[0-9]+\.[0-9]+\.[0-9]+([-.][0-9A-Za-z.]+)?$ ]] || fail "version must be semver-like: $version"
[[ "$release_tag" == "v${version}" || "$release_tag" == */"v${version}" ]] || fail "release tag must end in v${version}"
[[ "$repo" =~ ^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$ ]] || fail "invalid repository: $repo"
[[ "$image" =~ ^ghcr.io/[A-Za-z0-9._/-]+$ ]] || fail "image must be a GHCR image: $image"
[[ "$push" == false || "$smoke_test" == false ]] || fail "--push and --smoke-test cannot be used together"

if [[ "$dry_run" == true ]]; then
  log "would verify release $repo@$release_tag and build $image:$version"
  if [[ "$push" == true ]]; then
    log "would publish $image:$version"
  fi
  if [[ "$smoke_test" == true ]]; then
    log "would build and run local linux/amd64 and linux/arm64 smoke images"
  fi
  exit 0
fi

require_command gh
require_command git
require_command jq
require_command sha256sum
require_command tar

if [[ "$push" == true || "$smoke_test" == true ]]; then
  require_command docker
fi
if [[ "$push" == true ]]; then
  [[ -n "${GHCR_TOKEN:-}" ]] || fail "GHCR_TOKEN is required with --push"
fi

release_json="$(gh release view "$release_tag" --repo "$repo" --json isDraft,assets,url)"
if [[ "$(jq -r '.isDraft' <<<"$release_json")" == "true" ]]; then
  fail "release $release_tag is still a draft; publish it before building its container"
fi

release_url="$(jq -r '.url' <<<"$release_json")"
assets=(
  "nibid_${version}_checksums.txt"
  "nibid_${version}_darwin_all.tar.gz"
  "nibid_${version}_darwin_amd64.tar.gz"
  "nibid_${version}_darwin_arm64.tar.gz"
  "nibid_${version}_linux_amd64.tar.gz"
  "nibid_${version}_linux_arm64.tar.gz"
)
for asset in "${assets[@]}"; do
  jq -e --arg name "$asset" '.assets | any(.name == $name)' <<<"$release_json" >/dev/null || fail "release is missing asset: $asset"
done

tmp_dir="$(mktemp -d -t nibiru-release-image.XXXXXX)"
download_dir="$tmp_dir/downloads"
source_dir="$tmp_dir/source"
mkdir -p "$download_dir"

download_assets=(
  "nibid_${version}_checksums.txt"
  "nibid_${version}_linux_amd64.tar.gz"
  "nibid_${version}_linux_arm64.tar.gz"
)
for asset in "${download_assets[@]}"; do
  log "downloading $asset"
  gh release download "$release_tag" --repo "$repo" --dir "$download_dir" --pattern "$asset"
done

manifest="$download_dir/nibid_${version}_checksums.txt"
linux_manifest="$download_dir/linux-checksums.txt"
grep -E "^[[:xdigit:]]{64}  nibid_${version}_linux_(amd64|arm64)\\.tar\\.gz$" "$manifest" > "$linux_manifest"
[[ "$(wc -l < "$linux_manifest")" -eq 2 ]] || fail "checksum manifest must contain exactly two Linux archives"
(cd "$download_dir" && sha256sum -c "$(basename "$linux_manifest")")

log "cloning source for $release_tag"
git clone --quiet --depth 1 --branch "$release_tag" "https://github.com/$repo.git" "$source_dir"
release_commit="$(git -C "$source_dir" rev-parse HEAD)"
mkdir -p "$source_dir/dist/amd64" "$source_dir/dist/arm64"
tar -xzf "$download_dir/nibid_${version}_linux_amd64.tar.gz" -C "$source_dir/dist/amd64"
tar -xzf "$download_dir/nibid_${version}_linux_arm64.tar.gz" -C "$source_dir/dist/arm64"
[[ -x "$source_dir/dist/amd64/nibid" ]] || fail "amd64 archive does not contain an executable nibid"
[[ -x "$source_dir/dist/arm64/nibid" ]] || fail "arm64 archive does not contain an executable nibid"

log "verified release assets from $release_url"
if [[ "$smoke_test" == true ]]; then
  for architecture in amd64 arm64; do
    platform="linux/$architecture"
    smoke_tag="nibiru-release-smoke-$$:${version}-${architecture}"
    smoke_tags+=("$smoke_tag")

    log "building local $platform image from verified artifacts"
    docker buildx build "$source_dir" \
      --target release \
      --build-arg src=external \
      --platform "$platform" \
      --tag "$smoke_tag" \
      --load

    log "running nibid version --long in local $platform image"
    version_output="$(docker run --rm --platform "$platform" "$smoke_tag" version --long)"
    printf '%s\n' "$version_output"
    grep -Fx "version: $version" <<<"$version_output" >/dev/null || fail "local $platform image reports an unexpected nibid version"
  done
  log "local smoke test passed for linux/amd64 and linux/arm64"
  exit 0
fi

if [[ "$push" == false ]]; then
  log "verification complete. Re-run with --push to publish $image:$version."
  exit 0
fi

ghcr_username="${GHCR_USERNAME:-$(gh api user --jq .login)}"
log "logging in to GHCR as $ghcr_username"
printf '%s' "$GHCR_TOKEN" | docker login ghcr.io --username "$ghcr_username" --password-stdin

log "building and publishing $image:$version from verified artifacts"
docker buildx build "$source_dir" \
  --target release \
  --build-arg src=external \
  --platform linux/amd64,linux/arm64 \
  --tag "$image:$version" \
  --label "org.opencontainers.image.source=https://github.com/$repo" \
  --label "org.opencontainers.image.url=$release_url" \
  --label "org.opencontainers.image.revision=$release_commit" \
  --label "org.opencontainers.image.version=$version" \
  --push

manifest_json="$(docker buildx imagetools inspect --raw "$image:$version")"
for architecture in amd64 arm64; do
  jq -e --arg architecture "$architecture" \
    '.manifests | any(.platform.os == "linux" and .platform.architecture == $architecture)' \
    <<<"$manifest_json" >/dev/null || fail "published image is missing linux/$architecture"
done
log "published $image:$version with linux/amd64 and linux/arm64 manifests"
