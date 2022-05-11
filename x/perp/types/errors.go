package types

import sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

var (
	ErrPositionSizeZero = sdkerrors.Register(ModuleName, 1, "position size is zero")
)
