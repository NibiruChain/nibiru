package keeper

import (
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"

	"github.com/stretchr/testify/assert"

	epochtypes "github.com/NibiruChain/nibiru/x/epochs/types"
	"github.com/NibiruChain/nibiru/x/perp/types"
	pftypes "github.com/NibiruChain/nibiru/x/pricefeed/types"
	vpooltypes "github.com/NibiruChain/nibiru/x/vpool/types"
)

var pair = BtcNusdPair

func TestEndOfEpochTwapCalculation(t *testing.T) {
	tests := []struct {
		name                  string
		indexPrice, markPrice int64
		expectedFundingRate   string
	}{
		{
			name: "check empty price",
		},
		{
			name:      "empty index price",
			markPrice: 10,
		},
		{
			name:       "empty mark price",
			indexPrice: 10,
		},
		{
			name:                "equal prices",
			indexPrice:          10,
			markPrice:           10,
			expectedFundingRate: "0",
		},
		{
			name:                "calculate funding rate with higher index price",
			indexPrice:          462,
			markPrice:           19,
			expectedFundingRate: "-18.458333333333333333",
		},
		{
			name:                "calculate funding rate with higher mark price",
			indexPrice:          64,
			markPrice:           745,
			expectedFundingRate: "28.375000000000000000",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			keeper, mocks, ctx := getKeeper(t)
			initParams(ctx, keeper)
			setMockPrices(ctx, mocks, tc.indexPrice, tc.markPrice)
			keeper.AfterEpochEnd(ctx, "hour", 0)
			pair, err := keeper.PairMetadataState(ctx).Get(BtcNusdPair)
			require.NoError(t, err)
			assert.Equal(t, pair.Pair, BtcNusdPair.String())
			expected := []sdk.Dec{sdk.NewDec(0)}
			if tc.expectedFundingRate != "" {
				expected = append(expected, sdk.MustNewDecFromStr(tc.expectedFundingRate))
			}
			assert.Equal(t, expected, pair.CumulativePremiumFractions)
		})
	}
}

func initParams(ctx sdk.Context, k Keeper) {
	k.SetParams(ctx, types.Params{
		Stopped:                 false,
		MaintenanceMarginRatio:  sdk.OneDec(),
		FeePoolFeeRatio:         sdk.MustNewDecFromStr("0.00001"),
		EcosystemFundFeeRatio:   sdk.MustNewDecFromStr("0.000005"),
		LiquidationFeeRatio:     sdk.MustNewDecFromStr("0.000007"),
		PartialLiquidationRatio: sdk.MustNewDecFromStr("0.00001"),
		EpochIdentifier:         "hour",
	})
	k.PairMetadataState(ctx).Set(&types.PairMetadata{
		Pair: BtcNusdPair.String(),
		// start with one entry to ensure we append
		CumulativePremiumFractions: []sdk.Dec{sdk.ZeroDec()},
	})
}

func setMockPrices(ctx sdk.Context, mocks mockedDependencies, indexPrice, markPrice int64) {
	mocks.mockVpoolKeeper.EXPECT().ExistsPool(ctx, gomock.Any()).Return(true)
	if indexPrice != 0 && markPrice != 0 {
		mocks.mockEpochKeeper.EXPECT().GetEpochInfo(ctx, "hour").Return(
			epochtypes.EpochInfo{Duration: time.Hour},
		)
	}
	mocks.mockPricefeedKeeper.EXPECT().
		GetCurrentTWAPPrice(ctx, pair.Token0, pair.Token1).
		Return(pftypes.CurrentTWAP{
			PairID: BtcNusdPair.String(),
			Price:  sdk.NewDec(indexPrice),
		}, nil).MaxTimes(1)
	mocks.mockVpoolKeeper.EXPECT().
		GetCurrentTWAPPrice(ctx, pair).
		Return(vpooltypes.CurrentTWAP{
			PairID: BtcNusdPair.String(),
			Price:  sdk.NewDec(markPrice),
		}, nil).MaxTimes(1)
}
