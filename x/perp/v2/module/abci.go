package perp

import (
	abci "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/collections"

	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/perp/v2/keeper"
	"github.com/NibiruChain/nibiru/x/perp/v2/types"
)

// EndBlocker Called every block to store a snapshot of the perpamm.
func EndBlocker(ctx sdk.Context, k keeper.Keeper) []abci.ValidatorUpdate {
	for _, amm := range k.AMMs.Iterate(ctx, collections.Range[collections.Pair[asset.Pair, uint64]]{}).Values() {
		market, err := k.GetMarket(ctx, amm.Pair)
		if err != nil {
			k.Logger(ctx).Error("failed to fetch market", "pair", amm.Pair, "error", err)
			continue
		}

		// only snapshot enabled markets
		if !market.Enabled {
			continue
		}

		snapshot := types.ReserveSnapshot{
			Amm:         amm,
			TimestampMs: ctx.BlockTime().UnixMilli(),
		}
		k.ReserveSnapshots.Insert(ctx, collections.Join(amm.Pair, ctx.BlockTime()), snapshot)

		markTwap, err := k.CalcTwap(ctx, amm.Pair, types.TwapCalcOption_SPOT, types.Direction_DIRECTION_UNSPECIFIED, sdk.ZeroDec(), market.TwapLookbackWindow)
		if err != nil {
			k.Logger(ctx).Error("failed to fetch twap mark price", "market.Pair", market.Pair, "error", err)
			continue
		}

		indexTwap, err := k.OracleKeeper.GetExchangeRateTwap(ctx, amm.Pair)
		if err != nil {
			k.Logger(ctx).Error("failed to fetch twap index price", "market.Pair", market.Pair, "error", err)
			continue
		}

		if markTwap.IsNil() || markTwap.IsZero() {
			k.Logger(ctx).Error("mark price is zero", "market.Pair", market.Pair)
			continue
		}

		if indexTwap.IsNil() || indexTwap.IsZero() {
			k.Logger(ctx).Error("index price is zero", "market.Pair", market.Pair)
			continue
		}

		_ = ctx.EventManager().EmitTypedEvent(&types.AmmUpdatedEvent{
			FinalAmm:       amm,
			MarkPriceTwap:  markTwap,
			IndexPriceTwap: indexTwap,
		})

		_ = ctx.EventManager().EmitTypedEvent(&types.MarketUpdatedEvent{
			FinalMarket: market,
		})
	}

	return []abci.ValidatorUpdate{}
}
