package perp

import (
	"time"

	"github.com/NibiruChain/collections"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/perp/v2/keeper"
	types "github.com/NibiruChain/nibiru/x/perp/v2/types"
)

// InitGenesis initializes the capability module's state from a provided genesis
// state.
func InitGenesis(ctx sdk.Context, k keeper.Keeper, genState types.GenesisState) {
	// TODO: take care of last versions saved in the store
	if err := genState.Validate(); err != nil {
		panic(err)
	}

	for _, m := range genState.Markets {
		k.Markets.Insert(ctx, collections.Join(m.Pair, m.Version), m)
	}

	for _, g := range genState.MarketLastVersions {
		k.MarketLastVersion.Insert(ctx, g.Pair, types.MarketLastVersion{Version: g.Version})
	}

	for _, a := range genState.Amms {
		pair := a.Pair
		k.AMMs.Insert(ctx, pair, a)
		timestampMs := ctx.BlockTime().UnixMilli()
		k.ReserveSnapshots.Insert(
			ctx,
			collections.Join(pair, time.UnixMilli(timestampMs)),
			types.ReserveSnapshot{
				Amm:         a,
				TimestampMs: timestampMs,
			},
		)
	}

	for _, p := range genState.Positions {
		k.Positions.Insert(
			ctx,
			collections.Join(p.Pair, sdk.MustAccAddressFromBech32(p.TraderAddress)),
			p,
		)
	}
}

// ExportGenesis returns the capability module's exported genesis.
func ExportGenesis(ctx sdk.Context, k keeper.Keeper) *types.GenesisState {
	// TODO: take care of last versions saved in the store
	genesis := new(types.GenesisState)

	genesis.Markets = k.Markets.Iterate(ctx, collections.Range[collections.Pair[asset.Pair, uint64]]{}).Values()
	genesis.Amms = k.AMMs.Iterate(ctx, collections.Range[asset.Pair]{}).Values()
	genesis.Positions = k.Positions.Iterate(ctx, collections.PairRange[asset.Pair, sdk.AccAddress]{}).Values()
	genesis.ReserveSnapshots = k.ReserveSnapshots.Iterate(ctx, collections.PairRange[asset.Pair, time.Time]{}).Values()

	return genesis
}
