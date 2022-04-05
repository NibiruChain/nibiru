package keeper

import (
	"fmt"

	"github.com/MatrixDao/matrix/x/common"
	"github.com/MatrixDao/matrix/x/stablecoin/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

/*
The collateral ratio, or 'collRatio' (sdk.Dec), is a value beteween 0 and 1 that determines
what proportion of collateral and governance token is used during stablecoin mints
and burns.
*/

// GetCollRatio queries the 'collRatio'.
func (k *Keeper) GetCollRatio(ctx sdk.Context) (collRatio sdk.Dec) {
	return sdk.NewDec(k.GetParams(ctx).CollRatio).QuoInt64(1_000_000)
}

/*
SetCollRatio manually sets the 'collRatio'. This method is mainly used for
testing. When the chain is live, the collateral ratio cannot be manually set, only
adjusted by a fixed amount (e.g. 0.25%).
*/
func (k *Keeper) SetCollRatio(ctx sdk.Context, collRatio sdk.Dec) (err error) {
	collRatioTooHigh := collRatio.GT(sdk.OneDec())
	collRatioTooLow := collRatio.IsNegative()
	if collRatioTooHigh {
		return fmt.Errorf("input 'collRatio', %d, is higher than 1", collRatio)
	} else if collRatioTooLow {
		return fmt.Errorf("input 'collRatio', %d, is negative", collRatio)
	}
	params := types.NewParams(collRatio)
	k.ParamSubspace.SetParamSet(ctx, &params)
	return err
}

/*
UpdateCollRatio updaet the value of the current collateral ratio knowing the price is either up or down the peg
*/
func (k *Keeper) UpdateCollRatio(ctx sdk.Context, isPriceUp bool) (err error) {

	matrixStep := sdk.MustNewDecFromStr("0.0025")
	var adjustment sdk.Dec

	if isPriceUp {
		adjustment = matrixStep
	} else {
		adjustment = matrixStep.Mul(sdk.MustNewDecFromStr("-1"))
	}
	currCollRatio := k.GetCollRatio(ctx)
	k.SetCollRatio(ctx, currCollRatio.Add(adjustment))

	return err
}

/*
Evaluate Coll ratio updates the collateral ratio if the price is out of the bounds.
*/
func (k *Keeper) EvaluateCollRatio(ctx sdk.Context) (err error) {

	upperBound := sdk.MustNewDecFromStr("1.0001")
	lowerBound := sdk.MustNewDecFromStr("0.9999")

	// Should take TWAP price
	stablePrice, err := k.priceKeeper.GetCurrentTWAPPrice(ctx, common.CollPricePool)
	if err != nil {
		return err
	}

	if stablePrice.Price.GTE(upperBound) {
		err = k.UpdateCollRatio(ctx, true)
	} else if stablePrice.Price.LTE(lowerBound) {
		err = k.UpdateCollRatio(ctx, false)
	}
	if err != nil {
		return err
	}
	return nil
}
