package keeper_test

import (
	"fmt"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/collections"

	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/common/denoms"
	"github.com/NibiruChain/nibiru/x/common/testutil"
	. "github.com/NibiruChain/nibiru/x/common/testutil/action"
	. "github.com/NibiruChain/nibiru/x/common/testutil/assertion"
	"github.com/NibiruChain/nibiru/x/common/testutil/testapp"
	. "github.com/NibiruChain/nibiru/x/perp/v2/integration/action"
	. "github.com/NibiruChain/nibiru/x/perp/v2/integration/assertion"
	"github.com/NibiruChain/nibiru/x/perp/v2/keeper"
	types "github.com/NibiruChain/nibiru/x/perp/v2/types"
)

func TestShiftPegMultiplier(t *testing.T) {
	pair := asset.Registry.Pair(denoms.BTC, denoms.NUSD)

	tests := TestCases{
		TC("same price multiplier").
			Given(
				CreateCustomMarket(pair, WithTotalLong(sdk.NewDec(1000)), WithTotalShort(sdk.NewDec(500))),
				FundModule(types.VaultModuleAccount, sdk.NewCoins(sdk.NewCoin(types.TestingCollateralDenomNUSD, sdk.NewInt(1e6)))),
				FundModule(types.PerpFundModuleAccount, sdk.NewCoins(sdk.NewCoin(types.TestingCollateralDenomNUSD, sdk.NewInt(1e6)))),
			).
			When(
				ShiftPegMultiplier(pair, sdk.OneDec()),
			).
			Then(
				ModuleBalanceEqual(types.VaultModuleAccount, types.TestingCollateralDenomNUSD, sdk.NewInt(1e6)),
				ModuleBalanceEqual(types.PerpFundModuleAccount, types.TestingCollateralDenomNUSD, sdk.NewInt(1e6)),
				AMMShouldBeEqual(pair,
					AMM_SqrtDepthShouldBeEqual(sdk.NewDec(1e12)),
					AMM_BaseReserveShouldBeEqual(sdk.NewDec(1e12)),
					AMM_QuoteReserveShouldBeEqual(sdk.NewDec(1e12)),
					AMM_PriceMultiplierShouldBeEqual(sdk.OneDec()),
				),
			),

		TC("net bias zero").
			Given(
				CreateCustomMarket(pair,
					WithTotalLong(sdk.NewDec(1000)), WithTotalShort(sdk.NewDec(1000)),
					WithEnabled(true),
				),
				FundModule(types.VaultModuleAccount, sdk.NewCoins(sdk.NewCoin(types.TestingCollateralDenomNUSD, sdk.NewInt(1e6)))),
				FundModule(types.PerpFundModuleAccount, sdk.NewCoins(sdk.NewCoin(types.TestingCollateralDenomNUSD, sdk.NewInt(1e6)))),
			).
			When(
				ShiftPegMultiplier(pair, sdk.NewDec(10)),
			).
			Then(
				ModuleBalanceEqual(types.VaultModuleAccount, types.TestingCollateralDenomNUSD, sdk.NewInt(1e6)),
				ModuleBalanceEqual(types.PerpFundModuleAccount, types.TestingCollateralDenomNUSD, sdk.NewInt(1e6)),
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
				FundModule(types.VaultModuleAccount, sdk.NewCoins(sdk.NewCoin(types.TestingCollateralDenomNUSD, sdk.NewInt(1e6)))),
				FundModule(types.PerpFundModuleAccount, sdk.NewCoins(sdk.NewCoin(types.TestingCollateralDenomNUSD, sdk.NewInt(1e6)))),
			).
			When(
				ShiftPegMultiplier(pair, sdk.NewDec(10)),
			).
			Then(
				ModuleBalanceEqual(types.VaultModuleAccount, types.TestingCollateralDenomNUSD, sdk.NewInt(1004500)),
				ModuleBalanceEqual(types.PerpFundModuleAccount, types.TestingCollateralDenomNUSD, sdk.NewInt(995500)),
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
				FundModule(types.VaultModuleAccount, sdk.NewCoins(sdk.NewCoin(types.TestingCollateralDenomNUSD, sdk.NewInt(1e6)))),
				FundModule(types.PerpFundModuleAccount, sdk.NewCoins(sdk.NewCoin(types.TestingCollateralDenomNUSD, sdk.NewInt(1e6)))),
			).
			When(
				ShiftPegMultiplier(pair, sdk.MustNewDecFromStr("0.25")),
			).
			Then(
				ModuleBalanceEqual(types.VaultModuleAccount, types.TestingCollateralDenomNUSD, sdk.NewInt(999626)),
				ModuleBalanceEqual(types.PerpFundModuleAccount, types.TestingCollateralDenomNUSD, sdk.NewInt(1000374)),
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
				FundModule(types.VaultModuleAccount, sdk.NewCoins(sdk.NewCoin(types.TestingCollateralDenomNUSD, sdk.NewInt(1e6)))),
				FundModule(types.PerpFundModuleAccount, sdk.NewCoins(sdk.NewCoin(types.TestingCollateralDenomNUSD, sdk.NewInt(1e6)))),
			).
			When(
				ShiftPegMultiplier(pair, sdk.NewDec(10)),
			).
			Then(
				ModuleBalanceEqual(types.VaultModuleAccount, types.TestingCollateralDenomNUSD, sdk.NewInt(995500)),
				ModuleBalanceEqual(types.PerpFundModuleAccount, types.TestingCollateralDenomNUSD, sdk.NewInt(1004500)),
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
				FundModule(types.VaultModuleAccount, sdk.NewCoins(sdk.NewCoin(types.TestingCollateralDenomNUSD, sdk.NewInt(1e6)))),
				FundModule(types.PerpFundModuleAccount, sdk.NewCoins(sdk.NewCoin(types.TestingCollateralDenomNUSD, sdk.NewInt(1e6)))),
			).
			When(
				ShiftPegMultiplier(pair, sdk.MustNewDecFromStr("0.25")),
			).
			Then(
				ModuleBalanceEqual(types.VaultModuleAccount, types.TestingCollateralDenomNUSD, sdk.NewInt(1000376)),
				ModuleBalanceEqual(types.PerpFundModuleAccount, types.TestingCollateralDenomNUSD, sdk.NewInt(999624)),
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

// TestShiftPegMultiplier_Fail: Test scenarios for the `ShiftPegMultiplier`
// function that should error.
func TestShiftPegMultiplier_Fail(t *testing.T) {
	pair := asset.Registry.Pair(denoms.BTC, denoms.NUSD)
	app, ctx := testapp.NewNibiruTestAppAndContext()

	account := testutil.AccAddress()

	adminUser, err := sdk.AccAddressFromBech32(testutil.ADDR_SUDO_ROOT)
	require.NoError(t, err)

	err = app.PerpKeeperV2.Sudo().CreateMarket(
		ctx,
		keeper.ArgsCreateMarket{
			Pair:            pair,
			PriceMultiplier: sdk.NewDec(2),
			SqrtDepth:       sdk.NewDec(1_000_000),
		},
		adminUser,
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

	adminAddr := testapp.DefaultSudoRoot()
	// Error because of invalid pair
	err = app.PerpKeeperV2.Sudo().ShiftPegMultiplier(
		ctx, asset.MustNewPair("luna:usdt"), sdk.NewDec(-1), adminAddr)
	require.ErrorContains(t, err, "market luna:usdt not found")

	// Error because of invalid price multiplier
	err = app.PerpKeeperV2.Sudo().ShiftPegMultiplier(ctx, pair, sdk.NewDec(-1), adminAddr)
	require.ErrorIs(t, err, types.ErrAmmNonPositivePegMult)

	// Add market activity
	err = app.BankKeeper.MintCoins(ctx, "inflation", sdk.NewCoins(sdk.NewCoin(types.TestingCollateralDenomNUSD, sdk.NewInt(1020))))
	require.NoError(t, err)

	err = app.BankKeeper.SendCoinsFromModuleToAccount(ctx, "inflation", account, sdk.NewCoins(sdk.NewCoin(types.TestingCollateralDenomNUSD, sdk.NewInt(1020))))
	require.NoError(t, err)

	_, err = app.PerpKeeperV2.MarketOrder(
		ctx,
		pair,
		types.Direction_LONG,
		account,
		sdk.NewInt(1000),
		sdk.OneDec(),
		sdk.ZeroDec(),
	)
	require.NoError(t, err)

	// Error because no money in perp ef fund
	err = app.PerpKeeperV2.Sudo().ShiftPegMultiplier(ctx, pair, sdk.NewDec(3), adminAddr)
	require.ErrorContains(t, err, types.ErrNotEnoughFundToPayAction.Error())

	// Works because it goes in the other way
	err = app.PerpKeeperV2.Sudo().ShiftPegMultiplier(ctx, pair, sdk.NewDec(1), adminAddr)
	require.NoError(t, err)
}

// TestShiftSwapInvariant_Fail: Test scenarios for the `ShiftSwapInvariant`
// function that should error
func TestShiftSwapInvariant_Fail(t *testing.T) {
	pair := asset.Registry.Pair(denoms.BTC, denoms.NUSD)
	app, ctx := testapp.NewNibiruTestAppAndContext()
	account := testutil.AccAddress()

	adminUser, err := sdk.AccAddressFromBech32(testutil.ADDR_SUDO_ROOT)
	require.NoError(t, err)

	err = app.PerpKeeperV2.Sudo().CreateMarket(
		ctx,
		keeper.ArgsCreateMarket{
			Pair:            pair,
			PriceMultiplier: sdk.NewDec(2),
			SqrtDepth:       sdk.NewDec(1_000),
		},
		adminUser,
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

	adminAddr := testapp.DefaultSudoRoot()
	// Error because of invalid price multiplier
	err = app.PerpKeeperV2.Sudo().ShiftSwapInvariant(ctx, asset.MustNewPair("luna:usdt"), sdk.NewInt(-1), adminAddr)
	require.ErrorContains(t, err, "market luna:usdt not found")

	// Error because of invalid price multiplier
	err = app.PerpKeeperV2.Sudo().ShiftSwapInvariant(ctx, pair, sdk.NewInt(-1), adminAddr)
	require.ErrorIs(t, err, types.ErrAmmNonPositiveSwapInvariant)

	// Add market activity
	err = app.BankKeeper.MintCoins(ctx, "inflation", sdk.NewCoins(sdk.NewCoin(types.TestingCollateralDenomNUSD, sdk.NewInt(102))))
	require.NoError(t, err)

	err = app.BankKeeper.SendCoinsFromModuleToAccount(ctx, "inflation", account, sdk.NewCoins(sdk.NewCoin(types.TestingCollateralDenomNUSD, sdk.NewInt(102))))
	require.NoError(t, err)

	_, err = app.PerpKeeperV2.MarketOrder(
		ctx,
		pair,
		types.Direction_LONG,
		account,
		sdk.NewInt(100),
		sdk.OneDec(),
		sdk.ZeroDec(),
	)
	require.NoError(t, err)

	// Error because no money in perp ef fund
	err = app.PerpKeeperV2.Sudo().ShiftSwapInvariant(ctx, pair, sdk.NewInt(2_000_000), adminAddr)
	require.ErrorContains(t, err, types.ErrNotEnoughFundToPayAction.Error())

	// Fail at validate
	err = app.PerpKeeperV2.Sudo().ShiftSwapInvariant(ctx, pair, sdk.NewInt(0), adminAddr)
	require.ErrorContains(t, err, types.ErrAmmNonPositiveSwapInvariant.Error())

	// Works because it goes in the other way
	err = app.PerpKeeperV2.Sudo().ShiftSwapInvariant(ctx, pair, sdk.NewInt(500_000), adminAddr)
	require.NoError(t, err)
}

func TestShiftSwapInvariant(t *testing.T) {
	pair := asset.Registry.Pair(denoms.BTC, denoms.NUSD)

	tests := TestCases{
		TC("same swap invariant").
			Given(
				CreateCustomMarket(pair,
					WithTotalLong(sdk.NewDec(1000)),
					WithTotalShort(sdk.NewDec(500)),
					WithSqrtDepth(sdk.NewDec(1e6)),
				),
				FundModule(types.VaultModuleAccount, sdk.NewCoins(sdk.NewCoin(types.TestingCollateralDenomNUSD, sdk.NewInt(1e6)))),
				FundModule(types.PerpFundModuleAccount, sdk.NewCoins(sdk.NewCoin(types.TestingCollateralDenomNUSD, sdk.NewInt(1e6)))),
			).
			When(
				ShiftSwapInvariant(pair, sdk.NewInt(1e12)),
			).
			Then(
				ModuleBalanceEqual(types.VaultModuleAccount, types.TestingCollateralDenomNUSD, sdk.NewInt(1e6)),
				ModuleBalanceEqual(types.PerpFundModuleAccount, types.TestingCollateralDenomNUSD, sdk.NewInt(1e6)),
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
				FundModule(types.VaultModuleAccount, sdk.NewCoins(sdk.NewCoin(types.TestingCollateralDenomNUSD, sdk.NewInt(1e6)))),
				FundModule(types.PerpFundModuleAccount, sdk.NewCoins(sdk.NewCoin(types.TestingCollateralDenomNUSD, sdk.NewInt(1e6)))),
			).
			When(
				ShiftSwapInvariant(pair, sdk.NewInt(1e18)),
			).
			Then(
				ModuleBalanceEqual(types.VaultModuleAccount, types.TestingCollateralDenomNUSD, sdk.NewInt(1e6)),
				ModuleBalanceEqual(types.PerpFundModuleAccount, types.TestingCollateralDenomNUSD, sdk.NewInt(1e6)),
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
				FundModule(types.VaultModuleAccount, sdk.NewCoins(sdk.NewCoin(types.TestingCollateralDenomNUSD, sdk.NewInt(1e6)))),
				FundModule(types.PerpFundModuleAccount, sdk.NewCoins(sdk.NewCoin(types.TestingCollateralDenomNUSD, sdk.NewInt(1e6)))),
			).
			When(
				ShiftSwapInvariant(pair, sdk.NewInt(1e14)),
			).
			Then(
				ModuleBalanceEqual(types.VaultModuleAccount, types.TestingCollateralDenomNUSD, sdk.NewInt(1008101)),
				ModuleBalanceEqual(types.PerpFundModuleAccount, types.TestingCollateralDenomNUSD, sdk.NewInt(991899)),
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
				FundModule(types.VaultModuleAccount, sdk.NewCoins(sdk.NewCoin(types.TestingCollateralDenomNUSD, sdk.NewInt(1e6)))),
				FundModule(types.PerpFundModuleAccount, sdk.NewCoins(sdk.NewCoin(types.TestingCollateralDenomNUSD, sdk.NewInt(1e6)))),
			).
			When(
				ShiftSwapInvariant(pair, sdk.NewInt(1e6)),
			).
			Then(
				ModuleBalanceEqual(types.VaultModuleAccount, types.TestingCollateralDenomNUSD, sdk.NewInt(999991)),
				ModuleBalanceEqual(types.PerpFundModuleAccount, types.TestingCollateralDenomNUSD, sdk.NewInt(1000009)),
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
				FundModule(types.VaultModuleAccount, sdk.NewCoins(sdk.NewCoin(types.TestingCollateralDenomNUSD, sdk.NewInt(1e6)))),
				FundModule(types.PerpFundModuleAccount, sdk.NewCoins(sdk.NewCoin(types.TestingCollateralDenomNUSD, sdk.NewInt(1e6)))),
			).
			When(
				ShiftSwapInvariant(pair, sdk.NewInt(1e14)),
			).
			Then(
				ModuleBalanceEqual(types.VaultModuleAccount, types.TestingCollateralDenomNUSD, sdk.NewInt(1010102)),
				ModuleBalanceEqual(types.PerpFundModuleAccount, types.TestingCollateralDenomNUSD, sdk.NewInt(989898)),
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
				FundModule(types.VaultModuleAccount, sdk.NewCoins(sdk.NewCoin(types.TestingCollateralDenomNUSD, sdk.NewInt(1e6)))),
				FundModule(types.PerpFundModuleAccount, sdk.NewCoins(sdk.NewCoin(types.TestingCollateralDenomNUSD, sdk.NewInt(1e6)))),
			).
			When(
				ShiftSwapInvariant(pair, sdk.NewInt(1e6)),
			).
			Then(
				ModuleBalanceEqual(types.VaultModuleAccount, types.TestingCollateralDenomNUSD, sdk.NewInt(999989)),
				ModuleBalanceEqual(types.PerpFundModuleAccount, types.TestingCollateralDenomNUSD, sdk.NewInt(1000011)),
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

func TestKeeper_GetMarketByPairAndVersion(t *testing.T) {
	app, ctx := testapp.NewNibiruTestAppAndContext()

	pair := asset.Registry.Pair(denoms.BTC, denoms.NUSD)

	adminUser, err := sdk.AccAddressFromBech32(testutil.ADDR_SUDO_ROOT)
	require.NoError(t, err)

	err = app.PerpKeeperV2.Sudo().CreateMarket(
		ctx,
		keeper.ArgsCreateMarket{
			Pair:            pair,
			PriceMultiplier: sdk.NewDec(2),
			SqrtDepth:       sdk.NewDec(1_000_000),
		},
		adminUser,
	)
	require.NoError(t, err)

	market, err := app.PerpKeeperV2.Sudo().GetMarketByPairAndVersion(ctx, pair, 1)
	require.NoError(t, err)
	require.Equal(t, market.Version, uint64(1))
	require.Equal(t, market.Pair, pair)

	market, err = app.PerpKeeperV2.Sudo().GetMarketByPairAndVersion(ctx, pair, 2)
	require.ErrorContains(t, err, fmt.Sprintf("market with pair %s and version 2 not found", pair.String()))
}

func TestKeeper_GetAMMByPairAndVersion(t *testing.T) {
	app, ctx := testapp.NewNibiruTestAppAndContext()

	pair := asset.Registry.Pair(denoms.BTC, denoms.NUSD)

	adminUser, err := sdk.AccAddressFromBech32(testutil.ADDR_SUDO_ROOT)
	require.NoError(t, err)

	err = app.PerpKeeperV2.Sudo().CreateMarket(
		ctx,
		keeper.ArgsCreateMarket{
			Pair:            pair,
			PriceMultiplier: sdk.NewDec(2),
			SqrtDepth:       sdk.NewDec(1_000_000),
		},
		adminUser,
	)
	require.NoError(t, err)

	amm, err := app.PerpKeeperV2.Sudo().GetAMMByPairAndVersion(ctx, pair, 1)
	require.NoError(t, err)
	require.Equal(t, amm.Version, uint64(1))
	require.Equal(t, amm.Pair, pair)

	amm, err = app.PerpKeeperV2.Sudo().GetAMMByPairAndVersion(ctx, pair, 2)
	require.ErrorContains(t, err, fmt.Sprintf("amm with pair %s and version 2 not found", pair.String()))
}
