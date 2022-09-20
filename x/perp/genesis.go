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
		pm := *p
		k.PairsMetadata.Insert(ctx, p.Pair, pm)
	}

	// create positions
	for _, p := range genState.Positions {
		k.Positions.Insert(ctx, keys.Join(p.Pair, keys.String(p.TraderAddress)), *p)
	}

	// set params
	k.SetParams(ctx, genState.Params)

	// set prepaid debt position
	for _, pbd := range genState.PrepaidBadDebts {
		k.PrepaidBadDebtState(ctx).Set(pbd.Denom, pbd.Amount)
	}
}

// ExportGenesis returns the capability module's exported genesis.
func ExportGenesis(ctx sdk.Context, k keeper.Keeper) *types.GenesisState {
	genesis := new(types.GenesisState)

	genesis.Params = k.GetParams(ctx)

	// export positions
	positions := k.Positions.Iterate(ctx, keys.NewRange[keys.Pair[common.AssetPair, keys.StringKey]]()).Values()
	genesis.Positions = make([]*types.Position, len(positions))
	for i, pos := range k.Positions.Iterate(ctx, keys.NewRange[keys.Pair[common.AssetPair, keys.StringKey]]()).Values() {
		p := pos
		genesis.Positions[i] = &p
	}

	// export prepaid bad debt
	k.PrepaidBadDebtState(ctx).Iterate(func(denom string, amount sdk.Int) (stop bool) {
		genesis.PrepaidBadDebts = append(genesis.PrepaidBadDebts, &types.PrepaidBadDebt{
			Denom:  denom,
			Amount: amount,
		})
		return false
	})

	// export pairMetadata
	metadata := k.PairsMetadata.Iterate(ctx, keys.NewRange[common.AssetPair]()).Values()
	genesis.PairMetadata = make([]*types.PairMetadata, len(metadata))
	for i, m := range metadata {
		pm := m
		genesis.PairMetadata[i] = &pm
	}

	return genesis
}
