package keeper_test

import (
	"testing"
	"time"

	"github.com/cosmos/cosmos-sdk/simapp"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/common"

	"github.com/NibiruChain/nibiru/x/perp/types"
	"github.com/NibiruChain/nibiru/x/testutil/sample"
	"github.com/NibiruChain/nibiru/x/testutil/testapp"

	"github.com/stretchr/testify/require"
)

func TestGetAndSetPosition(t *testing.T) {
	testCases := []struct {
		name string
		test func()
	}{
		{
			name: "get - no positions set raises vpool not found error",
			test: func() {
				trader := sample.AccAddress()
				nibiruApp, ctx := testapp.NewNibiruAppAndContext(true)
				pair := common.MustNewAssetPair("osmo:nusd")

				_, err := nibiruApp.PerpKeeper.PositionsState(ctx).Get(pair, trader)
				require.Error(t, err)
				require.ErrorContains(t, err, types.ErrPositionNotFound.Error())
			},
		},
		{
			name: "set - creating position with set works and shows up in get",
			test: func() {
				vpoolPair := common.MustNewAssetPair("osmo:nusd")

				traderAddr := sample.AccAddress()
				nibiruApp, ctx := testapp.NewNibiruAppAndContext(true)

				_, err := nibiruApp.PerpKeeper.PositionsState(ctx).Get(vpoolPair, traderAddr)
				require.Error(t, err)
				require.ErrorContains(t, err, types.ErrPositionNotFound.Error())

				dummyPosition := &types.Position{
					TraderAddress: traderAddr.String(),
					Pair:          vpoolPair,
					Size_:         sdk.OneDec(),
					Margin:        sdk.OneDec(),
				}
				nibiruApp.PerpKeeper.PositionsState(ctx).Set(dummyPosition)
				outPosition, err := nibiruApp.PerpKeeper.PositionsState(ctx).Get(vpoolPair, traderAddr)
				require.NoError(t, err)
				require.EqualValues(t, dummyPosition, outPosition)
			},
		},
	}

	for _, testCase := range testCases {
		tc := testCase
		t.Run(tc.name, func(t *testing.T) {
			tc.test()
		})
	}
}

func TestDeletePosition(t *testing.T) {
	testCases := []struct {
		name string
		test func()
	}{
		{
			name: "set - creating position with set works and shows up in get",
			test: func() {
				vpoolPair := common.MustNewAssetPair("osmo:nusd")

				traders := []sdk.AccAddress{
					sample.AccAddress(), sample.AccAddress(),
				}
				nibiruApp, ctx := testapp.NewNibiruAppAndContext(true)

				t.Log("vpool contains no positions to start")
				for _, trader := range traders {
					_, err := nibiruApp.PerpKeeper.PositionsState(ctx).Get(vpoolPair, trader)
					require.Error(t, err)
					require.ErrorContains(t, err, types.ErrPositionNotFound.Error())
				}

				var dummyPositions []*types.Position
				for _, traderAddr := range traders {
					dummyPosition := &types.Position{
						TraderAddress: traderAddr.String(),
						Pair:          vpoolPair,
						Size_:         sdk.OneDec(),
						Margin:        sdk.OneDec(),
					}
					nibiruApp.PerpKeeper.PositionsState(ctx).Set(dummyPosition)
					outPosition, err := nibiruApp.PerpKeeper.PositionsState(ctx).Get(vpoolPair, traderAddr)
					require.NoError(t, err)
					require.EqualValues(t, dummyPosition, outPosition)
					t.Logf("position created successfully on vpool, %v, for trader %v",
						vpoolPair, traderAddr.String())
					dummyPositions = append(dummyPositions, dummyPosition)
				}

				t.Log("attempt to clear all positions")

				require.NoError(t,
					nibiruApp.PerpKeeper.PositionsState(ctx).Delete(vpoolPair, traders[0]),
				)

				outPosition, err := nibiruApp.PerpKeeper.PositionsState(ctx).Get(vpoolPair, traders[0])
				require.ErrorIs(t, err, types.ErrPositionNotFound)
				require.Nil(t, outPosition)

				outPosition, err = nibiruApp.PerpKeeper.PositionsState(ctx).Get(vpoolPair, traders[1])
				require.NoError(t, err)
				require.EqualValues(t, dummyPositions[1], outPosition)
				t.Log("trader 1 has a position and trader 0 does not.")

				t.Log("clearing position of trader 1...")
				require.NoError(t,
					nibiruApp.PerpKeeper.PositionsState(ctx).Delete(vpoolPair, traders[1]),
				)

				outPosition, err = nibiruApp.PerpKeeper.PositionsState(ctx).Get(vpoolPair, traders[1])
				require.ErrorIs(t, err, types.ErrPositionNotFound)
				require.Nil(t, outPosition)
				t.Log("Success, all trader positions have been cleared.")
			},
		},
	}

	for _, testCase := range testCases {
		tc := testCase
		t.Run(tc.name, func(t *testing.T) {
			tc.test()
		})
	}
}

func TestKeeperClosePosition(t *testing.T) {
	// TODO(mercilex): simulate funding payments
	t.Run("success", func(t *testing.T) {
		t.Log("Setup Nibiru app, pair, and trader")
		nibiruApp, ctx := testapp.NewNibiruAppAndContext(true)
		pair := common.MustNewAssetPair("xxx:yyy")

		t.Log("Set vpool defined by pair on VpoolKeeper")
		vpoolKeeper := &nibiruApp.VpoolKeeper
		vpoolKeeper.CreatePool(
			ctx,
			pair,
			/*tradeLimitRatio*/ sdk.MustNewDecFromStr("0.9"),
			/*quoteAssetReserve*/ sdk.NewDec(10_000_000),
			/*baseAssetReserve*/ sdk.NewDec(5_000_000),
			/*fluctuationLimitRatio*/ sdk.MustNewDecFromStr("0.1"),
			/*maxOracleSpreadRatio*/ sdk.MustNewDecFromStr("0.1"),
			/* maintenanceMarginRatio */ sdk.MustNewDecFromStr("0.0625"),
		)
		require.True(t, vpoolKeeper.ExistsPool(ctx, pair))
		nibiruApp.PricefeedKeeper.ActivePairsStore().Set(ctx, pair, true)

		t.Log("Set vpool defined by pair on PerpKeeper")
		perpKeeper := &nibiruApp.PerpKeeper
		perpKeeper.PairMetadataState(ctx).Set(
			&types.PairMetadata{
				Pair: pair,
				CumulativePremiumFractions: []sdk.Dec{
					sdk.MustNewDecFromStr("0.2")},
			},
		)

		t.Log("open position for alice - long")

		alice := sample.AccAddress()
		err := simapp.FundAccount(nibiruApp.BankKeeper, ctx, alice,
			sdk.NewCoins(sdk.NewInt64Coin("yyy", 300)))
		require.NoError(t, err)

		aliceSide := types.Side_BUY
		aliceQuote := sdk.NewInt(60)
		aliceLeverage := sdk.NewDec(10)
		aliceBaseLimit := sdk.NewDec(150)
		_, err = nibiruApp.PerpKeeper.OpenPosition(
			ctx, pair, aliceSide, alice, aliceQuote, aliceLeverage, aliceBaseLimit)
		require.NoError(t, err)

		t.Log("open position for bob - long")
		// force funding payments
		perpKeeper.PairMetadataState(ctx).Set(&types.PairMetadata{
			Pair: pair,
			CumulativePremiumFractions: []sdk.Dec{
				sdk.MustNewDecFromStr("0.3")},
		})
		bob := sample.AccAddress()
		err = simapp.FundAccount(nibiruApp.BankKeeper, ctx, bob,
			sdk.NewCoins(sdk.NewInt64Coin("yyy", 62)))
		require.NoError(t, err)

		bobSide := types.Side_BUY
		bobQuote := sdk.NewInt(60)
		bobLeverage := sdk.NewDec(10)
		bobBaseLimit := sdk.NewDec(150)
		_, err = nibiruApp.PerpKeeper.OpenPosition(
			ctx, pair, bobSide, bob, bobQuote, bobLeverage, bobBaseLimit)
		require.NoError(t, err)

		t.Log("testing close position")
		ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 1).
			WithBlockTime(ctx.BlockTime().Add(1 * time.Minute))

		posResp, err := nibiruApp.PerpKeeper.ClosePosition(ctx, pair, alice)
		require.NoError(t, err)
		require.True(t, posResp.BadDebt.IsZero())
		require.True(t, !posResp.FundingPayment.IsZero() && posResp.FundingPayment.IsPositive())

		position, err := nibiruApp.PerpKeeper.PositionsState(ctx).Get(pair, alice)
		require.ErrorIs(t, err, types.ErrPositionNotFound)
		require.Nil(t, position)

		// this tests the following issue https://github.com/NibiruChain/nibiru/issues/645
		// in which opening a position from the same address on the same pair
		// was not possible after calling close position, due to bad data clearance.
		_, err = nibiruApp.PerpKeeper.OpenPosition(ctx, pair, aliceSide, alice, aliceQuote, aliceLeverage, aliceBaseLimit)
		require.NoError(t, err)
	})
}
