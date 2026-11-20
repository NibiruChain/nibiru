#!/usr/bin/env bash
#
# build-nibiru.sh - Build or install the nibiru binary.
#
# The binary is named nibiru. A symlink named nibid sits beside it, because
# cosmovisor layouts, service files, and CI call nibid.
#
# Optional environment variables (inherited when set):
#   VERSION   Release version string (default: latest git tag, or branch-commit)
#   BUILDDIR  Output directory (default: $REPO_ROOT/build)
#   TEMPDIR   WasmVM artifact cache (default: $REPO_ROOT/temp)
#   NIBIRU_WASMVM_BUILD_FROM_SOURCE  Build the vendored runtime (default: false).
#   GOARCH    Target GOARCH for cross-compilation
#   GOOS      Target GOOS for cross-compilation
#   NIBID_STATIC_PIE  Build a Linux static PIE (default: false). Requires a
#                     supported musl toolchain such as the release container.
#   NIBID_EXTRA_TAGS  Space-separated Go build tags added to the defaults.
#                     Test-only, e.g. "nibiru_fault_injection".
#
# Usage:
#   contrib/scripts/build-nibiru.sh
#   contrib/scripts/build-nibiru.sh --help
#   contrib/scripts/build-nibiru.sh --run
#   contrib/scripts/build-nibiru.sh --run --just-build

set -Eeuo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd -P)"
SCRIPT_NAME="$(basename -- "${BASH_SOURCE[0]}")"
REPO_ROOT="$(cd -- "$SCRIPT_DIR/../.." && pwd -P)"

RUN=false
JUST_BUILD=false

trap 'log_error "failed on line $LINENO"; exit 1' ERR

# log_info: Print an informational message to stderr.
log_info() {
  printf '[build-nibiru] %s\n' "$*" >&2
}

# log_error: Print an error message to stderr.
log_error() {
  printf '[build-nibiru] ERROR: %s\n' "$*" >&2
}

# log_success: Print a success message to stderr.
log_success() {
  printf '[build-nibiru] OK: %s\n' "$*" >&2
}

# which_ok: Check if the given binary is in the $PATH or if it is something
# callable in a bash program.
# Returns code 0 on success and code 1 if the command fails.
which_ok() {

  # Runnable binary on $PATH? Ex: "jq", "bun", etc.
  # Alias? Ex: "ls" (I have it aliased to exa).
  # Built-in? Ex: "echo", "cd"
  if which "$1" >/dev/null 2>&1; then
    return 0
  fi

  # Function? An example for this is "nvm", which is a pure bash function.
  if type -a "$1" >/dev/null; then
    return 0
  fi

  log_error "$1 is not present in \$PATH"
  return 1
}

# require_cmds: Fail if any required command is missing.
require_cmds() {
  local cmd
  for cmd in "$@"; do
    which_ok "$cmd" || exit 1
  done
}

show_help() {
  cat <<EOF
Usage: $SCRIPT_NAME [OPTIONS]

Build or install the nibiru binary and its nibid symlink.

With no arguments, this script prints help and exits. Use --run to execute.

Options:
  --run         Build and install nibiru, plus a nibid symlink, to the Go binary
                directory.
  --just-build  Build ./build/nibiru and a ./build/nibid symlink instead of
                installing (requires --run).
  -h, --help    Show this help message and exit.

Environment:
  VERSION, BUILDDIR, TEMPDIR, GOARCH, GOOS, NIBID_STATIC_PIE - see script header comment.
EOF
}

parse_args() {
  if [[ $# -eq 0 ]]; then
    show_help
    exit 0
  fi

  while [[ $# -gt 0 ]]; do
    case "$1" in
    -h | --help)
      show_help
      exit 0
      ;;
    --run)
      RUN=true
      shift
      ;;
    --just-build)
      JUST_BUILD=true
      shift
      ;;
    *)
      log_error "unknown argument: $1"
      show_help
      exit 1
      ;;
    esac
  done

  if [[ "$JUST_BUILD" == true && "$RUN" != true ]]; then
    log_error "--just-build requires --run"
    show_help
    exit 1
  fi

  if [[ "$RUN" != true ]]; then
    show_help
    exit 0
  fi
}

# detect_os_name: Lowercase GOOS or uname -s (e.g. linux, darwin).
detect_os_name() {
  if [[ -n "${GOOS:-}" ]]; then
    printf '%s' "$GOOS"
    return 0
  fi
  uname -s | tr '[:upper:]' '[:lower:]'
}

# detect_arch_name: Platform lib directory suffix used by build.mk.
detect_arch_name() {
  local arch
  if [[ -n "${GOARCH:-}" ]]; then
    arch="$GOARCH"
  else
    arch="$(uname -m)"
  fi
  case "$arch" in
  x86_64 | amd64) printf '%s' "amd64" ;;
  *) printf '%s' "arm64" ;;
  esac
}

# compute_version: VERSION env, or git describe, or branch-commit fallback.
compute_version() {
  local branch commit described

  if [[ -n "${VERSION:-}" ]]; then
    printf '%s' "$VERSION"
    return 0
  fi

  if ! command -v git >/dev/null 2>&1 || ! git -C "$REPO_ROOT" rev-parse --git-dir >/dev/null 2>&1; then
    printf '%s' "unknown-version"
    return 0
  fi

  branch="$(git -C "$REPO_ROOT" rev-parse --abbrev-ref HEAD 2>/dev/null || printf '%s' "unknown")"
  commit="$(git -C "$REPO_ROOT" log -1 --format='%H' 2>/dev/null || printf '%s' "unknown")"
  described="$(git -C "$REPO_ROOT" describe --tags --match 'v[0-9]*' --abbrev=0 2>/dev/null || true)"
  if [[ -z "$described" ]]; then
    printf '%s-%s' "$branch" "$commit"
  else
    printf '%s' "$described"
  fi
}

# compute_commit: Return the git commit hash, or "unknown" outside git checkouts.
compute_commit() {
  if ! command -v git >/dev/null 2>&1 || ! git -C "$REPO_ROOT" rev-parse --git-dir >/dev/null 2>&1; then
    printf '%s' "unknown"
    return 0
  fi

  git -C "$REPO_ROOT" log -1 --format='%H' 2>/dev/null || printf '%s' "unknown"
}

# build_tags_for_os: Space-separated build tags matching build.mk.
build_tags_for_os() {
  local os_name="$1"
  if [[ "$os_name" == "darwin" ]]; then
    printf '%s' "netgo osusergo ledger static pebbledb static_wasm grocksdb_no_link"
  else
    printf '%s' "netgo osusergo ledger static pebbledb muslc"
  fi
}

# build_tags_csv: Comma-separated tags for ldflags BuildTags injection.
build_tags_csv() {
  local tags="$1"
  printf '%s' "${tags// /,}"
}

ensure_build_dir() {
  local builddir="$1"
  mkdir -p "$builddir"
}

ensure_temp_dir() {
  local tempdir="$1"
  mkdir -p "$tempdir"
}

# ensure_wasmvm_lib: Download a pinned, checksum-verified static runtime.
ensure_wasmvm_lib() (
  local tempdir="$1" os_name="$2" arch_name="$3" wasmvm_version="$4"
  local libdir="$tempdir/wasmvm/$wasmvm_version/lib/${os_name}_${arch_name}"
  if [[ "$wasmvm_version" == source ]]; then
    "$SCRIPT_DIR/build-wasmvm-source.sh" "$os_name" "$arch_name" "$libdir"
    return
  fi

  local artifact library digest actual download_dir
  case "$os_name:$arch_name" in
    linux:amd64) artifact="libwasmvm_muslc.x86_64.a"; library="libwasmvm_muslc.a" ;;
    linux:arm64) artifact="libwasmvm_muslc.aarch64.a"; library="libwasmvm_muslc.a" ;;
    darwin:*) artifact="libwasmvmstatic_darwin.a"; library="$artifact" ;;
    *) log_error "unsupported WasmVM platform: $os_name/$arch_name"; return 1 ;;
  esac
  digest="$(awk -v name="$artifact" '$2 == name {print $1}' "$SCRIPT_DIR/wasmvm-checksums.txt")"
  [[ "$digest" =~ ^[a-f0-9]{64}$ ]] || { log_error "missing pinned checksum for $artifact"; return 1; }
  hash_library() {
    if command -v sha256sum >/dev/null 2>&1; then
      sha256sum "$1" | awk '{print $1}'
    else
      shasum -a 256 "$1" | awk '{print $1}'
    fi
  }
  mkdir -p "$libdir"
  if [[ -f "$libdir/$library" ]] && [[ "$(hash_library "$libdir/$library")" == "$digest" ]]; then
    log_info "using verified WasmVM $wasmvm_version cache"
    return
  fi
  download_dir="$(mktemp -d "$libdir/download.XXXXXX")"
  trap 'rm -rf -- "$download_dir"' EXIT
  log_info "downloading WasmVM $wasmvm_version $artifact"
  wget -q -O "$download_dir/$library" "https://github.com/NibiruChain/nibiru/releases/download/lib/wasmvm/$wasmvm_version/$artifact"
  actual="$(hash_library "$download_dir/$library")"
  [[ "$actual" == "$digest" ]] || { log_error "WasmVM checksum mismatch: $artifact"; return 1; }
  mv "$download_dir/$library" "$libdir/$library"
)

verify_go_modules() {
  log_info "verifying go modules"
  (cd "$REPO_ROOT" && go mod verify)
}

run_go_compile() {
  local builddir="$1"
  local tempdir="$2"
  local os_name="$3"
  local arch_name="$4"
  local version="$5"
  local commit="$6"
  local build_tags="$7"
  local build_tags_csv="$8"
  local wasmvm_version="$9"
  local just_build="${10}"
  local static_pie="${11}"

  local cgo_ldflags ldflags go_build_mode out_dir
  go_build_mode=""

  cgo_ldflags="-L${tempdir}/wasmvm/${wasmvm_version}/lib/${os_name}_${arch_name}/"

  if [[ "$static_pie" == true ]]; then
    # Keep CGO library resolution separate from the final link mode. The
    # external linker receives -static-pie below, which makes the complete
    # Linux executable position independent instead of producing ET_EXEC.
    cgo_ldflags+=" -lm"
    go_build_mode="-buildmode=pie"
  elif [[ "$os_name" == "linux" ]]; then
    cgo_ldflags+=" -static -lm"
  fi

  ldflags="-X github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/version.Name=nibiru"
  ldflags+=" -X github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/version.AppName=nibid"
  ldflags+=" -X github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/version.Version=${version}"
  ldflags+=" -X github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/version.Commit=${commit}"
  ldflags+=" -X github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/version.BuildTags=${build_tags_csv}"
  # Nibiru BFT version tracks the nibid version, as bare semver (no "v").
  ldflags+=" -X github.com/NibiruChain/nibiru/v2/bft/version.NibiruBFTSemVer=${version#v}"
  # Logged as "commit_hash" when the consensus node starts.
  ldflags+=" -X github.com/NibiruChain/nibiru/v2/bft/version.TMGitCommitHash=${commit}"
  ldflags+=" -X github.com/cosmos/cosmos-sdk/types.DBBackend=pebbledb"
  ldflags+=" -linkmode=external -w -s"

  if [[ "$static_pie" == true ]]; then
    ldflags+=" -extldflags '-Wl,-z,muldefs -static-pie -z noexecstack'"
  fi

  if [[ "$just_build" == true ]]; then
    out_dir="$builddir"
    log_info "compiling nibiru to ${out_dir}/ (with a nibid symlink)"
  else
    out_dir="$(go_bin_dir)"
    mkdir -p "$out_dir"
    log_info "installing nibiru to ${out_dir}/nibiru (with a nibid symlink)"
  fi

  (
    cd "$REPO_ROOT"
    export GO111MODULE=on
    export CGO_ENABLED=1
    export CGO_LDFLAGS="$cgo_ldflags"

    go build \
      -mod=readonly \
      -trimpath \
      ${go_build_mode:+"$go_build_mode"} \
      -tags "$build_tags" \
      -ldflags "$ldflags" \
      -o "${out_dir}/nibiru" \
      ./x/cli
  )

  link_nibid "$out_dir"
}

# go_bin_dir: Directory where `go install` places binaries.
go_bin_dir() {
  local dir
  dir="$(go env GOBIN)"
  if [[ -z "$dir" ]]; then
    dir="$(go env GOPATH)/bin"
  fi
  printf '%s' "$dir"
}

# resolve_path: Print the path with symbolic links followed.
resolve_path() {
  readlink -f "$1" 2>/dev/null || realpath "$1" 2>/dev/null || printf '%s' "$1"
}

# path_copies: Print every executable named $1 on PATH, one per line, in PATH
# order. Duplicates that resolve to the same file print once.
path_copies() {
  local p seen=$'\n' real
  while IFS= read -r p; do
    real="$(resolve_path "$p")"
    [[ "$seen" == *$'\n'"$real"$'\n'* ]] && continue
    seen+="$real"$'\n'
    printf '%s\n' "$p"
  done < <(type -ap -- "$1" 2>/dev/null || true)
}

# check_path_name: Report how one name resolves on PATH relative to the file
# just installed ($2 is the installed path, $3 is its directory). Prints
# machine-readable marker lines to stdout:
#   nibiru-path-shadowed name=<n> runs=<path> installed=<path>
#   nibiru-path-ok name=<n> runs=<path> other=<path>
# and the human-readable warning to stderr. A copy that resolves to the
# installed file, such as the nibid symlink, is never "another copy".
check_path_name() {
  local name="$1" installed="$2" bin_dir="$3"
  local yellow="" cyan="" reset=""
  if [[ -t 2 && -z "${NO_COLOR:-}" ]]; then
    yellow=$'\033[33m' cyan=$'\033[36m' reset=$'\033[0m'
  fi

  local want runs
  want="$(resolve_path "$installed")"
  runs="$(command -v "$name" || true)"
  [[ -z "$runs" ]] && return 0

  if [[ "$(resolve_path "$runs")" != "$want" ]]; then
    printf 'nibiru-path-shadowed name=%s runs=%s installed=%s\n' \
      "$name" "$runs" "$installed"
    printf '%s[build-nibiru] WARNING: %s on PATH is %s, not the build just installed at %s.%s\n' \
      "$yellow" "$name" "$runs" "$installed" "$reset" >&2
    printf '  Fix it by putting %s earlier on PATH, or run the build by full path:\n' \
      "$bin_dir" >&2
    printf '    %s%s version --long%s\n' "$cyan" "$installed" "$reset" >&2
    return 0
  fi

  local other
  while IFS= read -r other; do
    [[ "$(resolve_path "$other")" == "$want" ]] && continue
    printf 'nibiru-path-ok name=%s runs=%s other=%s\n' "$name" "$runs" "$other"
    break
  done < <(path_copies "$name")
}

# warn_if_shadowed: Check nibiru and nibid on PATH. Never fails the install.
warn_if_shadowed() {
  local bin_dir="$1" name
  for name in nibiru nibid; do
    check_path_name "$name" "$bin_dir/$name" "$bin_dir"
  done
}

# link_nibid: Point ${1}/nibid at nibiru, replacing a previous nibid file or
# link. The link is relative, so the directory can move.
link_nibid() {
  local dir="$1"
  ln -sfn nibiru "${dir}/nibid"
}

main() {
  parse_args "$@"

  require_cmds go wget jq

  cd "$REPO_ROOT"

  local builddir="${BUILDDIR:-$REPO_ROOT/build}"
  local tempdir="${TEMPDIR:-$REPO_ROOT/temp}"
  local os_name arch_name version commit wasmvm_version build_tags tags_csv static_pie

  os_name="$(detect_os_name)"
  if [[ "$os_name" == darwin ]]; then
    export MACOSX_DEPLOYMENT_TARGET=14.5
  fi
  arch_name="$(detect_arch_name "$os_name")"
  version="$(compute_version)"
  commit="$(compute_commit)"
  wasmvm_version="v1.13.2"
  case "${NIBIRU_WASMVM_BUILD_FROM_SOURCE:-false}" in
    true) wasmvm_version="source" ;;
    false) ;;
    *) log_error "NIBIRU_WASMVM_BUILD_FROM_SOURCE must be true or false"; exit 1 ;;
  esac
  build_tags="$(build_tags_for_os "$os_name")"
  if [[ -n "${NIBID_EXTRA_TAGS:-}" ]]; then
    build_tags+=" ${NIBID_EXTRA_TAGS}"
    log_info "extra build tags: ${NIBID_EXTRA_TAGS}"
  fi
  tags_csv="$(build_tags_csv "$build_tags")"
  static_pie="${NIBID_STATIC_PIE:-false}"

  if [[ "$static_pie" != true && "$static_pie" != false ]]; then
    log_error "NIBID_STATIC_PIE must be true or false, got: $static_pie"
    exit 1
  fi
  if [[ "$static_pie" == true && "$os_name" != "linux" ]]; then
    log_error "NIBID_STATIC_PIE is supported only for Linux builds"
    exit 1
  fi

  verify_go_modules
  ensure_build_dir "$builddir"
  ensure_temp_dir "$tempdir"
  ensure_wasmvm_lib "$tempdir" "$os_name" "$arch_name" "$wasmvm_version"
  run_go_compile "$builddir" "$tempdir" "$os_name" "$arch_name" \
    "$version" "$commit" "$build_tags" "$tags_csv" "$wasmvm_version" "$JUST_BUILD" "$static_pie"

  local bin_dir bin_path
  if [[ "$JUST_BUILD" == true ]]; then
    bin_dir="$builddir"
  else
    bin_dir="$(go_bin_dir)"
  fi
  bin_path="${bin_dir}/nibiru"

  if [[ ! -x "$bin_path" || ! -L "${bin_dir}/nibid" ]]; then
    log_error "expected binary and nibid symlink not found in ${bin_dir}"
    exit 1
  fi

  log_success "built ${bin_path} (nibid -> nibiru)"
  # Run the checks on the real file.
  "$bin_path" version
  if [[ "$static_pie" == true ]]; then
    "$SCRIPT_DIR/nibid-elf-test.sh" --bin "$bin_path"
  fi

  if [[ "$JUST_BUILD" != true ]]; then
    command -v nibid >/dev/null 2>&1 || {
      log_error "nibid was installed, but it is not present in PATH"
      exit 1
    }
    warn_if_shadowed "$bin_dir"
  fi
}

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
  main "$@"
fi
