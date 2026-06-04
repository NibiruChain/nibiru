# v2.14 upgrade test artifacts

File `app/upgrades/v2_14_0_test.go` expects deployed-compatible CosmWasm
artifacts in this directory with these exact names:

- `cw3_flex_multisig.wasm`
- `cw4_group.wasm`

These bytecode payloads are downloaded from Nibiru mainnet so the upgrade test
instantiates the same code already used by the Treasury and Hot Wallet
contracts. Do not replace them with locally rebuilt artifacts unless the test is
intentionally validating rebuilt bytecode instead of deployed bytecode.

## Downloading from mainnet

Use mainnet contract metadata to resolve the deployed code IDs, then download
the bytecode into this directory:

```bash
NODE="https://rpc.archive.nibiru.fi:443"
CHAIN_ID="cataclysm-1"

TREASURY_CW3="nibi1l8dxzwz9d4peazcqjclnkj2mhvtj7mpnkqx85mg0ndrlhwrnh7gskkzg0v"
TREASURY_CW4="nibi1zvwvtpluyak4x7u2yf5j3qxqgvmnfdgn9y7dphthleq4sylrsaesmm9dnz"

CW3_CODE_ID="$(nibid q wasm contract "$TREASURY_CW3" \
  --node "$NODE" --chain-id "$CHAIN_ID" --output json \
  | jq -r '.contract_info.code_id')"

CW4_CODE_ID="$(nibid q wasm contract "$TREASURY_CW4" \
  --node "$NODE" --chain-id "$CHAIN_ID" --output json \
  | jq -r '.contract_info.code_id')"

nibid q wasm code "$CW3_CODE_ID" cw3_flex_multisig.wasm \
  --node "$NODE" --chain-id "$CHAIN_ID"

nibid q wasm code "$CW4_CODE_ID" cw4_group.wasm \
  --node "$NODE" --chain-id "$CHAIN_ID"
```

Mainnet values observed for the v2.14 upgrade test:

| Artifact | Mainnet code ID | SHA-256 |
|----------|-----------------|---------|
| `cw3_flex_multisig.wasm` | `2` | `715ee1e374074d61da6f9f31b3c645430099368c40f2d310ecec9035ab36bbb9` |
| `cw4_group.wasm` | `1` | `2d4c79cde9765e52c69af43ef9c9ae875ec064820665ec011bea7cf45dceac52` |

The Hot Wallet contracts use the same code IDs:

```bash
HOT_CW3="nibi15wd4ac2383fq65uymu72dg4u4u60t2du545fzxeakdw3kf7hd7yqtg45z7"
HOT_CW4="nibi1pqh0j0jzasj4f7mfm7hp6dq94fcuemrl9afas4k48pw0x3j8gxysdsy4n2"

nibid q wasm contract "$HOT_CW3" \
  --node "$NODE" --chain-id "$CHAIN_ID" --output json \
  | jq '.contract_info.code_id'

nibid q wasm contract "$HOT_CW4" \
  --node "$NODE" --chain-id "$CHAIN_ID" --output json \
  | jq '.contract_info.code_id'
```

## `cw3_flex_multisig.wasm`

Mainnet code ID `2` is used by:

- Treasury CW3 contract
  `nibi1l8dxzwz9d4peazcqjclnkj2mhvtj7mpnkqx85mg0ndrlhwrnh7gskkzg0v`
- Hot Wallet CW3 contract
  `nibi15wd4ac2383fq65uymu72dg4u4u60t2du545fzxeakdw3kf7hd7yqtg45z7`

Local source reference:
`nibi-wasm/contracts/core-cw3-flex-msig/src/msg.rs`.

Entrypoint message types:

- Struct `InstantiateMsg`
  - Field `group_addr`
  - Field `threshold`
  - Field `max_voting_period`
  - Field `executor`
  - Field `proposal_deposit`
- Enum `ExecuteMsg`
  - Variant `Propose`
  - Variant `Vote`
  - Variant `Execute`
  - Variant `Close`
  - Variant `MemberChangedHook`
- Enum `QueryMsg`
  - Variant `Threshold`
  - Variant `Proposal`
  - Variant `ListProposals`
  - Variant `ReverseProposals`
  - Variant `Vote`
  - Variant `ListVotes`
  - Variant `Voter`
  - Variant `ListVoters`
  - Variant `Config`

The local contract copy does not define a `MigrateMsg` type or a `migrate`
entrypoint. The v2.14 handler only relies on wasm module admin updates for this
contract; it does not execute a CW3 contract message.

## `cw4_group.wasm`

Mainnet code ID `1` is used by:

- Treasury CW4 group contract
  `nibi1zvwvtpluyak4x7u2yf5j3qxqgvmnfdgn9y7dphthleq4sylrsaesmm9dnz`
- Hot Wallet CW4 group contract
  `nibi1pqh0j0jzasj4f7mfm7hp6dq94fcuemrl9afas4k48pw0x3j8gxysdsy4n2`

Local source reference:
`wasm-cw-plus/contracts/cw4-group/src/msg.rs`.

Entrypoint message types:

- Struct `InstantiateMsg`
  - Field `admin`
  - Field `members`
- Enum `ExecuteMsg`
  - Variant `UpdateAdmin`
  - Variant `UpdateMembers`
  - Variant `AddHook`
  - Variant `RemoveHook`
- Enum `QueryMsg`
  - Variant `Admin`
  - Variant `TotalWeight`
  - Variant `ListMembers`
  - Variant `Member`
  - Variant `Hooks`

The local contract copy does not define a `MigrateMsg` type or a `migrate`
entrypoint. The v2.14 handler executes `UpdateMembers` and `UpdateAdmin` on the
Treasury CW4 group, and only updates wasm module admin metadata for the Hot
Wallet CW4 group.
