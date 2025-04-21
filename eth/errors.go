package eth

import (
	sdkioerrors "cosmossdk.io/errors"
)

var moduleErrorCodeIdx uint32 = 1

func registerError(msg string) *sdkioerrors.Error {
	moduleErrorCodeIdx += 1
	return sdkioerrors.Register("eth", moduleErrorCodeIdx, msg)
}

// Module "sentinel" errors
var (
	ErrInvalidChainID = registerError("invalid Ethereum chain ID")
)
