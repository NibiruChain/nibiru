package perp

import (
	"time"

	"github.com/NibiruChain/collections"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/perp/keeper/v2"
	types "github.com/NibiruChain/nibiru/x/perp/types/v2"
)

// InitGenesis initializes the capability module's state from a provided genesis
// state.
func InitGenesis(ctx sdk.Context, k keeper.Keeper, genState types.GenesisState) {
	k.SetParams(ctx, genState.Params)

	for _, m := range genState.Markets {
		k.Markets.Insert(ctx, m.Pair, m)
	}

	for _, a := range genState.Amms {
		k.AMMs.Insert(ctx, a.Pair, a)
	}

	for _, p := range genState.Positions {
		k.Positions.Insert(ctx, collections.Join(p.Pair, sdk.MustAccAddressFromBech32(p.TraderAddress)), p)
	}

	for _, p := range genState.ReserveSnapshots {
		k.ReserveSnapshots.Insert(ctx, collections.Join(p.Amm.Pair, time.UnixMilli(p.TimestampMs)), p)
	}
}

// ExportGenesis returns the capability module's exported genesis.
func ExportGenesis(ctx sdk.Context, k keeper.Keeper) *types.GenesisState {
	genesis := new(types.GenesisState)

	genesis.Params = k.GetParams(ctx)
	genesis.Markets = k.Markets.Iterate(ctx, collections.Range[asset.Pair]{}).Values()
	genesis.Amms = k.AMMs.Iterate(ctx, collections.Range[asset.Pair]{}).Values()
	genesis.Positions = k.Positions.Iterate(ctx, collections.PairRange[asset.Pair, sdk.AccAddress]{}).Values()
	genesis.ReserveSnapshots = k.ReserveSnapshots.Iterate(ctx, collections.PairRange[asset.Pair, time.Time]{}).Values()

	return genesis
}
