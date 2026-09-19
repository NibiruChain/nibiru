# Publishing coupled Rust crates

- Status: active
- Code scope: root `Cargo.toml`, crates `lib/nibiru-std/`,
  `lib/nibiru-ownable-derive/`, and `lib/nibiru-ownable/`
- Source of truth: script `contrib/scripts/publish-coupled.sh`

Three crates share the root workspace version and must publish in dependency
order:

```text
nibiru-std -> nibiru-ownable-derive -> nibiru-ownable
```

Crate `nibiru-ownable` exposes type `UserAddr` from crate `nibiru-std` and
depends on the procedural macros in crate `nibiru-ownable-derive`.

## Before publishing

1. Set `workspace.package.version` in root file `Cargo.toml` to the intended
   release version.
2. Keep the `nibiru-std` and `nibiru-ownable-derive` workspace dependency
   versions in file `Cargo.toml` equal to that release version.
3. Run the focused crate tests, documentation build, format check, and Clippy.
4. Commit and push the reviewed release changes.
5. Confirm that crates.io does not already contain the target versions.

## Dry run

From the repository root, run:

```bash
just rs publish
```

The command invokes script `contrib/scripts/publish-coupled.sh`. It dry-runs
crates `nibiru-std` and `nibiru-ownable-derive`, then inspects the package
contents for crate `nibiru-ownable`. Cargo cannot dry-run the final crate while
its release-version derive dependency is absent from crates.io.

## Publish

After the dry run and CI pass, authenticate Cargo for the Nibiru crates.io
owner account and run:

```bash
just rs publish-run
```

The script publishes the crates in dependency order and waits for each version
to appear on crates.io before publishing its dependent crate. A crates.io
publication cannot be undone.

## Verify

```bash
cargo search nibiru-std --limit 1
cargo search nibiru-ownable-derive --limit 1
cargo search nibiru-ownable --limit 1
```

Then update downstream contracts from local path dependencies to the published
version and run their focused tests.
