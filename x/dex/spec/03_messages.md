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
