# CW3 Flexible Multisig

This smart contract builds on [cw3-fixed-multisig](../cw3-fixed-multisig) with a more
powerful implementation of the [cw3 spec](https://github.com/CosmWasm/cw-plus/tree/v1.1.2/packages/cw3).
It is a multisig contract that is backed by a
[cw4 (group)](https://github.com/CosmWasm/cw-plus/tree/v1.1.2/packages/cw4) contract, which independently
maintains the voter set.

This provides several advantages:

* You can create two different multisigs with different voting thresholds
  backed by the same group. Thus, you can have a 50% vote, and a 67% vote
  that always use the same voter set, but can take other actions.
* In addition to the dynamic voting set, the main difference with the native
  Cosmos SDK multisig, is that it aggregates the signatures on chain, with
  visible proposals (like `x/gov` in the Cosmos SDK), rather than requiring
  signers to share signatures off chain.
* TODO: It allows dynamic multisig groups. Since the group can change,
  we can set one of the multisigs as the admin of the group contract,
  and the

## Usage Guide: CW3 Flex Multisig

This contract is a multisig contract that is backed by a cw4 (group) contract, which independently maintains the voter set.

### Instantiate

```js
{
    "group_addr": "cosmos1...", // this is the group contract that contains the member list
    "threshold": {
        "absolute_count": {"weight": 2},
        "absolute_percentage": {"percentage": 0.5},
        "threshold_quorum": { "threshold": 0.1, "quorum": 0.2 }
    },
    "max_voting_period": "3600s",
    // who is able to execute passed proposals
    // None means that anyone can execute
    "executor": {},
    /// The cost of creating a proposal (if any).
    "proposal_deposit": {
        "denom": "uusd",
        "amount": "1000000"
    },
}
```

### Execute

- **Propose** creates a message to be executed by the multisig. It can be executed by anyone.

```js
{
  "propose": {
    "title": "My proposal",
    "description": "This is a proposal",
    "msgs": [
      {
        "bank": {
          "send": {
            "from_address": "cosmos1...",
            "to_address": "cosmos1...",
            "amount": [{ "denom": "uusd", "amount": "1000000" }]
          }
        }
      }
    ],
    "latest": {
      "at_height": 123456
    }
  }
}
```

- **Vote** adds or replaces the sender's vote on an active proposal. It can be executed by any voter with non-zero weight.

```js
{
  "vote": {
    "proposal_id": 1,
    "vote": "yes"
  }
}
```

- **Execute** executes a passed proposal. It can be executed by anyone.

```js
{
  "execute": {
    "proposal_id": 1
  }
}
```

- **Close** closes an expired proposal. It can be executed by anyone.

```js
{
  "close": {
    "proposal_id": 1
  }
}
```

### Query

- **Threshold** returns the current threshold necessary for a proposal to be executed.

```js
{
  "threshold": {}
}
```

- **Proposal** fetches the details of a specific proposal given its ID.

```js
{
  "proposal": {
    "proposal_id": 1
  }
}
```

- **ListProposals** lists proposals with optional pagination. `start_after` specifies the ID after which to start listing, and `limit` sets the maximum number of proposals to return.

```js
{
  "list_proposals": {
    "start_after": 1,
    "limit": 10
  }
}
```

- **ReverseProposals** lists proposals in reverse order with optional pagination. `start_before` specifies the ID before which to start listing in reverse, and `limit` sets the maximum number of proposals to return.

```js
{
  "reverse_proposals": {
    "start_before": 10,
    "limit": 10
  }
}
```

- **Vote** retrieves the vote details for a given proposal ID and voter address.

```js
{
  "vote": {
    "proposal_id": 1,
    "voter": "cosmos1..."
  }
}
```

- **ListVotes** lists votes for a given proposal, with optional pagination. `start_after` specifies the address after which to start listing votes, and `limit` sets the maximum number of votes to return.

```js
{
  "list_votes": {
    "proposal_id": 1,
    "start_after": "cosmos1...",
    "limit": 10
  }
}
```

- **Voter** fetches details about a specific voter by their address.

```js
{
  "voter": {
    "address": "cosmos1..."
  }
}
```

- **ListVoters** lists voters with optional pagination. `start_after` specifies the address after which to start listing voters, and `limit` sets the maximum number of voters to return.

```js
{
  "list_voters": {
    "start_after": "cosmos1...",
    "limit": 10
  }
}
```

- **Config** retrieves the current configuration of the system.

```js
{
  "config": {}
}
```

## Concepts

### Instantiation

The first step to create such a multisig is to instantiate a cw4 contract
with the desired member set. For now, this only is supported by
[cw4-group](../cw4-group), but we will add a token-weighted group contract
(TODO).

If you create a `cw4-group` contract and want a multisig to be able
to modify its own group, do the following in multiple transactions:

  * instantiate cw4-group, with your personal key as admin
  * instantiate a multisig pointing to the group
  * `AddHook{multisig}` on the group contract
  * `UpdateAdmin{multisig}` on the group contract

This is the current practice to create such circular dependencies,
and depends on an external driver (hard to impossible to script such a
self-deploying contract on-chain). (TODO: document better).

When creating the multisig, you must set the required weight to pass a vote
as well as the max/default voting period. (TODO: allow more threshold types)

### Execution Process

First, a registered voter must submit a proposal. This also includes the
first "Yes" vote on the proposal by the proposer. The proposer can set
an expiration time for the voting process, or it defaults to the limit
provided when creating the contract (so proposals can be closed after several
days).

Before the proposal has expired, any voter with non-zero weight can add or
replace their vote. Replacing a vote updates the proposal tally and may move a
proposal between "Open", "Passed", and "Rejected" while it is still active. If
enough "Yes" votes were submitted before the proposal expiration date, the
status is set to "Passed".

Once a proposal is "Passed", anyone may submit an "Execute" message. This will
trigger the proposal to send all stored messages from the proposal and update
it's state to "Executed", so it cannot run again. (Note if the execution fails
for any reason - out of gas, insufficient funds, etc - the state update will
be reverted, and it will remain "Passed", so you can try again).

Once a proposal has expired without passing, anyone can submit a "Close"
message to mark it closed. This has no effect beyond cleaning up the UI/database.

## Building and Testing this contract

You will need Rust 1.44.1+ with `wasm32-unknown-unknown` target installed.

You can run unit tests on this via:

`cargo test`

Once you are happy with the content, you can compile it to wasm via:

```
RUSTFLAGS='-C link-arg=-s' cargo wasm
cp ../../target/wasm32-unknown-unknown/release/cw3_fixed_multisig.wasm .
ls -l cw3_fixed_multisig.wasm
sha256sum cw3_fixed_multisig.wasm
```

Or for a production-ready (optimized) build, run a build command in
the repository root: https://github.com/CosmWasm/cw-plus#compiling.