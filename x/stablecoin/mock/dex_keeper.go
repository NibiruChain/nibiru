package mock

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/dex/types"
)

type Keeper struct {
	pool types.Pool
}

func NewKeeper(pool types.Pool) Keeper {
	return Keeper{
		pool: pool,
	}
}

func (k Keeper) GetFromPair(ctx sdk.Context, denomA string, denomB string) (poolId uint64, err error) {
	return 1, nil
}

func (k Keeper) FetchPool(ctx sdk.Context, poolId uint64) (pool types.Pool, err error) {
	return k.pool, nil
}
