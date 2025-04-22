package types

import (
	sdkioerrors "cosmossdk.io/errors"
)

var moduleErrorCodeIdx uint32 = 1

func registerError(msg string) *sdkioerrors.Error {
	moduleErrorCodeIdx += 1
	return sdkioerrors.Register(ModuleName, moduleErrorCodeIdx, msg)
}

// Module "sentinel" errors
var (
	ErrInvalidGenesis         = registerError("invalid genesis")
	ErrInvalidDenom           = registerError("invalid token factory denom")
	ErrInvalidCreator         = registerError("invalid creator")
	ErrInvalidSubdenom        = registerError("invalid subdenom")
	ErrInvalidAdmin           = registerError("invalid denom admin")
	ErrDenomAlreadyRegistered = registerError("attempting to create denom that is already registered (has bank metadata)")
	ErrInvalidSender          = registerError("invalid msg sender")
	ErrInvalidModuleParams    = registerError("invalid module params")
	ErrGetAdmin               = registerError("failed to find admin for denom")
	ErrGetMetadata            = registerError("failed to find bank metadata for denom")
	ErrUnauthorized           = registerError("sender must be admin")
	// ErrBlockedAddress: error when the x/bank keeper has an address
	// blocked.
	ErrBlockedAddress = registerError("blocked address")
)
