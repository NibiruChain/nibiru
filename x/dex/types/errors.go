package types

// DONTCOVER

import (
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// x/dex module sentinel errors
var (
	ErrTooFewPoolAssets   = sdkerrors.Register(ModuleName, 1, "pool should have at least 2 assets, as they must be swapping between at least two assets")
	ErrTooManyPoolAssets  = sdkerrors.Register(ModuleName, 2, "pool has too many assets (currently capped at 2 assets per pool)")
	ErrInvalidSwapFee     = sdkerrors.Register(ModuleName, 3, "invalid pool swap fee, must be between [0, 1]")
	ErrInvalidExitFee     = sdkerrors.Register(ModuleName, 4, "invalid pool exit fee, must be between [0, 1]")
	ErrInvalidTokenWeight = sdkerrors.Register(ModuleName, 5, "token weight must be greater than zero")
	ErrTokenNotAllowed    = sdkerrors.Register(ModuleName, 8, "token not allowed")

	// create-pool tx cli errors
	ErrMissingPoolFileFlag   = sdkerrors.Register(ModuleName, 6, "must pass in a pool json using the --pool-file flag")
	ErrInvalidCreatePoolArgs = sdkerrors.Register(ModuleName, 7, "deposit tokens and token weights should have same length and denom order")
)
