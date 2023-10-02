package types

import (
	sdkerrors "cosmossdk.io/errors"
)

var moduleErrorCodeIdx uint32 = 1

func registerError(msg string) *sdkerrors.Error {
	moduleErrorCodeIdx += 1
	return sdkerrors.Register(ModuleName, moduleErrorCodeIdx, msg)
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
)
