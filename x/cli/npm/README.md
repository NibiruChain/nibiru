# nibid npm packaging

This private Bun workspace turns immutable `nibid_*_{os}_{arch}.tar.gz` release
artifacts into npm packages. The public package is `@nibiruchain/nibid`. Its
Node launcher resolves the matching platform package and starts its native
`nibid` binary.

Use a release version without the leading `v` and place release tarballs in
`artifacts/`:

```bash
NIBID_NPM_VERSION=2.19.0 bun run stage
bun run verify
bun run pack
```

For a one-platform local proof, set `NIBID_NPM_TARGETS=linux-x64`. CI should
stage and verify all four supported targets.
