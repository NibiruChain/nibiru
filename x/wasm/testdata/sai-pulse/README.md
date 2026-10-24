# Sai Pulse Wasm fixtures

These pinned contracts exercise the real Wasm VM and ABCI hook dispatcher in
`TestSaiPulseArtifacts`. The three files total approximately 1.1 MB. The test
package reads them from disk on first use and retains their byte slices in a
package-level `sync.OnceValues` cache. Concurrent tests share the same immutable
bytes. The loader lives in `_test.go`; these fixtures are not embedded or linked
into the production binary.

## Provenance

- Published release: [wasm-contracts/v1.52.0](https://github.com/NibiruChain/sai-perps/releases/tag/wasm-contracts/v1.52.0)

- Contract PR: https://github.com/NibiruChain/sai-perps/pull/454
- Follow-up issue: https://github.com/NibiruChain/sai-perps/issues/487
- Sai Perps main source commit: `890b60831ce50749fa2c34438eeebd3e2bdd5fb8`
- Main CI build: https://github.com/NibiruChain/sai-perps/actions/runs/39979483073
- Optimizer image: `cosmwasm/optimizer:0.17.0`, Linux x86_64.

All three binaries match the published release and main CI artifact manifests. They also match the
pre-merge CI artifacts from run `39977596123` byte-for-byte; the complete existing
keeper package tests passed with those bytes. Pulse includes the minimal
migration entrypoint.

To verify the files from this directory:

```bash
sha256sum -c checksums.txt
```

To run the integration test from the repository root:

```bash
go test ./x/wasm/keeper -run '^TestSaiPulseArtifacts$' -count=1 -v
```

The test also runs with the normal keeper package tests, without environment
variables, downloads, a local node, or a contract build. Missing files fail the
test instead of skipping it.

When updating fixtures, replace all three from the same reviewed build, update
their checksums and this provenance, and rerun the integration test. Fixture
updates are explicit repository changes rather than runtime downloads.
