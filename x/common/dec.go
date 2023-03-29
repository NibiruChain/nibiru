package common

import (
	"fmt"
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Computes the square root of the input decimal using its underlying big.Int
// The big.Int.Sqrt method is part of the standard library and thoroughly tested.
//   - NOTE, MustSqrtDec panics if it is called on a negative number, similar to the
//     sdk.NewCoin and SqrtBigInt functions.
func MustSqrtDec(dec sdk.Dec) sdk.Dec {
	sqrtBigInt := SqrtBigInt(dec.BigInt())
	precision := sdk.NewDecFromBigInt(PRECISION_MULT)
	return sdk.NewDecFromBigInt(sqrtBigInt).Quo(precision)
}

func SqrtDec(dec sdk.Dec) (sdk.Dec, error) {
	var sqrtDec sdk.Dec
	var panicErr error = TryCatch(func() {
		sqrtDec = MustSqrtDec(dec)
	})()
	return sqrtDec, panicErr
}

// SqrtBigInt returns the square root of the input.
//   - NOTE: SqrtBigInt panics if it is called on a negative number because it uses
//     the `big.Int.Sqrt` from the "math/big" package.
func SqrtBigInt(i *big.Int) *big.Int {
	sqrtInt := &big.Int{}
	return sqrtInt.Sqrt(i)
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

// calculate the precision multiplier
func calcPrecisionMultiplier(prec int64) *big.Int {
	if prec > PRECISION_SQRT {
		panic(fmt.Sprintf("too much precision, maximum %v, provided %v", PRECISION_SQRT, prec))
	}
	zerosToAdd := PRECISION_SQRT - prec
	multiplier := new(big.Int).Exp(tenInt, big.NewInt(zerosToAdd), nil)
	return multiplier
}
