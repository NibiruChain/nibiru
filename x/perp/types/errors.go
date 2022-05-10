package types

// DONTCOVER

import (
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// x/stablecoin module sentinel errors
var (
	MarginHighEnough = sdkerrors.Register(ModuleName, 1, "Margin is higher than required maintenant margin ratio")
)
