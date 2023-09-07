package types

import sdkerrors "cosmossdk.io/errors"

var ErrUnauthorized = sdkerrors.Register(ModuleName, 2, "unauthorized: missing sudo permissions")
