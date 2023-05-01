package common

import (
	"fmt"
	"math/big"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// MustSqrtDec computes the square root of the input decimal using its
// underlying big.Int. The big.Int.Sqrt method is part of the standard library,
// thoroughly tested, works at seemingly unbound precision (e.g. for numbers as
// large as 10**99.
//   - NOTE, MustSqrtDec panics if it is called on a negative number, similar to the
//     sdk.NewCoin and SqrtBigInt functions. A panic safe version of MustSqrtDec
//     is available in the SqrtDec method.
func MustSqrtDec(dec sdk.Dec) sdk.Dec {
	sqrtBigInt := MustSqrtBigInt(dec.BigInt())
	precision := sdk.NewDecFromBigInt(PRECISION_MULT)
	return sdk.NewDecFromBigInt(sqrtBigInt).Quo(precision)
}

// SqrtDec computes the square root of the input decimal using its
// underlying big.Int. SqrtDec is panic-safe and returns an error if the input
// decimal is negative.
//
// The big.Int.Sqrt method is part of the standard library,
// thoroughly tested, works at seemingly unbound precision (e.g. for numbers as
// large as 10**99.
func SqrtDec(dec sdk.Dec) (sdk.Dec, error) {
	var sqrtDec sdk.Dec
	var panicErr error = TryCatch(func() {
		sqrtDec = MustSqrtDec(dec)
	})()
	return sqrtDec, panicErr
}

// MustSqrtBigInt returns the square root of the input.
//   - NOTE: MustSqrtBigInt panics if it is called on a negative number because it uses
//     the `big.Int.Sqrt` from the "math/big" package.
func MustSqrtBigInt(i *big.Int) *big.Int {
	sqrtInt := &big.Int{}
	return sqrtInt.Sqrt(i)
}

// SqrtInt is the panic-safe version of MustSqrtBigInt
func SqrtBigInt(i *big.Int) (*big.Int, error) {
	sqrtInt := new(big.Int)
	var panicErr error = TryCatch(func() {
		*sqrtInt = *MustSqrtBigInt(i)
	})()
	return sqrtInt, panicErr
}

// BigIntPow10 returns a big int that is a power of 10, e.g. BigIngPow10(3)
// returns 1000. This function is useful for creating large numbers outside the
// range of an int64 or 18 decimal precision.
func BigIntPow10(power int64) *big.Int {
	bigInt, _ := new(big.Int).SetString("1"+strings.Repeat("0", int(power)), 10)
	return bigInt
}

// ————————————————————————————————————————————————
// Logic needed from private code in the Cosmos-SDK
// See https://github.com/cosmos/cosmos-sdk/blob/v0.45.12/types/decimal.go
//

const (
	PRECISION = 18
)

var (
	PRECISION_MULT = calcPrecisionMultiplier(0)
	PRECISION_SQRT = int64(PRECISION / 2)
	tenInt         = big.NewInt(10)
)

// calcPrecisionMultiplier computes a multiplier needed to maintain a target
// precision defined by 10 ** (PRECISION_SQRT - prec).
// The maximum available precision is PRECISION_SQRT (9).
func calcPrecisionMultiplier(prec int64) *big.Int {
	if prec > PRECISION_SQRT {
		panic(fmt.Sprintf("too much precision, maximum %v, provided %v", PRECISION_SQRT, prec))
	}
	zerosToAdd := PRECISION_SQRT - prec
	multiplier := new(big.Int).Exp(tenInt, big.NewInt(zerosToAdd), nil)
	return multiplier
}
