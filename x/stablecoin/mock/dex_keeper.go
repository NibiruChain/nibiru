package mock

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/spot/types"
)

type Keeper struct {
	pool types.Pool
}

func NewKeeper(pool types.Pool) Keeper {
	return Keeper{
		pool: pool,
	}
}

func (k Keeper) FetchPoolFromPair(ctx sdk.Context, denomA string, denomB string) (pool types.Pool, err error) {
	return k.pool, nil
}

func (k Keeper) FetchPool(ctx sdk.Context, poolId uint64) (pool types.Pool, err error) {
	return k.pool, nil
}
