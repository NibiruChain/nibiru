package perp

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	abci "github.com/tendermint/tendermint/abci/types"

	"cosmossdk.io/collections"

	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/perp/v2/keeper"
	"github.com/NibiruChain/nibiru/x/perp/v2/types"
)

// EndBlocker Called every block to store a snapshot of the perpamm.
func EndBlocker(ctx sdk.Context, k keeper.Keeper) []abci.ValidatorUpdate {
	for _, amm := range k.AMMs.Iterate(ctx, collections.Range[asset.Pair]{}).Values() {
		snapshot := types.ReserveSnapshot{
			Amm:         amm,
			TimestampMs: ctx.BlockTime().UnixMilli(),
		}
		k.ReserveSnapshots.Insert(ctx, collections.Join(amm.Pair, ctx.BlockTime()), snapshot)
	}
	return []abci.ValidatorUpdate{}
}
