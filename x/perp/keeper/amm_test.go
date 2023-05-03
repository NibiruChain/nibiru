package keeper_test

import (
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/common/denoms"
	"github.com/NibiruChain/nibiru/x/common/testutil"
	"github.com/NibiruChain/nibiru/x/common/testutil/testapp"
	perpammtypes "github.com/NibiruChain/nibiru/x/perp/amm/types"
	"github.com/NibiruChain/nibiru/x/perp/keeper"
	"github.com/NibiruChain/nibiru/x/perp/types"
)

func TestMsgServerRepeg(t *testing.T) {
	tests := []struct {
		name string

		initialPegMultiplier sdk.Dec
		initialBiasInQuote   sdk.Int

		newPegMultiplier sdk.Dec

		initialPerpEFFunds sdk.Coins
		initialVaultFunds  sdk.Coins

		expectedErr error

		expectedUnusdPerpEFFunds sdk.Int
		expectedUnusdVaultFunds  sdk.Int
	}{
		{
			name: "happy path - we pay the vault with perp ef",

			initialPegMultiplier: sdk.OneDec(),
			initialBiasInQuote:   sdk.NewInt(25),

			newPegMultiplier: sdk.NewDec(2),

			initialPerpEFFunds: sdk.NewCoins(sdk.NewInt64Coin("unusd", 25)),

			expectedUnusdPerpEFFunds: sdk.ZeroInt(),
			expectedUnusdVaultFunds:  sdk.NewInt(50), // 25 margin + 25 repeg
		},
		{
			name: "not happy path - we pay the vault with perp ef but not enough money",

			initialPegMultiplier: sdk.OneDec(),
			initialBiasInQuote:   sdk.NewInt(25),

			newPegMultiplier: sdk.NewDec(2),

			initialPerpEFFunds: sdk.NewCoins(sdk.NewInt64Coin("unusd", 24)),

			expectedErr: types.ErrNotEnoughFundToPayAction,

			expectedUnusdPerpEFFunds: sdk.NewInt(24),
			expectedUnusdVaultFunds:  sdk.NewInt(25),
		},
		{
			name: "happy path - we pay the perp ef with vault",

			initialPegMultiplier: sdk.OneDec(),
			initialBiasInQuote:   sdk.NewInt(-25),

			newPegMultiplier: sdk.NewDec(2),

			initialVaultFunds: sdk.NewCoins(sdk.NewInt64Coin("unusd", 25)),

			expectedUnusdPerpEFFunds: sdk.NewInt(25),
			expectedUnusdVaultFunds:  sdk.NewInt(25),
		},
		{
			name: "happy path - we pay the perp ef with vault but not enough money",

			initialPegMultiplier: sdk.OneDec(),
			initialBiasInQuote:   sdk.NewInt(-25),

			newPegMultiplier: sdk.NewDec(50),

			initialVaultFunds: sdk.NewCoins(sdk.NewInt64Coin("unusd", 24)),

			expectedUnusdPerpEFFunds: sdk.ZeroInt(),
			expectedUnusdVaultFunds:  sdk.NewInt(49), // 24 + 25
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			app, ctx := testapp.NewNibiruTestAppAndContext(true)
			ctx = ctx.WithBlockTime(time.Now())

			pair := asset.Registry.Pair(denoms.BTC, denoms.NUSD)
			traderAccount := testutil.AccAddress()

			t.Log("create market")
			assert.NoError(t, app.PerpAmmKeeper.CreatePool(
				/* ctx */ ctx,
				/* pair */ pair,
				/* quoteReserve */ sdk.NewDec(100),
				/* baseReserve */ sdk.NewDec(100),
				perpammtypes.MarketConfig{
					TradeLimitRatio:        sdk.OneDec(),
					FluctuationLimitRatio:  sdk.OneDec(),
					MaxOracleSpreadRatio:   sdk.OneDec(),
					MaintenanceMarginRatio: sdk.MustNewDecFromStr("0.0625"),
					MaxLeverage:            sdk.MustNewDecFromStr("15"),
				},
				/* pegMultiplier */ tc.initialPegMultiplier,
			))
			keeper.SetPairMetadata(app.PerpKeeper, ctx, types.PairMetadata{
				Pair:                            pair,
				LatestCumulativePremiumFraction: sdk.ZeroDec(),
			})
			ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 1).WithBlockTime(time.Now().Add(time.Minute))

			t.Log("create positions")
			require.NoError(t, testapp.FundAccount(app.BankKeeper, ctx, traderAccount, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, tc.initialBiasInQuote.Abs()))))

			dir := perpammtypes.Direction_DIRECTION_UNSPECIFIED
			if tc.initialBiasInQuote.IsPositive() {
				dir = perpammtypes.Direction_LONG
			} else if tc.initialBiasInQuote.IsNegative() {
				dir = perpammtypes.Direction_SHORT
			}

			_, err := app.PerpKeeper.OpenPosition(
				/* ctx */ ctx,
				/* pair */ pair,
				/* side */ dir,
				/* traderAddr */ traderAccount,
				/* quoteAssetAmount */ tc.initialBiasInQuote.Abs(),
				/* leverage */ sdk.OneDec(),
				/* baseAmtLimit */ sdk.ZeroDec(),
			)
			require.NoError(t, err)

			if tc.initialPerpEFFunds != nil {
				require.NoError(t, testapp.FundModuleAccount(app.BankKeeper, ctx, types.PerpEFModuleAccount, tc.initialPerpEFFunds))
			}
			if tc.initialVaultFunds != nil {
				require.NoError(t, testapp.FundModuleAccount(app.BankKeeper, ctx, types.VaultModuleAccount, tc.initialVaultFunds))
			}

			err = app.PerpKeeper.EditPoolPegMultiplier(ctx, sdk.AccAddress{}, pair, tc.newPegMultiplier)
			require.Equal(t, tc.expectedErr, err)

			pool, _ := app.PerpAmmKeeper.GetPool(ctx, pair)
			if tc.expectedErr != nil {
				require.Equal(t, tc.initialPegMultiplier, pool.PegMultiplier)
			} else {
				require.Equal(t, tc.newPegMultiplier, pool.PegMultiplier)
			}

			assert.EqualValues(t,
				tc.expectedUnusdVaultFunds,
				app.BankKeeper.GetBalance(
					ctx,
					app.AccountKeeper.GetModuleAddress(types.VaultModuleAccount),
					denoms.NUSD,
				).Amount,
			)

			assert.EqualValues(t,
				tc.expectedUnusdPerpEFFunds,
				app.BankKeeper.GetBalance(
					ctx,
					app.AccountKeeper.GetModuleAddress(types.PerpEFModuleAccount),
					denoms.NUSD,
				).Amount,
			)
		})
	}
}

func TestMsgServerUpdateSwapInvariant(t *testing.T) {
	tests := []struct {
		name string

		initialBiasInQuote sdk.Int

		swapInvariantMultiplier sdk.Dec

		initialPerpEFFunds sdk.Coins
		initialVaultFunds  sdk.Coins

		expectedErr error

		expectedUnusdPerpEFFunds sdk.Int
		expectedUnusdVaultFunds  sdk.Int
	}{
		{
			name: "happy path - we pay the vault with perp ef",

			initialBiasInQuote: sdk.NewInt(25),

			swapInvariantMultiplier: sdk.NewDec(2), // Cost would be 2

			initialPerpEFFunds: sdk.NewCoins(sdk.NewInt64Coin("unusd", 25)),

			expectedUnusdPerpEFFunds: sdk.NewInt(23),
			expectedUnusdVaultFunds:  sdk.NewInt(27),
		},
		{
			name: "not happy path - we pay the vault with perp ef but not enough money",

			initialBiasInQuote: sdk.NewInt(25),

			swapInvariantMultiplier: sdk.NewDec(2), // Cost would be 2
			expectedErr:             types.ErrNotEnoughFundToPayAction,

			initialPerpEFFunds: sdk.NewCoins(sdk.NewInt64Coin("unusd", 1)),

			expectedUnusdPerpEFFunds: sdk.NewInt(1),
			expectedUnusdVaultFunds:  sdk.NewInt(25),
		},
		{
			name: "happy path - we pay the perp ef with vault",

			initialBiasInQuote: sdk.NewInt(25),

			swapInvariantMultiplier: sdk.MustNewDecFromStr("0.5"), // Cost would be 1

			initialPerpEFFunds: sdk.NewCoins(sdk.NewInt64Coin("unusd", 25)),

			expectedUnusdPerpEFFunds: sdk.NewInt(26),
			expectedUnusdVaultFunds:  sdk.NewInt(24),
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			app, ctx := testapp.NewNibiruTestAppAndContext(true)
			ctx = ctx.WithBlockTime(time.Now())

			pair := asset.Registry.Pair(denoms.BTC, denoms.NUSD)
			traderAccount := testutil.AccAddress()

			t.Log("create market")
			assert.NoError(t, app.PerpAmmKeeper.CreatePool(
				/* ctx */ ctx,
				/* pair */ pair,
				/* quoteReserve */ sdk.NewDec(100),
				/* baseReserve */ sdk.NewDec(100),
				perpammtypes.MarketConfig{
					TradeLimitRatio:        sdk.OneDec(),
					FluctuationLimitRatio:  sdk.OneDec(),
					MaxOracleSpreadRatio:   sdk.OneDec(),
					MaintenanceMarginRatio: sdk.MustNewDecFromStr("0.0625"),
					MaxLeverage:            sdk.MustNewDecFromStr("15"),
				},
				/* pegMultiplier */ sdk.OneDec(),
			))
			keeper.SetPairMetadata(app.PerpKeeper, ctx, types.PairMetadata{
				Pair:                            pair,
				LatestCumulativePremiumFraction: sdk.ZeroDec(),
			})
			ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 1).WithBlockTime(time.Now().Add(time.Minute))

			t.Log("create positions")
			require.NoError(t, testapp.FundAccount(app.BankKeeper, ctx, traderAccount, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, tc.initialBiasInQuote.Abs()))))

			dir := perpammtypes.Direction_DIRECTION_UNSPECIFIED
			if tc.initialBiasInQuote.IsPositive() {
				dir = perpammtypes.Direction_LONG
			} else if tc.initialBiasInQuote.IsNegative() {
				dir = perpammtypes.Direction_SHORT
			}

			_, err := app.PerpKeeper.OpenPosition(
				/* ctx */ ctx,
				/* pair */ pair,
				/* side */ dir,
				/* traderAddr */ traderAccount,
				/* quoteAssetAmount */ tc.initialBiasInQuote.Abs(),
				/* leverage */ sdk.OneDec(),
				/* baseAmtLimit */ sdk.ZeroDec(),
			)
			require.NoError(t, err)

			if tc.initialPerpEFFunds != nil {
				require.NoError(t, testapp.FundModuleAccount(app.BankKeeper, ctx, types.PerpEFModuleAccount, tc.initialPerpEFFunds))
			}
			if tc.initialVaultFunds != nil {
				require.NoError(t, testapp.FundModuleAccount(app.BankKeeper, ctx, types.VaultModuleAccount, tc.initialVaultFunds))
			}

			err = app.PerpKeeper.EditPoolSwapInvariant(ctx, sdk.AccAddress{}, pair, tc.swapInvariantMultiplier)
			require.Equal(t, tc.expectedErr, err)

			pool, _ := app.PerpAmmKeeper.GetPool(ctx, pair)
			previousSwapInvariant := sdk.NewDec(10_000)
			newSwapInvariant := pool.SqrtDepth.Mul(pool.SqrtDepth)

			if tc.expectedErr != nil {
				require.Equal(t, previousSwapInvariant, newSwapInvariant)
			} else {
				approxNewSwapInvariant := previousSwapInvariant.Mul(tc.swapInvariantMultiplier)

				require.True(
					t,
					approxNewSwapInvariant.Sub(newSwapInvariant).Abs().LT(sdk.MustNewDecFromStr("0.0001")),
				)
			}

			assert.EqualValues(t,
				tc.expectedUnusdVaultFunds,
				app.BankKeeper.GetBalance(
					ctx,
					app.AccountKeeper.GetModuleAddress(types.VaultModuleAccount),
					denoms.NUSD,
				).Amount,
			)

			assert.EqualValues(t,
				tc.expectedUnusdPerpEFFunds,
				app.BankKeeper.GetBalance(
					ctx,
					app.AccountKeeper.GetModuleAddress(types.PerpEFModuleAccount),
					denoms.NUSD,
				).Amount,
			)
		})
	}
}
