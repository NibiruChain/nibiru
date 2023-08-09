package common

import (
	"fmt"
	"math/big"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	// number of decimal places
	PrecisionExponent = 18

	// bits required to represent the above precision
	// Ceiling[Log2[10^Precision - 1]]
	DecimalPrecisionBits = 60

	// decimalTruncateBits is the minimum number of bits removed
	// by a truncate operation. It is equal to
	// Floor[Log2[10^Precision - 1]].
	decimalTruncateBits = DecimalPrecisionBits - 1
	maxBitLen           = 256
	MaxDecBitLen        = maxBitLen + decimalTruncateBits
)

var (
	// precisionInt: 10 ** PrecisionExponent
	precisionInt = new(big.Int).Exp(big.NewInt(10), big.NewInt(PrecisionExponent), nil)
	// halfPrecisionInt: (10 ** PrecisionExponent) / 2
	halfPrecisionInt = new(big.Int).Quo(precisionInt, big.NewInt(2))
	oneInt           = big.NewInt(1)
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

//     ____
//  __|    |__   "chop 'em
//       ` \     round!"
// ___||  ~  _     -bankers
// |         |      __
// |       | |   __|__|__
// |_____:  /   | $$$    |
//              |________|

// ChopPrecisionAndRound: Remove a Precision amount of rightmost digits and
// perform bankers rounding on the remainder (gaussian rounding) on the digits
// which have been removed.
//
// Mutates the input. Use the non-mutative version if that is undesired
func ChopPrecisionAndRound(d *big.Int) *big.Int {
	// remove the negative and add it back when returning
	if d.Sign() == -1 {
		// make d positive, compute chopped value, and then un-mutate d
		d = d.Neg(d)
		d = ChopPrecisionAndRound(d)
		d = d.Neg(d)
		return d
	}

	// Divide out the 'precisionInt', which truncates to a quotient and remainder.
	quo, rem := d, big.NewInt(0)
	quo, rem = quo.QuoRem(d, precisionInt, rem)

	return BankersRound(quo, rem, halfPrecisionInt)
}

// BankersRound: Banker's rounding is a method commonly used in banking and
// accounting to reduce roudning bias when processing large volumes of rounded
// numbers.
//
// 1. If the remainder < half precision, round down
// 2. If the remainder > half precision, round up
// 3. If remainder == half precision,  round to the nearest even number
//
// The name comes from the idea that it provides egalitarian rounding that
// doesn't consistently favor one party over another (e.g. always rounding up).
// With this method, rounding errors tend to cancel out rather than
// accumulating in one direction.
func BankersRound(quo, rem, halfPrecision *big.Int) *big.Int {
	// Zero remainder after dividing precision means => no rounding is needed.
	if rem.Sign() == 0 {
		return quo
	}

	// Nonzero remainder after dividing precision means => do banker's rounding
	switch rem.Cmp(halfPrecision) {
	case -1:
		return quo
	case 1:
		return quo.Add(quo, oneInt)
	default:
		// default case: bankers rounding must take place
		// always round to an even number
		if quo.Bit(0) == 0 {
			return quo
		}
		return quo.Add(quo, oneInt)
	}
}
