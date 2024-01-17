# Spot Module

The x/spot module is responsible for creating, joining, and exiting liquidity pools. It also allows users to swap between two assets in an existing pool. It's a fully functional AMM.

- [Spot Module](#spot-module)
- [Concepts](#concepts)
  - [Pool](#pool)
    - [Creation of Pool](#creation-of-pool)
    - [Joining Pool](#joining-pool)
    - [Exiting Pool](#exiting-pool)
  - [Swap](#swap)
    - [Spot Price](#spot-price)
- [State](#state)
  - [Next Pool Number](#next-pool-number)
  - [Pools](#pools)
  - [Total Liquidity](#total-liquidity)
- [Messages](#messages)
  - [MsgCreatePool](#msgcreatepool)
    - [MsgCreatePoolResponse](#msgcreatepoolresponse)
  - [MsgJoinPool](#msgjoinpool)
    - [MsgJoinPoolResponse](#msgjoinpoolresponse)
- [CLI](#cli)
  - [Query](#query)
    - [params](#params)
    - [get-pool-number](#get-pool-number)
    - [get-pool](#get-pool)
    - [total-liquidity](#total-liquidity-1)
    - [pool-liquidity](#pool-liquidity)
  - [Transactions](#transactions)
    - [create-pool](#create-pool)
    - [join-pool](#join-pool)
- [GRPC and REST](#grpc-and-rest)
- [Parameters](#parameters)
  - [StartingPoolNumber](#startingpoolnumber)
  - [PoolCreationFee](#poolcreationfee)
- [Events](#events)
- [Hooks](#hooks)
  - [Begin Block](#begin-block)
  - [End Block](#end-block)
- [Future Improvements](#future-improvements)
- [Acceptance Tests](#acceptance-tests)

# Concepts

The `x/spot` module is responsible for for creating, joining, and exiting
liquidity pools that are dictated by an AMM for swaps.

## Pool

### Creation of Pool

When a pool is created, a fixed amount of 100 LP shares is minted and sent to the pool creator. The base pool share denom is in the format of nibiru/pool/{poolId} and is displayed in the format of NIBIRU-POOL-{poolId} to the user. One NIBIRU-POOL-{poolId} token is equivalent to 10^18 nibiru/pool/{poolId} tokens.

Pool assets are sorted in alphabetical order by default.

### Joining Pool

When joining a pool, users provide the tokens they are willing to deposit. The application will try to deposit as many tokens as it can while maintaining equal weight ratios across the pool's assets. Usually this means one asset acts as a limiting factor and all other tokens are deposited in proportion to the limited token.

For example, assume there is a 50/50 pool with 100 `tokenA` and 100 `tokenB`. A user wishes to LP 10 `tokenA` and 5 `tokenB` into the pool. Because `tokenB` is the limiting factor, all of `tokenB` will be deposited and only 5 of `tokenA` will be deposited. The user will be left with 5 `tokenA` and receive LP shares for the liquidity they provided.

### Exiting Pool

When exiting the pool, the user also provides the number of LP shares they are returning to the pool, and will receive assets in proportion to the LP shares returned. However, unlike joining a pool, exiting a pool requires the user to pay the exit fee, which is set as the param of the pool. The share of the user gets burnt.

For example, assume there is a 50/50 pool with 50 `tokenA` and 150 `tokenB` and 200 total LP shares minted. A user wishes to return 20 LP shares to the pool and withdraw their liquidity. Because 20/200 = 10%, the user will receive 5 `tokenA` and 15 `tokenB` from the pool, minus exit fees.

## Swap

During the process of swapping a specific asset, the token user is putting into the pool is justified as `tokenIn`, while the token that would be omitted after the swap is justified as `tokenOut`  throughout the module.

Given a tokenIn, the following calculations are done to calculate how much tokens are to be swapped and omitted from the pool.

- `tokenBalanceOut * [ 1 - { tokenBalanceIn / (tokenBalanceIn+(1-swapFee) * tokenAmountIn)}^(tokenWeightIn/tokenWeightOut)]`

The whole process is also able vice versa, the case where user provides tokenOut. The calculation  for the amount of token that the user should be putting in is done through the following formula.

- `tokenBalanceIn * [{tokenBalanceOut / (tokenBalanceOut - tokenAmountOut)}^(tokenWeightOut/tokenWeightIn)-1] / tokenAmountIn`

### Spot Price

Meanwhile, calculation of the spot price with a swap fee is done using the following formula

- `spotPrice / (1-swapFee)`

where spotPrice is

- `(tokenBalanceIn / tokenWeightIn) / (tokenBalanceOut / tokenWeightOut)`
# State

## Next Pool Number

The spot module stores a monotonically increasing counter denoting the next available integer pool number. Pool numbers start at 1 and increase every time a pool is created. The `Keeper.GetNextPoolNumberAndIncrement` function always fetches the next available pool number and increments the stored value by 1.

## Pools

Serialized protobufs representing pools are stored in the state, with the key 0x02 | poolId. See the [pool proto file](../../../proto/spot/v1/pool.proto) for what fields a pool has.

## Total Liquidity

The spot module also stores the total liquidity in the module's account, which is the sum of all assets aggregated across all pools. The total liquidity is updated every time a pool's liquidity is updated (either through creation, joining, exiting, or swaps).

The total liquidity is stored with key 0x03 | denom.
# Messages

## MsgCreatePool

Message to create a pool. Requires parameters specifying swap fee & exit fee, as well as the initial assets to deposit into the pool. The initial assets also determine the target weight of the pool (e.g. 50/50).

For now we only support two-asset pools, but could expand to >2 assets in the future.

### MsgCreatePoolResponse

Contains the poolId.

## MsgJoinPool

Message to join a pool. Users specify the poolId they wish to join and the assets they wish to deposit. The number of distinct assets provided by the user must match the number of distinct assets in the pool, or else the message will error.

### MsgJoinPoolResponse

Contains the updated pool liquidity, the number of LP shares minted and transferred to the user, and the remaining coins that could not be deposited due to a ratio mismatch (see [Concepts](01_concepts.md)).

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

# GRPC and REST

(<https://github.com/NibiruChain/nibiru/issues/220>): Add gRPC and REST docs.

# Parameters

The spot module contains the following parameters:

| Key                | Type      | Example      |
| ------------------ | --------- | ------------ |
| StartingPoolNumber | uint64    | 1            |
| PoolCreationFee    | sdk.Coins | 1000000ubini |

## StartingPoolNumber

The initial pool number to start creating pools at.

## PoolCreationFee

The amount of coins taken as a fee for creating a pool, from the pool creator's address.
# Events

| Event Type     | Attribute Key   | Attribute Value                              | Attribute Type |
|----------------|-----------------|----------------------------------------------|----------------|
| pool_joined    | sender          | sender's address                             | string         |
| pool_joined    | pool_id         | the numeric pool identifier                  | uint64         |
| pool_joined    | tokens_in       | the tokens sent by the user                  | sdk.Coins      |
| pool_joined    | pool_shares_out | the number of LP tokens returned to the user | sdk.Coin       |
| pool_joined    | rem_coins       | the tokens remaining after joining the pool  | sdk.Coins      |
| pool_created   | sender          | sender's address                             | string         |
| pool_created   | pool_id         | pool identifier                              | uint64         |
| pool_exited    | sender          | sender's address                             | string         |
| pool_exited    | pool_id         | pool identifier                              | uint64         |
| pool_exited    | num_shares_in   | number of LP tokens in                       | sdk.Coin       |
| pool_exited    | tokens_out      | tokens returned to the user                  | sdk.Coins      |
| assets_swapped | sender          | sender's address                             | string         |
| assets_swapped | pool_id         | pool identifier                              | uint64         |
| assets_swapped | token_in        | token to swap in                             | sdk.Coin       |
| assets_swapped | token_out       | token returned to user                       | sdk.Coin       |
# Hooks

As of this time, there are no hooks into the x/spot module.

## Begin Block

Nothing happens in begin block yet.

## End Block

Nothing happens in end block yet.

# Future Improvements

- Constant product solver for pools with different weights (<https://github.com/NibiruChain/nibiru/issues/141>)
- Safe shutdown where the pools freeze and swaps are not possible, but liquidity providers can still redeem their LP shares.

# Acceptance Tests

1. create pool
2. join pool
3. swap assets against pool
4. exit pool
