---
description: Nibiru is a decentralized blockchain governed by its community members.
order: 3
---

# Submitting Proposals <!-- omit in toc -->

This section describes how to submit governance proposals on Nibiru. {synopsis}

Any NIBI holder, whether bonded or unbonded, can submit proposals by sending a `TxGovProposal` transaction. This is possible using the `nibid` CLI. Each proposal type corresponds to a subcommand of `nibid tx gov submit-proposal`.

## Table of Contents <!-- omit in toc -->

- [Proposal Types](#proposal-types)
  - [Whitelisting an oracle address with `add-oracle`](#whitelisting-an-oracle-address-with-add-oracle)
  - [Create a virtual pool](#create-a-virtual-pool)
- [Querying a proposal](#querying-a-proposal)

## Proposal Types

| Proposal Type             | Description                                                                                                                                                                                                                                                          |
| ------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `add-oracle`              | Whitelists an account as a price feed, enabling it send `post-price` messages.                                                                                                                                                                                       |
| `cancel-software-upgrade` | Cancels a software upgrade.                                                                                                                                                                                                                                          |
| `community-pool-spend`    | Details a proposal for use of community funds, together with how many coins are proposed to be spent, and to which recipient account.                                                                                                                                |
| `create-pool`             | Creates a new Nibi-Perps trading pair by initializing a virtual AMM pool.                                                                                                                                                                                            |
| `ibc-upgrade`             | Updates the IBC client state in-place. An `upgraded_client_state.json` can be client-breaking.                                                                                                                                                                       |
| `param-change`            | Change module parameters.                                                                                                                                                                                                                                            |
| `software-upgrade`        | Upgrade the protocol code.                                                                                                                                                                                                                                           |
| `update-client`           | Substitutes the current IBC client for a new one. This proposal is useful for updating the light client in the case of misbehavior. See [ADR-026 of IBC-Go](https://ibc.cosmos.network/main/architecture/adr-026-ibc-client-recovery-mechanisms.html) for more info. |

### Whitelisting an oracle address with `add-oracle`

```bash
# parameters
nibid tx gov submit-proposal add-oracle [proposal-file] --deposit [deposit] [flags]

# example
nibid tx gov submit-proposal add-oracle /path/to/proposal.json --deposit 1000unibi --from validator
```

An JSON file the `add-oracle` proposal:

```json
{
 "title": "add Delphi oracle",
 "description": "Whitelists Delphi to post prices for BTC",
 "oracles": ["nibi1zaavvzxez0elundtn32qnk9lkm8kmcsz44g7xl"],
 "pairs": ["ubtc:unusd"]
}
```

### Create a virtual pool

```bash
# parameters
nibid tx gov submit-proposal create-pool proposalFile --deposit deposit [flags]

# example
nibid tx gov submit-proposal create-pool /path/to/proposal.json --deposit 1000unibi --from validator
```

Here's an example of a valid JSON file for the `create-pool` proposal.

```json
{
    "title": "Create vpool for BTC:NUSD",
    "description": "We want to allow leveraged BTC perp trading.",
    "pair": "ubtc:unusd",
    "quote_asset_reserve": "1000000",
    "base_asset_reserve": "1000000",
    "trade_limit_ratio": "0.1",
    "fluctuation_limit_ratio": "0.01",
    "max_oracle_spread_ratio": "0.1",
    "maintenance_margin_ratio": "0.0625"
}
```

## Querying a proposal

One can use the following command to query for proposals:

```bash
# parameters
nibid query gov proposal [proposal-id]

# example
nibid query gov proposal 1
```
