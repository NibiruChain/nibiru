package types

// DONTCOVER

import (
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// x/dex module sentinel errors
var (
	ErrTooFewPoolAssets         = sdkerrors.Register(ModuleName, 1, "pool should have at least 2 assets, as they must be swapping between at least two assets")
	ErrTooManyPoolAssets        = sdkerrors.Register(ModuleName, 2, "pool has too many assets (currently capped at 2 assets per pool)")
	ErrInvalidSwapFee           = sdkerrors.Register(ModuleName, 3, "invalid pool swap fee, must be between [0, 1]")
	ErrInvalidExitFee           = sdkerrors.Register(ModuleName, 4, "invalid pool exit fee, must be between [0, 1]")
	ErrInvalidTokenWeight       = sdkerrors.Register(ModuleName, 5, "token weight must be greater than zero")
	ErrTokenNotAllowed          = sdkerrors.Register(ModuleName, 8, "token not allowed")
	ErrPoolWithSameAssetsExists = sdkerrors.Register(ModuleName, 15, "a pool with the same denoms already exists")

	// create-pool tx cli errors
	ErrMissingPoolFileFlag   = sdkerrors.Register(ModuleName, 6, "must pass in a pool json using the --pool-file flag")
	ErrInvalidCreatePoolArgs = sdkerrors.Register(ModuleName, 7, "deposit tokens and token weights should have same length and denom order")

	// Invalid MsgSwapAsset
	ErrInvalidPoolId        = sdkerrors.Register(ModuleName, 9, "invalid pool id")
	ErrInvalidTokenIn       = sdkerrors.Register(ModuleName, 10, "invalid tokens in")
	ErrInvalidTokenOutDenom = sdkerrors.Register(ModuleName, 11, "invalid token out denom")

	// Errors when swapping assets
	ErrPoolNotFound       = sdkerrors.Register(ModuleName, 12, "pool not found")
	ErrTokenDenomNotFound = sdkerrors.Register(ModuleName, 13, "token denom not found in pool")
	ErrSameTokenDenom     = sdkerrors.Register(ModuleName, 14, "cannot use same token denom to swap in and out")
)
