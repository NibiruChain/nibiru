package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

/*
Checks if account has enough balance to spend coins.

args:
  - ctx: cosmos-sdk context
  - coinsToSpend: the coins that the account wishes to spend
  - account: the address of the account spending the coins

ret:
  - error: an error if insufficient balance
*/
func (k Keeper) CheckEnoughBalances(
	ctx sdk.Context,
	coinsToSpend sdk.Coins,
	account sdk.AccAddress,
) error {
	accCoins := k.bankKeeper.SpendableCoins(ctx, account)

	if accCoins.IsAllGTE(coinsToSpend) {
		return nil
	}

	return sdkerrors.ErrInsufficientFunds.Wrapf(
		"acc %s does not have enough to spend %s",
		account.String(),
		coinsToSpend.String(),
	)
}
