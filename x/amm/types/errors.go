package types

import sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

var (
	ErrStableNotSupported = sdkerrors.Register(ModuleName, 1, "stable coin not supported")
)
