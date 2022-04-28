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
	Name              string
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
			Name:              "Price higher than peg, wait for correct amount of time",
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
			Name:              "Price at peg, coll ratio should be the same",
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
			Name:              "Price higher than peg, but we don't wait for enough time, coll ratio should be the same",
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
			Name:              "Price higher than peg, and we wait for 2 updates, coll ratio should be updated twice",
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
			Name:              "Price higher than peg, and we wait for 2 updates but the last one is too close for update, coll ratio should be updated once",
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

	for _, tc := range tests {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
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

			_, err := app.PriceKeeper.SimSetPrice(ctx, common.StableDenom, common.CollDenom, tc.price, ctx.BlockTime().UTC().Add(time.Hour*1))
			require.NoError(t, err)

			err = app.PriceKeeper.SetCurrentPrices(ctx, common.StableDenom, common.CollDenom)
			require.NoError(t, err)

			err = app.StablecoinKeeper.SetCollRatio(ctx, tc.InCollRatio)
			require.NoError(t, err)

			tc.fn()

			currCollRatio := app.StablecoinKeeper.GetCollRatio(ctx)
			require.Equal(t, tc.ExpectedCollRatio, currCollRatio)
		})
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
	_, err := app.PriceKeeper.SimSetPrice(ctx, common.StableDenom, common.CollDenom, sdk.MustNewDecFromStr("0.9"), ctx.BlockTime().UTC().Add(time.Hour))
	require.NoError(t, err)

	err = app.PriceKeeper.SetCurrentPrices(ctx, common.StableDenom, common.CollDenom)
	require.NoError(t, err)

	err = app.StablecoinKeeper.SetCollRatio(ctx, sdk.MustNewDecFromStr("0.8"))
	require.NoError(t, err)

	// We wait for first epoch to start
	/*
		On the very first epoch in the chain, the epoch module waits until the second block having
		`epochInfo.StartTime.After(ctx.BlockTime())` to call the epoch end for the first time.
	*/
	runBlock(time.Minute*15 + time.Second)

	// Pass 1 second, this is the second block after the first 15min epoch is set, it will run the epochEnd hooks from
	// epoch module minute
	runBlock(time.Second)

	require.True(t, app.StablecoinKeeper.GetParams(ctx).IsCollateralRatioValid)

	// Pass 1 hour, collateral should be not valid because price are expired
	runBlock(time.Hour)        // Price are set as expired at the end of this block
	runBlock(time.Minute * 15) // Collateral ratio fail because no existing price since last block

	require.False(t, app.StablecoinKeeper.GetParams(ctx).IsCollateralRatioValid)

	// Post price, collateral should be valid again
	_, err = app.PriceKeeper.SimSetPrice(ctx, common.StableDenom, common.CollDenom, sdk.MustNewDecFromStr("0.9"), ctx.BlockTime().UTC().Add(time.Hour))
	require.NoError(t, err)

	runBlock(time.Second) // Median price and TWAP are computed again at the end of this block
	runBlock(time.Second) // The very next block have a collateral ratio valid without having to wait for the next epoch

	require.True(t, app.StablecoinKeeper.GetParams(ctx).IsCollateralRatioValid)
}
