package mock

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/dex/types"
)

// helper function to create dummy test pools
func DexPool(poolId uint64, assets sdk.Coins, shares int64) types.Pool {
	poolAssets := make([]types.PoolAsset, len(assets))
	for i, asset := range assets {
		poolAssets[i] = types.PoolAsset{
			Token:  asset,
			Weight: sdk.OneInt(),
		}
	}
	return types.Pool{
		Id: poolId,
		PoolParams: types.PoolParams{
			SwapFee: sdk.SmallestDec(),
			ExitFee: sdk.SmallestDec(),
		},
		PoolAssets:  poolAssets,
		TotalShares: sdk.NewInt64Coin(types.GetPoolShareBaseDenom(poolId), shares),
		TotalWeight: sdk.NewInt(2),
	}
}
