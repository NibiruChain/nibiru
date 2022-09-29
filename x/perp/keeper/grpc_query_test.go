package keeper_test

import (
	"testing"
	"time"

	"github.com/NibiruChain/nibiru/simapp"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/perp/keeper"
	"github.com/NibiruChain/nibiru/x/perp/types"
	"github.com/NibiruChain/nibiru/x/testutil/sample"
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
				Pair:                            common.Pair_BTC_NUSD,
				Size_:                           sdk.NewDec(10),
				OpenNotional:                    sdk.NewDec(10),
				Margin:                          sdk.NewDec(1),
				BlockNumber:                     1,
				LatestCumulativePremiumFraction: sdk.ZeroDec(),
			},
			quoteAssetReserve: sdk.NewDec(1_000_000),
			baseAssetReserve:  sdk.NewDec(500_000),

			expectedPositionNotional: sdk.MustNewDecFromStr("19.999600007999840003"),
			expectedUnrealizedPnl:    sdk.MustNewDecFromStr("9.999600007999840003"),
			expectedMarginRatio:      sdk.MustNewDecFromStr("0.549991"),
		},
		{
			name: "negative PnL, positive margin ratio",
			initialPosition: &types.Position{
				Pair:                            common.Pair_BTC_NUSD,
				Size_:                           sdk.NewDec(10),
				OpenNotional:                    sdk.NewDec(10),
				Margin:                          sdk.NewDec(1),
				BlockNumber:                     1,
				LatestCumulativePremiumFraction: sdk.ZeroDec(),
			},
			quoteAssetReserve: sdk.NewDec(1_000_000),
			baseAssetReserve:  sdk.NewDec(1_000_000),

			expectedPositionNotional: sdk.MustNewDecFromStr("9.99990000099999"),
			expectedUnrealizedPnl:    sdk.MustNewDecFromStr("-0.00009999900001"),
			expectedMarginRatio:      sdk.MustNewDecFromStr("0.099991"),
		},
		{
			name: "negative PnL, negative margin ratio",
			initialPosition: &types.Position{
				Pair:                            common.Pair_BTC_NUSD,
				Size_:                           sdk.NewDec(10),
				OpenNotional:                    sdk.NewDec(10),
				Margin:                          sdk.NewDec(1),
				BlockNumber:                     1,
				LatestCumulativePremiumFraction: sdk.ZeroDec(),
			},
			quoteAssetReserve: sdk.NewDec(500_000),
			baseAssetReserve:  sdk.NewDec(1_000_000),

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
			nibiruApp, ctx := simapp.NewTestNibiruAppAndContext(true)
			perpKeeper := &nibiruApp.PerpKeeper
			vpoolKeeper := &nibiruApp.VpoolKeeper
			queryServer := keeper.NewQuerier(*perpKeeper)

			t.Log("initialize vpool and pair")
			vpoolKeeper.CreatePool(
				ctx,
				common.Pair_BTC_NUSD,
				/* tradeLimitRatio */ sdk.OneDec(),
				/* quoteReserve */ tc.quoteAssetReserve,
				/* baseReserve */ tc.baseAssetReserve,
				/* fluctuationLimitRatio */ sdk.OneDec(),
				/* maxOracleSpreadRatio */ sdk.OneDec(),
				/* maintenanceMarginRatio */ sdk.MustNewDecFromStr("0.0625"),
				/* maxLeverage */ sdk.MustNewDecFromStr("15"),
			)
			setPairMetadata(nibiruApp.PerpKeeper, ctx, types.PairMetadata{
				Pair: common.Pair_BTC_NUSD,
				CumulativePremiumFractions: []sdk.Dec{
					sdk.ZeroDec(),
				},
			})

			t.Log("initialize position")
			setPosition(*perpKeeper, ctx, *tc.initialPosition)

			t.Log("query position")
			ctx = ctx.WithBlockTime(ctx.BlockTime().Add(time.Second))
			resp, err := queryServer.QueryTraderPosition(
				sdk.WrapSDKContext(ctx),
				&types.QueryTraderPositionRequest{
					Trader:    traderAddr.String(),
					TokenPair: common.Pair_BTC_NUSD.String(),
				},
			)
			require.NoError(t, err)

			t.Log("assert response")
			assert.EqualValues(t, tc.initialPosition, resp.Position)

			assert.Equal(t, tc.expectedPositionNotional, resp.PositionNotional)
			assert.Equal(t, tc.expectedUnrealizedPnl, resp.UnrealizedPnl)
			assert.Equal(t, tc.expectedMarginRatio, resp.MarginRatioMark)
			// assert.Equal(t, tc.expectedMarginRatioIndex, resp.MarginRatioIndex)
			// TODO https://github.com/NibiruChain/nibiru/issues/809
		})
	}
}

func TestQueryFundingRates(t *testing.T) {
	tests := []struct {
		name                string
		initialPairMetadata *types.PairMetadata

		query *types.QueryFundingRatesRequest

		expectErr            bool
		expectedFundingRates []sdk.Dec
	}{
		{
			name: "empty string pair",
			initialPairMetadata: &types.PairMetadata{
				Pair: common.Pair_BTC_NUSD,
				CumulativePremiumFractions: []sdk.Dec{
					sdk.ZeroDec(),
				},
			},
			query: &types.QueryFundingRatesRequest{
				Pair: "",
			},
			expectErr: true,
		},
		{
			name: "pair metadata not found",
			initialPairMetadata: &types.PairMetadata{
				Pair: common.Pair_BTC_NUSD,
				CumulativePremiumFractions: []sdk.Dec{
					sdk.ZeroDec(),
				},
			},
			query: &types.QueryFundingRatesRequest{
				Pair: "foo:bar",
			},
			expectErr: true,
		},
		{
			name: "returns single funding payment",
			initialPairMetadata: &types.PairMetadata{
				Pair: common.Pair_BTC_NUSD,
				CumulativePremiumFractions: []sdk.Dec{
					sdk.ZeroDec(),
				},
			},
			query: &types.QueryFundingRatesRequest{
				Pair: common.Pair_BTC_NUSD.String(),
			},
			expectErr: false,
			expectedFundingRates: []sdk.Dec{
				sdk.ZeroDec(),
			},
		},
		{
			name: "truncates to 48 funding payments",
			initialPairMetadata: &types.PairMetadata{
				Pair: common.Pair_BTC_NUSD,
				CumulativePremiumFractions: []sdk.Dec{
					sdk.ZeroDec(),
					sdk.NewDec(1),
					sdk.NewDec(2),
					sdk.NewDec(3),
					sdk.NewDec(4),
					sdk.NewDec(5),
					sdk.NewDec(6),
					sdk.NewDec(7),
					sdk.NewDec(8),
					sdk.NewDec(9),
					sdk.NewDec(10),
					sdk.NewDec(11),
					sdk.NewDec(12),
					sdk.NewDec(13),
					sdk.NewDec(14),
					sdk.NewDec(15),
					sdk.NewDec(16),
					sdk.NewDec(17),
					sdk.NewDec(18),
					sdk.NewDec(19),
					sdk.NewDec(20),
					sdk.NewDec(21),
					sdk.NewDec(22),
					sdk.NewDec(23),
					sdk.NewDec(24),
					sdk.NewDec(25),
					sdk.NewDec(26),
					sdk.NewDec(27),
					sdk.NewDec(28),
					sdk.NewDec(29),
					sdk.NewDec(30),
					sdk.NewDec(31),
					sdk.NewDec(32),
					sdk.NewDec(33),
					sdk.NewDec(34),
					sdk.NewDec(35),
					sdk.NewDec(36),
					sdk.NewDec(37),
					sdk.NewDec(38),
					sdk.NewDec(39),
					sdk.NewDec(40),
					sdk.NewDec(41),
					sdk.NewDec(42),
					sdk.NewDec(43),
					sdk.NewDec(44),
					sdk.NewDec(45),
					sdk.NewDec(46),
					sdk.NewDec(47),
					sdk.NewDec(48),
				},
			},
			query: &types.QueryFundingRatesRequest{
				Pair: common.Pair_BTC_NUSD.String(),
			},
			expectErr: false,
			expectedFundingRates: []sdk.Dec{
				sdk.NewDec(1),
				sdk.NewDec(2),
				sdk.NewDec(3),
				sdk.NewDec(4),
				sdk.NewDec(5),
				sdk.NewDec(6),
				sdk.NewDec(7),
				sdk.NewDec(8),
				sdk.NewDec(9),
				sdk.NewDec(10),
				sdk.NewDec(11),
				sdk.NewDec(12),
				sdk.NewDec(13),
				sdk.NewDec(14),
				sdk.NewDec(15),
				sdk.NewDec(16),
				sdk.NewDec(17),
				sdk.NewDec(18),
				sdk.NewDec(19),
				sdk.NewDec(20),
				sdk.NewDec(21),
				sdk.NewDec(22),
				sdk.NewDec(23),
				sdk.NewDec(24),
				sdk.NewDec(25),
				sdk.NewDec(26),
				sdk.NewDec(27),
				sdk.NewDec(28),
				sdk.NewDec(29),
				sdk.NewDec(30),
				sdk.NewDec(31),
				sdk.NewDec(32),
				sdk.NewDec(33),
				sdk.NewDec(34),
				sdk.NewDec(35),
				sdk.NewDec(36),
				sdk.NewDec(37),
				sdk.NewDec(38),
				sdk.NewDec(39),
				sdk.NewDec(40),
				sdk.NewDec(41),
				sdk.NewDec(42),
				sdk.NewDec(43),
				sdk.NewDec(44),
				sdk.NewDec(45),
				sdk.NewDec(46),
				sdk.NewDec(47),
				sdk.NewDec(48),
			},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Log("initialize app and keeper")
			nibiruApp, ctx := simapp.NewTestNibiruAppAndContext(true)
			queryServer := keeper.NewQuerier(nibiruApp.PerpKeeper)

			t.Log("initialize pair metadata")
			setPairMetadata(nibiruApp.PerpKeeper, ctx, *tc.initialPairMetadata)

			t.Log("query funding payments")
			resp, err := queryServer.FundingRates(sdk.WrapSDKContext(ctx), tc.query)

			if tc.expectErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)

				t.Log("assert response")
				assert.EqualValues(t, tc.expectedFundingRates, resp.CumulativeFundingRates)
			}
		})
	}
}
