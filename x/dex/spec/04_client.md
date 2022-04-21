<!--
order: 9
-->

# CLI

A user can query and interact with the `dex` module using the CLI.

## Query

The `query` commands allow users to query `dex` state.

```bash
nibid query dex --help
```

### params

The `params` command allows users to query genesis parameters for the dex module.

```bash
nibid query dex params [flags]
```

Example:

```bash
nibid query dex params
```

Example Output:

```bash
params:
  pool_creation_fee:
  - amount: "1000000000"
    denom: unibi
  startingPoolNumber: "1"
```

### get-pool-number

The `get-pool-number` command allows users to query the next available pool id number.

```bash
nibid query dex get-pool-number [flags]
```

Example:

```bash
nibid query dex get-pool-number
```

Example Output:

```bash
poolId: "1"
```

### get-pool

The `get-pool` command allows users to query a pool by id number.

```bash
nibid query dex get-pool [pool-id] [flags]
```

Example:

```bash
nibid query dex get-pool 1
```

Example Output:

```bash
pool:
  address: nibi1w00c7pqkr5z7ptewg5z87j2ncvxd88w43ug679
  id: "1"
  poolAssets:
  - token:
      amount: "100"
      denom: stake
    weight: "1073741824"
  - token:
      amount: "100"
      denom: validatortoken
    weight: "1073741824"
  poolParams:
    exitFee: "0.010000000000000000"
    swapFee: "0.010000000000000000"
  totalShares:
    amount: "100000000000000000000"
    denom: nibiru/pool/1
  totalWeight: "2147483648"
```

### total-liquidity

The `total-liquidity` command allows users to query the total amount of liquidity in the dex.

```bash
nibid query dex total-liquidity [flags]
```

Example:

```bash
nibid query dex total-liquidity
```

Example Output:

```bash
liquidity:
- amount: "100"
  denom: stake
- amount: "100"
  denom: validatortoken
```

### pool-liquidity

The `pool-liquidity` command allows users to query the total amount of liquidity in the dex.

```bash
nibid query dex pool-liquidity [pool-id] [flags]
```

Example:

```bash
nibid query dex pool-liquidity 1
```

Example Output:

```bash
liquidity:
- amount: "100"
  denom: stake
- amount: "100"
  denom: validatortoken
```

## Transactions

The `tx` commands allow users to interact with the `dex` module.

```bash
nibid tx dex --help
```

### create-pool

The `create-pool` command allows users to create pools.

```bash
nibid tx dex create-pool [flags]
```

Example:

```bash
nibid tx dex create-pool --pool-file ./new-pool.json
```

Where the pool file JSON has format:

```json
{
    "weights": "1stake,1validatortoken",
    "initial-deposit": "100stake,100validatortoken",
    "swap-fee": "0.01",
    "exit-fee": "0.01"
}
```

### join-pool

The `join-pool` command allows users to join pools with liquidty.

```bash
nibid tx dex join-pool [flags]
```

Example:

```bash
nibid tx dex join-pool --pool-id 1 --tokens-in 1validatortoken,1stake
```

# TODO

(<https://github.com/NibiruChain/nibiru/issues/220>): Add gRPC and REST docs.
