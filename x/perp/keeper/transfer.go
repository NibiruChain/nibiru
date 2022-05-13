package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Transfer moves the provided coins from the module address to the trader.
// It returns the actual moved coins as we do not know which is the quote asset.
// TODO(mercilex)
func (k Keeper) Transfer(ctx sdk.Context, denom string, trader sdk.AccAddress, amount sdk.Int) (transferred sdk.Coins, err error) {
	return sdk.NewCoins(sdk.NewCoin("todo", amount)), nil
}
