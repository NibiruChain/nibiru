package pricefeed

import (
	"errors"

	"github.com/NibiruChain/nibiru/x/pricefeed/keeper"
	"github.com/NibiruChain/nibiru/x/pricefeed/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// EndBlocker updates the current pricefeed
func EndBlocker(ctx sdk.Context, k keeper.Keeper) {
	// Update the current price of each asset.
	for _, pair := range k.GetPairs(ctx) {
		if !pair.Active {
			continue
		}

		err := k.SetCurrentPrices(ctx, pair.Token0, pair.Token1)
		if err != nil && !errors.Is(err, types.ErrNoValidPrice) {
			panic(err)
		}
	}
}
