package eth

import (
	sdkerrors "cosmossdk.io/errors"
)

var moduleErrorCodeIdx uint32 = 1

func registerError(msg string) *sdkerrors.Error {
	moduleErrorCodeIdx += 1
	return sdkerrors.Register("eth", moduleErrorCodeIdx, msg)
}

// Module "sentinel" errors
var (
	ErrInvalidChainID = registerError("invalid Ethereum chain ID")
)
