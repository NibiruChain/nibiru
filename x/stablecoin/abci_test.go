package stablecoin_test

import (
	"testing"
	"time"

	"github.com/NibiruChain/nibiru/app"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/common/denoms"
	"github.com/NibiruChain/nibiru/x/common/testutil/testapp"
	"github.com/NibiruChain/nibiru/x/epochs"
	otypes "github.com/NibiruChain/nibiru/x/oracle/types"
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
	var price sdk.Dec

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
				app.OracleKeeper.SetPrice(ctx, asset.Registry.Pair(denoms.USDC, denoms.NUSD), price)
				epochs.BeginBlocker(ctx, app.EpochsKeeper)
			},
		},
		{
			Name:              "Price at peg, coll ratio should be the same",
			InCollRatio:       sdk.MustNewDecFromStr("0.8"),
			price:             sdk.OneDec(),
			ExpectedCollRatio: sdk.MustNewDecFromStr("0.8"),
			fn: func() {
				ctx = ctx.WithBlockHeight(2).WithBlockTime(ctx.BlockTime().Add(time.Second))
				epochs.BeginBlocker(ctx, app.EpochsKeeper)

				ctx = ctx.WithBlockHeight(3).WithBlockTime(ctx.BlockTime().Add(time.Second * 60 * 16))
				app.OracleKeeper.SetPrice(ctx, asset.Registry.Pair(denoms.USDC, denoms.NUSD), price)
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
			price:             sdk.MustNewDecFromStr("1.1"),
			ExpectedCollRatio: sdk.MustNewDecFromStr("0.805"),
			fn: func() {
				ctx = ctx.WithBlockHeight(2).WithBlockTime(ctx.BlockTime().Add(time.Second))
				epochs.BeginBlocker(ctx, app.EpochsKeeper)

				ctx = ctx.WithBlockHeight(3).WithBlockTime(ctx.BlockTime().Add(time.Second + time.Minute*15))
				app.OracleKeeper.SetPrice(ctx, asset.Registry.Pair(denoms.USDC, denoms.NUSD), price)
				epochs.BeginBlocker(ctx, app.EpochsKeeper)

				ctx = ctx.WithBlockHeight(3).WithBlockTime(ctx.BlockTime().Add(time.Second + time.Minute*30))
				app.OracleKeeper.SetPrice(ctx, asset.Registry.Pair(denoms.USDC, denoms.NUSD), price)
				epochs.BeginBlocker(ctx, app.EpochsKeeper)
			},
		},
		{
			Name:              "Collateral price higher than stable, and we wait for 2 updates but the last one is too close for update, coll ratio should be updated once",
			InCollRatio:       sdk.MustNewDecFromStr("0.8"),
			price:             sdk.MustNewDecFromStr("1.1"),
			ExpectedCollRatio: sdk.MustNewDecFromStr("0.8025"),
			fn: func() {
				ctx = ctx.WithBlockHeight(2).WithBlockTime(ctx.BlockTime().Add(time.Second))
				epochs.BeginBlocker(ctx, app.EpochsKeeper)

				ctx = ctx.WithBlockHeight(3).WithBlockTime(ctx.BlockTime().Add(time.Second + time.Minute*14))
				app.OracleKeeper.SetPrice(ctx, asset.Registry.Pair(denoms.USDC, denoms.NUSD), price)
				epochs.BeginBlocker(ctx, app.EpochsKeeper)

				ctx = ctx.WithBlockHeight(3).WithBlockTime(ctx.BlockTime().Add(time.Second + time.Minute*16))
				app.OracleKeeper.SetPrice(ctx, asset.Registry.Pair(denoms.USDC, denoms.NUSD), price)
				epochs.BeginBlocker(ctx, app.EpochsKeeper)
			},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			app, ctx = testapp.NewNibiruTestAppAndContext(true)

			ctx = ctx.WithBlockHeight(1)
			price = tc.price

			require.NoError(t, app.StablecoinKeeper.SetCollRatio(ctx, tc.InCollRatio))

			tc.fn()

			currCollRatio := app.StablecoinKeeper.GetCollRatio(ctx)
			require.Equal(t, tc.ExpectedCollRatio, currCollRatio)
		})
	}
}

func TestEpochInfoChangesCollateralValidity(t *testing.T) {
	app, ctx := testapp.NewNibiruTestAppAndContext(true)

	runBlock := func(duration time.Duration) {
		ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 1).WithBlockTime(ctx.BlockTime().Add(duration))
		epochs.BeginBlocker(ctx, app.EpochsKeeper)
	}

	// start at t=1sec with blockheight 1
	ctx = ctx.WithBlockHeight(1).WithBlockTime(time.Now())
	epochs.BeginBlocker(ctx, app.EpochsKeeper)

	pairs := []asset.Pair{
		asset.Registry.Pair(denoms.USDC, denoms.NUSD),
	}
	params := otypes.DefaultParams()
	params.TwapLookbackWindow = 1 * time.Hour
	app.OracleKeeper.Params.Set(ctx, params)
	app.OracleKeeper.SetPrice(ctx, pairs[0], sdk.MustNewDecFromStr("0.9"))

	require.NoError(t, app.StablecoinKeeper.SetCollRatio(ctx, sdk.MustNewDecFromStr("0.8")))

	// Mint block #2
	runBlock(time.Minute * 15)
	require.True(t, app.StablecoinKeeper.GetParams(ctx).IsCollateralRatioValid)

	// Mint block #3, collateral should be not valid because price are expired
	runBlock(time.Hour) // Collateral ratio is set to invalid at the beginning of this block
	require.False(t, app.StablecoinKeeper.GetParams(ctx).IsCollateralRatioValid)

	// Post price, collateral should be valid again
	app.OracleKeeper.SetPrice(ctx, pairs[0], sdk.MustNewDecFromStr("0.9"))

	// Mint block #4, median price and TWAP are computed again at the end of a new block
	runBlock(15 * time.Minute) // Collateral ratio is set to valid at the next epoch
	require.True(t, app.StablecoinKeeper.GetParams(ctx).IsCollateralRatioValid)
}
