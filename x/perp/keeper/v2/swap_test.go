package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/x/common/testutil/mock"
	"github.com/NibiruChain/nibiru/x/common/testutil/testapp"
	v2types "github.com/NibiruChain/nibiru/x/perp/types/v2"
)

func TestSwapQuoteAsset(t *testing.T) {
	tests := []struct {
		name           string
		direction      v2types.Direction
		quoteAssetAmt  sdk.Dec
		baseAssetLimit sdk.Dec

		expectedBaseAssetDelta sdk.Dec
		expectedAMM            *v2types.AMM
		expectedErr            error
	}{
		{
			name:           "quote amount == 0",
			direction:      v2types.Direction_LONG,
			quoteAssetAmt:  sdk.NewDec(0),
			baseAssetLimit: sdk.NewDec(10),

			expectedAMM: mock.TestAMMDefault().
				WithPriceMultiplier(sdk.NewDec(2)),
			expectedBaseAssetDelta: sdk.ZeroDec(),
		},
		{
			name:           "normal swap add",
			direction:      v2types.Direction_LONG,
			quoteAssetAmt:  sdk.NewDec(100_000),
			baseAssetLimit: sdk.NewDec(49999),

			expectedAMM: mock.TestAMMDefault().
				WithQuoteReserve(sdk.NewDec(1_000_000_050_000)).
				WithBaseReserve(sdk.MustNewDecFromStr("999999950000.002499999875000006")).
				WithPriceMultiplier(sdk.NewDec(2)).
				WithTotalLong(sdk.MustNewDecFromStr("49999.997500000124999994")),
			expectedBaseAssetDelta: sdk.MustNewDecFromStr("49999.997500000124999994"),
		},
		{
			name:           "normal swap remove",
			direction:      v2types.Direction_SHORT,
			quoteAssetAmt:  sdk.NewDec(100_000),
			baseAssetLimit: sdk.NewDec(50_001),

			expectedAMM: mock.TestAMMDefault().
				WithQuoteReserve(sdk.NewDec(999_999_950_000)).
				WithBaseReserve(sdk.MustNewDecFromStr("1000000050000.002500000125000006")).
				WithPriceMultiplier(sdk.NewDec(2)).
				WithTotalShort(sdk.MustNewDecFromStr("50000.002500000125000006")),
			expectedBaseAssetDelta: sdk.MustNewDecFromStr("50000.002500000125000006"),
		},
		{
			name:           "base amount less than base limit in Long",
			direction:      v2types.Direction_LONG,
			quoteAssetAmt:  sdk.NewDec(500_000),
			baseAssetLimit: sdk.NewDec(454_500),

			expectedErr: v2types.ErrAssetFailsUserLimit,
		},
		{
			name:           "base amount more than base limit in Short",
			direction:      v2types.Direction_SHORT,
			quoteAssetAmt:  sdk.NewDec(1e6),
			baseAssetLimit: sdk.NewDec(454_500),

			expectedErr: v2types.ErrAssetFailsUserLimit,
		},
		{
			name:           "over reserve amount when removing quote",
			direction:      v2types.Direction_SHORT,
			quoteAssetAmt:  sdk.NewDec(2e12 + 1),
			baseAssetLimit: sdk.ZeroDec(),

			expectedErr: v2types.ErrQuoteReserveAtZero,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			app, ctx := testapp.NewNibiruTestAppAndContext(true)
			market := mock.TestMarket()
			amm := mock.TestAMMDefault().WithPriceMultiplier(sdk.NewDec(2))

			updatedAMM, baseAmt, err := app.PerpKeeperV2.SwapQuoteAsset(
				ctx,
				*market,
				*amm,
				tc.direction,
				tc.quoteAssetAmt,
				tc.baseAssetLimit,
			)

			if tc.expectedErr != nil {
				require.ErrorIs(t, err, tc.expectedErr)
			} else {
				require.NoError(t, err)
				assert.EqualValuesf(t, tc.expectedBaseAssetDelta, baseAmt, "base amount mismatch")

				require.NoError(t, err)
				assert.Equal(t, *tc.expectedAMM, *updatedAMM)
			}
		})
	}
}

func TestSwapBaseAsset(t *testing.T) {
	tests := []struct {
		name            string
		direction       v2types.Direction
		baseAssetAmt    sdk.Dec
		quoteAssetLimit sdk.Dec

		expectedAMM             *v2types.AMM
		expectedQuoteAssetDelta sdk.Dec
		expectedErr             error
	}{
		{
			name:            "zero base asset swap",
			direction:       v2types.Direction_LONG,
			baseAssetAmt:    sdk.ZeroDec(),
			quoteAssetLimit: sdk.ZeroDec(),

			expectedAMM: mock.TestAMMDefault().
				WithPriceMultiplier(sdk.NewDec(2)),
			expectedQuoteAssetDelta: sdk.ZeroDec(),
		},
		{
			name:            "go long",
			direction:       v2types.Direction_LONG,
			baseAssetAmt:    sdk.NewDec(100_000),
			quoteAssetLimit: sdk.NewDec(200_000),

			expectedAMM: mock.TestAMMDefault().
				WithBaseReserve(sdk.NewDec(999999900000)).
				WithQuoteReserve(sdk.MustNewDecFromStr("1000000100000.010000001000000100")).
				WithPriceMultiplier(sdk.NewDec((2))).
				WithTotalShort(sdk.NewDec(100_000)),
			expectedQuoteAssetDelta: sdk.MustNewDecFromStr("200000.020000002000000200"),
		},
		{
			name:            "go short",
			direction:       v2types.Direction_SHORT,
			baseAssetAmt:    sdk.NewDec(100_000),
			quoteAssetLimit: sdk.NewDec(200_000),

			expectedQuoteAssetDelta: sdk.MustNewDecFromStr("199999.980000001999999800"),
			expectedAMM: mock.TestAMMDefault().
				WithBaseReserve(sdk.NewDec(1000000100000)).
				WithQuoteReserve(sdk.MustNewDecFromStr("999999900000.009999999000000100")).
				WithPriceMultiplier(sdk.NewDec((2))).
				WithTotalShort(sdk.NewDec(-100_000)),
		},
		{
			name:            "quote asset amt less than quote limit in Long",
			direction:       v2types.Direction_LONG,
			baseAssetAmt:    sdk.NewDec(100_000),
			quoteAssetLimit: sdk.NewDec(200_001),

			expectedErr: v2types.ErrAssetFailsUserLimit,
		},
		{
			name:            "quote amount more than quote limit in Short",
			direction:       v2types.Direction_SHORT,
			baseAssetAmt:    sdk.NewDec(100_000),
			quoteAssetLimit: sdk.NewDec(199_999),

			expectedErr: v2types.ErrAssetFailsUserLimit,
		},
		{
			name:            "over reserve amount when removing base",
			direction:       v2types.Direction_LONG,
			baseAssetAmt:    sdk.NewDec(1e12 + 1),
			quoteAssetLimit: sdk.ZeroDec(),

			expectedErr: v2types.ErrBaseReserveAtZero,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			app, ctx := testapp.NewNibiruTestAppAndContext(true)
			market := mock.TestMarket()
			amm := mock.TestAMMDefault().WithPriceMultiplier(sdk.NewDec(2))

			updatedAMM, quoteAssetAmount, err := app.PerpKeeperV2.SwapBaseAsset(
				ctx,
				*market,
				*amm,
				tc.direction,
				tc.baseAssetAmt,
				tc.quoteAssetLimit,
			)

			if tc.expectedErr != nil {
				require.ErrorContains(t, err, tc.expectedErr.Error())
			} else {
				require.NoError(t, err)
				assert.EqualValuesf(t, tc.expectedQuoteAssetDelta, quoteAssetAmount,
					"expected %s; got %s", tc.expectedQuoteAssetDelta.String(), quoteAssetAmount.String())

				assert.Equal(t, *tc.expectedAMM, *updatedAMM)
			}
		})
	}
}
