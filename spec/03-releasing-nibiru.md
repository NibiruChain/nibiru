# Releasing Nibiru

- Status: active
- Code scope: workflow `.github/workflows/release.yml`, command `just`, and
  script `contrib/scripts/publish-release-image.sh`
- Source of truth: workflow `.github/workflows/release.yml`

A pushed Git tag that matches `v*` triggers the Nibiru release workflow. The
workflow builds Linux amd64 and arm64 binaries, Darwin amd64 and arm64 binaries,
a universal Darwin binary, a multi-architecture container image, a Chaosnet
container image, and a GitHub release with archive checksums.

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
