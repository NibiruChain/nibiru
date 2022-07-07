package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/perp/keeper"
	"github.com/NibiruChain/nibiru/x/perp/types"
	"github.com/NibiruChain/nibiru/x/testutil/sample"
	"github.com/NibiruChain/nibiru/x/testutil/testapp"
)

func TestQueryPosition(t *testing.T) {
	tests := []struct {
		name            string
		initialPosition *types.Position

		quoteAssetReserve sdk.Dec
		baseAssetReserve  sdk.Dec

		expectedPositionNotional sdk.Dec
		expectedUnrealizedPnl    sdk.Dec
		expectedMarginRatio      sdk.Dec
	}{
		{
			name: "positive PnL",
			initialPosition: &types.Position{
				Pair:                                common.PairBTCStable,
				Size_:                               sdk.NewDec(10),
				OpenNotional:                        sdk.NewDec(10),
				Margin:                              sdk.NewDec(1),
				BlockNumber:                         1,
				LastUpdateCumulativePremiumFraction: sdk.ZeroDec(),
			},
			quoteAssetReserve:        sdk.NewDec(1_000_000),
			baseAssetReserve:         sdk.NewDec(500_000),
			expectedPositionNotional: sdk.MustNewDecFromStr("19.999600007999840003"),
			expectedUnrealizedPnl:    sdk.MustNewDecFromStr("9.999600007999840003"),
			expectedMarginRatio:      sdk.MustNewDecFromStr("0.549991"),
		},
		{
			name: "negative PnL, positive margin ratio",
			initialPosition: &types.Position{
				Pair:                                common.PairBTCStable,
				Size_:                               sdk.NewDec(10),
				OpenNotional:                        sdk.NewDec(10),
				Margin:                              sdk.NewDec(1),
				BlockNumber:                         1,
				LastUpdateCumulativePremiumFraction: sdk.ZeroDec(),
			},
			quoteAssetReserve:        sdk.NewDec(1_000_000),
			baseAssetReserve:         sdk.NewDec(1_000_000),
			expectedPositionNotional: sdk.MustNewDecFromStr("9.99990000099999"),
			expectedUnrealizedPnl:    sdk.MustNewDecFromStr("-0.00009999900001"),
			expectedMarginRatio:      sdk.MustNewDecFromStr("0.099991"),
		},
		{
			name: "negative PnL, negative margin ratio",
			initialPosition: &types.Position{
				Pair:                                common.PairBTCStable,
				Size_:                               sdk.NewDec(10),
				OpenNotional:                        sdk.NewDec(10),
				Margin:                              sdk.NewDec(1),
				BlockNumber:                         1,
				LastUpdateCumulativePremiumFraction: sdk.ZeroDec(),
			},
			quoteAssetReserve:        sdk.NewDec(500_000),
			baseAssetReserve:         sdk.NewDec(1_000_000),
			expectedPositionNotional: sdk.MustNewDecFromStr("4.999950000499995"),
			expectedUnrealizedPnl:    sdk.MustNewDecFromStr("-5.000049999500005"),
			expectedMarginRatio:      sdk.MustNewDecFromStr("-0.800018"),
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Log("initialize trader address")
			traderAddr := sample.AccAddress()
			tc.initialPosition.TraderAddress = traderAddr.String()

			t.Log("initialize app and keeper")
			nibiruApp, ctx := testapp.NewNibiruAppAndContext(true)
			perpKeeper := &nibiruApp.PerpKeeper
			vpoolKeeper := &nibiruApp.VpoolKeeper
			queryServer := keeper.NewQuerier(*perpKeeper)

			t.Log("initialize vpool and pair")
			vpoolKeeper.CreatePool(
				ctx,
				common.PairBTCStable,
				/* tradeLimitRatio */ sdk.OneDec(),
				/* quoteReserve */ tc.quoteAssetReserve,
				/* baseReserve */ tc.baseAssetReserve,
				/* fluctuationLimitRatio */ sdk.OneDec(),
				/* maxOracleSpreadRatio */ sdk.OneDec(),
			)
			perpKeeper.PairMetadataState(ctx).Set(&types.PairMetadata{
				Pair: common.PairBTCStable,
				CumulativePremiumFractions: []sdk.Dec{
					sdk.ZeroDec(),
				},
			})

			t.Log("initialize position")
			perpKeeper.PositionsState(ctx).Set(common.PairBTCStable, traderAddr, tc.initialPosition)

			t.Log("query position")
			resp, err := queryServer.TraderPosition(
				sdk.WrapSDKContext(ctx),
				&types.QueryTraderPositionRequest{
					Trader:    traderAddr.String(),
					TokenPair: common.PairBTCStable.String(),
				},
			)
			require.NoError(t, err)

			t.Log("assert response")
			assert.EqualValues(t, tc.initialPosition, resp.Position)

			assert.Equal(t, tc.expectedPositionNotional, resp.PositionNotional)
			assert.Equal(t, tc.expectedUnrealizedPnl, resp.UnrealizedPnl)
			assert.Equal(t, tc.expectedMarginRatio, resp.MarginRatio)
		})
	}
}
