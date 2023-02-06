<!--
order: 1
-->

# Concepts

The `x/spot` module is responsible for for creating, joining, and exiting
liquidity pools that are dictated by an AMM for swaps.

## Pool

### Creation of Pool

When a pool is created, a fixed amount of 100 LP shares is minted and sent to the pool creator. The base pool share denom is in the format of nibiru/pool/{poolId} and is displayed in the format of NIBIRU-POOL-{poolId} to the user. One NIBIRU-POOL-{poolId} token is equivalent to 10^18 nibiru/pool/{poolId} tokens.

Pool assets are sorted in alphabetical order by defualt.

### Joining Pool

When joining a pool, users provide the tokens they are willing to deposit. The application will try to deposit as many tokens as it can while maintaining equal weight ratios across the pool's assets. Usually this means one asset acts as a limiting factor and all other tokens are deposited in proportion to the limited token.

For example, assume there is a 50/50 pool with 100 `tokenA` and 100 `tokenB`. A user wishes to LP 10 `tokenA` and 5 `tokenB` into the pool. Because `tokenB` is the limiting factor, all of `tokenB` will be deposited and only 5 of `tokenA` will be deposited. The user will be left with 5 `tokenA` and receive LP shares for the liquidity they provided.

### Exiting Pool

When exiting the pool, the user also provides the number of LP shares they are returning to the pool, and will receive assets in proportion to the LP shares returned. However, unlike joining a pool, exiting a pool requires the user to pay the exit fee, which is set as the param of the pool. The share of the user gets burnt.

For example, assume there is a 50/50 pool with 50 `tokenA` and 150 `tokenB` and 200 total LP shares minted. A user wishes to return 20 LP shares to the pool and withdraw their liquidity. Because 20/200 = 10%, the user will receive 5 `tokenA` and 15 `tokenB` from the pool, minus exit fees.

## Swap

During the process of swapping a specific asset, the token user is putting into the pool is justified as `tokenIn`, while the token that would be omitted after the swap is justified as `tokenOut`  throughout the module.

Given a tokenIn, the following calculations are done to calculate how much tokens are to be swapped and ommitted from the pool.

- `tokenBalanceOut * [ 1 - { tokenBalanceIn / (tokenBalanceIn+(1-swapFee) * tokenAmountIn)}^(tokenWeightIn/tokenWeightOut)]`

The whole process is also able vice versa, the case where user provides tokenOut. The calculation  for the amount of token that the user should be putting in is done through the following formula.

- `tokenBalanceIn * [{tokenBalanceOut / (tokenBalanceOut - tokenAmountOut)}^(tokenWeightOut/tokenWeightIn)-1] / tokenAmountIn`

### Spot Price

Meanwhile, calculation of the spot price with a swap fee is done using the following formula

- `spotPrice / (1-swapFee)`

where spotPrice is

- `(tokenBalanceIn / tokenWeightIn) / (tokenBalanceOut / tokenWeightOut)`
