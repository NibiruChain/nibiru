# Releasing Nibiru

- Status: active
- Code scope: workflow `.github/workflows/release.yml`, command `just`, and
  script `contrib/scripts/publish-release-image.sh`
- Source of truth: workflow `.github/workflows/release.yml`

A pushed Git tag that matches `v*` triggers the Nibiru release workflow. The
workflow builds Linux amd64 and arm64 binaries, Darwin amd64 and arm64 binaries,
a universal Darwin binary, a multi-architecture container image, a Chaosnet
container image, and a GitHub release with archive checksums.

## Compatible Release Lines

Branch `release/v2.23` maintains patches compatible with the deployed v2.23.0
state machine. It starts at `cedb14e6d`, whose node and CLI source match
`v2.23.0`; the intervening commits change documentation only. Target compatible
patch PRs at this branch. Tag approved release-line commits as `v2.23.1`,
`v2.23.2`, and so on.

CLI request construction and formatting, documentation, and off-chain registry
metadata may change on this line. Validator execution, ante handling,
precompiles, and query behavior reachable during contract execution must keep
the deployed semantics, including gas. A gRPC method being a query does not
make it safe to change in a patch if contracts can invoke it through Stargate.

Develop state-machine breaking changes against `main`, under Unreleased, for a
future coordinated upgrade. Carry compatible fixes forward from the release
line to `main`; do not merge `main` back into a maintained release line once it
contains breaking work. Cherry-pick only reviewed compatible commits when a
backport is needed.

For this split, the CLI converts USDC aliases to explicit addresses locally.
The node's alias resolver remains unchanged on `release/v2.23`. Changing that
resolver is separate breaking work for `main`.

Before tagging a patch, review the complete diff from the deployed release,
verify that contract-observable results and gas are unchanged, and require CI
on the exact approved head. The upgrade-handler checker confirms handler-name
compatibility only; it does not prove state-machine compatibility:

```bash
go run ./contrib/scripts/upgrade-handler-check.go v2.23.1
```

It must select the existing `v2.23.0` handler. Do not add a new upgrade handler
or schedule a governance upgrade for a compatible patch.

## Create a source release

Prepare and review the release commit on the intended branch, including any
required changelog changes. Tag that exact commit with a semantic version and
push the tag:

```bash
git tag -a vX.Y.Z -m "Release vX.Y.Z"
git push origin vX.Y.Z
```

Watch workflow `Release` in GitHub Actions. The workflow creates the GitHub
release after every binary build and the universal Darwin build succeed.

Do not use the removed Ignite release process or upload archives manually. The
workflow is responsible for building, checksumming, and attaching release
artifacts.

## Verify or publish a container from release artifacts

The release workflow publishes the primary Nibiru and Chaosnet images. Use the
following commands only when the release-artifact image workflow is needed,
such as a controlled republish after a public release exists:

```bash
just release-image-verify vX.Y.Z X.Y.Z
just release-image-smoke vX.Y.Z X.Y.Z
just release-image-publish vX.Y.Z X.Y.Z
```

Command `just release-image-verify` downloads and verifies the published Linux
archives. Command `just release-image-smoke` builds and runs local amd64 and
arm64 images. Command `just release-image-publish` requires `GHCR_TOKEN` and
pushes `ghcr.io/nibiruchain/nibiru:X.Y.Z`.
