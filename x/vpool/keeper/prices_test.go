package keeper

import (
	"testing"

	"github.com/NibiruChain/nibiru/x/common"
	pftypes "github.com/NibiruChain/nibiru/x/pricefeed/types"
	"github.com/NibiruChain/nibiru/x/testutil/mock"
	"github.com/NibiruChain/nibiru/x/vpool/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

func TestGetUnderlyingPrice(t *testing.T) {
	tests := []struct {
		name           string
		pair           common.TokenPair
		pricefeedPrice sdk.Dec
	}{
		{
			name:           "correctly fetch underlying price",
			pair:           common.TokenPair("btc:nusd"),
			pricefeedPrice: sdk.NewDec(40000),
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			mockPricefeedKeeper := mock.NewMockPriceKeeper(gomock.NewController(t))
			vpoolKeeper, ctx := VpoolKeeper(t, mockPricefeedKeeper)

			mockPricefeedKeeper.
				EXPECT().
				GetCurrentPrice(
					gomock.Eq(ctx),
					gomock.Eq(tc.pair.GetBaseTokenDenom()),
					gomock.Eq(tc.pair.GetQuoteTokenDenom()),
				).
				Return(
					pftypes.CurrentPrice{
						PairID: tc.pair.String(),
						Price:  tc.pricefeedPrice,
					}, nil,
				)

			price, err := vpoolKeeper.GetUnderlyingPrice(ctx, tc.pair)
			require.NoError(t, err)
			require.EqualValues(t, tc.pricefeedPrice, price)
		})
	}
}

func TestGetSpotPrice(t *testing.T) {
	tests := []struct {
		name              string
		pair              common.TokenPair
		quoteAssetReserve sdk.Dec
		baseAssetReserve  sdk.Dec
		expectedPrice     sdk.Dec
	}{
		{
			name:              "correctly fetch underlying price",
			pair:              common.TokenPair("btc:nusd"),
			quoteAssetReserve: sdk.NewDec(40_000),
			baseAssetReserve:  sdk.NewDec(1),
			expectedPrice:     sdk.NewDec(40000),
		},
		{
			name:              "complex price",
			pair:              common.TokenPair("btc:nusd"),
			quoteAssetReserve: sdk.NewDec(2_489_723_947),
			baseAssetReserve:  sdk.NewDec(34_597_234),
			expectedPrice:     sdk.MustNewDecFromStr("71.963092396345904415"),
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			vpoolKeeper, ctx := VpoolKeeper(t,
				mock.NewMockPriceKeeper(gomock.NewController(t)))

			vpoolKeeper.CreatePool(
				ctx,
				tc.pair.String(),
				/*tradeLimitRatio=*/ sdk.OneDec(),
				tc.quoteAssetReserve,
				tc.baseAssetReserve,
				/*fluctuationLimitratio=*/ sdk.OneDec(),
			)

			price, err := vpoolKeeper.GetSpotPrice(ctx, tc.pair)
			require.NoError(t, err)
			require.EqualValues(t, tc.expectedPrice, price)
		})
	}
}

func TestGetOutputPrice(t *testing.T) {
	tests := []struct {
		name                string
		pair                common.TokenPair
		quoteAssetReserve   sdk.Dec
		baseAssetReserve    sdk.Dec
		baseAmount          sdk.Dec
		direction           types.Direction
		expectedQuoteAmount sdk.Dec
		expectedErr         error
	}{
		{
			name:                "zero base asset means zero price",
			pair:                common.TokenPair("btc:nusd"),
			quoteAssetReserve:   sdk.NewDec(40_000),
			baseAssetReserve:    sdk.NewDec(10_000),
			baseAmount:          sdk.ZeroDec(),
			direction:           types.Direction_ADD_TO_POOL,
			expectedQuoteAmount: sdk.ZeroDec(),
		},
		{
			name:                "simple add base to pool",
			pair:                common.TokenPair("btc:nusd"),
			baseAssetReserve:    sdk.NewDec(1000),
			quoteAssetReserve:   sdk.NewDec(1000),
			baseAmount:          sdk.MustNewDecFromStr("500"),
			direction:           types.Direction_ADD_TO_POOL,
			expectedQuoteAmount: sdk.MustNewDecFromStr("333.333333333333333333"), // rounds down
		},
		{
			name:                "simple remove base from pool",
			pair:                common.TokenPair("btc:nusd"),
			baseAssetReserve:    sdk.NewDec(1000),
			quoteAssetReserve:   sdk.NewDec(1000),
			baseAmount:          sdk.MustNewDecFromStr("500"),
			direction:           types.Direction_REMOVE_FROM_POOL,
			expectedQuoteAmount: sdk.MustNewDecFromStr("1000"),
		},
		{
			name:              "too much base removed results in error",
			pair:              common.TokenPair("btc:nusd"),
			baseAssetReserve:  sdk.NewDec(1000),
			quoteAssetReserve: sdk.NewDec(1000),
			baseAmount:        sdk.MustNewDecFromStr("1000"),
			direction:         types.Direction_REMOVE_FROM_POOL,
			expectedErr:       types.ErrBaseReserveAtZero,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			vpoolKeeper, ctx := VpoolKeeper(t,
				mock.NewMockPriceKeeper(gomock.NewController(t)))

			vpoolKeeper.CreatePool(
				ctx,
				tc.pair.String(),
				/*tradeLimitRatio=*/ sdk.OneDec(),
				tc.quoteAssetReserve,
				tc.baseAssetReserve,
				/*fluctuationLimitRatio=*/ sdk.OneDec(),
			)

			quoteAmount, err := vpoolKeeper.GetOutputPrice(ctx, tc.pair, tc.direction, tc.baseAmount)
			if tc.expectedErr != nil {
				require.ErrorIs(t, err, tc.expectedErr,
					"expected error: %w, got: %w", tc.expectedErr, err)
			} else {
				require.NoError(t, err)
				require.EqualValuesf(t, tc.expectedQuoteAmount, quoteAmount,
					"expected quote: %s, got: %s", tc.expectedQuoteAmount.String(), quoteAmount.String(),
				)
			}
		})
	}
}

func TestGetInputPrice(t *testing.T) {
	tests := []struct {
		name               string
		pair               common.TokenPair
		quoteAssetReserve  sdk.Dec
		baseAssetReserve   sdk.Dec
		quoteAmount        sdk.Dec
		direction          types.Direction
		expectedBaseAmount sdk.Dec
		expectedErr        error
	}{
		{
			name:               "zero base asset means zero price",
			pair:               common.TokenPair("btc:nusd"),
			quoteAssetReserve:  sdk.NewDec(40_000),
			baseAssetReserve:   sdk.NewDec(10_000),
			quoteAmount:        sdk.ZeroDec(),
			direction:          types.Direction_ADD_TO_POOL,
			expectedBaseAmount: sdk.ZeroDec(),
		},
		{
			name:               "simple add base to pool",
			pair:               common.TokenPair("btc:nusd"),
			baseAssetReserve:   sdk.NewDec(1000),
			quoteAssetReserve:  sdk.NewDec(1000),
			quoteAmount:        sdk.NewDec(500),
			direction:          types.Direction_ADD_TO_POOL,
			expectedBaseAmount: sdk.MustNewDecFromStr("333.333333333333333333"), // rounds down
		},
		{
			name:               "simple remove base from pool",
			pair:               common.TokenPair("btc:nusd"),
			baseAssetReserve:   sdk.NewDec(1000),
			quoteAssetReserve:  sdk.NewDec(1000),
			quoteAmount:        sdk.NewDec(500),
			direction:          types.Direction_REMOVE_FROM_POOL,
			expectedBaseAmount: sdk.NewDec(1000),
		},
		{
			name:              "too much base removed results in error",
			pair:              common.TokenPair("btc:nusd"),
			baseAssetReserve:  sdk.NewDec(1000),
			quoteAssetReserve: sdk.NewDec(1000),
			quoteAmount:       sdk.NewDec(1000),
			direction:         types.Direction_REMOVE_FROM_POOL,
			expectedErr:       types.ErrQuoteReserveAtZero,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			vpoolKeeper, ctx := VpoolKeeper(t,
				mock.NewMockPriceKeeper(gomock.NewController(t)))

			vpoolKeeper.CreatePool(
				ctx,
				tc.pair.String(),
				/*tradeLimitRatio=*/ sdk.OneDec(),
				tc.quoteAssetReserve,
				tc.baseAssetReserve,
				/*fluctuationLimitRatio=*/ sdk.OneDec(),
			)

			baseAmount, err := vpoolKeeper.GetInputPrice(ctx, tc.pair, tc.direction, tc.quoteAmount)
			if tc.expectedErr != nil {
				require.ErrorIs(t, err, tc.expectedErr,
					"expected error: %w, got: %w", tc.expectedErr, err)
			} else {
				require.NoError(t, err)
				require.EqualValuesf(t, tc.expectedBaseAmount, baseAmount,
					"expected quote: %s, got: %s", tc.expectedBaseAmount.String(), baseAmount.String(),
				)
			}
		})
	}
}
