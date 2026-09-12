# Nibiru CLI distribution

This private Bun workspace owns the maintainer side of CLI distribution. It
turns immutable GitHub release archives into npm packages today, and keeps the
shared release provenance, cache, verification, and binary-test work in one
place for later distribution channels such as Brew. The public npm package is
`@nibiruchain/nibiru`. Its Node launcher resolves the matching platform package
and starts its native `nibid` binary.

Run the release CLI from this directory. It accepts an exact GitHub release tag,
a unique release version, or a Nibiru GitHub release URL. Exact tags avoid
ambiguity when a hotfix and a standard release share the same version.

```bash
bun run nibiru-dist --help
bun run nibiru-dist list
bun run nibiru-dist list --verify
bun run nibiru-dist get hotfix/v2.19.0
bun run nibiru-dist prepare hotfix/v2.19.0
bun run nibiru-dist publish --from-gh hotfix/v2.19.0 --dist-ver 2.19.0
bun run nibiru-dist publish --from-gh hotfix/v2.19.0 --dist-ver v2.19.0 --run
```

Command `list` reads `artifacts/releases/` only. Its default table shows each
cached GitHub release, binary version, four-platform archive count, total size,
state, and cache path. It is useful before a release build, when offline, or
when diagnosing why command `get` downloaded an archive again.

Option `--verify` hashes every cached archive against its checksum manifest. It
reports bad, incomplete, and malformed caches after inspecting all entries, then
exits unsuccessfully. Option `--json` writes structured cache records for shell
scripts and future cache maintenance. Neither option contacts GitHub.

Command `get` downloads the four npm target archives and the checksum manifest.
It stores them under `artifacts/releases/`, keyed by the GitHub release identity.
The command validates checksums on every cache hit, so a complete valid cache
skips the download.

Command `prepare` creates a versioned generated workspace and npm tarballs under
`dist/`. It does not change the package templates in `packages/`.

## GitHub binary version and npm distribution version

The GitHub release and npm package describe related but different things. The
GitHub archives identify the native binary version. For example,
`nibid_2.19.0_linux_amd64.tar.gz` contains a binary whose command `nibid version`
reports `2.19.0`.

Flag `--dist-ver` identifies the version of the npm wrapper packages. It may use
the same version or add a prerelease suffix for an npm-specific distribution:

```bash
bun run nibiru-dist publish \
  --from-gh hotfix/v2.19.0 \
  --dist-ver 2.19.0-npm.1
```

The helper requires both versions to have the same SemVer core,
`major.minor.patch`. The example packages verified `2.19.0` binaries as npm
version `2.19.0-npm.1`. It rejects `2.19.1-rc.1` because that label would claim
a different binary patch version. The generated workspace path includes the npm
distribution version, so separate prerelease packages from one GitHub release do
not overwrite one another.

Command `publish` checks that none of the five package versions already exist on
npm, then handles the native packages first and the umbrella package last. It
requires `--from-gh` to name the GitHub release source and `--dist-ver` to name
the npm package version. The version can include a leading `v` and a prerelease
suffix, but its SemVer core must match the downloaded assets. The command rejects
values such as `latest`, `next`, and build metadata.

Without `--run`, command `publish` prepares and validates the packages, checks
that the exact npm version does not exist, then prints the ordered publish
commands. It does not accept or pass a caller-selected npm registry tag.
