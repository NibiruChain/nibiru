package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

/* CalcFee calculates the total tx fee for exchanging `quoteAmt` of tokens on
the exchange.

Args:
	quoteAmt (sdk.Int):

Returns:
	toll (sdk.Int): Amount of tokens transferred to the the fee pool.
	spread (sdk.Int): Amount of tokens transferred to the PerpEF.
*/
func (k Keeper) CalcFee(ctx sdk.Context, quoteAmt sdk.Int) (toll sdk.Int, spread sdk.Int, err error) {
	if quoteAmt.Equal(sdk.ZeroInt()) {
		return sdk.ZeroInt(), sdk.ZeroInt(), nil
	}

	params := k.GetParams(ctx)

	tollRatio := params.GetTollRatioAsDec()
	spreadRatio := params.GetSpreadRatioAsDec()

	return sdk.NewDecFromInt(quoteAmt).Mul(tollRatio).TruncateInt(), sdk.NewDecFromInt(quoteAmt).Mul(spreadRatio).TruncateInt(), nil
}
