package pricefeed

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/pricefeed/keeper"
	"github.com/NibiruChain/nibiru/x/pricefeed/types"
)

// InitGenesis initializes the pricefeed module's state from a provided genesis state.
func InitGenesis(ctx sdk.Context, k keeper.Keeper, genState types.GenesisState) {
	k.SetParams(ctx, genState.Params)
	k.ActivePairsStore().AddActivePairs(ctx, genState.Params.Pairs)
	k.WhitelistOracles(ctx, common.StringsToAddrs(genState.GenesisOracles...))

	// If posted prices are not expired, set them in the store
	for _, pp := range genState.PostedPrices {
		if pp.Expiry.After(ctx.BlockTime()) {
			oracle := sdk.MustAccAddressFromBech32(pp.Oracle)
			if _, err := k.PostRawPrice(ctx, oracle, pp.PairID, pp.Price, pp.Expiry); err != nil {
				panic(err)
			}
		}
	}

	// Set the current price (if any) based on what's now in the store
	for _, pair := range genState.Params.Pairs {
		postedPrices := k.GetRawPrices(ctx, pair.String())

		if len(postedPrices) == 0 {
			continue
		}
		if err := k.GatherRawPrices(ctx, pair.Token0, pair.Token1); err != nil {
			panic(err)
		}
	}
}

// ExportGenesis returns the capability module's exported genesis.
func ExportGenesis(ctx sdk.Context, k keeper.Keeper) *types.GenesisState {
	genesis := &types.GenesisState{}

	// params
	genesis.Params = k.GetParams(ctx)

	// posted prices
	var postedPrices []types.PostedPrice
	for _, assetPair := range genesis.Params.Pairs {
		pp := k.GetRawPrices(ctx, assetPair.String())
		postedPrices = append(postedPrices, pp...)
	}
	genesis.PostedPrices = postedPrices

	// oracles
	foundOracles := make(map[string]bool)
	var exportOracles []string
	for _, assetPair := range genesis.Params.Pairs {
		for _, o := range k.GetOraclesForPair(ctx, assetPair.String()) {
			if _, found := foundOracles[o.String()]; !found {
				exportOracles = append(exportOracles, o.String())
				foundOracles[o.String()] = true
			}
		}
	}

	genesis.GenesisOracles = exportOracles

	return genesis
}
