package perp

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/collections/keys"
	"github.com/NibiruChain/nibiru/x/perp/keeper"
	"github.com/NibiruChain/nibiru/x/perp/types"
)

// InitGenesis initializes the capability module's state from a provided genesis
// state.
func InitGenesis(ctx sdk.Context, k keeper.Keeper, genState types.GenesisState) {
	// set pair metadata
	for _, p := range genState.PairMetadata {
		k.PairMetadata.Insert(ctx, p.Pair, *p)
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

	// set whitelist
	for _, whitelist := range genState.WhitelistedAddresses {
		addr, err := sdk.AccAddressFromBech32(whitelist)
		if err != nil {
			panic(err)
		}
		k.WhitelistState(ctx).Add(addr)
	}
}

// ExportGenesis returns the capability module's exported genesis.
func ExportGenesis(ctx sdk.Context, k keeper.Keeper) *types.GenesisState {
	genesis := new(types.GenesisState)

	genesis.Params = k.GetParams(ctx)

	// export positions
	positions := k.Positions.GetAll(ctx)
	genesis.Positions = make([]*types.Position, len(positions))
	for i, position := range positions {
		p := position
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

	// export whitelist
	k.WhitelistState(ctx).Iterate(func(addr sdk.AccAddress) (stop bool) {
		genesis.WhitelistedAddresses = append(genesis.WhitelistedAddresses, addr.String())
		return false
	})

	// export pairMetadata
	metadata := k.PairMetadata.GetAll(ctx)
	genesis.PairMetadata = make([]*types.PairMetadata, len(metadata))
	for i, m := range metadata {
		genesis.PairMetadata[i] = &m
	}

	return genesis
}
