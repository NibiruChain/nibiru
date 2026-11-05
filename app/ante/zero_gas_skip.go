package ante

import (
	sdk "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types"
)

// AnteDecSkipForZeroGas runs Next only for transactions that pay fees. For a
// zero-gas actor transaction, which [AnteDecZeroGasActors] marks with the
// fixed zero-gas meter and from which [DeductFeeDecorator] deducts nothing, it
// skips Next and continues the chain.
//
// It wraps the devgas payout decorator, which computes developer shares from
// the declared tx fee and pays them from the fee collector. Without the skip,
// a zero-gas actor transaction that declares a fee would produce a payout for
// a fee that was never collected.
type AnteDecSkipForZeroGas struct {
	Next sdk.AnteDecorator
}

func (d AnteDecSkipForZeroGas) AnteHandle(
	ctx sdk.Context,
	tx sdk.Tx,
	simulate bool,
	next sdk.AnteHandler,
) (sdk.Context, error) {
	if isZeroGasMeter(ctx) {
		return next(ctx, tx, simulate)
	}
	return d.Next.AnteHandle(ctx, tx, simulate, next)
}
