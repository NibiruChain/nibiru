#!/usr/bin/env bash
#
# build-nibiru.sh - Build or install the nibid binary.
#
# Optional environment variables (inherited when set):
#   VERSION   Release version string (default: latest git tag, or branch-commit)
#   BUILDDIR  Output directory (default: $REPO_ROOT/build)
#   TEMPDIR   WasmVM artifact cache (default: $REPO_ROOT/temp)
#   GOARCH    Target GOARCH for cross-compilation
#   GOOS      Target GOOS for cross-compilation
#   NIBID_STATIC_PIE  Build a Linux static PIE (default: false). Requires a
#                     supported musl toolchain such as the release container.
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

Build or install the nibid binary.

With no arguments, this script prints help and exits. Use --run to execute.

Options:
  --run         Build and install nibid to the Go binary directory.
  --just-build  Build nibid to ./build/nibid instead of installing (requires --run).
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
  local os_name="$1"
  if [[ "$os_name" == "darwin" ]]; then
    printf '%s' "all"
    return 0
  fi
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

  if [[ ! -d "$REPO_ROOT/.git" ]] || ! command -v git >/dev/null 2>&1; then
    printf '%s' "unknown-version"
    return 0
  fi

  branch="$(git -C "$REPO_ROOT" rev-parse --abbrev-ref HEAD 2>/dev/null || printf '%s' "unknown")"
  commit="$(git -C "$REPO_ROOT" log -1 --format='%H' 2>/dev/null || printf '%s' "unknown")"
  described="$(git -C "$REPO_ROOT" describe --tags --abbrev=0 2>/dev/null || true)"
  if [[ -z "$described" ]]; then
    printf '%s-%s' "$branch" "$commit"
  else
    printf '%s' "$described"
  fi
}

# compute_commit: Return the git commit hash, or "unknown" outside git checkouts.
compute_commit() {
  if [[ ! -d "$REPO_ROOT/.git" ]] || ! command -v git >/dev/null 2>&1; then
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

# ensure_wasmvm_lib: Download wasmvm static lib if missing.
ensure_wasmvm_lib() {
  local tempdir="$1"
  local os_name="$2"
  local arch_name="$3"
  local wasmvm_version="$4"

  local lib_dir wasmvm_gh_tag wasmvm_gh_tag_url base_url
  lib_dir="$tempdir/wasmvm/$wasmvm_version/lib/${os_name}_${arch_name}"
  wasmvm_gh_tag="lib/wasmvm/${wasmvm_version}"
  wasmvm_gh_tag_url=$(jq -nr --arg s "$wasmvm_gh_tag" '$s|@uri')
  base_url="https://github.com/NibiruChain/nibiru/releases/download/${wasmvm_gh_tag_url}"

  mkdir -p "$lib_dir"

  # shellcheck disable=SC2086
  if compgen -G "$lib_dir/libwasmvm*.a" >/dev/null; then
    return 0
  fi

  log_info "downloading wasmvm v$wasmvm_version (${os_name}_${arch_name})"
  if [[ "$os_name" == "darwin" ]]; then
    wget "${base_url}/libwasmvmstatic_darwin.a" \
      -O "$lib_dir/libwasmvmstatic_darwin.a"
  elif [[ "$arch_name" == "amd64" ]]; then
    wget "${base_url}/libwasmvm_muslc.x86_64.a" \
      -O "$lib_dir/libwasmvm_muslc.a"
  else
    wget "${base_url}/libwasmvm_muslc.aarch64.a" \
      -O "$lib_dir/libwasmvm_muslc.a"
  fi
}

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
  local cmt_version="$7"
  local build_tags="$8"
  local build_tags_csv="$9"
  local wasmvm_version="${10}"
  local just_build="${11}"
  local static_pie="${12}"

  local cgo_ldflags ldflags go_build_mode install_dir
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
  ldflags+=" -X github.com/cometbft/cometbft/version.CMTSemVer=${cmt_version}"
  ldflags+=" -X github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types.DBBackend=pebbledb"
  ldflags+=" -linkmode=external -w -s"

  if [[ "$static_pie" == true ]]; then
    ldflags+=" -extldflags '-Wl,-z,muldefs -static-pie -z noexecstack'"
  fi

  if [[ "$just_build" == true ]]; then
    log_info "compiling nibid to ${builddir}/"
  else
    install_dir="$(go env GOBIN)"
    if [[ -z "$install_dir" ]]; then
      install_dir="$(go env GOPATH)/bin"
    fi
    mkdir -p "$install_dir"
    log_info "installing nibid to ${install_dir}/nibid"
  fi

  (
    cd "$REPO_ROOT"
    export GO111MODULE=on
    export CGO_ENABLED=1
    export CGO_LDFLAGS="$cgo_ldflags"

    if [[ "$just_build" == true ]]; then
      go build \
        -mod=readonly \
        -trimpath \
        ${go_build_mode:+"$go_build_mode"} \
        -tags "$build_tags" \
        -ldflags "$ldflags" \
        -o "${builddir}/nibid" \
        ./x/cli
    else
      go build \
        -mod=readonly \
        -trimpath \
        ${go_build_mode:+"$go_build_mode"} \
        -tags "$build_tags" \
        -ldflags "$ldflags" \
        -o "${install_dir}/nibid" \
        ./x/cli
    fi
  )
}

main() {
  parse_args "$@"

  require_cmds go wget jq

  cd "$REPO_ROOT"

  local builddir="${BUILDDIR:-$REPO_ROOT/build}"
  local tempdir="${TEMPDIR:-$REPO_ROOT/temp}"
  local os_name arch_name version commit cmt_version wasmvm_version build_tags tags_csv static_pie

  os_name="$(detect_os_name)"
  arch_name="$(detect_arch_name "$os_name")"
  version="$(compute_version)"
  commit="$(compute_commit)"
  cmt_version="$(go list -m github.com/cometbft/cometbft | sed 's:.* ::')"
  wasmvm_version="v1.12.0" # tag name `lib/wasmvm/v*`
  build_tags="$(build_tags_for_os "$os_name")"
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
    "$version" "$commit" "$cmt_version" "$build_tags" "$tags_csv" "$wasmvm_version" "$JUST_BUILD" "$static_pie"

  if [[ "$JUST_BUILD" == true ]]; then
    if [[ ! -x "${builddir}/nibid" ]]; then
      log_error "expected binary not found: ${builddir}/nibid"
      exit 1
    fi

    log_success "built ${builddir}/nibid"
    "${builddir}/nibid" version
    if [[ "$static_pie" == true ]]; then
      "$SCRIPT_DIR/nibid-elf-test.sh" --bin "${builddir}/nibid"
    fi
  else
    local nibid_path
    nibid_path="$(command -v nibid)" || {
      log_error "nibid was installed, but it is not present in PATH"
      exit 1
    }

    log_success "installed $nibid_path"
    "$nibid_path" version
    if [[ "$static_pie" == true ]]; then
      "$SCRIPT_DIR/nibid-elf-test.sh" --bin "$nibid_path"
    fi
  fi
}

main "$@"
