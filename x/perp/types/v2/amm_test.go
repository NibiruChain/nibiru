package v2_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/x/common/testutil/mock"
	v2 "github.com/NibiruChain/nibiru/x/perp/types/v2"
)

func TestSwapBaseAsset(t *testing.T) {
	tests := []struct {
		name                    string
		baseAssetAmt            sdk.Dec
		dir                     v2.Direction
		expectedQuoteAssetDelta sdk.Dec
		expectedBaseReserve     sdk.Dec
		expectedQuoteReserve    sdk.Dec
		expectedBias            sdk.Dec
		expectedMarkPrice       sdk.Dec
		expectedErr             error
	}{
		{
			name:                    "long base asset",
			baseAssetAmt:            sdk.NewDec(1e11),
			dir:                     v2.Direction_LONG,
			expectedQuoteAssetDelta: sdk.MustNewDecFromStr("111111111111.111111111111111111"),
			expectedBaseReserve:     sdk.NewDec(900000000000),
			expectedQuoteReserve:    sdk.MustNewDecFromStr("1111111111111.111111111111111111"),
			expectedBias:            sdk.NewDec(100000000000),
			expectedMarkPrice:       sdk.MustNewDecFromStr("1.234567901234567901"),
		},
		{
			name:                    "short base asset",
			baseAssetAmt:            sdk.NewDec(1e11),
			dir:                     v2.Direction_SHORT,
			expectedQuoteAssetDelta: sdk.MustNewDecFromStr("90909090909.090909090909090909"),
			expectedBaseReserve:     sdk.NewDec(1100000000000),
			expectedQuoteReserve:    sdk.MustNewDecFromStr("909090909090.909090909090909091"),
			expectedBias:            sdk.NewDec(-100000000000),
			expectedMarkPrice:       sdk.MustNewDecFromStr("0.826446280991735537"),
		},
		{
			name:                    "long zero base asset",
			baseAssetAmt:            sdk.ZeroDec(),
			dir:                     v2.Direction_LONG,
			expectedQuoteAssetDelta: sdk.ZeroDec(),
			expectedBaseReserve:     sdk.NewDec(1e12),
			expectedQuoteReserve:    sdk.NewDec(1e12),
			expectedBias:            sdk.ZeroDec(),
			expectedMarkPrice:       sdk.OneDec(),
		},
		{
			name:                    "short zero base asset",
			baseAssetAmt:            sdk.ZeroDec(),
			dir:                     v2.Direction_SHORT,
			expectedQuoteAssetDelta: sdk.ZeroDec(),
			expectedBaseReserve:     sdk.NewDec(1e12),
			expectedQuoteReserve:    sdk.NewDec(1e12),
			expectedBias:            sdk.ZeroDec(),
			expectedMarkPrice:       sdk.OneDec(),
		},
		{
			name:         "not enough base in reserves",
			baseAssetAmt: sdk.NewDec(1e13),
			dir:          v2.Direction_LONG,
			expectedErr:  v2.ErrBaseReserveAtZero,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			amm := mock.TestAMM(sdk.NewDec(1e12), sdk.OneDec())

			quoteAssetDelta, err := amm.SwapBaseAsset(tc.baseAssetAmt, tc.dir)

			if tc.expectedErr != nil {
				require.ErrorIs(t, err, tc.expectedErr)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.expectedQuoteAssetDelta, quoteAssetDelta)
				assert.Equal(t, v2.AMM{
					Pair:            amm.Pair,
					BaseReserve:     tc.expectedBaseReserve,
					QuoteReserve:    tc.expectedQuoteReserve,
					SqrtDepth:       amm.SqrtDepth,
					PriceMultiplier: amm.PriceMultiplier,
					Bias:            tc.expectedBias,
				}, *amm)
				assert.Equal(t, tc.expectedMarkPrice, amm.MarkPrice())
			}
		})
	}
}

func TestSwapQuoteAsset(t *testing.T) {
	tests := []struct {
		name                   string
		quoteAssetAmt          sdk.Dec
		dir                    v2.Direction
		expectedBaseAssetDelta sdk.Dec
		expectedBaseReserve    sdk.Dec
		expectedQuoteReserve   sdk.Dec
		expectedBias           sdk.Dec
		expectedMarkPrice      sdk.Dec
		expectedErr            error
	}{
		{
			name:                   "long quote asset",
			quoteAssetAmt:          sdk.NewDec(1e11),
			dir:                    v2.Direction_LONG,
			expectedBaseAssetDelta: sdk.MustNewDecFromStr("47619047619.047619047619047619"),
			expectedBaseReserve:    sdk.MustNewDecFromStr("952380952380.952380952380952381"),
			expectedQuoteReserve:   sdk.NewDec(1050000000000),
			expectedBias:           sdk.MustNewDecFromStr("47619047619.047619047619047619"),
			expectedMarkPrice:      sdk.MustNewDecFromStr("2.205"),
		},
		{
			name:                   "short base asset",
			quoteAssetAmt:          sdk.NewDec(1e11),
			dir:                    v2.Direction_SHORT,
			expectedBaseAssetDelta: sdk.MustNewDecFromStr("52631578947.368421052631578947"),
			expectedBaseReserve:    sdk.MustNewDecFromStr("1052631578947.368421052631578947"),
			expectedQuoteReserve:   sdk.NewDec(950000000000),
			expectedBias:           sdk.MustNewDecFromStr("-52631578947.368421052631578947"),
			expectedMarkPrice:      sdk.MustNewDecFromStr("1.805"),
		},
		{
			name:                   "long zero base asset",
			quoteAssetAmt:          sdk.ZeroDec(),
			dir:                    v2.Direction_LONG,
			expectedBaseAssetDelta: sdk.ZeroDec(),
			expectedBaseReserve:    sdk.NewDec(1e12),
			expectedQuoteReserve:   sdk.NewDec(1e12),
			expectedBias:           sdk.ZeroDec(),
			expectedMarkPrice:      sdk.NewDec(2),
		},
		{
			name:                   "long zero base asset",
			quoteAssetAmt:          sdk.ZeroDec(),
			dir:                    v2.Direction_SHORT,
			expectedBaseAssetDelta: sdk.ZeroDec(),
			expectedBaseReserve:    sdk.NewDec(1e12),
			expectedQuoteReserve:   sdk.NewDec(1e12),
			expectedBias:           sdk.ZeroDec(),
			expectedMarkPrice:      sdk.NewDec(2),
		},
		{
			name:          "not enough base in reserves",
			quoteAssetAmt: sdk.NewDec(1e13),
			dir:           v2.Direction_SHORT,
			expectedErr:   v2.ErrQuoteReserveAtZero,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			amm := mock.TestAMM(sdk.NewDec(1e12), sdk.NewDec(2))

			quoteAssetDelta, err := amm.SwapQuoteAsset(tc.quoteAssetAmt, tc.dir)

			if tc.expectedErr != nil {
				require.ErrorIs(t, err, tc.expectedErr)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.expectedBaseAssetDelta, quoteAssetDelta)
				assert.Equal(t, v2.AMM{
					Pair:            amm.Pair,
					BaseReserve:     tc.expectedBaseReserve,
					QuoteReserve:    tc.expectedQuoteReserve,
					SqrtDepth:       amm.SqrtDepth,
					PriceMultiplier: amm.PriceMultiplier,
					Bias:            tc.expectedBias,
				}, *amm)
				assert.Equal(t, tc.expectedMarkPrice, amm.MarkPrice())
			}
		})
	}
}
