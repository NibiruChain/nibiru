package keeper_test

import (
	"testing"
	"time"

	"github.com/NibiruChain/collections"

	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/common/denoms"
	"github.com/NibiruChain/nibiru/x/testutil"
	testutilevents "github.com/NibiruChain/nibiru/x/testutil"

	"github.com/cosmos/cosmos-sdk/simapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	nibisimapp "github.com/NibiruChain/nibiru/simapp"
	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/perp/types"
	vpooltypes "github.com/NibiruChain/nibiru/x/vpool/types"
)

func TestOpenPositionSuccess(t *testing.T) {
	testCases := []struct {
		name        string
		traderFunds sdk.Coins

		initialPosition *types.Position

		side      types.Side
		margin    sdk.Int
		leverage  sdk.Dec
		baseLimit sdk.Dec

		expectedMargin           sdk.Dec
		expectedOpenNotional     sdk.Dec
		expectedSize             sdk.Dec
		expectedPositionNotional sdk.Dec
		expectedUnrealizedPnl    sdk.Dec
		expectedRealizedPnl      sdk.Dec
		expectedMarginToVault    sdk.Dec
		expectedMarkPrice        sdk.Dec
	}{
		{
			name:                     "new long position",
			traderFunds:              sdk.NewCoins(sdk.NewInt64Coin(denoms.NUSD, 1020)),
			initialPosition:          nil,
			side:                     types.Side_BUY,
			margin:                   sdk.NewInt(1000),
			leverage:                 sdk.NewDec(10),
			baseLimit:                sdk.ZeroDec(),
			expectedMargin:           sdk.NewDec(1000),
			expectedOpenNotional:     sdk.NewDec(10_000),
			expectedSize:             sdk.MustNewDecFromStr("9999.999900000001"),
			expectedPositionNotional: sdk.NewDec(10_000),
			expectedUnrealizedPnl:    sdk.ZeroDec(),
			expectedRealizedPnl:      sdk.ZeroDec(),
			expectedMarginToVault:    sdk.NewDec(1000),
			expectedMarkPrice:        sdk.MustNewDecFromStr("1.0000000200000001"),
		},
		{
			name:        "existing long position, go more long",
			traderFunds: sdk.NewCoins(sdk.NewInt64Coin(denoms.NUSD, 1020)),
			initialPosition: &types.Position{
				Pair:                            asset.Registry.Pair(denoms.BTC, denoms.NUSD),
				Size_:                           sdk.NewDec(10_000),
				Margin:                          sdk.NewDec(1000),
				OpenNotional:                    sdk.NewDec(10_000),
				LatestCumulativePremiumFraction: sdk.ZeroDec(),
				BlockNumber:                     1,
			},
			side:                     types.Side_BUY,
			margin:                   sdk.NewInt(1000),
			leverage:                 sdk.NewDec(10),
			baseLimit:                sdk.ZeroDec(),
			expectedMargin:           sdk.NewDec(2000),
			expectedOpenNotional:     sdk.NewDec(20_000),
			expectedSize:             sdk.MustNewDecFromStr("19999.999900000001"),
			expectedPositionNotional: sdk.MustNewDecFromStr("20000.000099999999"),
			expectedUnrealizedPnl:    sdk.MustNewDecFromStr("0.000099999999"),
			expectedRealizedPnl:      sdk.ZeroDec(),
			expectedMarginToVault:    sdk.NewDec(1000),
			expectedMarkPrice:        sdk.MustNewDecFromStr("1.0000000200000001"),
		},
		{
			name:        "existing long position, decrease a bit",
			traderFunds: sdk.NewCoins(sdk.NewInt64Coin(denoms.NUSD, 10)),
			initialPosition: &types.Position{
				Pair:                            asset.Registry.Pair(denoms.BTC, denoms.NUSD),
				Size_:                           sdk.NewDec(10_000),
				Margin:                          sdk.NewDec(1000),
				OpenNotional:                    sdk.NewDec(10_000),
				LatestCumulativePremiumFraction: sdk.ZeroDec(),
				BlockNumber:                     1,
			},
			side:                     types.Side_SELL,
			margin:                   sdk.NewInt(500),
			leverage:                 sdk.NewDec(10),
			baseLimit:                sdk.ZeroDec(),
			expectedMargin:           sdk.MustNewDecFromStr("999.99995000000025"),
			expectedOpenNotional:     sdk.MustNewDecFromStr("4999.99995000000025"),
			expectedSize:             sdk.MustNewDecFromStr("4999.999974999999875"),
			expectedPositionNotional: sdk.MustNewDecFromStr("4999.999900000001"),
			expectedUnrealizedPnl:    sdk.MustNewDecFromStr("-0.00004999999925"),
			expectedRealizedPnl:      sdk.MustNewDecFromStr("-0.00004999999975"),
			expectedMarginToVault:    sdk.ZeroDec(),
			expectedMarkPrice:        sdk.MustNewDecFromStr("0.999999990000000025"),
		},
		{
			name:        "existing long position, decrease a lot",
			traderFunds: sdk.NewCoins(sdk.NewInt64Coin(denoms.NUSD, 1060)),
			initialPosition: &types.Position{
				Pair:                            asset.Registry.Pair(denoms.BTC, denoms.NUSD),
				Size_:                           sdk.NewDec(10_000),
				Margin:                          sdk.NewDec(1000),
				OpenNotional:                    sdk.NewDec(10_000),
				LatestCumulativePremiumFraction: sdk.ZeroDec(),
				BlockNumber:                     1,
			},
			side:                     types.Side_SELL,
			margin:                   sdk.NewInt(3000),
			leverage:                 sdk.NewDec(10),
			baseLimit:                sdk.ZeroDec(),
			expectedMargin:           sdk.MustNewDecFromStr("2000.0000099999999"),
			expectedOpenNotional:     sdk.MustNewDecFromStr("20000.000099999999"),
			expectedSize:             sdk.MustNewDecFromStr("-20000.000900000027000001"),
			expectedPositionNotional: sdk.MustNewDecFromStr("20000.000099999999"),
			expectedUnrealizedPnl:    sdk.ZeroDec(),
			expectedRealizedPnl:      sdk.MustNewDecFromStr("-0.000099999999"),
			expectedMarginToVault:    sdk.MustNewDecFromStr("1000.0001099999989"),
			expectedMarkPrice:        sdk.MustNewDecFromStr("0.9999999400000009"),
		},
		{
			name:                     "new long position just under fluctuation limit",
			traderFunds:              sdk.NewCoins(sdk.NewInt64Coin(denoms.NUSD, 1*common.Precision*common.Precision)),
			initialPosition:          nil,
			side:                     types.Side_BUY,
			margin:                   sdk.NewInt(47_619_047_619),
			leverage:                 sdk.OneDec(),
			baseLimit:                sdk.ZeroDec(),
			expectedMargin:           sdk.NewDec(47_619_047_619),
			expectedOpenNotional:     sdk.NewDec(47_619_047_619),
			expectedSize:             sdk.MustNewDecFromStr("45454545454.502066115702477367"),
			expectedPositionNotional: sdk.NewDec(47_619_047_619),
			expectedUnrealizedPnl:    sdk.ZeroDec(),
			expectedRealizedPnl:      sdk.ZeroDec(),
			expectedMarginToVault:    sdk.NewDec(47_619_047_619),
			expectedMarkPrice:        sdk.MustNewDecFromStr("1.09750566893414059"),
		},
		{
			name:                     "new short position",
			traderFunds:              sdk.NewCoins(sdk.NewInt64Coin(denoms.NUSD, 1020)),
			initialPosition:          nil,
			side:                     types.Side_SELL,
			margin:                   sdk.NewInt(1000),
			leverage:                 sdk.NewDec(10),
			baseLimit:                sdk.ZeroDec(),
			expectedMargin:           sdk.NewDec(1000),
			expectedOpenNotional:     sdk.NewDec(10_000),
			expectedSize:             sdk.MustNewDecFromStr("-10000.000100000001"),
			expectedPositionNotional: sdk.NewDec(10_000),
			expectedUnrealizedPnl:    sdk.ZeroDec(),
			expectedRealizedPnl:      sdk.ZeroDec(),
			expectedMarginToVault:    sdk.NewDec(1000),
			expectedMarkPrice:        sdk.MustNewDecFromStr("0.9999999800000001"),
		},
		{
			name:        "existing short position, go more short",
			traderFunds: sdk.NewCoins(sdk.NewInt64Coin(denoms.NUSD, 1020)),
			initialPosition: &types.Position{
				Pair:                            asset.Registry.Pair(denoms.BTC, denoms.NUSD),
				Size_:                           sdk.NewDec(-10_000),
				Margin:                          sdk.NewDec(1000),
				OpenNotional:                    sdk.NewDec(10_000),
				LatestCumulativePremiumFraction: sdk.ZeroDec(),
				BlockNumber:                     1,
			},
			side:                     types.Side_SELL,
			margin:                   sdk.NewInt(1000),
			leverage:                 sdk.NewDec(10),
			baseLimit:                sdk.ZeroDec(),
			expectedMargin:           sdk.NewDec(2000),
			expectedOpenNotional:     sdk.NewDec(20_000),
			expectedSize:             sdk.MustNewDecFromStr("-20000.000100000001"),
			expectedPositionNotional: sdk.MustNewDecFromStr("19999.999899999999"),
			expectedUnrealizedPnl:    sdk.MustNewDecFromStr("0.000100000001"),
			expectedRealizedPnl:      sdk.ZeroDec(),
			expectedMarginToVault:    sdk.NewDec(1000),
			expectedMarkPrice:        sdk.MustNewDecFromStr("0.9999999800000001"),
		},
		{
			name:        "existing short position, decrease a bit",
			traderFunds: sdk.NewCoins(sdk.NewInt64Coin(denoms.NUSD, 10)),
			initialPosition: &types.Position{
				Pair:                            asset.Registry.Pair(denoms.BTC, denoms.NUSD),
				Size_:                           sdk.NewDec(-10_000),
				Margin:                          sdk.NewDec(1000),
				OpenNotional:                    sdk.NewDec(10_000),
				LatestCumulativePremiumFraction: sdk.ZeroDec(),
				BlockNumber:                     1,
			},
			side:                     types.Side_BUY,
			margin:                   sdk.NewInt(500),
			leverage:                 sdk.NewDec(10),
			baseLimit:                sdk.ZeroDec(),
			expectedMargin:           sdk.MustNewDecFromStr("999.99994999999975"),
			expectedOpenNotional:     sdk.MustNewDecFromStr("5000.00005000000025"),
			expectedSize:             sdk.MustNewDecFromStr("-5000.000024999999875"),
			expectedPositionNotional: sdk.MustNewDecFromStr("5000.000100000001"),
			expectedUnrealizedPnl:    sdk.MustNewDecFromStr("-0.00005000000075"),
			expectedRealizedPnl:      sdk.MustNewDecFromStr("-0.00005000000025"),
			expectedMarginToVault:    sdk.ZeroDec(),
			expectedMarkPrice:        sdk.MustNewDecFromStr("1.000000010000000025"),
		},
		{
			name:        "existing short position, decrease a lot",
			traderFunds: sdk.NewCoins(sdk.NewInt64Coin(denoms.NUSD, 1060)),
			initialPosition: &types.Position{
				Pair:                            asset.Registry.Pair(denoms.BTC, denoms.NUSD),
				Size_:                           sdk.NewDec(-10_000),
				Margin:                          sdk.NewDec(1000),
				OpenNotional:                    sdk.NewDec(10_000),
				LatestCumulativePremiumFraction: sdk.ZeroDec(),
				BlockNumber:                     1,
			},
			side:                     types.Side_BUY,
			margin:                   sdk.NewInt(3000),
			leverage:                 sdk.NewDec(10),
			baseLimit:                sdk.ZeroDec(),
			expectedMargin:           sdk.MustNewDecFromStr("1999.9999899999999"),
			expectedOpenNotional:     sdk.MustNewDecFromStr("19999.999899999999"),
			expectedSize:             sdk.MustNewDecFromStr("19999.999100000026999999"),
			expectedPositionNotional: sdk.MustNewDecFromStr("19999.999899999999"),
			expectedUnrealizedPnl:    sdk.ZeroDec(),
			expectedRealizedPnl:      sdk.MustNewDecFromStr("-0.000100000001"),
			expectedMarginToVault:    sdk.MustNewDecFromStr("1000.0000900000009"),
			expectedMarkPrice:        sdk.MustNewDecFromStr("1.0000000600000009"),
		},
		{
			name:                     "new short position just under fluctuation limit",
			traderFunds:              sdk.NewCoins(sdk.NewInt64Coin(denoms.NUSD, 1*common.Precision*common.Precision)),
			initialPosition:          nil,
			side:                     types.Side_SELL,
			margin:                   sdk.NewInt(47_619_047_619),
			leverage:                 sdk.OneDec(),
			baseLimit:                sdk.ZeroDec(),
			expectedMargin:           sdk.NewDec(47_619_047_619),
			expectedOpenNotional:     sdk.NewDec(47_619_047_619),
			expectedSize:             sdk.MustNewDecFromStr("-49999999999.947500000000002625"),
			expectedPositionNotional: sdk.NewDec(47_619_047_619),
			expectedUnrealizedPnl:    sdk.ZeroDec(),
			expectedRealizedPnl:      sdk.ZeroDec(),
			expectedMarginToVault:    sdk.NewDec(47_619_047_619),
			expectedMarkPrice:        sdk.MustNewDecFromStr("0.90702947845814059"),
		},
	}

	for _, testCase := range testCases {
		tc := testCase
		t.Run(tc.name, func(t *testing.T) {
			t.Log("Setup Nibiru app and constants")
			nibiruApp, ctx := nibisimapp.NewTestNibiruAppAndContext(true)
			traderAddr := testutil.AccAddress()
			exchangedSize := tc.expectedSize

			t.Log("initialize vpool")
			assert.NoError(t, nibiruApp.VpoolKeeper.CreatePool(
				ctx,
				asset.Registry.Pair(denoms.BTC, denoms.NUSD),
				/* quoteReserve */ sdk.NewDec(1*common.Precision*common.Precision),
				/* baseReserve */ sdk.NewDec(1*common.Precision*common.Precision),
				vpooltypes.VpoolConfig{
					FluctuationLimitRatio:  sdk.MustNewDecFromStr("0.1"),
					MaintenanceMarginRatio: sdk.MustNewDecFromStr("0.0625"),
					MaxLeverage:            sdk.MustNewDecFromStr("15"),
					MaxOracleSpreadRatio:   sdk.OneDec(), // 100%,
					TradeLimitRatio:        sdk.OneDec(),
				},
			))
			setPairMetadata(nibiruApp.PerpKeeper, ctx, types.PairMetadata{
				Pair:                            asset.Registry.Pair(denoms.BTC, denoms.NUSD),
				LatestCumulativePremiumFraction: sdk.ZeroDec(),
			})

			t.Log("initialize trader funds")
			require.NoError(t, simapp.FundAccount(nibiruApp.BankKeeper, ctx, traderAddr, tc.traderFunds))

			if tc.initialPosition != nil {
				t.Log("set initial position")
				tc.initialPosition.TraderAddress = traderAddr.String()
				setPosition(nibiruApp.PerpKeeper, ctx, *tc.initialPosition)
				exchangedSize = exchangedSize.Sub(tc.initialPosition.Size_)
			}

			ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 1).WithBlockTime(ctx.BlockTime().Add(time.Second * 5))
			resp, err := nibiruApp.PerpKeeper.OpenPosition(ctx, asset.Registry.Pair(denoms.BTC, denoms.NUSD), tc.side, traderAddr, tc.margin, tc.leverage, tc.baseLimit)
			require.NoError(t, err)

			t.Log("assert position response")
			assert.EqualValues(t, asset.Registry.Pair(denoms.BTC, denoms.NUSD), resp.Position.Pair)
			assert.EqualValues(t, traderAddr.String(), resp.Position.TraderAddress)
			assert.EqualValues(t, tc.expectedMargin, resp.Position.Margin, "margin")
			assert.EqualValues(t, tc.expectedOpenNotional, resp.Position.OpenNotional, "open notional")
			assert.EqualValues(t, tc.expectedSize, resp.Position.Size_, "position size")
			assert.EqualValues(t, ctx.BlockHeight(), resp.Position.BlockNumber)
			assert.EqualValues(t, sdk.ZeroDec(), resp.Position.LatestCumulativePremiumFraction)
			assert.EqualValues(t, tc.leverage.MulInt(tc.margin), resp.ExchangedNotionalValue)
			assert.EqualValues(t, exchangedSize, resp.ExchangedPositionSize)
			assert.EqualValues(t, sdk.ZeroDec(), resp.BadDebt)
			assert.EqualValues(t, sdk.ZeroDec(), resp.FundingPayment)
			assert.EqualValues(t, tc.expectedRealizedPnl, resp.RealizedPnl)
			assert.EqualValues(t, tc.expectedUnrealizedPnl, resp.UnrealizedPnlAfter)
			assert.EqualValues(t, tc.expectedMarginToVault, resp.MarginToVault)
			assert.EqualValues(t, tc.expectedPositionNotional, resp.PositionNotional)

			t.Log("assert position in state")
			position, err := nibiruApp.PerpKeeper.Positions.Get(ctx, collections.Join(asset.Registry.Pair(denoms.BTC, denoms.NUSD), traderAddr))
			require.NoError(t, err)
			assert.EqualValues(t, asset.Registry.Pair(denoms.BTC, denoms.NUSD), position.Pair)
			assert.EqualValues(t, traderAddr.String(), position.TraderAddress)
			assert.EqualValues(t, tc.expectedMargin, position.Margin, "margin")
			assert.EqualValues(t, tc.expectedOpenNotional, position.OpenNotional, "open notional")
			assert.EqualValues(t, tc.expectedSize, position.Size_, "position size")
			assert.EqualValues(t, ctx.BlockHeight(), position.BlockNumber)
			assert.EqualValues(t, sdk.ZeroDec(), position.LatestCumulativePremiumFraction)

			exchangedNotional := tc.leverage.MulInt(tc.margin)
			feePoolFee := nibiruApp.PerpKeeper.GetParams(ctx).FeePoolFeeRatio.Mul(exchangedNotional).RoundInt()
			ecosystemFundFee := nibiruApp.PerpKeeper.GetParams(ctx).EcosystemFundFeeRatio.Mul(exchangedNotional).RoundInt()

			testutilevents.RequireHasTypedEvent(t, ctx, &types.PositionChangedEvent{
				Pair:               asset.Registry.Pair(denoms.BTC, denoms.NUSD),
				TraderAddress:      traderAddr.String(),
				Margin:             sdk.NewCoin(denoms.NUSD, tc.expectedMargin.RoundInt()),
				PositionNotional:   tc.expectedPositionNotional,
				ExchangedNotional:  exchangedNotional,
				ExchangedSize:      exchangedSize,
				PositionSize:       tc.expectedSize,
				RealizedPnl:        tc.expectedRealizedPnl,
				UnrealizedPnlAfter: tc.expectedUnrealizedPnl,
				BadDebt:            sdk.NewCoin(denoms.NUSD, sdk.ZeroInt()),
				MarkPrice:          tc.expectedMarkPrice,
				FundingPayment:     sdk.ZeroDec(),
				TransactionFee:     sdk.NewCoin(denoms.NUSD, feePoolFee.Add(ecosystemFundFee)),
				BlockHeight:        ctx.BlockHeight(),
				BlockTimeMs:        ctx.BlockTime().UnixMilli(),
			})
		})
	}
}

func TestOpenPositionError(t *testing.T) {
	testCases := []struct {
		name        string
		traderFunds sdk.Coins

		// vpool params
		poolTradeLimitRatio sdk.Dec

		initialPosition *types.Position

		// position arguments
		side      types.Side
		margin    sdk.Int
		leverage  sdk.Dec
		baseLimit sdk.Dec

		expectedErr error
	}{
		{
			name:                "not enough trader funds",
			traderFunds:         sdk.NewCoins(sdk.NewInt64Coin(denoms.NUSD, 999)),
			poolTradeLimitRatio: sdk.OneDec(),
			initialPosition:     nil,
			side:                types.Side_BUY,
			margin:              sdk.NewInt(1000),
			leverage:            sdk.NewDec(10),
			baseLimit:           sdk.ZeroDec(),
			expectedErr:         sdkerrors.ErrInsufficientFunds,
		},
		{
			name:                "position has bad debt",
			traderFunds:         sdk.NewCoins(sdk.NewInt64Coin(denoms.NUSD, 999)),
			poolTradeLimitRatio: sdk.OneDec(),
			initialPosition: &types.Position{
				Pair:                            asset.Registry.Pair(denoms.BTC, denoms.NUSD),
				Size_:                           sdk.OneDec(),
				Margin:                          sdk.NewDec(1000),
				OpenNotional:                    sdk.NewDec(10_000),
				LatestCumulativePremiumFraction: sdk.ZeroDec(),
				BlockNumber:                     1,
			},
			side:        types.Side_BUY,
			margin:      sdk.NewInt(1),
			leverage:    sdk.OneDec(),
			baseLimit:   sdk.ZeroDec(),
			expectedErr: types.ErrMarginRatioTooLow,
		},
		{
			name:                "new long position not over base limit",
			traderFunds:         sdk.NewCoins(sdk.NewInt64Coin(denoms.NUSD, 1020)),
			poolTradeLimitRatio: sdk.OneDec(),
			initialPosition:     nil,
			side:                types.Side_BUY,
			margin:              sdk.NewInt(1000),
			leverage:            sdk.NewDec(10),
			baseLimit:           sdk.NewDec(10_000),
			expectedErr:         vpooltypes.ErrAssetFailsUserLimit,
		},
		{
			name:                "new short position not under base limit",
			traderFunds:         sdk.NewCoins(sdk.NewInt64Coin(denoms.NUSD, 1020)),
			poolTradeLimitRatio: sdk.OneDec(),
			initialPosition:     nil,
			side:                types.Side_SELL,
			margin:              sdk.NewInt(1000),
			leverage:            sdk.NewDec(10),
			baseLimit:           sdk.NewDec(10_000),
			expectedErr:         vpooltypes.ErrAssetFailsUserLimit,
		},
		{
			name:                "quote asset amount is zero",
			traderFunds:         sdk.NewCoins(sdk.NewInt64Coin(denoms.NUSD, 1020)),
			poolTradeLimitRatio: sdk.OneDec(),
			initialPosition:     nil,
			side:                types.Side_SELL,
			margin:              sdk.NewInt(0),
			leverage:            sdk.NewDec(10),
			baseLimit:           sdk.NewDec(10_000),
			expectedErr:         types.ErrQuoteAmountIsZero,
		},
		{
			name:                "leverage amount is zero",
			traderFunds:         sdk.NewCoins(sdk.NewInt64Coin(denoms.NUSD, 1020)),
			poolTradeLimitRatio: sdk.OneDec(),
			initialPosition:     nil,
			side:                types.Side_SELL,
			margin:              sdk.NewInt(1000),
			leverage:            sdk.NewDec(0),
			baseLimit:           sdk.NewDec(10_000),
			expectedErr:         types.ErrLeverageIsZero,
		},
		{
			name:                "leverage amount is too high - SELL",
			traderFunds:         sdk.NewCoins(sdk.NewInt64Coin(denoms.NUSD, 1020)),
			poolTradeLimitRatio: sdk.OneDec(),
			initialPosition:     nil,
			side:                types.Side_SELL,
			margin:              sdk.NewInt(100),
			leverage:            sdk.NewDec(100),
			baseLimit:           sdk.NewDec(11_000),
			expectedErr:         types.ErrLeverageIsTooHigh,
		},
		{
			name:                "leverage amount is too high - BUY",
			traderFunds:         sdk.NewCoins(sdk.NewInt64Coin(denoms.NUSD, 1020)),
			poolTradeLimitRatio: sdk.OneDec(),
			initialPosition:     nil,
			side:                types.Side_BUY,
			margin:              sdk.NewInt(100),
			leverage:            sdk.NewDec(16),
			baseLimit:           sdk.NewDec(0),
			expectedErr:         types.ErrLeverageIsTooHigh,
		},
		{
			name:                "new long position over fluctuation limit",
			traderFunds:         sdk.NewCoins(sdk.NewInt64Coin(denoms.NUSD, 1*common.Precision*common.Precision)),
			poolTradeLimitRatio: sdk.OneDec(),
			initialPosition:     nil,
			side:                types.Side_BUY,
			margin:              sdk.NewInt(100_000 * common.Precision),
			leverage:            sdk.OneDec(),
			baseLimit:           sdk.ZeroDec(),
			expectedErr:         vpooltypes.ErrOverFluctuationLimit,
		},
		{
			name:                "new short position over fluctuation limit",
			traderFunds:         sdk.NewCoins(sdk.NewInt64Coin(denoms.NUSD, 1*common.Precision*common.Precision)),
			poolTradeLimitRatio: sdk.OneDec(),
			initialPosition:     nil,
			side:                types.Side_SELL,
			margin:              sdk.NewInt(100_000 * common.Precision),
			leverage:            sdk.OneDec(),
			baseLimit:           sdk.ZeroDec(),
			expectedErr:         vpooltypes.ErrOverFluctuationLimit,
		},
		{
			name:                "new long position over trade limit",
			traderFunds:         sdk.NewCoins(sdk.NewInt64Coin(denoms.NUSD, 10_000*common.Precision)),
			poolTradeLimitRatio: sdk.MustNewDecFromStr("0.01"),
			initialPosition:     nil,
			side:                types.Side_BUY,
			margin:              sdk.NewInt(100_000 * common.Precision),
			leverage:            sdk.OneDec(),
			baseLimit:           sdk.ZeroDec(),
			expectedErr:         vpooltypes.ErrOverTradingLimit,
		},
		{
			name:                "new short position over trade limit",
			traderFunds:         sdk.NewCoins(sdk.NewInt64Coin(denoms.NUSD, 10_000*common.Precision)),
			poolTradeLimitRatio: sdk.MustNewDecFromStr("0.01"),
			initialPosition:     nil,
			side:                types.Side_SELL,
			margin:              sdk.NewInt(100_000 * common.Precision),
			leverage:            sdk.OneDec(),
			baseLimit:           sdk.ZeroDec(),
			expectedErr:         vpooltypes.ErrOverTradingLimit,
		},
	}

	for _, testCase := range testCases {
		tc := testCase
		t.Run(tc.name, func(t *testing.T) {
			t.Log("Setup Nibiru app and constants")
			nibiruApp, ctx := nibisimapp.NewTestNibiruAppAndContext(true)
			traderAddr := testutil.AccAddress()

			t.Log("initialize vpool")
			assert.NoError(t, nibiruApp.VpoolKeeper.CreatePool(
				ctx,
				asset.Registry.Pair(denoms.BTC, denoms.NUSD),
				/* tradeLimitRatio */
				/* quoteReserve */
				sdk.NewDec(1*common.Precision*common.Precision),
				/* baseReserve */ sdk.NewDec(1*common.Precision*common.Precision),
				vpooltypes.VpoolConfig{
					FluctuationLimitRatio:  sdk.MustNewDecFromStr("0.1"),
					MaintenanceMarginRatio: sdk.MustNewDecFromStr("0.0625"),
					MaxLeverage:            sdk.MustNewDecFromStr("15"),
					MaxOracleSpreadRatio:   sdk.OneDec(), // 100%,
					TradeLimitRatio:        tc.poolTradeLimitRatio,
				},
			))
			setPairMetadata(nibiruApp.PerpKeeper, ctx, types.PairMetadata{
				Pair:                            asset.Registry.Pair(denoms.BTC, denoms.NUSD),
				LatestCumulativePremiumFraction: sdk.ZeroDec(),
			})

			t.Log("initialize trader funds")
			require.NoError(t, simapp.FundAccount(nibiruApp.BankKeeper, ctx, traderAddr, tc.traderFunds))

			if tc.initialPosition != nil {
				t.Log("set initial position")
				tc.initialPosition.TraderAddress = traderAddr.String()
				setPosition(nibiruApp.PerpKeeper, ctx, *tc.initialPosition)
			}

			ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 1).WithBlockTime(ctx.BlockTime().Add(time.Second * 5))
			resp, err := nibiruApp.PerpKeeper.OpenPosition(ctx, asset.Registry.Pair(denoms.BTC, denoms.NUSD), tc.side, traderAddr, tc.margin, tc.leverage, tc.baseLimit)
			require.ErrorContains(t, err, tc.expectedErr.Error())
			require.Nil(t, resp)
		})
	}
}

func TestOpenPositionInvalidPair(t *testing.T) {
	testCases := []struct {
		name string
		test func()
	}{
		{
			name: "open pos - uninitialized pool raised pair not supported error",
			test: func() {
				t.Log("Setup Nibiru app, pair, and trader without a vpool.")
				nibiruApp, ctx := nibisimapp.NewTestNibiruAppAndContext(true)
				pair := common.MustNewAssetPair("xxx:yyy")

				trader := testutil.AccAddress()

				t.Log("open a position on invalid 'pair'")
				side := types.Side_BUY
				quote := sdk.NewInt(60)
				leverage := sdk.NewDec(10)
				baseLimit := sdk.NewDec(150)
				resp, err := nibiruApp.PerpKeeper.OpenPosition(
					ctx, pair, side, trader, quote, leverage, baseLimit)
				require.ErrorContains(t, err, types.ErrPairNotFound.Error())
				require.Nil(t, resp)
			},
		},
		{
			name: "open pos - vpool not set on the perp PairMetadata ",
			test: func() {
				t.Log("Setup Nibiru app, pair, and trader")
				nibiruApp, ctx := nibisimapp.NewTestNibiruAppAndContext(true)
				pair := common.MustNewAssetPair("xxx:yyy")

				t.Log("Set vpool defined by pair on VpoolKeeper")
				vpoolKeeper := &nibiruApp.VpoolKeeper
				assert.NoError(t, nibiruApp.VpoolKeeper.CreatePool(
					ctx,
					pair,
					sdk.NewDec(10*common.Precision), //
					sdk.NewDec(5*common.Precision),  // 5 tokens
					vpooltypes.VpoolConfig{
						FluctuationLimitRatio:  sdk.MustNewDecFromStr("0.1"),
						MaintenanceMarginRatio: sdk.MustNewDecFromStr("0.0625"),
						MaxLeverage:            sdk.MustNewDecFromStr("15"),
						MaxOracleSpreadRatio:   sdk.MustNewDecFromStr("0.1"), // 100%,
						TradeLimitRatio:        sdk.MustNewDecFromStr("0.9"),
					},
				))

				require.True(t, vpoolKeeper.ExistsPool(ctx, pair))

				t.Log("Attempt to open long position (expected unsuccessful)")
				trader := testutil.AccAddress()
				side := types.Side_BUY
				quote := sdk.NewInt(60)
				leverage := sdk.NewDec(10)
				baseLimit := sdk.NewDec(150)
				resp, err := nibiruApp.PerpKeeper.OpenPosition(
					ctx, pair, side, trader, quote, leverage, baseLimit)
				require.ErrorIs(t, err, collections.ErrNotFound)
				require.Nil(t, resp)
			},
		},
	}

	for _, testCase := range testCases {
		tc := testCase
		t.Run(tc.name, func(t *testing.T) {
			tc.test()
		})
	}
}
