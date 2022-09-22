package perp

import (
	"github.com/NibiruChain/nibiru/collections/keys"
	"github.com/NibiruChain/nibiru/x/common"

	sdk "github.com/cosmos/cosmos-sdk/types"

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
		k.Positions.Insert(ctx, keys.Join(p.Pair, keys.String(p.TraderAddress)), p)
	}

	// set params
	k.SetParams(ctx, genState.Params)

	// set prepaid debt position
	for _, pbd := range genState.PrepaidBadDebts {
		k.PrepaidBadDebt.Insert(ctx, keys.String(pbd.Denom), pbd)
	}
}

// ExportGenesis returns the capability module's exported genesis.
func ExportGenesis(ctx sdk.Context, k keeper.Keeper) *types.GenesisState {
	genesis := new(types.GenesisState)

	genesis.Params = k.GetParams(ctx)

	// export positions
	genesis.Positions = k.Positions.Iterate(ctx, keys.NewRange[keys.Pair[common.AssetPair, keys.StringKey]]()).Values()

	// export prepaid bad debt
	genesis.PrepaidBadDebts = k.PrepaidBadDebt.Iterate(ctx, keys.NewRange[keys.StringKey]()).Values()

	// export pairMetadata
	genesis.PairMetadata = k.PairsMetadata.Iterate(ctx, keys.NewRange[common.AssetPair]()).Values()

	return genesis
}
