# Private Release Repository

`NibiruChain/nibiru-priv` keeps a private copy of this repository for security fixes and release preparation. Keep shared workflow changes in public Nibiru so that a normal merge brings them into the private copy.

## Sync From Public Nibiru

1. Fetch both repositories and create a sync branch from private `main`. Merge public `main` into that branch without rebasing or copying private commits into the public repository.
2. Resolve conflicts and retain the private Dependabot configuration, which sets `open-pull-requests-limit: 0` for version updates. Apply that limit to any ecosystems added upstream. Dependabot security alerts and security updates are separate repository settings.
3. Open a private pull request, review the changes and run its checks. Merge it into private `main` after review. Its normal push checks then run on `main`.
4. Check the Actions tab after a scheduled time. Private scheduled runs should skip their jobs; public schedules continue. Manually dispatch workflows when a full run is needed.

For example, from a private checkout with a clean worktree:

```sh
git remote add upstream git@github.com:NibiruChain/nibiru.git # once
git fetch origin main
git fetch upstream main
git switch -c sync/public-main origin/main
git merge upstream/main
```

## Prepare a Private Release

The `Release` workflow builds binaries for `v*` tags and publishes a GitHub release in the repository where the tag was pushed. Its generated release notes include commit subjects. Review the tag, workflow, release notes, assets, and repository access before sharing anything outside the private repository. Private repository releases and artifacts require access to that repository.

The optional repository image uses `ghcr.io/<owner>/<repository>` when `PUBLISH_CONTAINERS` is `true`. Check the package visibility and access before enabling it. The shared `chaosnet` image and the public release image publisher run only in `NibiruChain/nibiru`; do not use them to publish a private build. Transfer approved binaries to the public release process only after disclosure is authorized.
