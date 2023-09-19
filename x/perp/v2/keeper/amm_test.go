package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/collections"

	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/common/denoms"
	. "github.com/NibiruChain/nibiru/x/common/testutil/action"
	. "github.com/NibiruChain/nibiru/x/common/testutil/assertion"
	"github.com/NibiruChain/nibiru/x/common/testutil/testapp"
	. "github.com/NibiruChain/nibiru/x/perp/v2/integration/action"
	. "github.com/NibiruChain/nibiru/x/perp/v2/integration/assertion"
	"github.com/NibiruChain/nibiru/x/perp/v2/keeper"
	types "github.com/NibiruChain/nibiru/x/perp/v2/types"
)

func TestEditPriceMultipler(t *testing.T) {
	pair := asset.Registry.Pair(denoms.BTC, denoms.NUSD)

	tests := TestCases{
		TC("same price multiplier").
			Given(
				CreateCustomMarket(pair, WithTotalLong(sdk.NewDec(1000)), WithTotalShort(sdk.NewDec(500))),
				FundModule(types.VaultModuleAccount, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(1e6)))),
				FundModule(types.PerpEFModuleAccount, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(1e6)))),
			).
			When(
				EditPriceMultiplier(pair, sdk.OneDec()),
			).
			Then(
				ModuleBalanceEqual(types.VaultModuleAccount, denoms.NUSD, sdk.NewInt(1e6)),
				ModuleBalanceEqual(types.PerpEFModuleAccount, denoms.NUSD, sdk.NewInt(1e6)),
				AMMShouldBeEqual(pair,
					AMM_SqrtDepthShouldBeEqual(sdk.NewDec(1e12)),
					AMM_BaseReserveShouldBeEqual(sdk.NewDec(1e12)),
					AMM_QuoteReserveShouldBeEqual(sdk.NewDec(1e12)),
					AMM_PriceMultiplierShouldBeEqual(sdk.OneDec()),
				),
			),

		TC("net bias zero").
			Given(
				CreateCustomMarket(pair, WithTotalLong(sdk.NewDec(1000)), WithTotalShort(sdk.NewDec(1000))),
				FundModule(types.VaultModuleAccount, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(1e6)))),
				FundModule(types.PerpEFModuleAccount, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(1e6)))),
			).
			When(
				EditPriceMultiplier(pair, sdk.NewDec(10)),
			).
			Then(
				ModuleBalanceEqual(types.VaultModuleAccount, denoms.NUSD, sdk.NewInt(1e6)),
				ModuleBalanceEqual(types.PerpEFModuleAccount, denoms.NUSD, sdk.NewInt(1e6)),
				AMMShouldBeEqual(pair,
					AMM_SqrtDepthShouldBeEqual(sdk.NewDec(1e12)),
					AMM_BaseReserveShouldBeEqual(sdk.NewDec(1e12)),
					AMM_QuoteReserveShouldBeEqual(sdk.NewDec(1e12)),
					AMM_PriceMultiplierShouldBeEqual(sdk.NewDec(10)),
				),
			),

		TC("long bias, increase price multiplier").
			Given(
				CreateCustomMarket(pair, WithTotalLong(sdk.NewDec(1000)), WithTotalShort(sdk.NewDec(500))),
				FundModule(types.VaultModuleAccount, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(1e6)))),
				FundModule(types.PerpEFModuleAccount, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(1e6)))),
			).
			When(
				EditPriceMultiplier(pair, sdk.NewDec(10)),
			).
			Then(
				ModuleBalanceEqual(types.VaultModuleAccount, denoms.NUSD, sdk.NewInt(1004500)),
				ModuleBalanceEqual(types.PerpEFModuleAccount, denoms.NUSD, sdk.NewInt(995500)),
				AMMShouldBeEqual(pair,
					AMM_SqrtDepthShouldBeEqual(sdk.NewDec(1e12)),
					AMM_BaseReserveShouldBeEqual(sdk.NewDec(1e12)),
					AMM_QuoteReserveShouldBeEqual(sdk.NewDec(1e12)),
					AMM_PriceMultiplierShouldBeEqual(sdk.NewDec(10)),
				),
			),

		TC("long bias, decrease price multiplier").
			Given(
				CreateCustomMarket(pair, WithTotalLong(sdk.NewDec(1000)), WithTotalShort(sdk.NewDec(500))),
				FundModule(types.VaultModuleAccount, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(1e6)))),
				FundModule(types.PerpEFModuleAccount, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(1e6)))),
			).
			When(
				EditPriceMultiplier(pair, sdk.MustNewDecFromStr("0.25")),
			).
			Then(
				ModuleBalanceEqual(types.VaultModuleAccount, denoms.NUSD, sdk.NewInt(999626)),
				ModuleBalanceEqual(types.PerpEFModuleAccount, denoms.NUSD, sdk.NewInt(1000374)),
				AMMShouldBeEqual(pair,
					AMM_SqrtDepthShouldBeEqual(sdk.NewDec(1e12)),
					AMM_BaseReserveShouldBeEqual(sdk.NewDec(1e12)),
					AMM_QuoteReserveShouldBeEqual(sdk.NewDec(1e12)),
					AMM_PriceMultiplierShouldBeEqual(sdk.MustNewDecFromStr("0.25")),
				),
			),

		TC("short bias, increase price multiplier").
			Given(
				CreateCustomMarket(pair, WithTotalLong(sdk.NewDec(500)), WithTotalShort(sdk.NewDec(1000))),
				FundModule(types.VaultModuleAccount, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(1e6)))),
				FundModule(types.PerpEFModuleAccount, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(1e6)))),
			).
			When(
				EditPriceMultiplier(pair, sdk.NewDec(10)),
			).
			Then(
				ModuleBalanceEqual(types.VaultModuleAccount, denoms.NUSD, sdk.NewInt(995500)),
				ModuleBalanceEqual(types.PerpEFModuleAccount, denoms.NUSD, sdk.NewInt(1004500)),
				AMMShouldBeEqual(pair,
					AMM_SqrtDepthShouldBeEqual(sdk.NewDec(1e12)),
					AMM_BaseReserveShouldBeEqual(sdk.NewDec(1e12)),
					AMM_QuoteReserveShouldBeEqual(sdk.NewDec(1e12)),
					AMM_PriceMultiplierShouldBeEqual(sdk.NewDec(10)),
				),
			),

		TC("short bias, decrease price multiplier").
			Given(
				CreateCustomMarket(pair, WithTotalLong(sdk.NewDec(500)), WithTotalShort(sdk.NewDec(1000))),
				FundModule(types.VaultModuleAccount, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(1e6)))),
				FundModule(types.PerpEFModuleAccount, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(1e6)))),
			).
			When(
				EditPriceMultiplier(pair, sdk.MustNewDecFromStr("0.25")),
			).
			Then(
				ModuleBalanceEqual(types.VaultModuleAccount, denoms.NUSD, sdk.NewInt(1000376)),
				ModuleBalanceEqual(types.PerpEFModuleAccount, denoms.NUSD, sdk.NewInt(999624)),
				AMMShouldBeEqual(pair,
					AMM_SqrtDepthShouldBeEqual(sdk.NewDec(1e12)),
					AMM_BaseReserveShouldBeEqual(sdk.NewDec(1e12)),
					AMM_QuoteReserveShouldBeEqual(sdk.NewDec(1e12)),
					AMM_PriceMultiplierShouldBeEqual(sdk.MustNewDecFromStr("0.25")),
				),
			),
	}

	NewTestSuite(t).WithTestCases(tests...).Run()
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

	// Error because of invalid pair
	err = app.PerpKeeperV2.Admin().EditPriceMultiplier(ctx, asset.MustNewPair("luna:usdt"), sdk.NewDec(-1))
	require.ErrorContains(t, err, "market luna:usdt not found")

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
	require.ErrorContains(t, err, "market luna:usdt not found")

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

	tests := TestCases{
		TC("same swap invariant").
			Given(
				CreateCustomMarket(pair,
					WithTotalLong(sdk.NewDec(1000)),
					WithTotalShort(sdk.NewDec(500)),
					WithSqrtDepth(sdk.NewDec(1e6)),
				),
				FundModule(types.VaultModuleAccount, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(1e6)))),
				FundModule(types.PerpEFModuleAccount, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(1e6)))),
			).
			When(
				EditSwapInvariant(pair, sdk.NewDec(1e12)),
			).
			Then(
				ModuleBalanceEqual(types.VaultModuleAccount, denoms.NUSD, sdk.NewInt(1e6)),
				ModuleBalanceEqual(types.PerpEFModuleAccount, denoms.NUSD, sdk.NewInt(1e6)),
				AMMShouldBeEqual(pair,
					AMM_SqrtDepthShouldBeEqual(sdk.NewDec(1e6)),
					AMM_BaseReserveShouldBeEqual(sdk.NewDec(1e6)),
					AMM_QuoteReserveShouldBeEqual(sdk.NewDec(1e6)),
					AMM_PriceMultiplierShouldBeEqual(sdk.OneDec()),
				),
			),

		TC("net bias zero").
			Given(
				CreateCustomMarket(pair,
					WithTotalLong(sdk.NewDec(1000)),
					WithTotalShort(sdk.NewDec(1000)),
					WithSqrtDepth(sdk.NewDec(1e6)),
				),
				FundModule(types.VaultModuleAccount, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(1e6)))),
				FundModule(types.PerpEFModuleAccount, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(1e6)))),
			).
			When(
				EditSwapInvariant(pair, sdk.NewDec(1e18)),
			).
			Then(
				ModuleBalanceEqual(types.VaultModuleAccount, denoms.NUSD, sdk.NewInt(1e6)),
				ModuleBalanceEqual(types.PerpEFModuleAccount, denoms.NUSD, sdk.NewInt(1e6)),
				AMMShouldBeEqual(pair,
					AMM_SqrtDepthShouldBeEqual(sdk.NewDec(1e9)),
					AMM_BaseReserveShouldBeEqual(sdk.NewDec(1e9)),
					AMM_QuoteReserveShouldBeEqual(sdk.NewDec(1e9)),
					AMM_PriceMultiplierShouldBeEqual(sdk.OneDec()),
				),
			),

		TC("long bias, increase swap invariant").
			Given(
				CreateCustomMarket(pair, WithTotalLong(sdk.NewDec(1e5)), WithSqrtDepth(sdk.NewDec(1e6))),
				FundModule(types.VaultModuleAccount, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(1e6)))),
				FundModule(types.PerpEFModuleAccount, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(1e6)))),
			).
			When(
				EditSwapInvariant(pair, sdk.NewDec(1e14)),
			).
			Then(
				ModuleBalanceEqual(types.VaultModuleAccount, denoms.NUSD, sdk.NewInt(1008101)),
				ModuleBalanceEqual(types.PerpEFModuleAccount, denoms.NUSD, sdk.NewInt(991899)),
				AMMShouldBeEqual(pair,
					AMM_SqrtDepthShouldBeEqual(sdk.NewDec(1e7)),
					AMM_BaseReserveShouldBeEqual(sdk.NewDec(1e7)),
					AMM_QuoteReserveShouldBeEqual(sdk.NewDec(1e7)),
					AMM_PriceMultiplierShouldBeEqual(sdk.OneDec()),
				),
			),

		TC("long bias, decrease swap invariant").
			Given(
				CreateCustomMarket(pair, WithTotalLong(sdk.NewDec(1e2)), WithSqrtDepth(sdk.NewDec(1e6))),
				FundModule(types.VaultModuleAccount, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(1e6)))),
				FundModule(types.PerpEFModuleAccount, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(1e6)))),
			).
			When(
				EditSwapInvariant(pair, sdk.NewDec(1e6)),
			).
			Then(
				ModuleBalanceEqual(types.VaultModuleAccount, denoms.NUSD, sdk.NewInt(999991)),
				ModuleBalanceEqual(types.PerpEFModuleAccount, denoms.NUSD, sdk.NewInt(1000009)),
				AMMShouldBeEqual(pair,
					AMM_SqrtDepthShouldBeEqual(sdk.NewDec(1e3)),
					AMM_BaseReserveShouldBeEqual(sdk.NewDec(1e3)),
					AMM_QuoteReserveShouldBeEqual(sdk.NewDec(1e3)),
					AMM_PriceMultiplierShouldBeEqual(sdk.OneDec()),
				),
			),

		TC("short bias, increase swap invariant").
			Given(
				CreateCustomMarket(pair, WithTotalShort(sdk.NewDec(1e5)), WithSqrtDepth(sdk.NewDec(1e6))),
				FundModule(types.VaultModuleAccount, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(1e6)))),
				FundModule(types.PerpEFModuleAccount, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(1e6)))),
			).
			When(
				EditSwapInvariant(pair, sdk.NewDec(1e14)),
			).
			Then(
				ModuleBalanceEqual(types.VaultModuleAccount, denoms.NUSD, sdk.NewInt(1010102)),
				ModuleBalanceEqual(types.PerpEFModuleAccount, denoms.NUSD, sdk.NewInt(989898)),
				AMMShouldBeEqual(pair,
					AMM_SqrtDepthShouldBeEqual(sdk.NewDec(1e7)),
					AMM_BaseReserveShouldBeEqual(sdk.NewDec(1e7)),
					AMM_QuoteReserveShouldBeEqual(sdk.NewDec(1e7)),
					AMM_PriceMultiplierShouldBeEqual(sdk.OneDec()),
				),
			),

		TC("short bias, decrease swap invariant").
			Given(
				CreateCustomMarket(pair, WithTotalShort(sdk.NewDec(1e2)), WithSqrtDepth(sdk.NewDec(1e6))),
				FundModule(types.VaultModuleAccount, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(1e6)))),
				FundModule(types.PerpEFModuleAccount, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(1e6)))),
			).
			When(
				EditSwapInvariant(pair, sdk.NewDec(1e6)),
			).
			Then(
				ModuleBalanceEqual(types.VaultModuleAccount, denoms.NUSD, sdk.NewInt(999989)),
				ModuleBalanceEqual(types.PerpEFModuleAccount, denoms.NUSD, sdk.NewInt(1000011)),
				AMMShouldBeEqual(pair,
					AMM_SqrtDepthShouldBeEqual(sdk.NewDec(1e3)),
					AMM_BaseReserveShouldBeEqual(sdk.NewDec(1e3)),
					AMM_QuoteReserveShouldBeEqual(sdk.NewDec(1e3)),
					AMM_PriceMultiplierShouldBeEqual(sdk.OneDec()),
				),
			),
	}

	NewTestSuite(t).WithTestCases(tests...).Run()
}
