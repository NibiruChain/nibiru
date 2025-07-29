package types

import sdkioerrors "cosmossdk.io/errors"

// x/txfees module errors.
var (
	ErrNoBaseDenom = sdkioerrors.Register(ModuleName, 1, "no base denom was set")
)
