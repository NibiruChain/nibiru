# Publishing the coupled Rust crates

Version `0.8.0` publishes three coupled crates:

```text
nibiru-std -> nibiru-ownable-derive -> nibiru-ownable
```

`nibiru-ownable` exposes type `UserAddr` from `nibiru-std` and depends on the
procedural macros in `nibiru-ownable-derive`. Publish all three at the same
workspace version, in that order.

## Before publishing

1. Set `workspace.package.version` and the `nibiru-std` and
   `nibiru-ownable-derive` workspace dependency versions in root file
   `Cargo.toml` to the release version.
2. Run the focused crate tests, documentation build, format check, and Clippy.
3. Commit the release changes and push the reviewed commit.
4. Confirm that the target version does not already exist on crates.io.

## Dry run

From the repository root, run:

```bash
just rs publish
```

The command runs file `contrib/scripts/publish-coupled.sh`. It dry-runs
`nibiru-std` and `nibiru-ownable-derive`, then lists the
`nibiru-ownable` package contents. Cargo cannot dry-run the final crate until
its unpublished 0.8.0 derive dependency is available from crates.io.

## Publish

After the dry run and CI pass, use a Cargo session authenticated for the Nibiru
crates.io owner account:

```bash
just rs publish-run
```

This command invokes `cargo publish --allow-dirty` for `nibiru-std`,
`nibiru-ownable-derive`, then `nibiru-ownable`. The script waits for each
dependency version to appear on crates.io before it publishes the dependent
crate. Publishing cannot be undone.

## Verify

```bash
cargo search nibiru-std --limit 1
cargo search nibiru-ownable-derive --limit 1
cargo search nibiru-ownable --limit 1
```

Then update downstream contracts from local path dependencies to the published
version and rerun their focused tests.
