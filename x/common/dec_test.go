package common_test

import (
	"fmt"
	"math/big"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"

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
		dec     sdk.Dec
		sqrtDec sdk.Dec
	}{
		// --------------------------------------------------------------------
		// Cases: 1 or higher
		{dec: sdk.OneDec(), sqrtDec: sdk.OneDec()},
		{dec: sdk.NewDec(4), sqrtDec: sdk.NewDec(2)},
		{dec: sdk.NewDec(250_000), sqrtDec: sdk.NewDec(500)},
		{dec: sdk.NewDec(4_819_136_400), sqrtDec: sdk.NewDec(69_420)},

		// --------------------------------------------------------------------
		// Cases: Between 0 and 1
		{dec: sdk.MustNewDecFromStr("0.81"), sqrtDec: sdk.MustNewDecFromStr("0.9")},
		{dec: sdk.MustNewDecFromStr("0.25"), sqrtDec: sdk.MustNewDecFromStr("0.5")},
		// ↓ dec 1e-12, sqrtDec: 1e-6
		{dec: sdk.MustNewDecFromStr("0.000000000001"), sqrtDec: sdk.MustNewDecFromStr("0.000001")},

		// --------------------------------------------------------------------
		// The math/big library panics if you call sqrt() on a negative number.
	}

	t.Run("negative sqrt should panic", func(t *testing.T) {
		panicString := common.TryCatch(func() {
			common.MustSqrtDec(sdk.NewDec(-9))
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
		value       sdk.Dec
		clampValue  sdk.Dec
		expected    sdk.Dec
		description string
	}{
		{
			value:       sdk.NewDec(15),
			clampValue:  sdk.NewDec(1),
			expected:    sdk.NewDec(1),
			description: "Clamping 15 to 1",
		},
		{
			value:       sdk.NewDec(-15),
			clampValue:  sdk.NewDec(1),
			expected:    sdk.NewDec(-1),
			description: "Clamping -1.5 to 1",
		},
		{
			value:       sdk.MustNewDecFromStr("0.5"),
			clampValue:  sdk.NewDec(1),
			expected:    sdk.MustNewDecFromStr("0.5"),
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
