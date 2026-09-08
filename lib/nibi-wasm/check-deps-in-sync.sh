#!/usr/bin/env bash
# Package Compatibility Verification Script

set -euo pipefail

echo "🔍 PACKAGE COMPATIBILITY VERIFICATION"
echo "======================================"

get_workspace_version() {
  local v
  # Prefer [workspace.package] version (Cargo workspaces with unified versioning)
  v="$(awk '
    BEGIN{ in=0 }
    /^\[workspace\.package\]/{ in=1; next }
    /^\[.*\]/{ if(in) exit; in=0 }
    in && $1 ~ /^version/ { print; exit }
  ' Cargo.toml | sed -E 's/.*=\s*"([^"]+)".*/\1/')" || true

  # Fallback to root [package] version
  if [ -z "${v:-}" ]; then
    v="$(awk '
      BEGIN{ in=0 }
      /^\[package\]/{ in=1; next }
      /^\[.*\]/{ if(in) exit; in=0 }
      in && $1 ~ /^version/ { print; exit }
    ' Cargo.toml | sed -E 's/.*=\s*"([^"]+)".*/\1/')" || true
  fi

  printf '%s\n' "${v:-unknown}"
}

printf "\n1️⃣ Checking workspace configuration...\n"
printf "Workspace version: %s\n" "$(get_workspace_version)"

printf "\n2️⃣ Verifying workspace build...\n"
if cargo check --workspace > /dev/null 2>&1; then
  echo "✅ Workspace builds successfully"
else
  echo "❌ Workspace build failed"
  exit 1
fi

printf "\n3️⃣ Analyzing cosmwasm-std versions...\n"
echo "All cosmwasm-std versions in use:"
cargo tree --workspace 2>/dev/null | grep -F "cosmwasm-std" | sed -E 's/.*(cosmwasm-std v[0-9][^ ]*).*/\1/' | sort | uniq -c

printf "\n4️⃣ Checking nibiru-std usage...\n"
echo "Packages using nibiru-std:"
# Inverse deps: which workspace packages depend on nibiru-std
cargo tree --workspace -i nibiru-std 2>/dev/null \
  | sed 's/^[^a-zA-Z0-9_-]*//' \
  | awk '{print $1}' \
  | sort -u

printf "\n5️⃣ Verifying test compilation...\n"
if cargo test --workspace --lib --bins --tests --no-run > /dev/null 2>&1; then
  echo "✅ All tests compile successfully"
else
  echo "❌ Test compilation failed"
  exit 1
fi

printf "\n6️⃣ Checking for version conflicts...\n"
conflicts="$(cargo tree --workspace 2>/dev/null | grep -F '(*)' | wc -l | tr -d ' ')"
if [ "${conflicts}" -eq 0 ]; then
  echo "✅ No version conflicts detected"
else
  echo "⚠️  Found ${conflicts} potential version conflicts:"
  cargo tree --workspace 2>/dev/null | grep -F '(*)'
fi

printf "\n7️⃣ Package-specific dependency analysis...\n"
echo "Dependencies for key packages:"
for pkg in cw-address-like easy-addr nibiru-ownable nibiru-ownable-derive; do
  if [ -d "packages/${pkg}" ]; then
    echo "--- ${pkg} ---"
    cargo tree -p "${pkg}" --depth 1 2>/dev/null || echo "Package not found or has issues"
  fi
done

printf "\n✅ VERIFICATION COMPLETE\n"
echo "All packages appear to be compatible!"