package mock

import "github.com/MatrixDao/matrix/x/dex/types"

// helper function to create dummy test pools
func DexPool(assets sdk.Coins, shares int64) types.Pool {
	poolAssets := make([]types.PoolAsset, len(assets))
	for i, asset := range assets {
		poolAssets[i] = types.PoolAsset{
			Token:  asset,
			Weight: sdk.OneInt(),
		}
	}
	return types.Pool{
		Id: 1,
		PoolParams: types.PoolParams{
			SwapFee: sdk.SmallestDec(),
			ExitFee: sdk.SmallestDec(),
		},
		PoolAssets:  poolAssets,
		TotalShares: sdk.NewInt64Coin(shareDenom, shares),
		TotalWeight: sdk.NewInt(2),
	}
}
