package keeper_test

import (
	"testing"
	"time"

	"github.com/NibiruChain/collections"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/common/testutil"
	"github.com/NibiruChain/nibiru/x/common/testutil/testapp"
	perpammtypes "github.com/NibiruChain/nibiru/x/perp/amm/types"
	"github.com/NibiruChain/nibiru/x/perp/keeper"
	"github.com/NibiruChain/nibiru/x/perp/types"
)

func TestKeeperClosePosition(t *testing.T) {
	// TODO(mercilex): simulate funding payments
	t.Run("success", func(t *testing.T) {
		t.Log("Setup Nibiru app, pair, and trader")
		nibiruApp, ctx := testapp.NewNibiruTestAppAndContext(true)
		ctx = ctx.WithBlockTime(time.Now())
		pair := asset.MustNewPair("xxx:yyy")

		t.Log("Set market defined by pair on PerpAmmKeeper")
		perpammKeeper := &nibiruApp.PerpAmmKeeper
		require.NoError(t, perpammKeeper.CreatePool(
			ctx,
			pair,
			/*quoteReserve*/ sdk.NewDec(10*common.TO_MICRO),
			/*baseReserve*/ sdk.NewDec(10*common.TO_MICRO),
			perpammtypes.MarketConfig{
				TradeLimitRatio:        sdk.MustNewDecFromStr("0.9"),
				FluctuationLimitRatio:  sdk.MustNewDecFromStr("0.1"),
				MaxOracleSpreadRatio:   sdk.MustNewDecFromStr("0.1"),
				MaintenanceMarginRatio: sdk.MustNewDecFromStr("0.0625"),
				MaxLeverage:            sdk.MustNewDecFromStr("15"),
			},
			sdk.ZeroDec(),
			sdk.OneDec(),
		))
		require.True(t, perpammKeeper.ExistsPool(ctx, pair))

		t.Log("Set market defined by pair on PerpKeeper")
		keeper.SetPairMetadata(nibiruApp.PerpKeeper, ctx, types.PairMetadata{
			Pair:                            pair,
			LatestCumulativePremiumFraction: sdk.MustNewDecFromStr("0.2"),
		},
		)

		t.Log("open position for alice - long")
		ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 1).WithBlockTime(ctx.BlockTime().Add(time.Minute))

		alice := testutil.AccAddress()
		err := testapp.FundAccount(nibiruApp.BankKeeper, ctx, alice,
			sdk.NewCoins(sdk.NewInt64Coin("yyy", 300)))
		require.NoError(t, err)

		aliceSide := perpammtypes.Direction_LONG
		aliceQuote := sdk.NewInt(60)
		aliceLeverage := sdk.NewDec(5)
		aliceBaseLimit := sdk.NewDec(150)

		nibiruApp.OracleKeeper.SetPrice(ctx, pair, sdk.NewDec(20))

		_, err = nibiruApp.PerpKeeper.OpenPosition(
			ctx, pair, aliceSide, alice, aliceQuote, aliceLeverage, aliceBaseLimit)
		require.NoError(t, err)

		t.Log("open position for bob - long")
		// force funding payments
		keeper.SetPairMetadata(nibiruApp.PerpKeeper, ctx, types.PairMetadata{
			Pair:                            pair,
			LatestCumulativePremiumFraction: sdk.MustNewDecFromStr("0.3"),
		})
		bob := testutil.AccAddress()
		err = testapp.FundAccount(nibiruApp.BankKeeper, ctx, bob,
			sdk.NewCoins(sdk.NewInt64Coin("yyy", 62)))
		require.NoError(t, err)

		bobSide := perpammtypes.Direction_LONG
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

		position, err := nibiruApp.PerpKeeper.Positions.Get(ctx, collections.Join(pair, alice))
		require.ErrorIs(t, err, collections.ErrNotFound)
		require.Empty(t, position)

		// this tests the following issue https://github.com/NibiruChain/nibiru/issues/645
		// in which opening a position from the same address on the same pair
		// was not possible after calling close position, due to bad data clearance.
		_, err = nibiruApp.PerpKeeper.OpenPosition(ctx, pair, aliceSide, alice, aliceQuote, aliceLeverage, aliceBaseLimit)
		require.NoError(t, err)
	})
}
