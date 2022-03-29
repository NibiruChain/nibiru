package types

// DONTCOVER

import (
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// x/dex module sentinel errors
var (
	ErrTooFewPoolAssets  = sdkerrors.Register(ModuleName, 1, "pool should have at least 2 assets, as they must be swapping between at least two assets")
	ErrTooManyPoolAssets = sdkerrors.Register(ModuleName, 2, "pool has too many assets (currently capped at 8 assets per pool)")
)
