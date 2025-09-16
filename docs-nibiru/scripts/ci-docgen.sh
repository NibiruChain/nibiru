#!/usr/bin/env bash
set -e

# Generate nibijs docs from a fresh clone of ts-sdk
generate_nibijs_docs() {
  if yarn docgen-nibijs; then
    echo "✅ Success - generated docs"
  else
    echo "❌ Failure - failed to generate docs"
    exit 1
  fi
}

# Verify that no files have changed
verify_no_changes() {
  git update-index -q --refresh
  local changes
  local changes_to_nibijs
  changes=$(git diff-index --name-only HEAD --)
  changes_to_nibijs=$(echo "$changes" | grep "dev/tools/nibijs")

  if [ -n "$changes_to_nibijs" ]; then
    echo "❌ Generated nibijs documentation differs the those on the git HEAD. \
      Try running \"yarn docgen-nibijs\" in the docs-nibiru directory and \
      committing any new changes."
    echo "changes to nibijs: "
    echo "$changes_to_nibijs"
    exit 1
  else
    echo "✅ Success - generated docs match the ones on the git HEAD."
    exit 0
  fi
}

generate_nibijs_docs
verify_no_changes
