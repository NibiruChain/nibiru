#!/usr/bin/env bash
# Script to publish coupled packages in the correct order
# Usage: ./scripts/publish-coupled.sh [--run] [--help]

set -Eeuo pipefail

repo_root="$(git rev-parse --show-toplevel)"
cd "$repo_root"

# Default values - DRY RUN BY DEFAULT for safety
DRY_RUN=true
PACKAGES=("nibiru-std" "nibiru-ownable-derive" "nibiru-ownable")

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --run)
            DRY_RUN=false
            shift
            ;;
        -h|--help)
            echo "Usage: $0 [--run] [--help]"
            echo "  --run        Actually publish to crates.io (default is dry-run)"
            echo "  --help       Show this help message"
            echo ""
            echo "This script reads the workspace version from Cargo.toml and publishes"
            echo "coupled packages in dependency order. Dry-run is the default for safety."
            exit 0
            ;;
        *)
            echo "Unknown option $1"
            echo "Use --help for usage information"
            exit 1
            ;;
    esac
done

# Read workspace version from Cargo.toml
VERSION=$(grep '^package.version' Cargo.toml | sed 's/.*= *"\([^"]*\)".*/\1/')
echo "📖 Reading workspace version from Cargo.toml: $VERSION"

# Verify the workspace dependency has the correct version
for dependency in nibiru-std nibiru-ownable-derive; do
    workspace_dep_version=$(grep "^$dependency.*=.*{ path = \"lib/$dependency\"" Cargo.toml | sed 's/.*version = "\([^"]*\)".*/\1/' | head -1)
    if [ "$workspace_dep_version" != "$VERSION" ]; then
        echo "⚠️  Warning: $dependency workspace version ($workspace_dep_version) doesn't match $VERSION"
        echo "   Please update the workspace dependency version in Cargo.toml"
        exit 1
    fi
done

if [ "$DRY_RUN" = true ]; then
    echo "📦 DRY RUN: Publishing coupled packages version $VERSION"
    echo "========================================================"
else
    echo "📦 Publishing coupled packages version $VERSION"
    echo "================================================"
fi

wait_for_registry_package() {
    local package="$1"
    local attempts=30

    for ((attempt = 1; attempt <= attempts; attempt++)); do
        if cargo info "${package}@${VERSION}" >/dev/null 2>&1; then
            return 0
        fi
        echo "Waiting for ${package}@${VERSION} to reach crates.io (${attempt}/${attempts})..."
        sleep 5
    done

    echo "${package}@${VERSION} is not visible on crates.io after $((attempts * 5)) seconds." >&2
    return 1
}

publish_package() {
    local package=$1
    local package_dir="lib/$package"

    echo "📦 Publishing $package..."

    if [ ! -d "$package_dir" ]; then
        echo "❌ Package directory $package_dir not found"
        exit 1
    fi

    cd "$package_dir"

    if [ "$DRY_RUN" = false ]; then
        echo "🚀 Publishing $package@$VERSION to crates.io..."
        cargo publish --allow-dirty
    fi

    cd - > /dev/null
    echo "✅ $package published successfully"
    echo ""
}

if [ "$DRY_RUN" = true ]; then
    # cargo publish --dry-run resolves dependencies from crates.io. The final
    # package cannot resolve this release's unpublished derive crate, so dry-run
    # the publishable leaves and inspect the final package's archive instead.
    for package in nibiru-std nibiru-ownable-derive; do
        echo "🔍 Dry run: would publish $package@$VERSION"
        (
            cd "lib/$package"
            cargo publish --dry-run --allow-dirty
        )
    done

    echo "🔍 Inspecting nibiru-ownable@$VERSION package contents"
    cargo package --package nibiru-ownable --list --allow-dirty
    echo "nibiru-ownable publish validation runs after its dependencies are public."
else
    publish_package nibiru-std
    wait_for_registry_package nibiru-std
    publish_package nibiru-ownable-derive
    wait_for_registry_package nibiru-ownable-derive
    publish_package nibiru-ownable
fi

if [ "$DRY_RUN" = true ]; then
    echo "✅ Coupled release dry run passed."
    echo ""
    echo "Dry-run package set:"
else
    echo "✅ All coupled packages published successfully!"
    echo ""
    echo "Published packages:"
fi
for package in "${PACKAGES[@]}"; do
    echo "  - $package@$VERSION"
done
