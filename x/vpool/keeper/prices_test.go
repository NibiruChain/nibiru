package keeper

import (
	"testing"

	"github.com/NibiruChain/nibiru/x/common"
	pftypes "github.com/NibiruChain/nibiru/x/pricefeed/types"
	"github.com/NibiruChain/nibiru/x/testutil/mock"
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
			pricefeedPrice: sdk.MustNewDecFromStr("40000"),
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
		quoteAssetReserve sdk.Int
		baseAssetReserve  sdk.Int
		expectedPrice     sdk.Dec
	}{
		{
			name:              "correctly fetch underlying price",
			pair:              common.TokenPair("btc:nusd"),
			quoteAssetReserve: sdk.NewIntFromUint64(40_000),
			baseAssetReserve:  sdk.NewIntFromUint64(1),
			expectedPrice:     sdk.MustNewDecFromStr("40000"),
		},
		{
			name:              "complex price",
			pair:              common.TokenPair("btc:nusd"),
			quoteAssetReserve: sdk.NewIntFromUint64(2_489_723_947),
			baseAssetReserve:  sdk.NewIntFromUint64(34_597_234),
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
