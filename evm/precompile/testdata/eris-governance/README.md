# Eris governance replay

Script `capture.py` downloads public mainnet code and complete contract storage
at a single block. It never signs or submits transactions. Snapshot `snapshot.json`
records height `47677314`, the block time, contract info, code hashes, code history,
and raw storage entries for the hub, escrow, allocation gauge, and proposal gauge.
Compressed bytecode files use deterministic gzip headers. The replay checks each
uncompressed SHA-256 hash before importing Wasm genesis state.

The governance test runs the canonical ErisEvm implementation behind a delegate
proxy, using real Wasm and FunToken precompiles. A new test account creates its
position through contract execution. The escrow invokes both live gauge
callbacks. Tests do not edit the captured Wasm state to make those calls succeed.

The ERC20 receipt contract is a standard six-decimal ERC20Minter deployed locally.
Its initialized code and storage are copied to the production receipt address,
and its ownership transfers to the EVM module. Bank receipt backing and the test
account's funds are synthetic. This exercises Bank-origin FunToken conversion
without claiming that the entire mainnet module state has been reproduced.

A shape-valid address outside the hub allowlist makes the vote fail after an
extension and its callbacks. The test checks that the extension and refreshed
preferences both roll back. Expired withdrawal returns only newly withdrawn
receipts as ERC20 and preserves the caller's existing Bank receipt balance.

Regenerate only when intentionally updating the replay evidence:

```sh
python3 evm/precompile/testdata/eris-governance/capture.py
go test ./evm/precompile -run TestEris -count=1
```

Captured Wasm derives from Eris contracts distributed under GPL-3.0.
The public source clone used for schema review is at commit
`7c3acdb32907d30727bd8f4313bd91a539aaa8d8` of
[contracts-tokenfactory](https://github.com/erisprotocol/contracts-tokenfactory).
The fixture's live code hashes are authoritative for execution; the source clone
is a reference for message schemas and semantics, not proof of a reproducible
bytecode build.
