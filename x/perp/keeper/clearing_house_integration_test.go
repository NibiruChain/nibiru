package keeper_test

import (
	"testing"
	"time"

	"github.com/cosmos/cosmos-sdk/simapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	nibisimapp "github.com/NibiruChain/nibiru/simapp"
	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/perp/types"
	"github.com/NibiruChain/nibiru/x/testutil/sample"
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
	}{
		{
			name:                     "new long position",
			traderFunds:              sdk.NewCoins(sdk.NewInt64Coin(common.DenomStable, 1020)),
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
		},
		{
			name:        "existing long position, go more long",
			traderFunds: sdk.NewCoins(sdk.NewInt64Coin(common.DenomStable, 1020)),
			initialPosition: &types.Position{
				Pair:                           common.PairBTCStable,
				Size_:                          sdk.NewDec(10_000),
				Margin:                         sdk.NewDec(1000),
				OpenNotional:                   sdk.NewDec(10_000),
				LatestCumulativeFundingPayment: sdk.ZeroDec(),
				BlockNumber:                    1,
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
		},
		{
			name:        "existing long position, decrease a bit",
			traderFunds: sdk.NewCoins(sdk.NewInt64Coin(common.DenomStable, 10)),
			initialPosition: &types.Position{
				Pair:                           common.PairBTCStable,
				Size_:                          sdk.NewDec(10_000),
				Margin:                         sdk.NewDec(1000),
				OpenNotional:                   sdk.NewDec(10_000),
				LatestCumulativeFundingPayment: sdk.ZeroDec(),
				BlockNumber:                    1,
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
		},
		{
			name:        "existing long position, decrease a lot",
			traderFunds: sdk.NewCoins(sdk.NewInt64Coin(common.DenomStable, 1060)),
			initialPosition: &types.Position{
				Pair:                           common.PairBTCStable,
				Size_:                          sdk.NewDec(10_000),
				Margin:                         sdk.NewDec(1000),
				OpenNotional:                   sdk.NewDec(10_000),
				LatestCumulativeFundingPayment: sdk.ZeroDec(),
				BlockNumber:                    1,
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
		},
		{
			name:                     "new long position just under fluctuation limit",
			traderFunds:              sdk.NewCoins(sdk.NewInt64Coin(common.DenomStable, 1_000_000_000_000)),
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
		},
		{
			name:                     "new short position",
			traderFunds:              sdk.NewCoins(sdk.NewInt64Coin(common.DenomStable, 1020)),
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
		},
		{
			name:        "existing short position, go more short",
			traderFunds: sdk.NewCoins(sdk.NewInt64Coin(common.DenomStable, 1020)),
			initialPosition: &types.Position{
				Pair:                           common.PairBTCStable,
				Size_:                          sdk.NewDec(-10_000),
				Margin:                         sdk.NewDec(1000),
				OpenNotional:                   sdk.NewDec(10_000),
				LatestCumulativeFundingPayment: sdk.ZeroDec(),
				BlockNumber:                    1,
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
		},
		{
			name:        "existing short position, decrease a bit",
			traderFunds: sdk.NewCoins(sdk.NewInt64Coin(common.DenomStable, 10)),
			initialPosition: &types.Position{
				Pair:                           common.PairBTCStable,
				Size_:                          sdk.NewDec(-10_000),
				Margin:                         sdk.NewDec(1000),
				OpenNotional:                   sdk.NewDec(10_000),
				LatestCumulativeFundingPayment: sdk.ZeroDec(),
				BlockNumber:                    1,
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
		},
		{
			name:        "existing short position, decrease a lot",
			traderFunds: sdk.NewCoins(sdk.NewInt64Coin(common.DenomStable, 1060)),
			initialPosition: &types.Position{
				Pair:                           common.PairBTCStable,
				Size_:                          sdk.NewDec(-10_000),
				Margin:                         sdk.NewDec(1000),
				OpenNotional:                   sdk.NewDec(10_000),
				LatestCumulativeFundingPayment: sdk.ZeroDec(),
				BlockNumber:                    1,
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
		},
		{
			name:                     "new short position just under fluctuation limit",
			traderFunds:              sdk.NewCoins(sdk.NewInt64Coin(common.DenomStable, 1_000_000_000_000)),
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
		},
	}

	for _, testCase := range testCases {
		tc := testCase
		t.Run(tc.name, func(t *testing.T) {
			t.Log("Setup Nibiru app and constants")
			nibiruApp, ctx := nibisimapp.NewTestNibiruAppAndContext(true)
			traderAddr := sample.AccAddress()
			oracle := sample.AccAddress()
			exchangedSize := tc.expectedSize

			t.Log("set pricefeed oracle")
			nibiruApp.PricefeedKeeper.WhitelistOracles(ctx, []sdk.AccAddress{oracle})
			require.NoError(t, nibiruApp.PricefeedKeeper.PostRawPrice(ctx, oracle, common.PairBTCStable.String(), sdk.OneDec(), time.Now().Add(time.Hour)))
			require.NoError(t, nibiruApp.PricefeedKeeper.GatherRawPrices(ctx, common.DenomBTC, common.DenomStable))

			t.Log("initialize vpool")
			nibiruApp.VpoolKeeper.CreatePool(
				ctx,
				common.PairBTCStable,
				/* tradeLimitRatio */ sdk.OneDec(),
				/* quoteReserve */ sdk.NewDec(1_000_000_000_000),
				/* baseReserve */ sdk.NewDec(1_000_000_000_000),
				/* fluctuationLimit */ sdk.MustNewDecFromStr("0.1"),
				/* maxOracleSpreadRatio */ sdk.OneDec(),
				/* maintenanceMarginRatio */ sdk.MustNewDecFromStr("0.0625"),
				/* maxLeverage */ sdk.MustNewDecFromStr("15"),
			)
			nibiruApp.PerpKeeper.PairMetadataState(ctx).Set(&types.PairMetadata{
				Pair:                   common.PairBTCStable,
				CumulativeFundingRates: []sdk.Dec{sdk.ZeroDec()},
			})

			t.Log("initialize trader funds")
			require.NoError(t, simapp.FundAccount(nibiruApp.BankKeeper, ctx, traderAddr, tc.traderFunds))

			if tc.initialPosition != nil {
				t.Log("set initial position")
				tc.initialPosition.TraderAddress = traderAddr.String()
				nibiruApp.PerpKeeper.PositionsState(ctx).Set(tc.initialPosition)
				exchangedSize = exchangedSize.Sub(tc.initialPosition.Size_)
			}

			ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 1).WithBlockTime(ctx.BlockTime().Add(time.Second * 5))
			resp, err := nibiruApp.PerpKeeper.OpenPosition(ctx, common.PairBTCStable, tc.side, traderAddr, tc.margin, tc.leverage, tc.baseLimit)
			require.NoError(t, err)

			t.Log("assert position response")
			assert.EqualValues(t, common.PairBTCStable, resp.Position.Pair)
			assert.EqualValues(t, traderAddr.String(), resp.Position.TraderAddress)
			assert.EqualValues(t, tc.expectedMargin, resp.Position.Margin, "margin")
			assert.EqualValues(t, tc.expectedOpenNotional, resp.Position.OpenNotional, "open notional")
			assert.EqualValues(t, tc.expectedSize, resp.Position.Size_, "position size")
			assert.EqualValues(t, ctx.BlockHeight(), resp.Position.BlockNumber)
			assert.EqualValues(t, sdk.ZeroDec(), resp.Position.LatestCumulativeFundingPayment)
			assert.EqualValues(t, tc.leverage.MulInt(tc.margin), resp.ExchangedNotionalValue)
			assert.EqualValues(t, exchangedSize, resp.ExchangedPositionSize)
			assert.EqualValues(t, sdk.ZeroDec(), resp.BadDebt)
			assert.EqualValues(t, sdk.ZeroDec(), resp.FundingPayment)
			assert.EqualValues(t, tc.expectedRealizedPnl, resp.RealizedPnl)
			assert.EqualValues(t, tc.expectedUnrealizedPnl, resp.UnrealizedPnlAfter)
			assert.EqualValues(t, tc.expectedMarginToVault, resp.MarginToVault)
			assert.EqualValues(t, tc.expectedPositionNotional, resp.PositionNotional)

			t.Log("assert position in state")
			position, err := nibiruApp.PerpKeeper.PositionsState(ctx).Get(common.PairBTCStable, traderAddr)
			require.NoError(t, err)
			assert.EqualValues(t, common.PairBTCStable, position.Pair)
			assert.EqualValues(t, traderAddr.String(), position.TraderAddress)
			assert.EqualValues(t, tc.expectedMargin, position.Margin, "margin")
			assert.EqualValues(t, tc.expectedOpenNotional, position.OpenNotional, "open notional")
			assert.EqualValues(t, tc.expectedSize, position.Size_, "position size")
			assert.EqualValues(t, ctx.BlockHeight(), position.BlockNumber)
			assert.EqualValues(t, sdk.ZeroDec(), position.LatestCumulativeFundingPayment)
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
			traderFunds:         sdk.NewCoins(sdk.NewInt64Coin(common.DenomStable, 999)),
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
			traderFunds:         sdk.NewCoins(sdk.NewInt64Coin(common.DenomStable, 999)),
			poolTradeLimitRatio: sdk.OneDec(),
			initialPosition: &types.Position{
				Pair:                           common.PairBTCStable,
				Size_:                          sdk.OneDec(),
				Margin:                         sdk.NewDec(1000),
				OpenNotional:                   sdk.NewDec(10_000),
				LatestCumulativeFundingPayment: sdk.ZeroDec(),
				BlockNumber:                    1,
			},
			side:        types.Side_BUY,
			margin:      sdk.NewInt(1),
			leverage:    sdk.OneDec(),
			baseLimit:   sdk.ZeroDec(),
			expectedErr: types.ErrMarginRatioTooLow,
		},
		{
			name:                "new long position not over base limit",
			traderFunds:         sdk.NewCoins(sdk.NewInt64Coin(common.DenomStable, 1020)),
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
			traderFunds:         sdk.NewCoins(sdk.NewInt64Coin(common.DenomStable, 1020)),
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
			traderFunds:         sdk.NewCoins(sdk.NewInt64Coin(common.DenomStable, 1020)),
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
			traderFunds:         sdk.NewCoins(sdk.NewInt64Coin(common.DenomStable, 1020)),
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
			traderFunds:         sdk.NewCoins(sdk.NewInt64Coin(common.DenomStable, 1020)),
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
			traderFunds:         sdk.NewCoins(sdk.NewInt64Coin(common.DenomStable, 1020)),
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
			traderFunds:         sdk.NewCoins(sdk.NewInt64Coin(common.DenomStable, 1_000_000_000_000)),
			poolTradeLimitRatio: sdk.OneDec(),
			initialPosition:     nil,
			side:                types.Side_BUY,
			margin:              sdk.NewInt(100_000_000_000),
			leverage:            sdk.OneDec(),
			baseLimit:           sdk.ZeroDec(),
			expectedErr:         vpooltypes.ErrOverFluctuationLimit,
		},
		{
			name:                "new short position over fluctuation limit",
			traderFunds:         sdk.NewCoins(sdk.NewInt64Coin(common.DenomStable, 1_000_000_000_000)),
			poolTradeLimitRatio: sdk.OneDec(),
			initialPosition:     nil,
			side:                types.Side_SELL,
			margin:              sdk.NewInt(100_000_000_000),
			leverage:            sdk.OneDec(),
			baseLimit:           sdk.ZeroDec(),
			expectedErr:         vpooltypes.ErrOverFluctuationLimit,
		},
		{
			name:                "new long position over trade limit",
			traderFunds:         sdk.NewCoins(sdk.NewInt64Coin(common.DenomStable, 10_000_000_000)),
			poolTradeLimitRatio: sdk.MustNewDecFromStr("0.01"),
			initialPosition:     nil,
			side:                types.Side_BUY,
			margin:              sdk.NewInt(100_000_000_000),
			leverage:            sdk.OneDec(),
			baseLimit:           sdk.ZeroDec(),
			expectedErr:         vpooltypes.ErrOverTradingLimit,
		},
		{
			name:                "new short position over trade limit",
			traderFunds:         sdk.NewCoins(sdk.NewInt64Coin(common.DenomStable, 10_000_000_000)),
			poolTradeLimitRatio: sdk.MustNewDecFromStr("0.01"),
			initialPosition:     nil,
			side:                types.Side_SELL,
			margin:              sdk.NewInt(100_000_000_000),
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
			traderAddr := sample.AccAddress()
			oracle := sample.AccAddress()

			t.Log("set pricefeed oracle")
			nibiruApp.PricefeedKeeper.WhitelistOracles(ctx, []sdk.AccAddress{oracle})
			require.NoError(t, nibiruApp.PricefeedKeeper.PostRawPrice(ctx, oracle, common.PairBTCStable.String(), sdk.OneDec(), time.Now().Add(time.Hour)))
			require.NoError(t, nibiruApp.PricefeedKeeper.GatherRawPrices(ctx, common.DenomBTC, common.DenomStable))

			t.Log("initialize vpool")
			nibiruApp.VpoolKeeper.CreatePool(
				ctx,
				common.PairBTCStable,
				/* tradeLimitRatio */ tc.poolTradeLimitRatio,
				/* quoteReserve */ sdk.NewDec(1_000_000_000_000),
				/* baseReserve */ sdk.NewDec(1_000_000_000_000),
				/* fluctuationLimit */ sdk.MustNewDecFromStr("0.1"),
				/* maxOracleSpreadRatio */ sdk.OneDec(),
				/* maintenanceMarginRatio */ sdk.MustNewDecFromStr("0.0625"),
				/* maxLeverage */ sdk.MustNewDecFromStr("15"),
			)
			nibiruApp.PerpKeeper.PairMetadataState(ctx).Set(&types.PairMetadata{
				Pair:                   common.PairBTCStable,
				CumulativeFundingRates: []sdk.Dec{sdk.ZeroDec()},
			})

			t.Log("initialize trader funds")
			require.NoError(t, simapp.FundAccount(nibiruApp.BankKeeper, ctx, traderAddr, tc.traderFunds))

			if tc.initialPosition != nil {
				t.Log("set initial position")
				tc.initialPosition.TraderAddress = traderAddr.String()
				nibiruApp.PerpKeeper.PositionsState(ctx).Set(tc.initialPosition)
			}

			ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 1).WithBlockTime(ctx.BlockTime().Add(time.Second * 5))
			resp, err := nibiruApp.PerpKeeper.OpenPosition(ctx, common.PairBTCStable, tc.side, traderAddr, tc.margin, tc.leverage, tc.baseLimit)
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

				trader := sample.AccAddress()

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
				vpoolKeeper.CreatePool(
					ctx,
					pair,
					sdk.MustNewDecFromStr("0.9"), // 0.9 ratio
					sdk.NewDec(10_000_000),       //
					sdk.NewDec(5_000_000),        // 5 tokens
					sdk.MustNewDecFromStr("0.1"), // 0.9 ratio
					sdk.MustNewDecFromStr("0.1"),
					/* maintenanceMarginRatio */ sdk.MustNewDecFromStr("0.0625"),
					/* maxLeverage */ sdk.MustNewDecFromStr("15"),
				)
				nibiruApp.PricefeedKeeper.ActivePairsStore().Set(ctx, pair, true)

				require.True(t, vpoolKeeper.ExistsPool(ctx, pair))

				t.Log("Attempt to open long position (expected unsuccessful)")
				trader := sample.AccAddress()
				side := types.Side_BUY
				quote := sdk.NewInt(60)
				leverage := sdk.NewDec(10)
				baseLimit := sdk.NewDec(150)
				resp, err := nibiruApp.PerpKeeper.OpenPosition(
					ctx, pair, side, trader, quote, leverage, baseLimit)
				require.ErrorContains(t, err, types.ErrPairMetadataNotFound.Error())
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
