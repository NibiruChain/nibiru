package stablecoin_test

import (
	"testing"
	"time"

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
			price:             sdk.MustNewDecFromStr("1.1"),
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
			price:             sdk.MustNewDecFromStr("1.1"),
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
			price:             sdk.MustNewDecFromStr("1.1"),
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
			price:             sdk.MustNewDecFromStr("1.1"),
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
		markets := ptypes.NewParams([]ptypes.Market{

			{
				MarketID:   common.CollStablePool,
				BaseAsset:  common.CollDenom,
				QuoteAsset: common.StableDenom,
				Oracles:    []sdk.AccAddress{oracle},
				Active:     true,
			},
		})

		app.PriceKeeper.SetParams(ctx, markets)

		app.PriceKeeper.SimSetPrice(ctx, common.CollStablePool, test.price)
		app.PriceKeeper.SetCurrentPrices(ctx, common.CollStablePool)

		app.StablecoinKeeper.SetCollRatio(ctx, test.InCollRatio)

		test.fn()

		currCollRatio := app.StablecoinKeeper.GetCollRatio(ctx)
		require.Equal(t, test.ExpectedCollRatio, currCollRatio)

	}
}
