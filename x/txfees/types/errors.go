package types

import  sdkioerrors "cosmossdk.io/errors"


// x/txfees module errors.
var (
	ErrNoBaseDenom                  = sdkioerrors.Register(ModuleName, 1, "no base denom was set")
	ErrTooManyFeeCoins              = sdkioerrors.Register(ModuleName, 2, "too many fee coins. only accepts fees in one denom")
	ErrInvalidFeeToken              = sdkioerrors.Register(ModuleName, 3, "invalid fee token")
	ErrNotWhitelistedFeeTokenSetter = sdkioerrors.Register(ModuleName, 4, "not whitelisted fee token setter")
)
