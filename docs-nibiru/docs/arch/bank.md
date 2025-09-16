---
order: 2
---
# Module: Bank

bank module allows you to manage assets for accounts loaded into the local keys
module {synopsis}

## Available Commands

#### Transactions

| `nibid tx bank` | Description |
| :--- | :--- |
| [send](#nibid-tx-bank-send) | Send funds from one account to another.

#### Queries

| `nibid query bank` | Description |
| :--- | :--- |
| [balances](#nibid-query-bank-balances) | Query for account balances by address |
| [total](#nibid-query-bank-total) | Query the total supply of coins of the chain |
| [denom-metadata](#nibid-query-denom-metadata) | Query the client metadata for coin denominations | 

---

### nibid query bank balances 

Query the total balance of an account or of a specific denomination.

```bash
nibid query bank balances [address] [flags]
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

### nibid query bank total

Query total supply of coins that are held by accounts in the chain.

```text
nibid query bank total [flags]
```

**Flags:**

| Name, shorthand |  Description |
| :---            |  :---        |
| --help, -h | Help for coin-type |
| --denom | The specific balance denomination to query for |

### nibid tx bank send

Sending tokens to another address, this command includes `generate`, `sign` and `broadcast` steps.

```text
nibid tx bank send [from_key_or_address] [to_address] [amount] [flags]
```

**Flags:**

| Name, shorthand |  Description |
| :---            |  :---        |
| --help, -h | Help for send |
