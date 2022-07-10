package keeper_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/cosmos/cosmos-sdk/simapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/perp/types"
	"github.com/NibiruChain/nibiru/x/testutil/sample"
	"github.com/NibiruChain/nibiru/x/testutil/testapp"
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

		expectedMargin       sdk.Dec
		expectedOpenNotional sdk.Dec
		expectedSize         sdk.Dec
	}{
		{
			name:                 "new long position",
			traderFunds:          sdk.NewCoins(sdk.NewInt64Coin(common.DenomStable, 1020)),
			initialPosition:      nil,
			side:                 types.Side_BUY,
			margin:               sdk.NewInt(1000),
			leverage:             sdk.NewDec(10),
			baseLimit:            sdk.ZeroDec(),
			expectedMargin:       sdk.NewDec(1000),
			expectedOpenNotional: sdk.NewDec(10_000),
			expectedSize:         sdk.MustNewDecFromStr("9999.999900000001"),
		},
		{
			name:        "existing long position, go more long",
			traderFunds: sdk.NewCoins(sdk.NewInt64Coin(common.DenomStable, 1020)),
			initialPosition: &types.Position{
				Pair:                                common.PairBTCStable,
				Size_:                               sdk.NewDec(10_000),
				Margin:                              sdk.NewDec(1000),
				OpenNotional:                        sdk.NewDec(10_000),
				LastUpdateCumulativePremiumFraction: sdk.ZeroDec(),
				BlockNumber:                         1,
			},
			side:                 types.Side_BUY,
			margin:               sdk.NewInt(1000),
			leverage:             sdk.NewDec(10),
			baseLimit:            sdk.ZeroDec(),
			expectedMargin:       sdk.NewDec(2000),
			expectedOpenNotional: sdk.NewDec(20_000),
			expectedSize:         sdk.MustNewDecFromStr("19999.999900000001"),
		},
		{
			name:        "existing long position, decrease a bit",
			traderFunds: sdk.NewCoins(sdk.NewInt64Coin(common.DenomStable, 10)),
			initialPosition: &types.Position{
				Pair:                                common.PairBTCStable,
				Size_:                               sdk.NewDec(10_000),
				Margin:                              sdk.NewDec(1000),
				OpenNotional:                        sdk.NewDec(10_000),
				LastUpdateCumulativePremiumFraction: sdk.ZeroDec(),
				BlockNumber:                         1,
			},
			side:                 types.Side_SELL,
			margin:               sdk.NewInt(500),
			leverage:             sdk.NewDec(10),
			baseLimit:            sdk.ZeroDec(),
			expectedMargin:       sdk.MustNewDecFromStr("999.99995000000025"),
			expectedOpenNotional: sdk.MustNewDecFromStr("4999.99995000000025"),
			expectedSize:         sdk.MustNewDecFromStr("4999.999974999999875"),
		},
		{
			name:        "existing long position, decrease a lot",
			traderFunds: sdk.NewCoins(sdk.NewInt64Coin(common.DenomStable, 1060)),
			initialPosition: &types.Position{
				Pair:                                common.PairBTCStable,
				Size_:                               sdk.NewDec(10_000),
				Margin:                              sdk.NewDec(1000),
				OpenNotional:                        sdk.NewDec(10_000),
				LastUpdateCumulativePremiumFraction: sdk.ZeroDec(),
				BlockNumber:                         1,
			},
			side:                 types.Side_SELL,
			margin:               sdk.NewInt(3000),
			leverage:             sdk.NewDec(10),
			baseLimit:            sdk.ZeroDec(),
			expectedMargin:       sdk.MustNewDecFromStr("2000.0000099999999"),
			expectedOpenNotional: sdk.MustNewDecFromStr("20000.000099999999"),
			expectedSize:         sdk.MustNewDecFromStr("-20000.000900000027000001"),
		},
		{
			name:                 "new short position",
			traderFunds:          sdk.NewCoins(sdk.NewInt64Coin(common.DenomStable, 1020)),
			initialPosition:      nil,
			side:                 types.Side_SELL,
			margin:               sdk.NewInt(1000),
			leverage:             sdk.NewDec(10),
			baseLimit:            sdk.ZeroDec(),
			expectedMargin:       sdk.NewDec(1000),
			expectedOpenNotional: sdk.NewDec(10_000),
			expectedSize:         sdk.MustNewDecFromStr("-10000.000100000001"),
		},
		{
			name:        "existing short position, go more short",
			traderFunds: sdk.NewCoins(sdk.NewInt64Coin(common.DenomStable, 1020)),
			initialPosition: &types.Position{
				Pair:                                common.PairBTCStable,
				Size_:                               sdk.NewDec(-10_000),
				Margin:                              sdk.NewDec(1000),
				OpenNotional:                        sdk.NewDec(10_000),
				LastUpdateCumulativePremiumFraction: sdk.ZeroDec(),
				BlockNumber:                         1,
			},
			side:                 types.Side_SELL,
			margin:               sdk.NewInt(1000),
			leverage:             sdk.NewDec(10),
			baseLimit:            sdk.ZeroDec(),
			expectedMargin:       sdk.NewDec(2000),
			expectedOpenNotional: sdk.NewDec(20_000),
			expectedSize:         sdk.MustNewDecFromStr("-20000.000100000001"),
		},
		{
			name:        "existing short position, decrease a bit",
			traderFunds: sdk.NewCoins(sdk.NewInt64Coin(common.DenomStable, 10)),
			initialPosition: &types.Position{
				Pair:                                common.PairBTCStable,
				Size_:                               sdk.NewDec(-10_000),
				Margin:                              sdk.NewDec(1000),
				OpenNotional:                        sdk.NewDec(10_000),
				LastUpdateCumulativePremiumFraction: sdk.ZeroDec(),
				BlockNumber:                         1,
			},
			side:                 types.Side_BUY,
			margin:               sdk.NewInt(500),
			leverage:             sdk.NewDec(10),
			baseLimit:            sdk.ZeroDec(),
			expectedMargin:       sdk.MustNewDecFromStr("999.99994999999975"),
			expectedOpenNotional: sdk.MustNewDecFromStr("5000.00005000000025"),
			expectedSize:         sdk.MustNewDecFromStr("-5000.000024999999875"),
		},
		{
			name:        "existing short position, decrease a lot",
			traderFunds: sdk.NewCoins(sdk.NewInt64Coin(common.DenomStable, 1060)),
			initialPosition: &types.Position{
				Pair:                                common.PairBTCStable,
				Size_:                               sdk.NewDec(-10_000),
				Margin:                              sdk.NewDec(1000),
				OpenNotional:                        sdk.NewDec(10_000),
				LastUpdateCumulativePremiumFraction: sdk.ZeroDec(),
				BlockNumber:                         1,
			},
			side:                 types.Side_BUY,
			margin:               sdk.NewInt(3000),
			leverage:             sdk.NewDec(10),
			baseLimit:            sdk.ZeroDec(),
			expectedMargin:       sdk.MustNewDecFromStr("1999.9999899999999"),
			expectedOpenNotional: sdk.MustNewDecFromStr("19999.999899999999"),
			expectedSize:         sdk.MustNewDecFromStr("19999.999100000026999999"),
		},
	}

	for _, testCase := range testCases {
		tc := testCase
		t.Run(tc.name, func(t *testing.T) {
			t.Log("Setup Nibiru app and constants")
			nibiruApp, ctx := testapp.NewNibiruAppAndContext(true)
			traderAddr := sample.AccAddress()
			oracle := sample.AccAddress()

			t.Log("set pricefeed oracle")
			nibiruApp.PricefeedKeeper.WhitelistOracles(ctx, []sdk.AccAddress{oracle})
			_, err := nibiruApp.PricefeedKeeper.SetPrice(ctx, oracle, common.PairBTCStable.String(), sdk.OneDec(), time.Now().Add(time.Hour))
			require.NoError(t, err)
			require.NoError(t, nibiruApp.PricefeedKeeper.SetCurrentPrices(ctx, common.DenomAxlBTC, common.DenomStable))

			t.Log("initialize vpool")
			nibiruApp.VpoolKeeper.CreatePool(
				ctx,
				common.PairBTCStable,
				/* tradeLimitRatio */ sdk.OneDec(),
				/* quoteReserve */ sdk.NewDec(1_000_000_000_000),
				/* baseReserve */ sdk.NewDec(1_000_000_000_000),
				/* fluctuationLimit */ sdk.OneDec(),
				/* maxOracleSpreadRatio */ sdk.OneDec(),
			)
			nibiruApp.PerpKeeper.PairMetadataState(ctx).Set(&types.PairMetadata{
				Pair:                       common.PairBTCStable,
				CumulativePremiumFractions: []sdk.Dec{sdk.ZeroDec()},
			})

			t.Log("initialize trader funds")
			require.NoError(t, simapp.FundAccount(nibiruApp.BankKeeper, ctx, traderAddr, tc.traderFunds))

			if tc.initialPosition != nil {
				t.Log("set initial position")
				tc.initialPosition.TraderAddress = traderAddr.String()
				nibiruApp.PerpKeeper.PositionsState(ctx).Set(common.PairBTCStable, traderAddr, tc.initialPosition)
			}

			ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 1).WithBlockTime(ctx.BlockTime().Add(time.Second * 5))
			err = nibiruApp.PerpKeeper.OpenPosition(ctx, common.PairBTCStable, tc.side, traderAddr, tc.margin, tc.leverage, tc.baseLimit)
			require.NoError(t, err)

			position, err := nibiruApp.PerpKeeper.PositionsState(ctx).Get(common.PairBTCStable, traderAddr)
			require.NoError(t, err)
			assert.EqualValues(t, common.PairBTCStable, position.Pair)
			assert.EqualValues(t, traderAddr.String(), position.TraderAddress)
			assert.EqualValues(t, tc.expectedMargin, position.Margin, "margin")
			assert.EqualValues(t, tc.expectedOpenNotional, position.OpenNotional, "open notional")
			assert.EqualValues(t, tc.expectedSize, position.Size_, "position size")
			assert.EqualValues(t, ctx.BlockHeight(), position.BlockNumber)
			assert.EqualValues(t, sdk.ZeroDec(), position.LastUpdateCumulativePremiumFraction)
		})
	}
}

func TestOpenPositionError(t *testing.T) {
	testCases := []struct {
		name        string
		traderFunds sdk.Coins

		initialPosition *types.Position

		side      types.Side
		margin    sdk.Int
		leverage  sdk.Dec
		baseLimit sdk.Dec

		expectedErr error
	}{
		{
			name:            "not enough trader funds",
			traderFunds:     sdk.NewCoins(sdk.NewInt64Coin(common.DenomStable, 999)),
			initialPosition: nil,
			side:            types.Side_BUY,
			margin:          sdk.NewInt(1000),
			leverage:        sdk.NewDec(10),
			baseLimit:       sdk.ZeroDec(),
			expectedErr:     sdkerrors.ErrInsufficientFunds,
		},
		{
			name:        "position has bad debt",
			traderFunds: sdk.NewCoins(sdk.NewInt64Coin(common.DenomStable, 999)),
			initialPosition: &types.Position{
				Pair:                                common.PairBTCStable,
				Size_:                               sdk.NewDec(1),
				Margin:                              sdk.NewDec(1000),
				OpenNotional:                        sdk.NewDec(10_000),
				LastUpdateCumulativePremiumFraction: sdk.ZeroDec(),
				BlockNumber:                         1,
			},
			side:        types.Side_BUY,
			margin:      sdk.NewInt(1),
			leverage:    sdk.NewDec(1),
			baseLimit:   sdk.ZeroDec(),
			expectedErr: fmt.Errorf("margin ratio did not meet criteria"),
		},
		{
			name:            "new long position not over base limit",
			traderFunds:     sdk.NewCoins(sdk.NewInt64Coin(common.DenomStable, 1020)),
			initialPosition: nil,
			side:            types.Side_BUY,
			margin:          sdk.NewInt(1000),
			leverage:        sdk.NewDec(10),
			baseLimit:       sdk.NewDec(10_000),
			expectedErr:     vpooltypes.ErrAssetOverUserLimit,
		},
		{
			name:            "new short position not under base limit",
			traderFunds:     sdk.NewCoins(sdk.NewInt64Coin(common.DenomStable, 1020)),
			initialPosition: nil,
			side:            types.Side_SELL,
			margin:          sdk.NewInt(1000),
			leverage:        sdk.NewDec(10),
			baseLimit:       sdk.NewDec(10_000),
			expectedErr:     vpooltypes.ErrAssetOverUserLimit,
		},
	}

	for _, testCase := range testCases {
		tc := testCase
		t.Run(tc.name, func(t *testing.T) {
			t.Log("Setup Nibiru app and constants")
			nibiruApp, ctx := testapp.NewNibiruAppAndContext(true)
			traderAddr := sample.AccAddress()
			oracle := sample.AccAddress()

			t.Log("set pricefeed oracle")
			nibiruApp.PricefeedKeeper.WhitelistOracles(ctx, []sdk.AccAddress{oracle})
			_, err := nibiruApp.PricefeedKeeper.SetPrice(ctx, oracle, common.PairBTCStable.String(), sdk.OneDec(), time.Now().Add(time.Hour))
			require.NoError(t, err)
			require.NoError(t, nibiruApp.PricefeedKeeper.SetCurrentPrices(ctx, common.DenomAxlBTC, common.DenomStable))

			t.Log("initialize vpool")
			nibiruApp.VpoolKeeper.CreatePool(
				ctx,
				common.PairBTCStable,
				/* tradeLimitRatio */ sdk.OneDec(),
				/* quoteReserve */ sdk.NewDec(1_000_000_000_000),
				/* baseReserve */ sdk.NewDec(1_000_000_000_000),
				/* fluctuationLimit */ sdk.OneDec(),
				/* maxOracleSpreadRatio */ sdk.OneDec(),
			)
			nibiruApp.PerpKeeper.PairMetadataState(ctx).Set(&types.PairMetadata{
				Pair:                       common.PairBTCStable,
				CumulativePremiumFractions: []sdk.Dec{sdk.ZeroDec()},
			})

			t.Log("initialize trader funds")
			require.NoError(t, simapp.FundAccount(nibiruApp.BankKeeper, ctx, traderAddr, tc.traderFunds))

			if tc.initialPosition != nil {
				t.Log("set initial position")
				tc.initialPosition.TraderAddress = traderAddr.String()
				nibiruApp.PerpKeeper.PositionsState(ctx).Set(common.PairBTCStable, traderAddr, tc.initialPosition)
			}

			ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 1).WithBlockTime(ctx.BlockTime().Add(time.Second * 5))
			err = nibiruApp.PerpKeeper.OpenPosition(ctx, common.PairBTCStable, tc.side, traderAddr, tc.margin, tc.leverage, tc.baseLimit)
			require.ErrorContains(t, err, tc.expectedErr.Error())
		})
	}
}
