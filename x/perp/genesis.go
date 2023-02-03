package perp

import (
	"github.com/NibiruChain/collections"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/perp/keeper"
	"github.com/NibiruChain/nibiru/x/perp/types"
)

// InitGenesis initializes the capability module's state from a provided genesis
// state.
func InitGenesis(ctx sdk.Context, k keeper.Keeper, genState types.GenesisState) {
	// set pair metadata
	for _, p := range genState.PairMetadata {
		k.PairsMetadata.Insert(ctx, p.Pair, p)
	}

	// create positions
	for _, p := range genState.Positions {
		k.Positions.Insert(ctx, collections.Join(p.Pair, sdk.MustAccAddressFromBech32(p.TraderAddress)), p)
	}

	// set params
	k.SetParams(ctx, genState.Params)

	// set prepaid debt position
	for _, pbd := range genState.PrepaidBadDebts {
		k.PrepaidBadDebt.Insert(ctx, pbd.Denom, pbd)
	}
}

// ExportGenesis returns the capability module's exported genesis.
func ExportGenesis(ctx sdk.Context, k keeper.Keeper) *types.GenesisState {
	genesis := new(types.GenesisState)

	genesis.Params = k.GetParams(ctx)

	// export positions
	genesis.Positions = k.Positions.Iterate(ctx, collections.PairRange[asset.Pair, sdk.AccAddress]{}).Values()

	// export prepaid bad debt
	genesis.PrepaidBadDebts = k.PrepaidBadDebt.Iterate(ctx, collections.Range[string]{}).Values()

	// export pairMetadata
	genesis.PairMetadata = k.PairsMetadata.Iterate(ctx, collections.Range[asset.Pair]{}).Values()

	return genesis
}
