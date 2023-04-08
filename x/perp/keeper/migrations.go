package keeper

import (
	"github.com/NibiruChain/collections"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"

	"github.com/NibiruChain/nibiru/x/common/asset"
	perpammkeeper "github.com/NibiruChain/nibiru/x/perp/amm/keeper"
	"github.com/NibiruChain/nibiru/x/perp/types"
)

func From2To3(perpKeeper Keeper, perpammKeeper types.PerpAmmKeeper) module.MigrationHandler {
	return func(ctx sdk.Context) error {
		k, ok := perpammKeeper.(perpammkeeper.Keeper)
		if !ok {
			panic("market keeper is not perpammkeeper.Keeper")
		}

		iterator := k.Pools.Iterate(ctx, collections.Range[asset.Pair]{}).Values()
		for _, pool := range iterator {
			sumBias := sdk.ZeroDec()

			positions := perpKeeper.Positions.Iterate(ctx, collections.PairRange[asset.Pair, sdk.AccAddress]{}).Values()
			for _, position := range positions {
				sumBias = sumBias.Add(position.Size_)
			}

			pool.Bias = sumBias
			pool.PegMultiplier = sdk.OneDec()
			k.Pools.Insert(ctx, pool.Pair, pool)
		}

		return nil
	}
}
