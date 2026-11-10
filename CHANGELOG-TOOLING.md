# Tooling Changelog

Track compatible CLI, developer-tool, documentation, and off-chain token
registry changes here. These entries must leave validator execution and
contract-observable results and gas unchanged for identical chain inputs.
State-machine changes belong in [CHANGELOG.md](CHANGELOG.md).

Add pending changes under Unreleased in the same PR that implements them,
with a PR link and the user-visible behavior. Call out changes to CLI arguments,
JSON output, or alias meanings explicitly. Consensus compatibility does not
imply that a CLI interface change is backward-compatible for scripts.

During release preparation, move only entries present in the approved release
commit into a version section such as `## v2.23.1`. Keep Unreleased for later
work. Use that version section for the patch release notes. Tag compatible
patches from `release/v2.23`, independently of coordinated chain releases.

## Unreleased

### CLI

- Resolve case-insensitive `USDC` to Sai USDC and `USDC.e` to legacy Stargate
  USDC locally before sending explicit token addresses to the node. Existing
  command arguments and JSON fields stay the same, but the CLI alias `USDC`
  now selects Sai USDC. Explicit identifiers retain their meaning.
  [PR #2908](https://github.com/NibiruChain/nibiru/pull/2908).
- Keep node and Wasm Stargate semantics unchanged: direct native `USDC` queries
  still select legacy USDC. The native alias change is separate coordinated
  upgrade work in [PR #2909](https://github.com/NibiruChain/nibiru/pull/2909).

### Documentation

- Document the compatible `release/v2.23` line and distinguish Sai USDC from
  legacy USDC.e in balance examples and wallet token listings.
  [PR #2908](https://github.com/NibiruChain/nibiru/pull/2908).

### Token Registry

- Add Sai USDC at `0x357c97796aA8e2A6b5A64b1Fc19d5baB9668E190` and its
  bank denom. Label Stargate `USDC.e` as legacy without changing its identifier
  or symbol. These are off-chain registry updates; node behavior is unchanged.
  [PR #2908](https://github.com/NibiruChain/nibiru/pull/2908),
  [PR #2909](https://github.com/NibiruChain/nibiru/pull/2909).
