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
bun run main.ts --help
bun run main.ts list
bun run main.ts list --verify
bun run main.ts get hotfix/v2.19.0
bun run main.ts prepare hotfix/v2.19.0
bun run main.ts publish --from-gh hotfix/v2.19.0 --dist-ver 2.19.0
bun run main.ts publish --from-gh hotfix/v2.19.0 --dist-ver v2.19.0 --run
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
`nibid_2.19.0_linux_amd64.tar.gz` contains the native `nibid` binary. The npm
package exposes it through `nibiru version`, which reports `2.19.0`.

Flag `--dist-ver` identifies the version of the npm wrapper packages. It may use
the same version or add a prerelease suffix for an npm-specific distribution:

```bash
bun run main.ts publish \
  --from-gh hotfix/v2.19.0 \
  --dist-ver 2.19.0-npm.1
```

The helper requires both versions to have the same SemVer core,
`major.minor.patch`. The example packages verified `2.19.0` binaries as npm
version `2.19.0-npm.1`. It rejects `2.19.1-rc.1` because that label would claim
a different binary patch version. The generated workspace path includes the npm
distribution version, so separate prerelease packages from one GitHub release do
not overwrite one another.

Command `publish` checks tools and required Bun options before resolving the
release. For uploads, it verifies authentication with `bun pm whoami`. It checks
that none of the five package versions already exist on npm before downloading
or packing binaries, then handles the native packages first and the umbrella
package last. It requires `--from-gh` to name the GitHub release source and `--dist-ver` to name
the npm package version. The version can include a leading `v` and a prerelease
suffix, but its SemVer core must match the downloaded assets. The command rejects
values such as `latest`, `next`, and build metadata.

Without `--run`, command `publish` prepares and validates the packages, checks
that the exact npm version does not exist, then prints the ordered publish
commands. It does not accept or pass a caller-selected npm registry tag.

With `--run`, command `publish` writes to npm. Authenticate first with command
`bunx npm login`, then verify that command `bun pm whoami` succeeds. The helper
stops before processing release assets if Bun cannot verify authentication.
A successful preflight confirms login, but npm can still require write-time
confirmation or reject uploads if the account lacks package write permission.
If the npm account requires two-factor authentication for writes, Bun prints
the browser or one-time-password prompt in the terminal and waits for the
operator to complete it.

## Authentication and progress

Older Bun versions can read a different credential file than npm when
environment variable `XDG_CONFIG_HOME` is set. On this workstation, upgrading
from Bun 1.3.14 to 1.4.3 restored authentication with `XDG_CONFIG_HOME` set and
credentials in `~/.npmrc`. Try command `bun upgrade`, then `bun pm whoami`,
before using an environment override.

If upgrading is unavailable, npm recognizes your login, and Bun still reports
missing authentication with credentials in `~/.npmrc`, check this command:

```bash
env -u XDG_CONFIG_HOME bun pm whoami
```

If that succeeds, use the same environment override for the release helper:

```bash
env -u XDG_CONFIG_HOME bun run main.ts publish \
  --from-gh v2.19.0 --dist-ver 2.19.0 --run
```

The override applies only to that command. It does not change your shell or
credential files. Run the release helper from this directory. Running command
`bun publish` directly here attempts to publish the private tooling workspace;
the helper publishes from generated public package directories under `dist/`.

Each publish run starts with an explanation of its source, version, and purpose.
Phase 0 checks prerequisites and npm versions. Phase 1 stages verified binaries,
phase 2 packs five npm packages, and phase 3 publishes or previews the commands.
Dry runs skip authentication and explicitly report that nothing will be uploaded.
The npm version check runs once, during phase 0.

If an upload fails, the helper names the failed package and lists packages whose
uploads completed successfully. Inspect npm before retrying. Published versions
cannot be overwritten, and the helper refuses a run if any requested package
version already exists. A failed upload can leave a partial release.
