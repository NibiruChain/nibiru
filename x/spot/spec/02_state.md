# State

## Next Pool Number

The spot module stores a monotonically increasing counter denoting the next available integer pool number. Pool numbers start at 1 and increase every time a pool is created. The `Keeper.GetNextPoolNumberAndIncrement` function always fetches the next availble pool number and increments the stored value by 1.

## Pools

Serialized protobufs representing pools are stored in the state, with the key 0x02 | poolId. See the [pool proto file](../../../proto/dex/v1/pool.proto) for what fields a pool has.

## Total Liquidity

The spot module also stores the total liquidity in the module's account, which is the sum of all assets aggregated across all pools. The total liquidity is updated every time a pool's liquidity is updated (either through creation, joining, exiting, or swaps).

The total liquidity is stored with key 0x03 | denom.
