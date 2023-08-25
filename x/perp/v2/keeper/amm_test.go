package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/collections"

	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/common/denoms"
	tutilaction "github.com/NibiruChain/nibiru/x/common/testutil/action"
	tutilassertion "github.com/NibiruChain/nibiru/x/common/testutil/assertion"
	"github.com/NibiruChain/nibiru/x/common/testutil/testapp"
	perpaction "github.com/NibiruChain/nibiru/x/perp/v2/integration/action"
	perpassertion "github.com/NibiruChain/nibiru/x/perp/v2/integration/assertion"
	"github.com/NibiruChain/nibiru/x/perp/v2/keeper"
	types "github.com/NibiruChain/nibiru/x/perp/v2/types"
)

func TestEditPriceMultipler(t *testing.T) {
	pair := asset.Registry.Pair(denoms.BTC, denoms.NUSD)

	tests := tutilaction.TestCases{
		tutilaction.TC("same price multiplier").
			Given(
				perpaction.CreateCustomMarket(pair, perpaction.WithTotalLong(sdk.NewDec(1000)), perpaction.WithTotalShort(sdk.NewDec(500))),
				tutilaction.FundModule(types.VaultModuleAccount, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(1e6)))),
				tutilaction.FundModule(types.PerpEFModuleAccount, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(1e6)))),
			).
			When(
				perpaction.EditPriceMultiplier(pair, sdk.OneDec()),
			).
			Then(
				tutilassertion.ModuleBalanceEqual(types.VaultModuleAccount, denoms.NUSD, sdk.NewInt(1e6)),
				tutilassertion.ModuleBalanceEqual(types.PerpEFModuleAccount, denoms.NUSD, sdk.NewInt(1e6)),
				perpassertion.AMMShouldBeEqual(pair,
					perpassertion.AMM_SqrtDepthShouldBeEqual(sdk.NewDec(1e12)),
					perpassertion.AMM_BaseReserveShouldBeEqual(sdk.NewDec(1e12)),
					perpassertion.AMM_QuoteReserveShouldBeEqual(sdk.NewDec(1e12)),
					perpassertion.AMM_PriceMultiplierShouldBeEqual(sdk.OneDec()),
				),
			),

		tutilaction.TC("net bias zero").
			Given(
				perpaction.CreateCustomMarket(pair, perpaction.WithTotalLong(sdk.NewDec(1000)), perpaction.WithTotalShort(sdk.NewDec(1000))),
				tutilaction.FundModule(types.VaultModuleAccount, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(1e6)))),
				tutilaction.FundModule(types.PerpEFModuleAccount, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(1e6)))),
			).
			When(
				perpaction.EditPriceMultiplier(pair, sdk.NewDec(10)),
			).
			Then(
				tutilassertion.ModuleBalanceEqual(types.VaultModuleAccount, denoms.NUSD, sdk.NewInt(1e6)),
				tutilassertion.ModuleBalanceEqual(types.PerpEFModuleAccount, denoms.NUSD, sdk.NewInt(1e6)),
				perpassertion.AMMShouldBeEqual(pair,
					perpassertion.AMM_SqrtDepthShouldBeEqual(sdk.NewDec(1e12)),
					perpassertion.AMM_BaseReserveShouldBeEqual(sdk.NewDec(1e12)),
					perpassertion.AMM_QuoteReserveShouldBeEqual(sdk.NewDec(1e12)),
					perpassertion.AMM_PriceMultiplierShouldBeEqual(sdk.NewDec(10)),
				),
			),

		tutilaction.TC("long bias, increase price multiplier").
			Given(
				perpaction.CreateCustomMarket(pair, perpaction.WithTotalLong(sdk.NewDec(1000)), perpaction.WithTotalShort(sdk.NewDec(500))),
				tutilaction.FundModule(types.VaultModuleAccount, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(1e6)))),
				tutilaction.FundModule(types.PerpEFModuleAccount, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(1e6)))),
			).
			When(
				perpaction.EditPriceMultiplier(pair, sdk.NewDec(10)),
			).
			Then(
				tutilassertion.ModuleBalanceEqual(types.VaultModuleAccount, denoms.NUSD, sdk.NewInt(1004500)),
				tutilassertion.ModuleBalanceEqual(types.PerpEFModuleAccount, denoms.NUSD, sdk.NewInt(995500)),
				perpassertion.AMMShouldBeEqual(pair,
					perpassertion.AMM_SqrtDepthShouldBeEqual(sdk.NewDec(1e12)),
					perpassertion.AMM_BaseReserveShouldBeEqual(sdk.NewDec(1e12)),
					perpassertion.AMM_QuoteReserveShouldBeEqual(sdk.NewDec(1e12)),
					perpassertion.AMM_PriceMultiplierShouldBeEqual(sdk.NewDec(10)),
				),
			),

		tutilaction.TC("long bias, decrease price multiplier").
			Given(
				perpaction.CreateCustomMarket(pair, perpaction.WithTotalLong(sdk.NewDec(1000)), perpaction.WithTotalShort(sdk.NewDec(500))),
				tutilaction.FundModule(types.VaultModuleAccount, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(1e6)))),
				tutilaction.FundModule(types.PerpEFModuleAccount, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(1e6)))),
			).
			When(
				perpaction.EditPriceMultiplier(pair, sdk.MustNewDecFromStr("0.25")),
			).
			Then(
				tutilassertion.ModuleBalanceEqual(types.VaultModuleAccount, denoms.NUSD, sdk.NewInt(999626)),
				tutilassertion.ModuleBalanceEqual(types.PerpEFModuleAccount, denoms.NUSD, sdk.NewInt(1000374)),
				perpassertion.AMMShouldBeEqual(pair,
					perpassertion.AMM_SqrtDepthShouldBeEqual(sdk.NewDec(1e12)),
					perpassertion.AMM_BaseReserveShouldBeEqual(sdk.NewDec(1e12)),
					perpassertion.AMM_QuoteReserveShouldBeEqual(sdk.NewDec(1e12)),
					perpassertion.AMM_PriceMultiplierShouldBeEqual(sdk.MustNewDecFromStr("0.25")),
				),
			),

		tutilaction.TC("short bias, increase price multiplier").
			Given(
				perpaction.CreateCustomMarket(pair, perpaction.WithTotalLong(sdk.NewDec(500)), perpaction.WithTotalShort(sdk.NewDec(1000))),
				tutilaction.FundModule(types.VaultModuleAccount, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(1e6)))),
				tutilaction.FundModule(types.PerpEFModuleAccount, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(1e6)))),
			).
			When(
				perpaction.EditPriceMultiplier(pair, sdk.NewDec(10)),
			).
			Then(
				tutilassertion.ModuleBalanceEqual(types.VaultModuleAccount, denoms.NUSD, sdk.NewInt(995500)),
				tutilassertion.ModuleBalanceEqual(types.PerpEFModuleAccount, denoms.NUSD, sdk.NewInt(1004500)),
				perpassertion.AMMShouldBeEqual(pair,
					perpassertion.AMM_SqrtDepthShouldBeEqual(sdk.NewDec(1e12)),
					perpassertion.AMM_BaseReserveShouldBeEqual(sdk.NewDec(1e12)),
					perpassertion.AMM_QuoteReserveShouldBeEqual(sdk.NewDec(1e12)),
					perpassertion.AMM_PriceMultiplierShouldBeEqual(sdk.NewDec(10)),
				),
			),

		tutilaction.TC("short bias, decrease price multiplier").
			Given(
				perpaction.CreateCustomMarket(pair, perpaction.WithTotalLong(sdk.NewDec(500)), perpaction.WithTotalShort(sdk.NewDec(1000))),
				tutilaction.FundModule(types.VaultModuleAccount, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(1e6)))),
				tutilaction.FundModule(types.PerpEFModuleAccount, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(1e6)))),
			).
			When(
				perpaction.EditPriceMultiplier(pair, sdk.MustNewDecFromStr("0.25")),
			).
			Then(
				tutilassertion.ModuleBalanceEqual(types.VaultModuleAccount, denoms.NUSD, sdk.NewInt(1000376)),
				tutilassertion.ModuleBalanceEqual(types.PerpEFModuleAccount, denoms.NUSD, sdk.NewInt(999624)),
				perpassertion.AMMShouldBeEqual(pair,
					perpassertion.AMM_SqrtDepthShouldBeEqual(sdk.NewDec(1e12)),
					perpassertion.AMM_BaseReserveShouldBeEqual(sdk.NewDec(1e12)),
					perpassertion.AMM_QuoteReserveShouldBeEqual(sdk.NewDec(1e12)),
					perpassertion.AMM_PriceMultiplierShouldBeEqual(sdk.MustNewDecFromStr("0.25")),
				),
			),
	}

	tutilaction.NewTestSuite(t).WithTestCases(tests...).Run()
}

func TestEditPriceMultiplerFail(t *testing.T) {
	pair := asset.Registry.Pair(denoms.BTC, denoms.NUSD)
	app, ctx := testapp.NewNibiruTestAppAndContext()
	account := sdk.MustAccAddressFromBech32("cosmos1zaavvzxez0elundtn32qnk9lkm8kmcszzsv80v")

	err := app.PerpKeeperV2.Admin().CreateMarket(
		ctx,
		keeper.ArgsCreateMarket{
			Pair:            pair,
			PriceMultiplier: sdk.NewDec(2),
			SqrtDepth:       sdk.NewDec(1_000_000),
		},
	)
	app.PerpKeeperV2.ReserveSnapshots.Insert(
		ctx,
		collections.Join(pair, ctx.BlockTime()),
		types.ReserveSnapshot{
			Amm: types.AMM{
				Pair:            pair,
				BaseReserve:     sdk.NewDec(1_000),
				QuoteReserve:    sdk.NewDec(1_000),
				SqrtDepth:       sdk.NewDec(1_000_000),
				PriceMultiplier: sdk.NewDec(2),
				TotalLong:       sdk.NewDec(100),
				TotalShort:      sdk.ZeroDec(),
			},
			TimestampMs: ctx.BlockTime().UnixMilli(),
		})
	require.NoError(t, err)

	// Error because of invalid price multiplier
	err = app.PerpKeeperV2.Admin().EditPriceMultiplier(ctx, asset.MustNewPair("luna:usdt"), sdk.NewDec(-1))
	require.ErrorContains(t, err, "collections: not found")

	// Error because of invalid price multiplier
	err = app.PerpKeeperV2.Admin().EditPriceMultiplier(ctx, pair, sdk.NewDec(-1))
	require.ErrorIs(t, err, types.ErrNonPositivePegMultiplier)

	// Add market activity
	err = app.BankKeeper.MintCoins(ctx, "inflation", sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(1020))))
	require.NoError(t, err)

	err = app.BankKeeper.SendCoinsFromModuleToAccount(ctx, "inflation", account, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(1020))))
	require.NoError(t, err)

	_, err = app.PerpKeeperV2.MarketOrder(
		ctx,
		pair,
		types.Direction_LONG,
		sdk.MustAccAddressFromBech32("cosmos1zaavvzxez0elundtn32qnk9lkm8kmcszzsv80v"),
		sdk.NewInt(1000),
		sdk.OneDec(),
		sdk.ZeroDec(),
	)
	require.NoError(t, err)

	// Error because no money in perp ef fund
	err = app.PerpKeeperV2.Admin().EditPriceMultiplier(ctx, pair, sdk.NewDec(3))
	require.ErrorContains(t, err, "not enough fund in perp ef to pay for repeg")

	// Works because it goes in the other way
	err = app.PerpKeeperV2.Admin().EditPriceMultiplier(ctx, pair, sdk.NewDec(1))
	require.NoError(t, err)
}

func TestEditSwapInvariantFail(t *testing.T) {
	pair := asset.Registry.Pair(denoms.BTC, denoms.NUSD)
	app, ctx := testapp.NewNibiruTestAppAndContext()
	account := sdk.MustAccAddressFromBech32("cosmos1zaavvzxez0elundtn32qnk9lkm8kmcszzsv80v")

	err := app.PerpKeeperV2.Admin().CreateMarket(
		ctx,
		keeper.ArgsCreateMarket{
			Pair:            pair,
			PriceMultiplier: sdk.NewDec(2),
			SqrtDepth:       sdk.NewDec(1_000),
		},
	)
	app.PerpKeeperV2.ReserveSnapshots.Insert(
		ctx,
		collections.Join(pair, ctx.BlockTime()),
		types.ReserveSnapshot{
			Amm: types.AMM{
				Pair:            pair,
				BaseReserve:     sdk.NewDec(1_000),
				QuoteReserve:    sdk.NewDec(1_000),
				SqrtDepth:       sdk.NewDec(1_000),
				PriceMultiplier: sdk.NewDec(2),
				TotalLong:       sdk.NewDec(100),
				TotalShort:      sdk.ZeroDec(),
			},
			TimestampMs: ctx.BlockTime().UnixMilli(),
		})
	require.NoError(t, err)

	// Error because of invalid price multiplier
	err = app.PerpKeeperV2.Admin().EditSwapInvariant(ctx, asset.MustNewPair("luna:usdt"), sdk.NewDec(-1))
	require.ErrorContains(t, err, "collections: not found")

	// Error because of invalid price multiplier
	err = app.PerpKeeperV2.Admin().EditSwapInvariant(ctx, pair, sdk.NewDec(-1))
	require.ErrorIs(t, err, types.ErrNegativeSwapInvariant)

	// Add market activity
	err = app.BankKeeper.MintCoins(ctx, "inflation", sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(102))))
	require.NoError(t, err)

	err = app.BankKeeper.SendCoinsFromModuleToAccount(ctx, "inflation", account, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(102))))
	require.NoError(t, err)

	_, err = app.PerpKeeperV2.MarketOrder(
		ctx,
		pair,
		types.Direction_LONG,
		sdk.MustAccAddressFromBech32("cosmos1zaavvzxez0elundtn32qnk9lkm8kmcszzsv80v"),
		sdk.NewInt(100),
		sdk.OneDec(),
		sdk.ZeroDec(),
	)
	require.NoError(t, err)

	// Error because no money in perp ef fund
	err = app.PerpKeeperV2.Admin().EditSwapInvariant(ctx, pair, sdk.NewDec(2_000_000))
	require.ErrorContains(t, err, "not enough fund in perp ef to pay for repeg")

	// Fail at validate
	err = app.PerpKeeperV2.Admin().EditSwapInvariant(ctx, pair, sdk.NewDec(0))
	require.ErrorContains(t, err, "swap multiplier must be > 0")

	// Works because it goes in the other way
	err = app.PerpKeeperV2.Admin().EditSwapInvariant(ctx, pair, sdk.NewDec(500_000))
	require.NoError(t, err)
}

func TestEditSwapInvariant(t *testing.T) {
	pair := asset.Registry.Pair(denoms.BTC, denoms.NUSD)

	tests := tutilaction.TestCases{
		tutilaction.TC("same swap invariant").
			Given(
				perpaction.CreateCustomMarket(pair,
					perpaction.WithTotalLong(sdk.NewDec(1000)),
					perpaction.WithTotalShort(sdk.NewDec(500)),
					perpaction.WithSqrtDepth(sdk.NewDec(1e6)),
				),
				tutilaction.FundModule(types.VaultModuleAccount, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(1e6)))),
				tutilaction.FundModule(types.PerpEFModuleAccount, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(1e6)))),
			).
			When(
				perpaction.EditSwapInvariant(pair, sdk.NewDec(1e12)),
			).
			Then(
				tutilassertion.ModuleBalanceEqual(types.VaultModuleAccount, denoms.NUSD, sdk.NewInt(1e6)),
				tutilassertion.ModuleBalanceEqual(types.PerpEFModuleAccount, denoms.NUSD, sdk.NewInt(1e6)),
				perpassertion.AMMShouldBeEqual(pair,
					perpassertion.AMM_SqrtDepthShouldBeEqual(sdk.NewDec(1e6)),
					perpassertion.AMM_BaseReserveShouldBeEqual(sdk.NewDec(1e6)),
					perpassertion.AMM_QuoteReserveShouldBeEqual(sdk.NewDec(1e6)),
					perpassertion.AMM_PriceMultiplierShouldBeEqual(sdk.OneDec()),
				),
			),

		tutilaction.TC("net bias zero").
			Given(
				perpaction.CreateCustomMarket(pair,
					perpaction.WithTotalLong(sdk.NewDec(1000)),
					perpaction.WithTotalShort(sdk.NewDec(1000)),
					perpaction.WithSqrtDepth(sdk.NewDec(1e6)),
				),
				tutilaction.FundModule(types.VaultModuleAccount, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(1e6)))),
				tutilaction.FundModule(types.PerpEFModuleAccount, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(1e6)))),
			).
			When(
				perpaction.EditSwapInvariant(pair, sdk.NewDec(1e18)),
			).
			Then(
				tutilassertion.ModuleBalanceEqual(types.VaultModuleAccount, denoms.NUSD, sdk.NewInt(1e6)),
				tutilassertion.ModuleBalanceEqual(types.PerpEFModuleAccount, denoms.NUSD, sdk.NewInt(1e6)),
				perpassertion.AMMShouldBeEqual(pair,
					perpassertion.AMM_SqrtDepthShouldBeEqual(sdk.NewDec(1e9)),
					perpassertion.AMM_BaseReserveShouldBeEqual(sdk.NewDec(1e9)),
					perpassertion.AMM_QuoteReserveShouldBeEqual(sdk.NewDec(1e9)),
					perpassertion.AMM_PriceMultiplierShouldBeEqual(sdk.OneDec()),
				),
			),

		tutilaction.TC("long bias, increase swap invariant").
			Given(
				perpaction.CreateCustomMarket(pair, perpaction.WithTotalLong(sdk.NewDec(1e5)), perpaction.WithSqrtDepth(sdk.NewDec(1e6))),
				tutilaction.FundModule(types.VaultModuleAccount, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(1e6)))),
				tutilaction.FundModule(types.PerpEFModuleAccount, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(1e6)))),
			).
			When(
				perpaction.EditSwapInvariant(pair, sdk.NewDec(1e14)),
			).
			Then(
				tutilassertion.ModuleBalanceEqual(types.VaultModuleAccount, denoms.NUSD, sdk.NewInt(1008101)),
				tutilassertion.ModuleBalanceEqual(types.PerpEFModuleAccount, denoms.NUSD, sdk.NewInt(991899)),
				perpassertion.AMMShouldBeEqual(pair,
					perpassertion.AMM_SqrtDepthShouldBeEqual(sdk.NewDec(1e7)),
					perpassertion.AMM_BaseReserveShouldBeEqual(sdk.NewDec(1e7)),
					perpassertion.AMM_QuoteReserveShouldBeEqual(sdk.NewDec(1e7)),
					perpassertion.AMM_PriceMultiplierShouldBeEqual(sdk.OneDec()),
				),
			),

		tutilaction.TC("long bias, decrease swap invariant").
			Given(
				perpaction.CreateCustomMarket(pair, perpaction.WithTotalLong(sdk.NewDec(1e2)), perpaction.WithSqrtDepth(sdk.NewDec(1e6))),
				tutilaction.FundModule(types.VaultModuleAccount, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(1e6)))),
				tutilaction.FundModule(types.PerpEFModuleAccount, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(1e6)))),
			).
			When(
				perpaction.EditSwapInvariant(pair, sdk.NewDec(1e6)),
			).
			Then(
				tutilassertion.ModuleBalanceEqual(types.VaultModuleAccount, denoms.NUSD, sdk.NewInt(999991)),
				tutilassertion.ModuleBalanceEqual(types.PerpEFModuleAccount, denoms.NUSD, sdk.NewInt(1000009)),
				perpassertion.AMMShouldBeEqual(pair,
					perpassertion.AMM_SqrtDepthShouldBeEqual(sdk.NewDec(1e3)),
					perpassertion.AMM_BaseReserveShouldBeEqual(sdk.NewDec(1e3)),
					perpassertion.AMM_QuoteReserveShouldBeEqual(sdk.NewDec(1e3)),
					perpassertion.AMM_PriceMultiplierShouldBeEqual(sdk.OneDec()),
				),
			),

		tutilaction.TC("short bias, increase swap invariant").
			Given(
				perpaction.CreateCustomMarket(pair, perpaction.WithTotalShort(sdk.NewDec(1e5)), perpaction.WithSqrtDepth(sdk.NewDec(1e6))),
				tutilaction.FundModule(types.VaultModuleAccount, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(1e6)))),
				tutilaction.FundModule(types.PerpEFModuleAccount, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(1e6)))),
			).
			When(
				perpaction.EditSwapInvariant(pair, sdk.NewDec(1e14)),
			).
			Then(
				tutilassertion.ModuleBalanceEqual(types.VaultModuleAccount, denoms.NUSD, sdk.NewInt(1010102)),
				tutilassertion.ModuleBalanceEqual(types.PerpEFModuleAccount, denoms.NUSD, sdk.NewInt(989898)),
				perpassertion.AMMShouldBeEqual(pair,
					perpassertion.AMM_SqrtDepthShouldBeEqual(sdk.NewDec(1e7)),
					perpassertion.AMM_BaseReserveShouldBeEqual(sdk.NewDec(1e7)),
					perpassertion.AMM_QuoteReserveShouldBeEqual(sdk.NewDec(1e7)),
					perpassertion.AMM_PriceMultiplierShouldBeEqual(sdk.OneDec()),
				),
			),

		tutilaction.TC("short bias, decrease swap invariant").
			Given(
				perpaction.CreateCustomMarket(pair, perpaction.WithTotalShort(sdk.NewDec(1e2)), perpaction.WithSqrtDepth(sdk.NewDec(1e6))),
				tutilaction.FundModule(types.VaultModuleAccount, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(1e6)))),
				tutilaction.FundModule(types.PerpEFModuleAccount, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(1e6)))),
			).
			When(
				perpaction.EditSwapInvariant(pair, sdk.NewDec(1e6)),
			).
			Then(
				tutilassertion.ModuleBalanceEqual(types.VaultModuleAccount, denoms.NUSD, sdk.NewInt(999989)),
				tutilassertion.ModuleBalanceEqual(types.PerpEFModuleAccount, denoms.NUSD, sdk.NewInt(1000011)),
				perpassertion.AMMShouldBeEqual(pair,
					perpassertion.AMM_SqrtDepthShouldBeEqual(sdk.NewDec(1e3)),
					perpassertion.AMM_BaseReserveShouldBeEqual(sdk.NewDec(1e3)),
					perpassertion.AMM_QuoteReserveShouldBeEqual(sdk.NewDec(1e3)),
					perpassertion.AMM_PriceMultiplierShouldBeEqual(sdk.OneDec()),
				),
			),
	}

	tutilaction.NewTestSuite(t).WithTestCases(tests...).Run()
}
