package keeper

import (
	"fmt"

	"github.com/MatrixDao/matrix/x/stablecoin/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

/*
The collateral ratio, or 'collRatio' (sdk.Dec), is a value beteween 0 and 1 that determines
what proportion of collateral and governance token is used during stablecoin mints
and burns.
*/

/* GetCollRatio queries the 'collRatio'. */
func (k *Keeper) GetCollRatio(ctx sdk.Context) (collRatio sdk.Dec) {
	return sdk.NewInt(k.GetParams(ctx).CollRatio).ToDec().QuoInt64(1_000_000)
}

/*
SetCollRatio manually sets the 'collRatio'. This method is mainly used for
testing. When the chain is live, the collateral ratio cannot be manually set, only
adjusted by a fixed amount (0.25%).
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
