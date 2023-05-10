package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/collections"

	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/common/denoms"
	"github.com/NibiruChain/nibiru/x/common/testutil"
	"github.com/NibiruChain/nibiru/x/common/testutil/mock"
	"github.com/NibiruChain/nibiru/x/common/testutil/testapp"

	v2types "github.com/NibiruChain/nibiru/x/perp/types/v2"
)

func TestClosePosition(t *testing.T) {
	app.SetPrefixes(app.AccountAddressPrefix)
	tests := []struct {
		name string

		initialPosition    v2types.Position
		newPriceMultiplier sdk.Dec
		newLatestCPF       sdk.Dec

		expectedFundingPayment         sdk.Dec
		expectedBadDebt                sdk.Dec
		expectedRealizedPnl            sdk.Dec
		expectedMarginToVault          sdk.Dec
		expectedExchangedNotionalValue sdk.Dec
	}{
		{
			name: "long position, positive PnL",
			// user bought in at 100 BTC for 10 NUSD at 10x leverage (1 BTC = 1 NUSD)
			// notional value is 100 NUSD
			// BTC doubles in value, now its price is 1 BTC = 2 NUSD
			// user has position notional value of 200 NUSD and unrealized PnL of +100 NUSD
			// user closes position
			// user ends up with realized PnL of +100 NUSD, unrealized PnL after of 0 NUSD,
			//   position notional value of 0 NUSD
			initialPosition: v2types.Position{
				TraderAddress:                   testutil.AccAddress().String(),
				Pair:                            asset.Registry.Pair(denoms.BTC, denoms.NUSD),
				Size_:                           sdk.NewDec(100), // 100 BTC
				Margin:                          sdk.NewDec(10),  // 10 NUSD
				OpenNotional:                    sdk.NewDec(100), // 100 NUSD
				LatestCumulativePremiumFraction: sdk.ZeroDec(),
				LastUpdatedBlockNumber:          0,
			},
			newPriceMultiplier: sdk.NewDec(2),
			newLatestCPF:       sdk.MustNewDecFromStr("0.02"),

			expectedExchangedNotionalValue: sdk.MustNewDecFromStr("199.999999980000000002"),
			expectedBadDebt:                sdk.ZeroDec(),
			expectedFundingPayment:         sdk.NewDec(2),
			expectedRealizedPnl:            sdk.MustNewDecFromStr("99.999999980000000002"),
			expectedMarginToVault:          sdk.MustNewDecFromStr("-107.999999980000000002"),
		},
		{
			name: "close long position, negative PnL",
			// user bought in at 100 BTC for 10 NUSD at 10x leverage (1 BTC = 1 NUSD)
			//   position and open notional value is 100 NUSD
			// BTC drops in value, now its price is 1 BTC = 0.95 NUSD
			// user has position notional value of 195 NUSD and unrealized PnL of -5 NUSD
			// user closes position
			// user ends up with realized PnL of -5 NUSD, unrealized PnL of 0 NUSD,
			//   position notional value of 0 NUSD
			initialPosition: v2types.Position{
				TraderAddress:                   testutil.AccAddress().String(),
				Pair:                            asset.Registry.Pair(denoms.BTC, denoms.NUSD),
				Size_:                           sdk.NewDec(100),
				Margin:                          sdk.NewDec(10),
				OpenNotional:                    sdk.NewDec(100),
				LatestCumulativePremiumFraction: sdk.ZeroDec(),
				LastUpdatedBlockNumber:          0,
			},
			newPriceMultiplier: sdk.MustNewDecFromStr("0.95"),
			newLatestCPF:       sdk.MustNewDecFromStr("0.02"),

			expectedBadDebt:                sdk.ZeroDec(),
			expectedFundingPayment:         sdk.NewDec(2),
			expectedRealizedPnl:            sdk.MustNewDecFromStr("-5.000000009499999999"),
			expectedMarginToVault:          sdk.MustNewDecFromStr("-2.999999990500000001"), // 10(old) + (-5)(realized PnL) - (2)(funding payment)
			expectedExchangedNotionalValue: sdk.MustNewDecFromStr("94.999999990500000001"),
		},

		/*==========================SHORT POSITIONS===========================*/
		{
			name: "close short position, positive PnL",
			// user bought in at 100 BTC for 10 NUSD at 10x leverage (1 BTC = 1 NUSD)
			// position and open notional value is 100 NUSD
			// BTC drops in value, now its price is 1 BTC = 0.95 NUSD
			// user has position notional value of 95 NUSD and unrealized PnL of 5 NUSD
			// user closes position
			// user ends up with realized PnL of 5 NUSD, unrealized PnL of 0 NUSD,
			//   position notional value of 0 NUSD
			initialPosition: v2types.Position{
				TraderAddress:                   testutil.AccAddress().String(),
				Pair:                            asset.Registry.Pair(denoms.BTC, denoms.NUSD),
				Size_:                           sdk.NewDec(-100),
				Margin:                          sdk.NewDec(10),
				OpenNotional:                    sdk.NewDec(100),
				LatestCumulativePremiumFraction: sdk.ZeroDec(),
				LastUpdatedBlockNumber:          0,
			},
			newPriceMultiplier: sdk.MustNewDecFromStr("0.95"),
			newLatestCPF:       sdk.MustNewDecFromStr("0.02"),

			expectedBadDebt:                sdk.ZeroDec(),
			expectedFundingPayment:         sdk.NewDec(-2),
			expectedRealizedPnl:            sdk.MustNewDecFromStr("4.999999990499999999"),
			expectedMarginToVault:          sdk.MustNewDecFromStr("-16.999999990499999999"), // old(10) + (5)(realizedPnL) - (-2)(fundingPayment)
			expectedExchangedNotionalValue: sdk.MustNewDecFromStr("95.000000009500000001"),
		},
		{
			name: "decrease short position, negative PnL",
			// user bought in at 100 BTC for 10 NUSD at 10x leverage (1 BTC = 1 NUSD)
			// position and open notional value is 100 NUSD
			// BTC increases in value, now its price is 1 BTC = 1.05 NUSD
			// user has position notional value of 105 NUSD and unrealized PnL of -5 NUSD
			// user closes their position
			// user ends up with realized PnL of -5 NUSD, unrealized PnL of 0 NUSD
			//   position notional value of 0 NUSD
			initialPosition: v2types.Position{
				TraderAddress:                   testutil.AccAddress().String(),
				Pair:                            asset.Registry.Pair(denoms.BTC, denoms.NUSD),
				Size_:                           sdk.NewDec(-100), // -100 BTC
				Margin:                          sdk.NewDec(10),   // 10 NUSD
				OpenNotional:                    sdk.NewDec(100),  // 100 NUSD
				LatestCumulativePremiumFraction: sdk.ZeroDec(),
				LastUpdatedBlockNumber:          0,
			},
			newPriceMultiplier: sdk.MustNewDecFromStr("1.05"),
			newLatestCPF:       sdk.MustNewDecFromStr("0.02"),

			expectedBadDebt:                sdk.ZeroDec(),
			expectedFundingPayment:         sdk.NewDec(-2),
			expectedRealizedPnl:            sdk.MustNewDecFromStr("-5.000000010500000001"),
			expectedMarginToVault:          sdk.MustNewDecFromStr("-6.999999989499999999"), // old(10) + (-5)(realizedPnL) - (-2)(fundingPayment)
			expectedExchangedNotionalValue: sdk.MustNewDecFromStr("105.000000010500000001"),
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			app, ctx := testapp.NewNibiruTestAppAndContext(true)
			traderAddr := sdk.MustAccAddressFromBech32(tc.initialPosition.TraderAddress)

			market := mock.TestMarket().WithLatestCumulativePremiumFraction(tc.newLatestCPF)
			amm := mock.TestAMMDefault().WithPriceMultiplier(tc.newPriceMultiplier)
			app.PerpKeeperV2.Markets.Insert(ctx, tc.initialPosition.Pair, *market)
			app.PerpKeeperV2.AMMs.Insert(ctx, tc.initialPosition.Pair, *amm)
			app.PerpKeeperV2.ReserveSnapshots.Insert(ctx, collections.Join(tc.initialPosition.Pair, ctx.BlockTime()), v2types.ReserveSnapshot{
				Amm:         *amm,
				TimestampMs: ctx.BlockTime().UnixMilli(),
			})
			app.PerpKeeperV2.Positions.Insert(ctx, collections.Join(tc.initialPosition.Pair, traderAddr), tc.initialPosition)
			require.NoError(t, testapp.FundModuleAccount(app.BankKeeper, ctx, v2types.VaultModuleAccount, sdk.NewCoins(sdk.NewInt64Coin(denoms.NUSD, 1e18))))
			require.NoError(t, testapp.FundModuleAccount(app.BankKeeper, ctx, v2types.PerpEFModuleAccount, sdk.NewCoins(sdk.NewInt64Coin(denoms.NUSD, 1e18))))

			resp, err := app.PerpKeeperV2.ClosePosition(
				ctx,
				tc.initialPosition.Pair,
				traderAddr,
			)

			require.NoError(t, err)
			assert.Equal(t, v2types.PositionResp{
				Position: &v2types.Position{
					TraderAddress:                   tc.initialPosition.TraderAddress,
					Pair:                            tc.initialPosition.Pair,
					Size_:                           sdk.ZeroDec(),
					Margin:                          sdk.ZeroDec(),
					OpenNotional:                    sdk.ZeroDec(),
					LatestCumulativePremiumFraction: tc.newLatestCPF,
					LastUpdatedBlockNumber:          ctx.BlockHeight(),
				},
				ExchangedNotionalValue: tc.expectedExchangedNotionalValue,
				ExchangedPositionSize:  tc.initialPosition.Size_.Neg(),
				BadDebt:                tc.expectedBadDebt,
				FundingPayment:         tc.expectedFundingPayment,
				RealizedPnl:            tc.expectedRealizedPnl,
				UnrealizedPnlAfter:     sdk.ZeroDec(),
				MarginToVault:          tc.expectedMarginToVault,
				PositionNotional:       sdk.ZeroDec(),
			}, *resp)

			testutil.RequireHasTypedEvent(t, ctx, &v2types.PositionChangedEvent{
				Pair:               tc.initialPosition.Pair,
				TraderAddress:      tc.initialPosition.TraderAddress,
				Margin:             sdk.NewInt64Coin(denoms.NUSD, 0),
				PositionNotional:   sdk.ZeroDec(),
				ExchangedNotional:  tc.expectedExchangedNotionalValue,
				ExchangedSize:      tc.initialPosition.Size_.Neg(),
				PositionSize:       sdk.ZeroDec(),
				RealizedPnl:        tc.expectedRealizedPnl,
				UnrealizedPnlAfter: sdk.ZeroDec(),
				BadDebt:            sdk.NewCoin(denoms.NUSD, sdk.ZeroInt()),
				FundingPayment:     tc.expectedFundingPayment,
				TransactionFee:     sdk.NewInt64Coin(denoms.NUSD, 0),
				BlockHeight:        ctx.BlockHeight(),
				BlockTimeMs:        ctx.BlockTime().UnixMilli(),
			})
		})
	}
}
