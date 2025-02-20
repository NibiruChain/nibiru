package ante

import (
	sdkerrors "cosmossdk.io/errors"
	"cosmossdk.io/math"
)

var errorCodeIdx uint32 = 1

func registerError(errMsg string) *sdkerrors.Error {
	errorCodeIdx += 1
	return sdkerrors.Register("ante-nibiru", errorCodeIdx, errMsg)
}

// app/ante "sentinel" errors
var (
	ErrOracleAnte             = registerError("oracle ante error")
	ErrMaxValidatorCommission = registerError("validator commission rate is above max")
)

func NewErrMaxValidatorCommission(gotCommission math.LegacyDec) error {
	return ErrMaxValidatorCommission.Wrapf(
		"got (%s), max rate is (%s)", gotCommission, MAX_COMMISSION())
}
