package types

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
)
