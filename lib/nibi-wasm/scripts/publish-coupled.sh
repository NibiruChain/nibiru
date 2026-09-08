#!/usr/bin/env bash
# Script to publish coupled packages in the correct order
# Usage: ./scripts/publish-coupled.sh [--run] [--help]

set -euo pipefail

# Default values - DRY RUN BY DEFAULT for safety
DRY_RUN=true
PACKAGES=("nibiru-ownable-derive" "nibiru-ownable")

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
WORKSPACE_DEP_VERSION=$(grep 'nibiru-ownable-derive.*=.*{ path = "packages/nibiru-ownable-derive"' Cargo.toml | sed 's/.*version = "\([^"]*\)".*/\1/' | head -1)
if [ "$WORKSPACE_DEP_VERSION" != "$VERSION" ]; then
    echo "⚠️  Warning: Workspace dependency version ($WORKSPACE_DEP_VERSION) doesn't match workspace version ($VERSION)"
    echo "   Please update the workspace dependency version in Cargo.toml to match the workspace version"
    exit 1
fi

if [ "$DRY_RUN" = true ]; then
    echo "📦 DRY RUN: Publishing coupled packages version $VERSION"
    echo "========================================================"
else
    echo "📦 Publishing coupled packages version $VERSION"
    echo "================================================"
fi

# Function to publish a package
publish_package() {
    local package=$1
    local package_dir="packages/$package"

    echo "📦 Publishing $package..."

    if [ ! -d "$package_dir" ]; then
        echo "❌ Package directory $package_dir not found"
        exit 1
    fi

    cd "$package_dir"

    if [ "$DRY_RUN" = true ]; then
        echo "🔍 Dry run: would publish $package@$VERSION"
        cargo publish --dry-run --allow-dirty
    else
        echo "🚀 Publishing $package@$VERSION to crates.io..."
        cargo publish --allow-dirty
    fi

    cd - > /dev/null
    echo "✅ $package published successfully"
    echo ""
}

# Publish packages in dependency order
for package in "${PACKAGES[@]}"; do
    publish_package "$package"
done

echo "✅ All coupled packages published successfully!"
echo ""
echo "Published packages:"
for package in "${PACKAGES[@]}"; do
    echo "  - $package@$VERSION"
done