package stablecoin_test

import (
	"testing"
	"time"

	"github.com/NibiruChain/nibiru/x/pricefeed"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/epochs"
	ptypes "github.com/NibiruChain/nibiru/x/pricefeed/types"
	"github.com/NibiruChain/nibiru/x/testutil"
	"github.com/NibiruChain/nibiru/x/testutil/sample"
	"github.com/stretchr/testify/require"
)

type test struct {
	InCollRatio       sdk.Dec
	ExpectedCollRatio sdk.Dec
	price             sdk.Dec
	fn                func()
}

func TestEpochInfoChangesBeginBlockerAndInitGenesis(t *testing.T) {
	var app *app.NibiruApp
	var ctx sdk.Context

	tests := []test{
		{
			// Price higher than peg, wait for correct amount of time
			InCollRatio:       sdk.MustNewDecFromStr("0.8"),
			price:             sdk.MustNewDecFromStr("0.9"),
			ExpectedCollRatio: sdk.MustNewDecFromStr("0.8025"),
			fn: func() {
				ctx = ctx.WithBlockHeight(2).WithBlockTime(ctx.BlockTime().Add(time.Second))
				epochs.BeginBlocker(ctx, app.EpochsKeeper)

				ctx = ctx.WithBlockHeight(3).WithBlockTime(ctx.BlockTime().Add(time.Second * 60 * 16))
				epochs.BeginBlocker(ctx, app.EpochsKeeper)
			},
		},
		{
			// Price at peg, coll ratio should be the same
			InCollRatio:       sdk.MustNewDecFromStr("0.8"),
			price:             sdk.MustNewDecFromStr("1"),
			ExpectedCollRatio: sdk.MustNewDecFromStr("0.8"),
			fn: func() {
				ctx = ctx.WithBlockHeight(2).WithBlockTime(ctx.BlockTime().Add(time.Second))
				epochs.BeginBlocker(ctx, app.EpochsKeeper)

				ctx = ctx.WithBlockHeight(3).WithBlockTime(ctx.BlockTime().Add(time.Second * 60 * 16))
				epochs.BeginBlocker(ctx, app.EpochsKeeper)
			},
		},
		{
			// Price higher than peg, but we don't wait for enough time, coll ratio should be the same
			InCollRatio:       sdk.MustNewDecFromStr("0.8"),
			price:             sdk.MustNewDecFromStr("0.9"),
			ExpectedCollRatio: sdk.MustNewDecFromStr("0.8"),
			fn: func() {
				ctx = ctx.WithBlockHeight(2).WithBlockTime(ctx.BlockTime().Add(time.Second))
				epochs.BeginBlocker(ctx, app.EpochsKeeper)

				ctx = ctx.WithBlockHeight(3).WithBlockTime(ctx.BlockTime().Add(time.Second * 2))
				epochs.BeginBlocker(ctx, app.EpochsKeeper)

				ctx = ctx.WithBlockHeight(3).WithBlockTime(ctx.BlockTime().Add(time.Second * 3))
				epochs.BeginBlocker(ctx, app.EpochsKeeper)
			},
		},
		{
			// Price higher than peg, and we wait for 2 updates, coll ratio should be updated twice
			InCollRatio:       sdk.MustNewDecFromStr("0.8"),
			price:             sdk.MustNewDecFromStr("0.9"),
			ExpectedCollRatio: sdk.MustNewDecFromStr("0.805"),
			fn: func() {
				ctx = ctx.WithBlockHeight(2).WithBlockTime(ctx.BlockTime().Add(time.Second))
				epochs.BeginBlocker(ctx, app.EpochsKeeper)

				ctx = ctx.WithBlockHeight(3).WithBlockTime(ctx.BlockTime().Add(time.Second + time.Minute*15))
				epochs.BeginBlocker(ctx, app.EpochsKeeper)

				ctx = ctx.WithBlockHeight(3).WithBlockTime(ctx.BlockTime().Add(time.Second + time.Minute*30))
				epochs.BeginBlocker(ctx, app.EpochsKeeper)
			},
		},
		{
			// Price higher than peg, and we wait for 2 updates but the last one is too close for update, coll ratio should be updated once
			InCollRatio:       sdk.MustNewDecFromStr("0.8"),
			price:             sdk.MustNewDecFromStr("0.9"),
			ExpectedCollRatio: sdk.MustNewDecFromStr("0.8025"),
			fn: func() {
				ctx = ctx.WithBlockHeight(2).WithBlockTime(ctx.BlockTime().Add(time.Second))
				epochs.BeginBlocker(ctx, app.EpochsKeeper)

				ctx = ctx.WithBlockHeight(3).WithBlockTime(ctx.BlockTime().Add(time.Second + time.Minute*14))
				epochs.BeginBlocker(ctx, app.EpochsKeeper)

				ctx = ctx.WithBlockHeight(3).WithBlockTime(ctx.BlockTime().Add(time.Second + time.Minute*16))
				epochs.BeginBlocker(ctx, app.EpochsKeeper)
			},
		},
	}

	for _, test := range tests {
		app, ctx = testutil.NewNibiruApp(true)

		ctx = ctx.WithBlockHeight(1)

		oracle := sample.AccAddress()
		markets := ptypes.NewParams([]ptypes.Pair{

			{
				Token0:  common.CollDenom,
				Token1:  common.StableDenom,
				Oracles: []sdk.AccAddress{oracle},
				Active:  true,
			},
		})

		app.PriceKeeper.SetParams(ctx, markets)

		_, err := app.PriceKeeper.SimSetPrice(ctx, common.StableDenom, common.CollDenom, test.price)
		require.NoError(t, err)

		err = app.PriceKeeper.SetCurrentPrices(ctx, common.StableDenom, common.CollDenom)
		require.NoError(t, err)

		err = app.StablecoinKeeper.SetCollRatio(ctx, test.InCollRatio)
		require.NoError(t, err)

		test.fn()

		currCollRatio := app.StablecoinKeeper.GetCollRatio(ctx)
		require.Equal(t, test.ExpectedCollRatio, currCollRatio)
	}
}

func TestEpochInfoChangesCollateralValidity(t *testing.T) {
	var app *app.NibiruApp
	var ctx sdk.Context

	runBlock := func(duration time.Duration) {
		ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 1).WithBlockTime(ctx.BlockTime().Add(duration))
		epochs.BeginBlocker(ctx, app.EpochsKeeper)
		pricefeed.EndBlocker(ctx, app.PriceKeeper)
	}

	app, ctx = testutil.NewNibiruApp(true)

	ctx = ctx.WithBlockHeight(1)

	oracle := sample.AccAddress()
	markets := ptypes.NewParams([]ptypes.Pair{

		{
			Token0:  common.CollDenom,
			Token1:  common.StableDenom,
			Oracles: []sdk.AccAddress{oracle},
			Active:  true,
		},
	})

	app.PriceKeeper.SetParams(ctx, markets)

	// Sim set price set the price for one hour
	_, err := app.PriceKeeper.SimSetPrice(ctx, common.StableDenom, common.CollDenom, sdk.MustNewDecFromStr("0.9"))
	require.NoError(t, err)

	err = app.PriceKeeper.SetCurrentPrices(ctx, common.StableDenom, common.CollDenom)
	require.NoError(t, err)

	err = app.StablecoinKeeper.SetCollRatio(ctx, sdk.MustNewDecFromStr("0.8"))
	require.NoError(t, err)

	// Pass 15 minute, collateral should be valid
	runBlock(time.Minute * 16)
	runBlock(time.Minute)

	require.True(t, app.StablecoinKeeper.GetParams(ctx).IsCollateralRatioValid)

	// Pass 1 hour, collateral should be not valid because price are expired
	runBlock(time.Hour + time.Minute*16)
	runBlock(time.Minute)

	require.False(t, app.StablecoinKeeper.GetParams(ctx).IsCollateralRatioValid)

	// Post price, collateral should be valid again
	_, err = app.PriceKeeper.SimSetPrice(ctx, common.StableDenom, common.CollDenom, sdk.MustNewDecFromStr("0.9"))
	require.NoError(t, err)

	runBlock(time.Minute * 16)
	runBlock(time.Minute)

	require.True(t, app.StablecoinKeeper.GetParams(ctx).IsCollateralRatioValid)
}
