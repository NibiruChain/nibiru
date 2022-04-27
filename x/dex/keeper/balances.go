package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// checkEnoughBalance
func (k Keeper) checkEnoughBalance(
	ctx sdk.Context,
	coinToSpend sdk.Coin,
	acc sdk.AccAddress,
) error {
	accCoins := k.bankKeeper.SpendableCoins(ctx, acc)
	for _, coin := range accCoins {
		if coin.Denom == coinToSpend.Denom {
			hasEnoughBalance := coin.Amount.GTE(coinToSpend.Amount)
			if hasEnoughBalance {
				return nil
			} else {
				break
			}
		}
	}

	return sdkerrors.ErrInsufficientFunds.Wrapf(
		"acc %s does not have enough to spend %s",
		acc.String(),
		coinToSpend.String(),
	)
}

// CheckEnoughBalances checks if account address has enough balance of coins.
func (k Keeper) CheckEnoughBalances(
	ctx sdk.Context,
	coins sdk.Coins,
	account sdk.AccAddress,
) error {
	for _, coin := range coins {
		err := k.checkEnoughBalance(ctx, coin, account)
		if err != nil {
			return err
		}
	}

	return nil
}
