package types

// DONTCOVER

import (
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// x/derivatives module sentinel errors
var (
	ErrNotRunning = sdkerrors.Register(ModuleName, 1, "derivatives platform is not running")
	ErrLeverage   = sdkerrors.Register(ModuleName, 2, "excessive leverage")
)
