# ErisEvm governance extension

The canonical implementation is file `contracts/ErisEvm.sol`. It extends the
existing liquid stake, unstake, and redeem methods with receipt escrow and
validator allocation voting. Constants and functions add no storage slots to the
proxy. Method `governanceVersion()` returns `1` so clients can distinguish an
upgraded proxy from a proxy that only supports the older staking methods.

## Sender and funds

```text
EOA -> proxy delegatecall -> ErisEvm delegatecall -> Wasm / FunToken precompile
                            msg.sender stays the EOA
```

Methods accept six-decimal receipt amounts, seconds of lock duration, validator
operator addresses, and integer basis points. They expose no recipient override,
arbitrary Wasm JSON, generic execute function, or owner tuning operation.

Receipt deposits query the caller's Bank and ERC20 balances. Bank receipts fund
the deposit first. Method `sendToBank` converts only the ERC20 shortfall to the
same caller's Bank account. The precompile transfers from the delegated caller
directly; this operation requires no ERC20 approval. The escrow then receives
exactly the requested receipt amount as Wasm funds.

Method `withdrawEscrow` reads the caller's Bank receipt balance, executes
`{"withdraw":{}}`, and converts only the positive balance increase to ERC20
receipts for the caller. Previously held Bank receipts remain Bank receipts.
Withdrawal unlocks stNIBI, not NIBI. Queueing an unbond and redeeming remain
separate hub operations.

## Vote encoding and rollback

Method `vote` requires matching arrays, one to 30 unique validator addresses,
positive weights, and a sum of `10000`. The Rust gauge allows a sum below that
value; the wrapper deliberately represents a complete allocation. Operator
addresses contain the prefix `nibivaloper1` plus 38 Bech32 characters. Restricting
length, prefix, and alphabet prevents JSON interpolation from accepting quotes
or structural delimiters. The Rust hub allowlist checks the actual address.

Method `extendLockAndVote` validates encoding before executing either operation.
The escrow extension invokes gauge callbacks that refresh existing preferences.
The following explicit vote replaces them. A failure in any callback or the vote
reverts the entire EVM transaction, including Bank conversion and Wasm writes.
Test `TestErisGovernanceReplay` executes these calls against captured live Wasm
and verifies a vote failure after extension rolls back both lock and preferences.

## Contract schemas

Rust references are in the local clone
`/home/realu/ki/web3-apps/erisprotocol-contracts-tokenfactory` at commit
`7c3acdb32907d30727bd8f4313bd91a539aaa8d8`:

- [Escrow messages](https://github.com/erisprotocol/contracts-tokenfactory/blob/7c3acdb32907d30727bd8f4313bd91a539aaa8d8/packages/eris/src/voting_escrow.rs)
  in file `packages/eris/src/voting_escrow.rs` define `create_lock`,
  `extend_lock_amount`, `extend_lock_time`, and `withdraw`.
- [Escrow execution](https://github.com/erisprotocol/contracts-tokenfactory/blob/7c3acdb32907d30727bd8f4313bd91a539aaa8d8/contracts/amp-governance/voting_escrow/src/contract.rs)
  enforces duration, lock, blacklist, decommissioning, and callback behavior.
- [Allocation messages](https://github.com/erisprotocol/contracts-tokenfactory/blob/7c3acdb32907d30727bd8f4313bd91a539aaa8d8/packages/eris/src/amp_gauges.rs)
  in file `packages/eris/src/amp_gauges.rs` define `vote` as a vector of
  validator-address and basis-point tuples.
- [Gauge execution](https://github.com/erisprotocol/contracts-tokenfactory/blob/7c3acdb32907d30727bd8f4313bd91a539aaa8d8/contracts/amp-governance/amp_gauges/src/contract.rs)
  schedules preferences for the next period and authenticates escrow callbacks.

## Deployment boundary

The paired web feature is `/stake/eris-vote` in NibiruChain/nibi-webapp.
The page checks the capability getter before enabling EVM governance and
simulates calls before asking for a signature. Its checked-in ABI comes from the
canonical generated artifact in this repository.

Keep both PRs unmerged until integration and security review finish. Deployment
and a proxy upgrade are separate authorized operations. Before enabling the EVM
path, confirm the proxy admin, target implementation, runtime bytecode, storage
compatibility, and `governanceVersion() == 1`, then run read-only simulations
against the upgraded proxy. This implementation task submits no mainnet upgrade
or wallet transaction.
