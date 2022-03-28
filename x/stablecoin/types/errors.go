package types

// DONTCOVER

import (
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// x/stablecoin module sentinel errors
var (
	ErrSample        = sdkerrors.Register(ModuleName, 1100, "sample error")
	NoCoinFound      = sdkerrors.Register(ModuleName, 1, "No coin found")
	NotEnoughBalance = sdkerrors.Register(ModuleName, 2, "Not enough balance")
)
