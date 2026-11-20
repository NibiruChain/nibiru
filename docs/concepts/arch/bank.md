---
order: 2
description: >-
  bank module allows you to manage assets for accounts loaded into the local keys
  module
---
# Module: Bank

{{ $frontmatter.description }}

## Available Commands

#### Transactions

| `nibiru tx bank` | Description |
| :--- | :--- |
| [send](#nibiru-tx-bank-send) | Send funds from one account to another |
| multi-send | Send funds from one account to two or more accounts |

#### Queries

| `nibiru q bank` | Description |
| :--- | :--- |
| balance | Query a single account balance by address and denom |
| [balances](#nibiru-q-bank-balances) | Query for account balances by address |
| spendable-balances | Query for account spendable balances by address |
| send-enabled | Query for send enabled entries |
| [total](#nibiru-q-bank-total) | Query the total supply of coins of the chain |
| denom-metadata | Query the client metadata for coin denominations |

---

### nibiru q bank balances
Query the total balance of an account or of a specific denomination.

```bash
nibiru q bank balances [address] [flags]
```

**Args:**

| Name    | Description | 
| ---     | ----------- |
| address | Bech32 address that the query will return balances for |

**Flags:**

| Name, shorthand |  Description |
| :---            |  :---        |
| --help, -h    |  Help for balances |
| --denom       |  The specific balance denomination to query for |
| --count-total |  Count total number of records in all balances to query for |
| --height  | Use a specific block height to query state at (this can error if the node is pruning state) |

### nibiru q bank total
Query total supply of coins that are held by accounts in the chain.

```text
nibiru q bank total [flags]
```

**Flags:**

| Name, shorthand |  Description |
| :---            |  :---        |
| --help, -h | Help for total |
| --denom | The specific balance denomination to query for |

### nibiru tx bank send

Sending tokens to another address, this command includes `generate`, `sign` and `broadcast` steps.

```text
nibiru tx bank send [from_key_or_address] [to_address] [amount] [flags]
```

**Flags:**

| Name, shorthand |  Description |
| :---            |  :---        |
| --help, -h | Help for send |
