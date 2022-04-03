package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/MatrixDao/matrix/x/stablecoin/types"
)

// CheckEnoughBalance
func (k Keeper) _checkEnoughBalance(ctx sdk.Context, coinToSpend sdk.Coin, acc sdk.AccAddress) error {
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
	return types.NotEnoughBalance.Wrap(coinToSpend.String())
}

// CheckEnoughBalances
func (k Keeper) CheckEnoughBalances(ctx sdk.Context, coins sdk.Coins, fromAddr sdk.AccAddress) error {
	for _, coin := range coins {
		err := k._checkEnoughBalance(ctx, coin, fromAddr)
		if err != nil {
			return err
		}
	}
	return nil
}
