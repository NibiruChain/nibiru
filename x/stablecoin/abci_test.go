package stablecoin_test

import (
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/epochs"
	"github.com/NibiruChain/nibiru/x/pricefeed"
	ptypes "github.com/NibiruChain/nibiru/x/pricefeed/types"
	"github.com/NibiruChain/nibiru/x/testutil/sample"
	"github.com/NibiruChain/nibiru/x/testutil/testapp"
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
			Name:              "Collateral price higher than stable, wait for correct amount of time",
			InCollRatio:       sdk.MustNewDecFromStr("0.8"),
			price:             sdk.MustNewDecFromStr("0.9"),
			ExpectedCollRatio: sdk.MustNewDecFromStr("0.7975"),
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
			Name:              "Collateral price higher than stable, and we wait for 2 updates, coll ratio should be updated twice",
			InCollRatio:       sdk.MustNewDecFromStr("0.8"),
			price:             sdk.MustNewDecFromStr("0.9"),
			ExpectedCollRatio: sdk.MustNewDecFromStr("0.795"),
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
			Name:              "Collateral price higher than stable, and we wait for 2 updates but the last one is too close for update, coll ratio should be updated once",
			InCollRatio:       sdk.MustNewDecFromStr("0.8"),
			price:             sdk.MustNewDecFromStr("0.9"),
			ExpectedCollRatio: sdk.MustNewDecFromStr("0.7975"),
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
			app, ctx = testapp.NewNibiruAppAndContext(true)

			ctx = ctx.WithBlockHeight(1)

			oracle := sample.AccAddress()
			pairs := common.AssetPairs{
				common.PairCollStable,
			}
			markets := ptypes.NewParams(pairs)
			app.PricefeedKeeper.SetParams(ctx, markets)
			app.PricefeedKeeper.WhitelistOracles(ctx, []sdk.AccAddress{oracle})

			_, err := app.PricefeedKeeper.SetPrice(
				ctx,
				oracle,
				pairs[0].String(),
				/* price */ tc.price,
				/* expiry */ ctx.BlockTime().UTC().Add(time.Hour*1))
			require.NoError(t, err)

			err = app.PricefeedKeeper.SetCurrentPrices(ctx, pairs[0].Token0, pairs[0].Token1)
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
	app, ctx := testapp.NewNibiruAppAndContext(true)

	runBlock := func(duration time.Duration) {
		ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 1).WithBlockTime(ctx.BlockTime().Add(duration))
		pricefeed.BeginBlocker(ctx, app.PricefeedKeeper)
		epochs.BeginBlocker(ctx, app.EpochsKeeper)
	}

	// start at t=1sec with blockheight 1
	ctx = ctx.WithBlockHeight(1).WithBlockTime(time.Unix(1, 0))
	pricefeed.BeginBlocker(ctx, app.PricefeedKeeper)
	epochs.BeginBlocker(ctx, app.EpochsKeeper)

	oracle := sample.AccAddress()
	pairs := common.AssetPairs{
		{Token0: common.DenomColl, Token1: common.DenomStable},
	}
	markets := ptypes.NewParams(pairs)
	app.PricefeedKeeper.SetParams(ctx, markets)
	app.PricefeedKeeper.WhitelistOracles(ctx, []sdk.AccAddress{oracle})
	app.PricefeedKeeper.SetParams(ctx, markets)

	// Sim set price set the price for one hour
	_, err := app.PricefeedKeeper.SetPrice(
		ctx, oracle, pairs[0].String(), sdk.MustNewDecFromStr("0.9"), ctx.BlockTime().Add(time.Hour))
	require.NoError(t, err)
	require.NoError(t, app.PricefeedKeeper.SetCurrentPrices(ctx, pairs[0].Token0, pairs[0].Token1))
	require.NoError(t, app.StablecoinKeeper.SetCollRatio(ctx, sdk.MustNewDecFromStr("0.8")))

	// Mint block #2
	runBlock(time.Minute * 15)
	require.True(t, app.StablecoinKeeper.GetParams(ctx).IsCollateralRatioValid)

	// Mint block #3, collateral should be not valid because price are expired
	runBlock(time.Hour) // Collateral ratio is set to invalid at the beginning of this block
	require.False(t, app.StablecoinKeeper.GetParams(ctx).IsCollateralRatioValid)

	// Post price, collateral should be valid again
	_, err = app.PricefeedKeeper.SetPrice(
		ctx, oracle, pairs[0].String(), sdk.MustNewDecFromStr("0.9"), ctx.BlockTime().UTC().Add(time.Hour))
	require.NoError(t, err)

	// Mint block #4, median price and TWAP are computed again at the end of a new block
	runBlock(time.Second) // Collateral ratio is set to valid at the beginning of this block
	require.True(t, app.StablecoinKeeper.GetParams(ctx).IsCollateralRatioValid)
}
