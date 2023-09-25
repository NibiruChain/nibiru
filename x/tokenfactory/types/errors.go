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
	ErrInvalidGenesis           = registerError("invalid genesis")
	ErrInvalidDenom             = registerError("invalid token factory denom")
	ErrInvalidCreator           = registerError("invalid creator")
	ErrInvalidSubdenom          = registerError("invalid subdenom")
	ErrInvalidAuthorityMetadata = registerError("invalid denom authority metadata")
	ErrDenomAlreadyRegistered   = registerError("attempting to create denom that is already registered (has bank metadata)")
)
