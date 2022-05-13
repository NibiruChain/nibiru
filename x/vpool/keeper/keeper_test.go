package keeper

import (
	"fmt"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/testutil/mock"
	"github.com/NibiruChain/nibiru/x/vpool/types"
)

func TestSwapQuoteForBase_Errors(t *testing.T) {
	tests := []struct {
		name        string
		pair        common.TokenPair
		direction   types.Direction
		quoteAmount sdk.Dec
		baseLimit   sdk.Dec
		error       error
	}{
		{
			"pair not supported",
			"BTC:UST",
			types.Direction_ADD_TO_POOL,
			sdk.NewDec(10),
			sdk.NewDec(10),
			types.ErrPairNotSupported,
		},
		{
			"base amount less than base limit in Long",
			NUSDPair,
			types.Direction_ADD_TO_POOL,
			sdk.NewDec(500_000),
			sdk.NewDec(454_500),
			fmt.Errorf("base amount (238095) is less than selected limit (454500)"),
		},
		{
			"base amount more than base limit in Short",
			NUSDPair,
			types.Direction_REMOVE_FROM_POOL,
			sdk.NewDec(1_000_000),
			sdk.NewDec(454_500),
			fmt.Errorf("base amount (555556) is greater than selected limit (454500)"),
		},
		{
			"quote input bigger than reserve ratio",
			NUSDPair,
			types.Direction_REMOVE_FROM_POOL,
			sdk.NewDec(10_000_000),
			sdk.NewDec(10),
			types.ErrOvertradingLimit,
		},
		{
			"over fluctuation limit fails",
			NUSDPair,
			types.Direction_ADD_TO_POOL,
			sdk.NewDec(1_000_000),
			sdk.NewDec(454544),
			fmt.Errorf("error updating reserve: %w", types.ErrOverFluctuationLimit),
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			vpoolKeeper, ctx := VpoolKeeper(t,
				mock.NewMockPriceKeeper(gomock.NewController(t)),
			)

			vpoolKeeper.CreatePool(
				ctx,
				NUSDPair,
				sdk.MustNewDecFromStr("0.9"), // 0.9 ratio
				sdk.NewDec(10_000_000),       // 10
				sdk.NewDec(5_000_000),        // 5
				sdk.MustNewDecFromStr("0.1"), // 0.1 fluctuation limit ratio
			)

			_, err := vpoolKeeper.SwapQuoteForBase(
				ctx,
				tc.pair,
				tc.direction,
				tc.quoteAmount,
				tc.baseLimit,
			)
			require.Error(t, err)
		})
	}
}

func TestSwapQuoteForBase_HappyPath(t *testing.T) {
	tests := []struct {
		name                 string
		direction            types.Direction
		quoteAmount          sdk.Dec
		baseLimit            sdk.Dec
		expectedQuoteReserve sdk.Dec
		expectedBaseReserve  sdk.Dec
		resp                 sdk.Dec
	}{
		{
			"quote amount == 0",
			types.Direction_ADD_TO_POOL,
			sdk.NewDec(0),
			sdk.NewDec(10),
			sdk.NewDec(10_000_000),
			sdk.NewDec(5_000_000),
			sdk.ZeroDec(),
		},
		{
			"normal swap add",
			types.Direction_ADD_TO_POOL,
			sdk.NewDec(1_000_000),
			sdk.NewDec(454_500),
			sdk.NewDec(11_000_000),
			sdk.MustNewDecFromStr("4545454.545454545454545455"),
			sdk.MustNewDecFromStr("454545.454545454545454545"),
		},
		{
			"normal swap remove",
			types.Direction_REMOVE_FROM_POOL,
			sdk.NewDec(1_000_000),
			sdk.NewDec(555_560),
			sdk.NewDec(9_000_000),
			sdk.MustNewDecFromStr("5555555.555555555555555556"),
			sdk.MustNewDecFromStr("555555.555555555555555556"),
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			vpoolKeeper, ctx := VpoolKeeper(t,
				mock.NewMockPriceKeeper(gomock.NewController(t)),
			)

			vpoolKeeper.CreatePool(
				ctx,
				NUSDPair,
				sdk.MustNewDecFromStr("0.9"),  // 0.9 ratio
				sdk.NewDec(10_000_000),        // 10 tokens
				sdk.NewDec(5_000_000),         // 5 tokens
				sdk.MustNewDecFromStr("0.25"), // 0.25 ratio
			)

			res, err := vpoolKeeper.SwapQuoteForBase(
				ctx,
				NUSDPair,
				tc.direction,
				tc.quoteAmount,
				tc.baseLimit,
			)
			require.NoError(t, err)
			require.Equal(t, tc.resp, res)

			pool, err := vpoolKeeper.getPool(ctx, NUSDPair)
			require.NoError(t, err)
			require.Equal(t, tc.expectedQuoteReserve, pool.QuoteAssetReserve)
			require.Equal(t, tc.expectedBaseReserve, pool.BaseAssetReserve)
		})
	}
}

func TestSwapBaseForQuote(t *testing.T) {
	tests := []struct {
		name                     string
		initialQuoteReserve      sdk.Dec
		initialBaseReserve       sdk.Dec
		direction                types.Direction
		baseAssetAmount          sdk.Dec
		quoteAssetLimit          sdk.Dec
		expectedQuoteReserve     sdk.Dec
		expectedBaseReserve      sdk.Dec
		expectedQuoteAssetAmount sdk.Dec
		expectedErr              error
	}{
		{
			name:                     "zero base asset swap",
			initialQuoteReserve:      sdk.NewDec(10_000_000),
			initialBaseReserve:       sdk.NewDec(5_000_000),
			direction:                types.Direction_ADD_TO_POOL,
			baseAssetAmount:          sdk.ZeroDec(),
			quoteAssetLimit:          sdk.ZeroDec(),
			expectedQuoteReserve:     sdk.NewDec(10_000_000),
			expectedBaseReserve:      sdk.NewDec(5_000_000),
			expectedQuoteAssetAmount: sdk.ZeroDec(),
		},
		{
			name:                     "add base asset swap",
			initialQuoteReserve:      sdk.NewDec(10_000_000),
			initialBaseReserve:       sdk.NewDec(5_000_000),
			direction:                types.Direction_ADD_TO_POOL,
			baseAssetAmount:          sdk.NewDec(1_000_000),
			quoteAssetLimit:          sdk.NewDec(1_666_666),
			expectedQuoteReserve:     sdk.MustNewDecFromStr("8333333.333333333333333333"),
			expectedBaseReserve:      sdk.NewDec(6_000_000),
			expectedQuoteAssetAmount: sdk.MustNewDecFromStr("1666666.666666666666666667"),
		},
		{
			name:                     "remove base asset",
			initialQuoteReserve:      sdk.NewDec(10_000_000),
			initialBaseReserve:       sdk.NewDec(5_000_000),
			direction:                types.Direction_REMOVE_FROM_POOL,
			baseAssetAmount:          sdk.NewDec(1_000_000),
			quoteAssetLimit:          sdk.NewDec(2_500_001),
			expectedQuoteReserve:     sdk.NewDec(12_500_000),
			expectedBaseReserve:      sdk.NewDec(4_000_000),
			expectedQuoteAssetAmount: sdk.NewDec(2_500_000),
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			vpoolKeeper, ctx := VpoolKeeper(t,
				mock.NewMockPriceKeeper(gomock.NewController(t)),
			)

			vpoolKeeper.CreatePool(
				ctx,
				NUSDPair,
				sdk.OneDec(),
				tc.initialQuoteReserve,
				tc.initialBaseReserve,
				sdk.OneDec(),
			)

			quoteAssetAmount, err := vpoolKeeper.SwapBaseForQuote(
				ctx,
				NUSDPair,
				tc.direction,
				tc.baseAssetAmount,
				tc.quoteAssetLimit,
			)

			if tc.expectedErr != nil {
				require.Error(t, err)
			} else {
				pool, err := vpoolKeeper.getPool(ctx, NUSDPair)
				require.NoError(t, err)

				require.EqualValuesf(t, tc.expectedQuoteAssetAmount, quoteAssetAmount,
					"expected %s; got %s", tc.expectedQuoteAssetAmount.String(), quoteAssetAmount.String())
				require.NoError(t, err)
				require.Equal(t, tc.expectedQuoteReserve, pool.QuoteAssetReserve)
				require.Equal(t, tc.expectedBaseReserve, pool.BaseAssetReserve)

				snapshot, _, err := vpoolKeeper.getLatestReserveSnapshot(ctx, NUSDPair)
				require.NoError(t, err)
				require.EqualValues(t, tc.expectedQuoteReserve, snapshot.QuoteAssetReserve)
				require.EqualValues(t, tc.expectedBaseReserve, snapshot.BaseAssetReserve)
			}
		})
	}
}
