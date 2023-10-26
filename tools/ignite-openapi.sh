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

# ensure_path: Ensures that the script execution starts from the root of the
# repo.
ensure_path() {

  # Get the current directory
  local current_dir
  current_dir=$(pwd)

  # Get the directory of the script
  SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

  # # Ensure the script is being run from the root of the repo
  # if [ "$current_dir" != "${SCRIPT_DIR%/tools}" ]; then
  #   echo "Error: You must run this script from the repo root."
  #   return 1
  # fi
  # Get the root directory of the repo
  local repo_root
  repo_root="${SCRIPT_DIR%/tools}"


  # Check if we're in the root directory or tools directory of the repo
  if [ "$current_dir" == "$repo_root" ]; then
      # We're in the root, nothing to do
      :
  elif [ "$current_dir" == "$SCRIPT_DIR" ]; then
      # We're in the tools directory, move one level up to the root
      cd "$repo_root"
  else
      # We're neither in the root nor in the tools directory
      log_error "You must run this script from the repo root or the tools directory."
      log_debug "current_dir: $current_dir"
      log_debug "SCRIPT_DIR: $SCRIPT_DIR"
      log_debug "repo_root: $repo_root"
      exit 1
  fi
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

# ensure_deps: Make sure expected dependencies are installed.
ensure_deps() {
  if ! which_ok ignite; then 
    log_debug "Installing Ignite CLI (ignite)..."
    curl https://get.ignite.com/cli! | bash
    if ! which_ok ignite; then
      log_error "Failed to install ignite."
      return 1
    fi
  fi

  which_ok nibid
}

# Ignite expects files at the base of the repo. Rather than keeping a config
# there just for the Ignite CLI, it's cleaner to symlink the file just for the
# script to run.
symlink_config_to_repo_base() {
  local source_file
  source_file="tools/ignite-config.yml"
  if [[ ! -f "$source_file" ]]; then
    log_error "Source file '$source_file' does not exist."
    return 1
  fi

  ln -sf $source_file ./config.yml
  log_success "symlinked config"
}

check_ignite_config_health() {
  ignite doctor
}

# ------------------------------    main    -----------------------------
ensure_path
ensure_deps
symlink_config_to_repo_base
check_ignite_config_health
ignite generate openapi --yes 
log_success "Completed generation of OpenAPI spec"
