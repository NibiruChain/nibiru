package v2

import sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

var (
	ErrPairNotSupported     = sdkerrors.Register(ModuleName, 1, "pair not supported")
	ErrOverTradingLimit     = sdkerrors.Register(ModuleName, 2, "over trading limit")
	ErrQuoteReserveAtZero   = sdkerrors.Register(ModuleName, 3, "quote reserve after at zero")
	ErrBaseReserveAtZero    = sdkerrors.Register(ModuleName, 4, "base reserve after at zero")
	ErrNoLastSnapshotSaved  = sdkerrors.Register(ModuleName, 5, "There was no last snapshot, could be that you did not do snapshot on pool creation")
	ErrOverFluctuationLimit = sdkerrors.Register(ModuleName, 6, "price is over fluctuation limit")
	ErrAssetFailsUserLimit  = sdkerrors.Register(ModuleName, 7, "amount of assets traded does not meet user-defined limit")
	ErrNoValidPrice         = sdkerrors.Register(ModuleName, 8, "no valid prices available")
	ErrNoValidTWAP          = sdkerrors.Register(ModuleName, 9, "TWAP price not found")
	// Could replace ErrBaseReserveAtZero and ErrQUoteReserveAtZero if wrapped
	ErrNonPositiveReserves = sdkerrors.Register(ModuleName, 10,
		"base and quote reserves must always be positive")
	ErrLiquidityDepth = sdkerrors.Register(ModuleName, 11,
		"liquidity depth must be positive and equal to the square of the reserves")
	ErrInvalidAmount                      = sdkerrors.Register(ModuleName, 12, "invalid amount")
	ErrMarginRatioTooHigh                 = sdkerrors.Register(ModuleName, 13, "margin ratio is too healthy to liquidate")
	ErrPairNotFound                       = sdkerrors.Register(ModuleName, 14, "pair doesn't have live market")
	ErrPositionZero                       = sdkerrors.Register(ModuleName, 15, "position is zero")
	ErrFailedRemoveMarginCanCauseBadDebt  = sdkerrors.Register(ModuleName, 16, "failed to remove margin; position would have bad debt if removed")
	ErrQuoteAmountIsZero                  = sdkerrors.Register(ModuleName, 17, "quote amount cannot be zero")
	ErrLeverageIsZero                     = sdkerrors.Register(ModuleName, 18, "leverage cannot be zero")
	ErrMarginRatioTooLow                  = sdkerrors.Register(ModuleName, 19, "margin ratio did not meet maintenance margin ratio")
	ErrLeverageIsTooHigh                  = sdkerrors.Register(ModuleName, 20, "leverage cannot be higher than market parameter")
	ErrUnauthorized                       = sdkerrors.Register(ModuleName, 21, "operation not authorized")
	ErrAllLiquidationsFailed              = sdkerrors.Register(ModuleName, 22, "all liquidations failed")
	ErrPositionHealthy                    = sdkerrors.Register(ModuleName, 23, "position is healthy")
	ErrLiquidityDepthOverflow             = sdkerrors.Register(ModuleName, 24, "liquidty depth overflow")
	ErrMarketNotEnabled                   = sdkerrors.Register(ModuleName, 25, "market is not enabled, you can only fully close your position")
	ErrNonPositivePegMultiplier           = sdkerrors.Register(ModuleName, 26, "peg multiplier must be > 0")
	ErrNonPositiveSwapInvariantMutliplier = sdkerrors.Register(ModuleName, 27, "swap multiplier must be > 0")
	ErrNilSwapInvariantMutliplier         = sdkerrors.Register(ModuleName, 28, "swap multiplier must be not nil")
)
