package perp

import (
	"log"

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
		k.PairsMetadata.Insert(ctx, p.Pair, *p)
	}

	// create positions
	for _, p := range genState.Positions {
		k.Positions.Insert(ctx, keys.Join(p.Pair, keys.String(p.TraderAddress)), *p)
	}

	// set params
	k.SetParams(ctx, genState.Params)

	// set prepaid debt position
	for _, pbd := range genState.PrepaidBadDebts {
		k.BadDebt.Insert(ctx, keys.String(pbd.Denom), *pbd)
	}
}

// ExportGenesis returns the capability module's exported genesis.
func ExportGenesis(ctx sdk.Context, k keeper.Keeper) *types.GenesisState {
	genesis := new(types.GenesisState)

	genesis.Params = k.GetParams(ctx)

	// export positions
	positions := k.Positions.Iterate(ctx, keys.NewRange[keys.Pair[common.AssetPair, keys.StringKey]]()).Values()
	genesis.Positions = make([]*types.Position, len(positions))
	for i, pos := range positions {
		p := pos
		genesis.Positions[i] = &p
	}

	// export prepaid bad debt
	pbd := k.BadDebt.Iterate(ctx, keys.NewRange[keys.StringKey]()).Values()
	genesis.PrepaidBadDebts = make([]*types.PrepaidBadDebt, len(pbd))
	for i, p := range pbd {
		x := p
		genesis.PrepaidBadDebts[i] = &x
	}

	// export pairMetadata
	metadata := k.PairsMetadata.Iterate(ctx, keys.NewRange[common.AssetPair]()).Values()
	genesis.PairMetadata = make([]*types.PairMetadata, len(metadata))
	for i, m := range metadata {
		log.Print(m)
		pm := m
		genesis.PairMetadata[i] = &pm
	}

	return genesis
}
