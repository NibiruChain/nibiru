package perp

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/perp/keeper"
	"github.com/NibiruChain/nibiru/x/perp/types"
)

// InitGenesis initializes the capability module's state from a provided genesis
// state.
func InitGenesis(ctx sdk.Context, k keeper.Keeper, genState types.GenesisState) {
	// set pair metadata
	for _, p := range genState.PairMetadata {
		k.PairMetadataState(ctx).Set(p)
	}

	// create positions
	for _, p := range genState.Positions {
		err := k.PositionsState(ctx).Create(p)
		if err != nil {
			panic(fmt.Errorf("unable to re-create position %s: %w", p, err))
		}
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
	k.PositionsState(ctx).Iterate(func(position *types.Position) (stop bool) {
		genesis.Positions = append(genesis.Positions, position)
		return false
	})

	// export prepaid bad debt
	k.PrepaidBadDebtState(ctx).Iterate(func(denom string, amount sdk.Int) (stop bool) {
		genesis.PrepaidBadDebts = append(genesis.PrepaidBadDebts, &types.PrepaidBadDebt{
			Denom:  denom,
			Amount: amount,
		})
		return false
	})

	// export pairMetadata
	metadata := k.PairMetadataState(ctx).GetAll()
	genesis.PairMetadata = metadata

	return genesis
}
