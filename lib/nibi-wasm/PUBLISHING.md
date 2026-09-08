# Publishing Guide for Coupled Packages

This document explains how to publish the coupled packages `nibiru-ownable-derive` and `nibiru-ownable` that share the same version.

## The Problem

When publishing packages that depend on each other, Cargo requires:
1. Dependencies to be published first
2. Exact version numbers (not just `{ workspace = true }`)

## Solution: Automated Publishing Script

We've created a script that handles the publishing order automatically.

### Quick Commands

```bash
# Dry run (default - safe to run anytime)
just publish

# Actually publish to crates.io
just publish-run
```

### Manual Usage

```bash
# Dry run (default behavior - safe)
./scripts/publish-coupled.sh

# Actually publish to crates.io
./scripts/publish-coupled.sh --run

# Show help
./scripts/publish-coupled.sh --help
```

## Workflow for New Versions

### 1. Update Workspace Version

```bash
# Edit Cargo.toml and update:
package.version = "X.Y.Z"
```

### 2. Update Workspace Dependency Version

```bash
# Also update the workspace dependency version to match:
nibiru-ownable-derive = { path = "packages/nibiru-ownable-derive", version = "X.Y.Z" }
```

### 3. Publish Coupled Packages

```bash
# Test first (dry run is default - safe to run)
just publish

# If everything looks good, actually publish
just publish-run
```

### 4. Verify Publication

Check that both packages are published on crates.io:
- https://crates.io/crates/nibiru-ownable-derive
- https://crates.io/crates/nibiru-ownable

## Alternative: Using cargo-workspaces

If you prefer a more general solution:

```bash
# Install the tool
cargo install cargo-workspaces

# Publish all workspace packages in dependency order
cargo workspaces publish --from-git
```

## How It Works

The script:
1. Reads the workspace version from `Cargo.toml`
2. Verifies the workspace dependency version matches the workspace version
3. Publishes `nibiru-ownable-derive` first (the dependency)
4. Publishes `nibiru-ownable` second (the dependent package)
5. Uses dry-run by default for safety

## Troubleshooting

### "Package already exists"
If you get this error, you need to bump the version number.

### "Dependency not found"
Make sure `nibiru-ownable-derive` is published before `nibiru-ownable`.

### "Version mismatch"
Ensure both packages use the same version in their `Cargo.toml` files.
