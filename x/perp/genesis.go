package perp

import (
	"github.com/NibiruChain/nibiru/x/common"
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
		k.PrepaidBadDebt.Insert(ctx, keys.String(pbd.Denom), types.PrepaidBadDebt{
			Denom:  pbd.Denom,
			Amount: pbd.Amount,
		})
	}

	// set whitelist
	for _, whitelist := range genState.WhitelistedAddresses {
		addr, err := sdk.AccAddressFromBech32(whitelist)
		if err != nil {
			panic(err)
		}
		k.Whitelist.Insert(ctx, keys.String(addr.String()))
	}
}

// ExportGenesis returns the capability module's exported genesis.
func ExportGenesis(ctx sdk.Context, k keeper.Keeper) *types.GenesisState {
	genesis := new(types.GenesisState)

	genesis.Params = k.GetParams(ctx)

	// export positions
	positions := k.Positions.Iterate(
		ctx,
		keys.Unbounded[keys.Pair[common.AssetPair, keys.StringKey]](),
		keys.Unbounded[keys.Pair[common.AssetPair, keys.StringKey]](),
		keys.OrderAscending,
	).Values()
	genesis.Positions = make([]*types.Position, len(positions))
	for i, position := range positions {
		p := position
		genesis.Positions[i] = &p
	}

	// export prepaid bad debt
	pbd := k.PrepaidBadDebt.Iterate(
		ctx,
		keys.Unbounded[keys.StringKey](),
		keys.Unbounded[keys.StringKey](),
		keys.OrderAscending,
	).Values()
	genesis.PrepaidBadDebts = make([]*types.PrepaidBadDebt, len(pbd))
	for i, p := range pbd {
		p := p
		genesis.PrepaidBadDebts[i] = &p
	}

	// export whitelist
	whitelist := k.Whitelist.GetAll(ctx)
	genesis.WhitelistedAddresses = make([]string, len(whitelist))
	for i, addr := range whitelist {
		addr := addr
		genesis.WhitelistedAddresses[i] = addr.String()
	}

	// export pairMetadata
	metadata := k.PairMetadata.Iterate(
		ctx,
		keys.Unbounded[common.AssetPair](),
		keys.Unbounded[common.AssetPair](),
		keys.OrderAscending,
	).Values()
	genesis.PairMetadata = make([]*types.PairMetadata, len(metadata))
	for i, m := range metadata {
		genesis.PairMetadata[i] = &m
	}

	return genesis
}
