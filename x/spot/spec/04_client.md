<!--
order: 9
-->

# CLI

A user can query and interact with the `spot` module using the CLI.

## Query

The `query` commands allow users to query `spot` state.

```bash
nibid query spot --help
```

### params

The `params` command allows users to query genesis parameters for the spot module.

```bash
nibid query spot params [flags]
```

Example:

```bash
nibid query spot params
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
nibid query spot get-pool-number [flags]
```

Example:

```bash
nibid query spot get-pool-number
```

Example Output:

```bash
poolId: "1"
```

### get-pool

The `get-pool` command allows users to query a pool by id number.

```bash
nibid query spot get-pool [pool-id] [flags]
```

Example:

```bash
nibid query spot get-pool 1
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

The `total-liquidity` command allows users to query the total amount of liquidity in the spot.

```bash
nibid query spot total-liquidity [flags]
```

Example:

```bash
nibid query spot total-liquidity
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

The `pool-liquidity` command allows users to query the total amount of liquidity in the spot.

```bash
nibid query spot pool-liquidity [pool-id] [flags]
```

Example:

```bash
nibid query spot pool-liquidity 1
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

The `tx` commands allow users to interact with the `spot` module.

```bash
nibid tx spot --help
```

### create-pool

The `create-pool` command allows users to create pools.

```bash
nibid tx spot create-pool [flags]
```

Example:

```bash
nibid tx spot create-pool --pool-file ./new-pool.json
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
nibid tx spot join-pool [flags]
```

Example:

```bash
nibid tx spot join-pool --pool-id 1 --tokens-in 1validatortoken,1stake
```

# TODO

(<https://github.com/NibiruChain/nibiru/issues/220>): Add gRPC and REST docs.
