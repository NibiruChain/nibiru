package pricefeed

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/pricefeed/keeper"
	"github.com/NibiruChain/nibiru/x/pricefeed/types"
)

// InitGenesis initializes the capability module's state from a provided genesis
// state.
func InitGenesis(ctx sdk.Context, k keeper.Keeper, genState types.GenesisState) {
	k.SetParams(ctx, genState.Params)
	k.ActivePairsStore().
		AddActivePairs(ctx, common.NewAssetPairs(genState.Params.Pairs))
	k.WhitelistOracles(ctx, genState.GenesisOracles)

	// If posted prices are not expired, set them in the store
	for _, pp := range genState.PostedPrices {
		if pp.Expiry.After(ctx.BlockTime()) {
			_, err := k.SetPrice(ctx, pp.OracleAddress, pp.PairID, pp.Price, pp.Expiry)
			if err != nil {
				panic(err)
			}
		}
	}
	params := k.GetParams(ctx)

	// Set the current price (if any) based on what's now in the store
	for _, pairID := range params.Pairs {
		pair := common.MustNewAssetPair(pairID)
		if !k.ActivePairsStore().Get(ctx, pair) {
			continue
		}
		postedPrices := k.GetRawPrices(ctx, pair.AsString())

		if len(postedPrices) == 0 {
			continue
		}
		err := k.SetCurrentPrices(ctx, pair.Token0, pair.Token1)
		if err != nil {
			panic(err)
		}
	}
}

// ExportGenesis returns the capability module's exported genesis.
func ExportGenesis(ctx sdk.Context, k keeper.Keeper) *types.GenesisState {
	genesis := types.DefaultGenesis()
	genesis.Params = k.GetParams(ctx)
	var postedPrices []types.PostedPrice
	for _, assetPair := range k.GetPairs(ctx) {
		pp := k.GetRawPrices(ctx, assetPair.AsString())
		postedPrices = append(postedPrices, pp...)
	}
	genesis.PostedPrices = postedPrices

	return genesis
}
