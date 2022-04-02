package keeper

import (
	"github.com/MatrixDao/matrix/x/stablecoin/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// GetParams get all parameters as types.Params
func (k Keeper) GetParams(ctx sdk.Context) types.Params {
	var scParams types.Params
	k.paramstore.GetParamSet(ctx, &scParams)
	return scParams
}

// SetParams set the params
func (k Keeper) SetParams(ctx sdk.Context, params types.Params) {
	k.paramstore.SetParamSet(ctx, &params)
}

func (k Keeper) CheckEnoughBalance(ctx sdk.Context, coinToSpend sdk.Coin, acc sdk.AccAddress) (bool, error) {
	accCoins := k.bankKeeper.SpendableCoins(ctx, acc)

	for _, coin := range accCoins {
		if coin.Denom == coinToSpend.Denom {
			return coin.Amount.GTE(coinToSpend.Amount), nil
		}
	}

	return false, sdkerrors.Wrap(types.NoCoinFound, coinToSpend.Denom)

}

func (k Keeper) CheckEnoughBalances(ctx sdk.Context, coins sdk.Coins, acc sdk.AccAddress) error {

	fromAddr := acc
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
