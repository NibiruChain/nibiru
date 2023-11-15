package types

import sdkerrors "cosmossdk.io/errors"

// highestErrorCode = 33
// NOTE: Please increment this when you add an error to make it easier for
// other developers to know which "code" value should be used next.

var (
	ErrPairNotSupported    = sdkerrors.Register(ModuleName, 1, "pair not supported")
	ErrQuoteReserveAtZero  = sdkerrors.Register(ModuleName, 2, "quote reserve after at zero")
	ErrBaseReserveAtZero   = sdkerrors.Register(ModuleName, 3, "base reserve after at zero")
	ErrNoLastSnapshotSaved = sdkerrors.Register(ModuleName, 4, "There was no last snapshot, could be that you did not do snapshot on pool creation")
	ErrAssetFailsUserLimit = sdkerrors.Register(ModuleName, 5, "amount of assets traded does not meet user-defined limit")

	// Price-related errors
	ErrNoValidPrice = sdkerrors.Register(ModuleName, 8, "no valid prices available")
	ErrNoValidTWAP  = sdkerrors.Register(ModuleName, 9, "TWAP price not found")

	// Could replace ErrBaseReserveAtZero and ErrQUoteReserveAtZero if wrapped
	ErrInvalidAmmReserves = sdkerrors.Register(ModuleName, 10,
		"base and quote reserves must always be positive")
	ErrLiquidityDepth = sdkerrors.Register(ModuleName, 11,
		"liquidity depth must be positive and equal to the square of the reserves")
	ErrMarginRatioTooHigh        = sdkerrors.Register(ModuleName, 13, "margin ratio is too healthy to liquidate")
	ErrPairNotFound              = sdkerrors.Register(ModuleName, 14, "pair doesn't have live market")
	ErrMarketWithVersionNotFound = sdkerrors.Register(ModuleName, 33, "market with version not found")
	ErrAMMWithVersionNotFound    = sdkerrors.Register(ModuleName, 34, "amm with version not found")
	ErrPositionNotFound          = sdkerrors.Register(ModuleName, 35, "position not found")
	ErrPositionZero              = sdkerrors.Register(ModuleName, 15, "position is zero")
	ErrBadDebt                   = sdkerrors.Register(ModuleName, 16, "position is underwater")
	ErrInputQuoteAmtNegative     = sdkerrors.Register(ModuleName, 17, "quote amount cannot be zero")
	ErrInputBaseAmtNegative      = sdkerrors.Register(ModuleName, 30, "base amount cannot be zero")

	ErrUserLeverageNegative     = sdkerrors.Register(ModuleName, 18, "leverage cannot be zero")
	ErrMarginRatioTooLow        = sdkerrors.Register(ModuleName, 19, "margin ratio did not meet maintenance margin ratio")
	ErrLeverageIsTooHigh        = sdkerrors.Register(ModuleName, 20, "leverage cannot be higher than market parameter")
	ErrUnauthorized             = sdkerrors.Register(ModuleName, 21, "operation not authorized")
	ErrAllLiquidationsFailed    = sdkerrors.Register(ModuleName, 22, "all liquidations failed")
	ErrParseLiquidateResponse   = sdkerrors.Register(ModuleName, 31, "failed to JSON parse liquidate responses")
	ErrPositionHealthy          = sdkerrors.Register(ModuleName, 23, "position is healthy")
	ErrLiquidityDepthOverflow   = sdkerrors.Register(ModuleName, 24, "liquidty depth overflow")
	ErrMarketNotEnabled         = sdkerrors.Register(ModuleName, 25, "market is not enabled, you can only fully close your position")
	ErrNonPositivePegMultiplier = sdkerrors.Register(ModuleName, 26, "peg multiplier must be > 0")
	ErrNegativeSwapInvariant    = sdkerrors.Register(ModuleName, 27, "swap multiplier must be > 0")
	ErrNilSwapInvariant         = sdkerrors.Register(ModuleName, 28, "swap multiplier must be not nil")
	ErrNotEnoughFundToPayAction = sdkerrors.Register(ModuleName, 29, "not enough fund in perp EF to pay for action")

	ErrSettlementPositionMarketEnabled   = sdkerrors.Register(ModuleName, 32, "market is enabled, you can only settle position on disabled market")
	ErrCollateralTokenFactoryDenomNotSet = sdkerrors.Register(ModuleName, 36, "no collateral denom set for the perp keeper")
	ErrInvalidCollateral                 = sdkerrors.Register(ModuleName, 37, "invalid collateral denom")
)
