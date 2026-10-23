# Sai Pulse Wasm fixtures

These pinned contracts exercise the real Wasm VM and ABCI hook dispatcher in
`TestSaiPulseArtifacts`. The three files total approximately 1.1 MB. The test
package reads them from disk on first use and retains their byte slices in a
package-level `sync.OnceValues` cache. Concurrent tests share the same immutable
bytes. The loader lives in `_test.go`; these fixtures are not embedded or linked
into the production binary.

## Provenance

- Contract PR: https://github.com/NibiruChain/sai-perps/pull/454
- Sai Perps source commit: `49adc56e51ad7bbbb1e2a5d0029e39a107472ab5`
- CI build: https://github.com/NibiruChain/sai-perps/actions/runs/39880735839
- CI tested merge commit: `e95d8b9865693edcc8554fa6f7581caa5512e06b`
- Optimizer image: `cosmwasm/optimizer:0.17.0`
- Independent build: homebox, Ubuntu 24.04 WSL2, x86_64, using `just wasm-all`
  at the source commit above. All three binaries match CI byte-for-byte.

Homebox used a Windows-backed source mount because its Ubuntu 24 Docker
integration was disabled. Compilation succeeded, but that mount blocked writes
of the build metadata. Provenance was recorded after copying the unchanged Wasm
into the Linux worktree. Binary checksums matched the CI manifest.

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
