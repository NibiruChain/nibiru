package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	"github.com/MatrixDao/matrix/x/stablecoin/types"
)

// CheckEnoughBalance
// TODO Tests
func (k Keeper) CheckEnoughBalance(ctx sdk.Context, coinToSpend sdk.Coin, acc sdk.AccAddress) (bool, error) {
	accCoins := k.bankKeeper.SpendableCoins(ctx, acc)

	for _, coin := range accCoins {
		if coin.Denom == coinToSpend.Denom {
			return coin.Amount.GTE(coinToSpend.Amount), nil
		}
	}

	return false, sdkerrors.Wrap(types.NoCoinFound, coinToSpend.Denom)
}

// CheckEnoughBalances
// TODO Tests
func (k Keeper) CheckEnoughBalances(ctx sdk.Context, coins sdk.Coins, fromAddr sdk.AccAddress) error {
	for _, coin := range coins {
		hasEnoughBalance, err := k.CheckEnoughBalance(ctx, coin, fromAddr)
		if err != nil {
			return err
		}
		if !hasEnoughBalance {
			return types.NotEnoughBalance.Wrap(coin.String())
		}
	}

	return nil
}
