#!/usr/bin/env bash
# Install tools for publishing workspace packages

echo "Installing cargo-workspaces for workspace publishing..."
cargo install cargo-workspaces

echo "Installing cargo-publish-all as alternative..."
cargo install cargo-publish-all

echo "Tools installed! You can now use:"
echo "  cargo workspaces publish --help"
echo "  cargo publish-all --help"

