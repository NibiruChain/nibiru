---
description: >-
  Nibiru is a decentralized blockchain governed by its community members.
  This section describes how to submit governance proposals on Nibiru.
order: 7
---

# Submitting Proposals <!-- omit in toc -->

{{ $frontmatter.description }}

Any NIBI holder, whether bonded or unbonded, can submit proposals with the `nibiru` CLI. There are two submission paths:

- `nibiru tx gov submit-proposal <proposal-json-file>` submits a proposal that carries one or more messages, a deposit, and metadata, all defined in a JSON file.
- `nibiru tx gov submit-legacy-proposal [type]` submits a legacy proposal. Pass a `--proposal` JSON file, or set `--title`, `--description`, `--type`, and `--deposit` directly.

## Table of Contents <!-- omit in toc -->

- [Proposal Types](#proposal-types)
  - [Submitting a Proposal With Messages](#submitting-a-proposal-with-messages)
- [Querying a Proposal](#querying-a-proposal)

## Proposal Types

The legacy subcommands of `nibiru tx gov submit-legacy-proposal` are:

| Proposal Type             | Description |
| ------------------------- | ----------- |
| `cancel-software-upgrade` | Cancels a software upgrade. |
| `ibc-upgrade`             | Updates the IBC client state in-place. An `upgraded_client_state.json` can be client-breaking. |
| `param-change`            | Change module parameters. |
| `software-upgrade`        | Upgrade the protocol code. |
| `update-client`           | Substitutes the current IBC client for a new one. This proposal is useful for updating the light client in the case of misbehavior. See [ADR-026 of IBC-Go](https://ibc.cosmos.network/main/architecture/adr-026-ibc-client-recovery-mechanisms.html) for more info. |

Other actions, such as spending community pool funds, are messages inside a
`submit-proposal` file.

### Submitting a Proposal With Messages

```bash
# parameters
nibiru tx gov submit-proposal [proposal-json-file] [flags]

# example
nibiru tx gov submit-proposal /path/to/proposal.json --from validator
```

The file sets the deposit with the `deposit` field. For example:

```json
{
  "messages": [
    {
      "@type": "/cosmos.distribution.v1beta1.MsgCommunityPoolSpend",
      "authority": "<gov-module-account-address>",
      "recipient": "nibi1...",
      "amount": [{ "denom": "unibi", "amount": "1000000" }]
    }
  ],
  "metadata": "",
  "deposit": "1000unibi",
  "title": "Spend From the Community Pool",
  "summary": "Fund a grant."
}
```

## Querying a Proposal

One can use the following command to query for proposals:

```bash
# parameters
nibiru q gov proposal [proposal-id]

# example
nibiru q gov proposal 1
```
