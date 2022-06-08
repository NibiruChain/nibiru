package keeper_test

import (
	"github.com/cosmos/cosmos-sdk/simapp"
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/common"

	"github.com/NibiruChain/nibiru/x/perp/types"
	"github.com/NibiruChain/nibiru/x/testutil"
	"github.com/NibiruChain/nibiru/x/testutil/sample"

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
				nibiruApp, ctx := testutil.NewNibiruApp(true)
				pair, err := common.NewAssetPairFromStr("osmo:nusd")
				require.NoError(t, err)

				_, err = nibiruApp.PerpKeeper.GetPosition(
					ctx, pair, trader)
				require.Error(t, err)
				require.ErrorContains(t, err, types.ErrPositionNotFound.Error())
			},
		},
		{
			name: "set - creating position with set works and shows up in get",
			test: func() {
				vpoolPair, err := common.NewAssetPairFromStr("osmo:nusd")
				require.NoError(t, err)

				traderAddr := sample.AccAddress()
				nibiruApp, ctx := testutil.NewNibiruApp(true)

				_, err = nibiruApp.PerpKeeper.GetPosition(
					ctx, vpoolPair, traderAddr)
				require.Error(t, err)
				require.ErrorContains(t, err, types.ErrPositionNotFound.Error())

				dummyPosition := &types.Position{
					TraderAddress: traderAddr,
					Pair:          vpoolPair.String(),
					Size_:         sdk.OneDec(),
					Margin:        sdk.OneDec(),
				}
				nibiruApp.PerpKeeper.SetPosition(
					ctx, vpoolPair, traderAddr, dummyPosition)
				outPosition, err := nibiruApp.PerpKeeper.GetPosition(
					ctx, vpoolPair, traderAddr)
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

func TestClearPosition(t *testing.T) {
	testCases := []struct {
		name string
		test func()
	}{
		{
			name: "set - creating position with set works and shows up in get",
			test: func() {
				vpoolPair, err := common.NewAssetPairFromStr("osmo:nusd")
				require.NoError(t, err)

				traders := []sdk.AccAddress{
					sample.AccAddress(), sample.AccAddress(),
				}
				nibiruApp, ctx := testutil.NewNibiruApp(true)

				t.Log("vpool contains no positions to start")
				for _, trader := range traders {
					_, err := nibiruApp.PerpKeeper.GetPosition(
						ctx, vpoolPair, trader)
					require.Error(t, err)
					require.ErrorContains(t, err, types.ErrPositionNotFound.Error())
				}

				var dummyPositions []*types.Position
				for _, traderAddr := range traders {
					dummyPosition := &types.Position{
						TraderAddress: traderAddr,
						Pair:          vpoolPair.String(),
						Size_:         sdk.OneDec(),
						Margin:        sdk.OneDec(),
					}
					nibiruApp.PerpKeeper.SetPosition(
						ctx, vpoolPair, traderAddr, dummyPosition)
					outPosition, err := nibiruApp.PerpKeeper.GetPosition(
						ctx, vpoolPair, traderAddr)
					require.NoError(t, err)
					require.EqualValues(t, dummyPosition, outPosition)
					t.Logf("position created successfully on vpool, %v, for trader %v",
						vpoolPair, traderAddr.String())
					dummyPositions = append(dummyPositions, dummyPosition)
				}

				t.Log("attempt to clear all positions")

				require.NoError(t,
					nibiruApp.PerpKeeper.ClearPosition(
						ctx, vpoolPair, traders[0]),
				)

				outPosition, err := nibiruApp.PerpKeeper.GetPosition(
					ctx, vpoolPair, traders[0])
				require.NoError(t, err)
				require.EqualValues(t,
					types.ZeroPosition(ctx, vpoolPair, traders[0]),
					outPosition,
				)

				outPosition, err = nibiruApp.PerpKeeper.GetPosition(
					ctx, vpoolPair, traders[1])
				require.NoError(t, err)
				require.EqualValues(t, dummyPositions[1], outPosition)
				t.Log("trader 1 has a position and trader 0 does not.")

				t.Log("clearing position of trader 1...")
				require.NoError(t,
					nibiruApp.PerpKeeper.ClearPosition(
						ctx, vpoolPair, traders[1]),
				)
				outPosition, err = nibiruApp.PerpKeeper.GetPosition(
					ctx, vpoolPair, traders[1])
				require.NoError(t, err)
				require.EqualValues(t,
					types.ZeroPosition(ctx, vpoolPair, traders[1]),
					outPosition,
				)
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

func TestKeeper_ClosePosition(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		t.Log("Setup Nibiru app, pair, and trader")
		nibiruApp, ctx := testutil.NewNibiruApp(true)
		pair, err := common.NewAssetPairFromStr("xxx:yyy")
		require.NoError(t, err)

		t.Log("Set vpool defined by pair on VpoolKeeper")
		vpoolKeeper := &nibiruApp.VpoolKeeper
		vpoolKeeper.CreatePool(
			ctx,
			pair,
			sdk.MustNewDecFromStr("0.9"), // 0.9 ratio
			sdk.NewDec(10_000_000),       //
			sdk.NewDec(5_000_000),        // 5 tokens
			sdk.MustNewDecFromStr("0.1"), // 0.9 ratio
			sdk.MustNewDecFromStr("0.1"),
		)
		require.True(t, vpoolKeeper.ExistsPool(ctx, pair))

		t.Log("Set vpool defined by pair on PerpKeeper")
		perpKeeper := &nibiruApp.PerpKeeper
		perpKeeper.PairMetadata().Set(ctx, &types.PairMetadata{
			Pair: pair.String(),
			CumulativePremiumFractions: []sdk.Dec{
				sdk.MustNewDecFromStr("0.2")},
		})

		t.Log("open position for alice - long")

		alice := sample.AccAddress()
		err = simapp.FundAccount(nibiruApp.BankKeeper, ctx, alice,
			sdk.NewCoins(sdk.NewInt64Coin("yyy", 60)))
		require.NoError(t, err)

		aliceSide := types.Side_BUY
		aliceQuote := sdk.NewInt(60)
		aliceLeverage := sdk.NewDec(10)
		aliceBaseLimit := sdk.NewDec(150)
		err = nibiruApp.PerpKeeper.OpenPosition(
			ctx, pair, aliceSide, alice, aliceQuote, aliceLeverage, aliceBaseLimit)

		require.NoError(t, err)

		t.Log("open position for bob - long")
		// force funding payments
		perpKeeper.PairMetadata().Set(ctx, &types.PairMetadata{
			Pair: pair.String(),
			CumulativePremiumFractions: []sdk.Dec{
				sdk.MustNewDecFromStr("0.3")},
		})
		bob := sample.AccAddress()
		err = simapp.FundAccount(nibiruApp.BankKeeper, ctx, bob,
			sdk.NewCoins(sdk.NewInt64Coin("yyy", 60)))
		require.NoError(t, err)

		bobSide := types.Side_BUY
		bobQuote := sdk.NewInt(60)
		bobLeverage := sdk.NewDec(10)
		bobBaseLimit := sdk.NewDec(150)
		err = nibiruApp.PerpKeeper.OpenPosition(
			ctx, pair, bobSide, bob, bobQuote, bobLeverage, bobBaseLimit)

		require.NoError(t, err)

		t.Log("testing close position")
		ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 1).
			WithBlockTime(ctx.BlockTime().Add(1 * time.Minute))

		err = nibiruApp.PerpKeeper.ClosePosition(ctx, pair, alice)
		require.NoError(t, err)

		position, err := nibiruApp.PerpKeeper.Positions().Get(ctx, pair, alice)
		require.NoError(t, err)

		require.True(t, position.Size_.IsZero())
		require.True(t, position.Margin.IsZero())
		require.True(t, position.OpenNotional.IsZero())
	})
}
