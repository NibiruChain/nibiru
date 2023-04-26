package keeper

import (
	"testing"

	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/common/denoms"
	"github.com/NibiruChain/nibiru/x/common/testutil/mock"
	v2types "github.com/NibiruChain/nibiru/x/perp/types/v2"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSwapQuoteAsset(t *testing.T) {
	tests := []struct {
		name               string
		setMocks           func(ctx sdk.Context, mocks mockedDependencies)
		side               v2types.Direction
		expectedBaseAmount sdk.Dec
	}{
		{
			name: "long position - buy",
			setMocks: func(ctx sdk.Context, mocks mockedDependencies) {
				market := v2types.Market{Pair: asset.Registry.Pair(denoms.BTC, denoms.NUSD)}
				mocks.mockPerpAmmKeeper.EXPECT().
					SwapQuoteForBase(
						ctx,
						market,
						v2types.Direction_LONG,
						/*quoteAmount=*/ sdk.NewDec(10),
						/*baseLimit=*/ sdk.NewDec(1),
						/* skipFluctuationLimitCheck */ false,
					).Return(market, sdk.NewDec(5), nil)
			},
			side:               v2types.Direction_LONG,
			expectedBaseAmount: sdk.NewDec(5),
		},
		{
			name: "short position - sell",
			setMocks: func(ctx sdk.Context, mocks mockedDependencies) {
				market := v2types.Market{Pair: asset.Registry.Pair(denoms.BTC, denoms.NUSD)}
				mocks.mockPerpAmmKeeper.EXPECT().
					SwapQuoteForBase(
						ctx,
						market,
						v2types.Direction_SHORT,
						/*quoteAmount=*/ sdk.NewDec(10),
						/*baseLimit=*/ sdk.NewDec(1),
						/* skipFluctuationLimitCheck */ false,
					).Return(market, sdk.NewDec(5), nil)
			},
			side:               v2types.Direction_SHORT,
			expectedBaseAmount: sdk.NewDec(-5),
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			perpKeeper, mocks, ctx := getKeeper(t)

			tc.setMocks(ctx, mocks)

			market := v2types.Market{Pair: asset.Registry.Pair(denoms.BTC, denoms.NUSD)}

			_, baseAmount, err := perpKeeper.swapQuoteAsset(
				ctx,
				market,
				*mock.TestAMM(),
				tc.side,
				sdk.NewDec(10),
				sdk.NewDec(1),
				false,
			)

			require.NoError(t, err)
			assert.EqualValues(t, tc.expectedBaseAmount, baseAmount)
		})
	}
}
