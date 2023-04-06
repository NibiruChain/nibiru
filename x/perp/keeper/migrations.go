package keeper

import (
	"github.com/NibiruChain/collections"
	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/perp/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
)

func From2To3(perpKeeper Keeper, vpoolKeeper types.VpoolKeeper) module.MigrationHandler {
	return func(ctx sdk.Context) error {
		iterator := vpoolKeeper.Pools.Iterate(ctx, collections.Range[asset.Pair]{}).Values()
		for _, pool := range iterator {
			sumBias := sdk.ZeroDec()

			positions := perpKeeper.Positions.Iterate(ctx, collections.PairRange[asset.Pair, sdk.AccAddress]{}).Values()
			for _, position := range positions {
				sumBias = sumBias.Add(position.Size_)
			}

			pool.Bias = sumBias
			vpoolKeeper.Pools.Insert(ctx, pool.Pair, pool)
		}

		return nil
	}
}
