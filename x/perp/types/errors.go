package types

import (
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// x/perp module sentinel errors
var (
	MarginHighEnough = sdkerrors.Register(ModuleName, 1, "Margin is higher than required maintenant margin ratio")
)
