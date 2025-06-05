package types

// DONTCOVER

import (
	sdkioerrors "cosmossdk.io/errors"
)

// x/epochs module sentinel errors.
var (
	ErrSample = sdkioerrors.Register(ModuleName, 1100, "sample error")
)
