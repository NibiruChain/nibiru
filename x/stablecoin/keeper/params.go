package keeper

import (
	"github.com/MatrixDao/matrix/x/stablecoin/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// GetParams get all parameters as types.Params
func (k Keeper) GetParams(ctx sdk.Context) types.Params {
	return types.NewParams()
}

// SetParams set the params
func (k Keeper) SetParams(ctx sdk.Context, params types.Params) {
	k.paramstore.SetParamSet(ctx, &params)
}

func (k Keeper) checkEnoughBalance(ctx sdk.Context, coinToSpend sdk.Coin, acc sdk.AccAddress) (bool, error) {
	accCoins := k.bankKeeper.SpendableCoins(ctx, acc)

	for _, coin := range accCoins {
		if coin.Denom == coinToSpend.Denom {
			return coin.Amount.GTE(coinToSpend.Amount), nil
		}
	}

	return false, sdkerrors.Wrap(types.NoCoinFound, coinToSpend.Denom)

}
