package common_test

import (
	"fmt"
	"math/big"
	"testing"

	"cosmossdk.io/math"

	"github.com/stretchr/testify/assert"

	"github.com/NibiruChain/nibiru/x/common"
)

func TestSqrtBigInt(t *testing.T) {
	testCases := []struct {
		bigInt     *big.Int
		sqrtBigInt *big.Int
	}{
		{bigInt: big.NewInt(1), sqrtBigInt: big.NewInt(1)},
		{bigInt: big.NewInt(4), sqrtBigInt: big.NewInt(2)},
		{bigInt: big.NewInt(250_000), sqrtBigInt: big.NewInt(500)},
		{bigInt: big.NewInt(4_819_136_400), sqrtBigInt: big.NewInt(69_420)},
		{
			bigInt:     new(big.Int).Mul(big.NewInt(4_819_136_400), common.BigIntPow10(32)),
			sqrtBigInt: new(big.Int).Mul(big.NewInt(69_420), common.BigIntPow10(16)),
		},
		{
			bigInt:     new(big.Int).Mul(big.NewInt(9), common.BigIntPow10(100)),
			sqrtBigInt: new(big.Int).Mul(big.NewInt(3), common.BigIntPow10(50)),
		},
	}

	for _, testCase := range testCases {
		tc := testCase
		t.Run(fmt.Sprintf(`bigInt: %s, sqrtBigInt: %s`, tc.bigInt, tc.sqrtBigInt), func(t *testing.T) {
			sqrtInt, err := common.SqrtBigInt(tc.bigInt)
			assert.NoError(t, err)
			assert.Equal(t, tc.sqrtBigInt.String(), sqrtInt.String())
		})
	}
}

func TestSqrtDec(t *testing.T) {
	testCases := []struct {
		dec     math.LegacyDec
		sqrtDec math.LegacyDec
	}{
		// --------------------------------------------------------------------
		// Cases: 1 or higher
		{dec: math.LegacyOneDec(), sqrtDec: math.LegacyOneDec()},
		{dec: math.LegacyNewDec(4), sqrtDec: math.LegacyNewDec(2)},
		{dec: math.LegacyNewDec(250_000), sqrtDec: math.LegacyNewDec(500)},
		{dec: math.LegacyNewDec(4_819_136_400), sqrtDec: math.LegacyNewDec(69_420)},

		// --------------------------------------------------------------------
		// Cases: Between 0 and 1
		{dec: math.LegacyMustNewDecFromStr("0.81"), sqrtDec: math.LegacyMustNewDecFromStr("0.9")},
		{dec: math.LegacyMustNewDecFromStr("0.25"), sqrtDec: math.LegacyMustNewDecFromStr("0.5")},
		// ↓ dec 1e-12, sqrtDec: 1e-6
		{dec: math.LegacyMustNewDecFromStr("0.000000000001"), sqrtDec: math.LegacyMustNewDecFromStr("0.000001")},

		// --------------------------------------------------------------------
		// The math/big library panics if you call sqrt() on a negative number.
	}

	t.Run("negative sqrt should panic", func(t *testing.T) {
		panicString := common.TryCatch(func() {
			common.MustSqrtDec(math.LegacyNewDec(-9))
		})().Error()

		assert.Contains(t, panicString, "square root of negative number")
	})

	for _, testCase := range testCases {
		tc := testCase
		t.Run(fmt.Sprintf(`dec: %s, sqrtDec: %s`, tc.dec, tc.sqrtDec), func(t *testing.T) {
			sqrtDec, err := common.SqrtDec(tc.dec)
			assert.NoError(t, err)
			assert.Equal(t, tc.sqrtDec.String(), sqrtDec.String())
		})
	}
}

func TestBankersRound(t *testing.T) {
	quo := big.NewInt(56789)
	halfPrecision := big.NewInt(50000)

	testCases := []struct {
		name    string
		quo     *big.Int
		rem     *big.Int
		rounded *big.Int
	}{
		{
			name:    "Remainder < half precision => round down",
			rem:     big.NewInt(49_999),
			rounded: quo,
		},
		{
			name:    "Remainder > half precision => round up",
			rem:     big.NewInt(50_001),
			rounded: big.NewInt(56_790), // = quo + 1
		},
		{
			name:    "Remainder = half precision, quotient is odd => round up",
			rem:     halfPrecision,
			rounded: big.NewInt(56_742),
			quo:     big.NewInt(56_741),
		},
		{
			name:    "Remainder = half precision, quotient is even => no change",
			rem:     halfPrecision,
			rounded: quo,
		},
		{
			name:    "Remainder = 0 => no change",
			rem:     big.NewInt(0),
			rounded: quo,
		},
	}

	for _, tc := range testCases {
		tcQuo := quo
		if tc.quo != nil {
			tcQuo = tc.quo
		}
		rounded := common.BankersRound(tcQuo, tc.rem, halfPrecision)
		assert.EqualValues(t, tc.rounded, rounded)
	}
}

func TestClamp(t *testing.T) {
	tests := []struct {
		value       math.LegacyDec
		clampValue  math.LegacyDec
		expected    math.LegacyDec
		description string
	}{
		{
			value:       math.LegacyNewDec(15),
			clampValue:  math.LegacyNewDec(1),
			expected:    math.LegacyNewDec(1),
			description: "Clamping 15 to 1",
		},
		{
			value:       math.LegacyNewDec(-15),
			clampValue:  math.LegacyNewDec(1),
			expected:    math.LegacyNewDec(-1),
			description: "Clamping -15 to 1",
		},
		{
			value:       math.LegacyMustNewDecFromStr("0.5"),
			clampValue:  math.LegacyNewDec(1),
			expected:    math.LegacyMustNewDecFromStr("0.5"),
			description: "Clamping 0.5 to 1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			result := common.Clamp(tt.value, tt.clampValue)
			assert.Equal(t, tt.expected, result)
		})
	}
}
