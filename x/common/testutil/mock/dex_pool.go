package mock

import (
	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/spot/types"
)

// helper function to create dummy test pools
func SpotPool(poolId uint64, assets sdk.Coins, shares int64) types.Pool {
	poolAssets := make([]types.PoolAsset, len(assets))
	for i, asset := range assets {
		poolAssets[i] = types.PoolAsset{
			Token:  asset,
			Weight: sdkmath.OneInt(),
		}
	}
	return types.Pool{
		Id: poolId,
		PoolParams: types.PoolParams{
			SwapFee:  sdkmath.LegacySmallestDec(),
			ExitFee:  sdkmath.LegacySmallestDec(),
			PoolType: types.PoolType_BALANCER,
			A:        sdkmath.ZeroInt(),
		},
		PoolAssets:  poolAssets,
		TotalShares: sdk.NewInt64Coin(types.GetPoolShareBaseDenom(poolId), shares),
		TotalWeight: sdkmath.NewInt(2),
	}
}

// helper function to create dummy test pools
func SpotStablePool(poolId uint64, assets sdk.Coins, shares int64) types.Pool {
	poolAssets := make([]types.PoolAsset, len(assets))
	for i, asset := range assets {
		poolAssets[i] = types.PoolAsset{
			Token:  asset,
			Weight: sdkmath.OneInt(),
		}
	}
	return types.Pool{
		Id: poolId,
		PoolParams: types.PoolParams{
			SwapFee:  sdkmath.LegacyZeroDec(),
			ExitFee:  sdkmath.LegacyZeroDec(),
			PoolType: types.PoolType_STABLESWAP,
			A:        sdkmath.NewInt(100),
		},
		PoolAssets:  poolAssets,
		TotalShares: sdk.NewInt64Coin(types.GetPoolShareBaseDenom(poolId), shares),
		TotalWeight: sdkmath.NewInt(2),
	}
}
