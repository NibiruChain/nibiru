package perp

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	abci "github.com/tendermint/tendermint/abci/types"

	"github.com/NibiruChain/collections"

	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/perp/keeper"
	types "github.com/NibiruChain/nibiru/x/perp/types/v1"
)

// EndBlocker Called every block to store metrics.
func EndBlocker(ctx sdk.Context, k keeper.Keeper) []abci.ValidatorUpdate {
	for _, metrics := range k.Metrics.Iterate(ctx, collections.Range[asset.Pair]{}).Values() {
		_ = ctx.EventManager().EmitTypedEvent(&types.MetricsEvent{
			Pair:        metrics.Pair,
			NetSize:     metrics.NetSize,
			BlockHeight: ctx.BlockHeight(),
			BlockTimeMs: ctx.BlockTime().Unix(),
		})
	}
	return []abci.ValidatorUpdate{}
}
