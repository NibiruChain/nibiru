package ante

import (
	sdkioerrors "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

var errorCodeIdx uint32 = 1

func registerError(errMsg string) *sdkioerrors.Error {
	errorCodeIdx += 1
	return sdkioerrors.Register("ante-nibiru", errorCodeIdx, errMsg)
}

// app/ante "sentinel" errors
var (
	ErrOracleAnte             = registerError("oracle ante error")
	ErrMaxValidatorCommission = registerError("validator commission rate is above max")
)

func NewErrMaxValidatorCommission(gotCommission sdk.Dec) error {
	return ErrMaxValidatorCommission.Wrapf(
		"got (%s), max rate is (%s)", gotCommission, MAX_COMMISSION())
}
