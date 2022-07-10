package pricefeed

import (
	"errors"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/pricefeed/keeper"
	"github.com/NibiruChain/nibiru/x/pricefeed/types"
)

// BeginBlocker updates the current pricefeed
func BeginBlocker(ctx sdk.Context, k keeper.Keeper) {
	// Update the current price of each asset.
	for _, pair := range k.GetPairs(ctx) {
		if !k.IsActivePair(ctx, pair.String()) {
			continue
		}

		err := k.GatherRawPrices(ctx, pair.Token0, pair.Token1)
		if err != nil && !errors.Is(err, types.ErrNoValidPrice) {
			panic(err)
		}
	}
}
