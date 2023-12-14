package types

import sdkerrors "cosmossdk.io/errors"

var moduleErrorCodeIdx uint32 = 1

// registerError: Cleaner way of using 'sdkerrors.Register' without as much time
// manually writing integers.
func registerError(msg string) *sdkerrors.Error {
	moduleErrorCodeIdx += 1
	return sdkerrors.Register(ModuleName, moduleErrorCodeIdx, msg)
}

var (
	ErrPairNotSupported    = registerError("pair not supported")
	ErrAssetFailsUserLimit = registerError("amount of assets traded does not meet user-defined limit")

	ErrNoValidTWAP = registerError("TWAP price not found")

	ErrAmmNonpositiveReserves      = errorAmm("base and quote reserves must always be positive")
	ErrLiquidityDepth              = errorAmm("liquidity depth must be positive and equal to the square of the reserves")
	ErrAmmBaseSupplyNonpositive    = errorAmm("base supply must be > 0")
	ErrAmmQuoteSupplyNonpositive   = errorAmm("quote supply must be > 0")
	ErrAmmLiquidityDepthOverflow   = errorAmm("liquidty depth overflow")
	ErrAmmNonPositivePegMult       = errorAmm("peg multiplier must be > 0")
	ErrAmmNonPositiveSwapInvariant = errorAmm("swap invariant (and sqrt depth) must be > 0")
	ErrNilSwapInvariant            = errorAmm("swap invariant (and sqrt depth) must not be nil")

	ErrPairNotFound         = registerError("pair doesn't have live market")
	ErrPositionNotFound     = registerError("position not found")
	ErrBadDebt              = registerError("position is underwater")
	ErrInputBaseAmtNegative = registerError("base amount cannot be zero")

	ErrInputQuoteAmtNegative = errorMarketOrder("quote amount cannot be zero")
	ErrUserLeverageNegative  = errorMarketOrder("leverage cannot be zero")
	ErrLeverageIsTooHigh     = errorMarketOrder("leverage cannot be higher than market parameter")

	ErrMarginRatioTooLow        = registerError("margin ratio did not meet maintenance margin ratio")
	ErrAllLiquidationsFailed    = registerError("all liquidations failed")
	ErrParseLiquidateResponse   = registerError("failed to JSON parse liquidate responses")
	ErrPositionHealthy          = registerError("position is healthy")
	ErrMarketNotEnabled         = registerError("market is not enabled, you can only fully close your position")
	ErrNotEnoughFundToPayAction = registerError("not enough fund in perp EF to pay for action")

	ErrSettlementPositionMarketEnabled = registerError("market is enabled, you can only settle position on disabled market")
	ErrCollateralDenomNotSet           = registerError("ErrorCollateral: no collateral denom set for the perp keeper")
	ErrInvalidCollateral               = registerError("ErrorCollateral: invalid collateral denom")
)

// Register error instance for "ErrorMarketOrder"
func errorMarketOrder(msg string) *sdkerrors.Error {
	return registerError("ErrorMarketOrder: " + msg)
}

func errorAmm(msg string) *sdkerrors.Error {
	return registerError("ErrorAMM: " + msg)
}
