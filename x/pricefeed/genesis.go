package pricefeed

import (
	"github.com/MatrixDao/matrix/x/pricefeed/keeper"
	"github.com/MatrixDao/matrix/x/pricefeed/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// InitGenesis initializes the capability module's state from a provided genesis
// state.
func InitGenesis(ctx sdk.Context, k keeper.Keeper, genState types.GenesisState) {
	// this line is used by starport scaffolding # genesis/module/init
	k.SetParams(ctx, genState.Params)

	// Iterate through the posted prices and set them in the store if they are not expired
	for _, pp := range genState.PostedPrices {
		if pp.Expiry.After(ctx.BlockTime()) {
			_, err := k.SetPrice(ctx, pp.OracleAddress, pp.MarketID, pp.Price, pp.Expiry)
			if err != nil {
				panic(err)
			}
		}
	}
	params := k.GetParams(ctx)

	// Set the current price (if any) based on what's now in the store
	for _, market := range params.Markets {
		if !market.Active {
			continue
		}
		rps := k.GetRawPrices(ctx, market.MarketID)

		if len(rps) == 0 {
			continue
		}
		err := k.SetCurrentPrices(ctx, market.MarketID)
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
	for _, market := range k.GetMarkets(ctx) {
		pp := k.GetRawPrices(ctx, market.MarketID)
		postedPrices = append(postedPrices, pp...)
	}
	genesis.PostedPrices = postedPrices

	// this line is used by starport scaffolding # genesis/module/export

	return genesis
}
