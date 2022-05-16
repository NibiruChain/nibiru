package keeper_test

import (
	"testing"
	"time"

	"github.com/cosmos/cosmos-sdk/simapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/perp/events"
	"github.com/NibiruChain/nibiru/x/perp/types"
	"github.com/NibiruChain/nibiru/x/testutil"
	"github.com/NibiruChain/nibiru/x/testutil/sample"
)

func TestOpenPosition_Setup(t *testing.T) {
	testCases := []struct {
		name string
		test func()
	}{
		{
			name: "open pos - uninitialized pool raised pair not supported error",
			test: func() {
				t.Log("Setup Nibiru app, pair, and trader without a vpool.")
				nibiruApp, ctx := testutil.NewNibiruApp(true)
				pair := common.TokenPair("xxx:yyy")
				alice := sample.AccAddress()

				t.Log("open a position on invalid 'pair'")
				side := types.Side_BUY
				quote := sdk.NewDec(60)
				leverage := sdk.NewDec(10)
				baseLimit := sdk.NewDec(150)
				err := nibiruApp.PerpKeeper.OpenPosition(
					ctx, pair, side, alice, quote, leverage, baseLimit)
				require.Error(t, err)
				require.ErrorContains(t, err, types.ErrPairNotFound.Error())
			},
		},
		{
			name: "open pos - vpool not set on the perp PairMetadata ",
			test: func() {
				t.Log("Setup Nibiru app, pair, and trader")
				nibiruApp, ctx := testutil.NewNibiruApp(true)
				pair := common.TokenPair("xxx:yyy")

				t.Log("Set vpool defined by pair on VpoolKeeper")
				vpoolKeeper := &nibiruApp.VpoolKeeper
				vpoolKeeper.CreatePool(
					ctx,
					pair.String(),
					sdk.MustNewDecFromStr("0.9"), // 0.9 ratio
					sdk.NewDec(10_000_000),       //
					sdk.NewDec(5_000_000),        // 5 tokens
					sdk.MustNewDecFromStr("0.1"), // 0.9 ratio
				)

				require.True(t, vpoolKeeper.ExistsPool(ctx, pair))

				t.Log("Attempt to open long position (expected unsuccessful)")
				alice := sample.AccAddress()
				side := types.Side_BUY
				quote := sdk.NewDec(60)
				leverage := sdk.NewDec(10)
				baseLimit := sdk.NewDec(150)
				err := nibiruApp.PerpKeeper.OpenPosition(
					ctx, pair, side, alice, quote, leverage, baseLimit)

				require.Error(t, err)
				require.ErrorContains(t, err, types.ErrPairNotFound.Error())
			},
		},
		{
			name: "open pos - happy path 1",
			test: func() {
				t.Log("Setup Nibiru app, pair, and trader")
				nibiruApp, ctx := testutil.NewNibiruApp(true)
				pair := common.TokenPair("xxx:yyy")

				t.Log("Set vpool defined by pair on VpoolKeeper")
				vpoolKeeper := &nibiruApp.VpoolKeeper
				vpoolKeeper.CreatePool(
					ctx,
					pair.String(),
					sdk.MustNewDecFromStr("0.9"), // 0.9 ratio
					sdk.NewDec(10_000_000),       //
					sdk.NewDec(5_000_000),        // 5 tokens
					sdk.MustNewDecFromStr("0.1"), // 0.9 ratio
				)
				require.True(t, vpoolKeeper.ExistsPool(ctx, pair))

				t.Log("Set vpool defined by pair on PerpKeeper")
				perpKeeper := &nibiruApp.PerpKeeper
				perpKeeper.PairMetadata().Set(ctx, &types.PairMetadata{
					Pair: pair.String(),
					CumulativePremiumFractions: []sdk.Dec{
						sdk.OneDec()},
				})

				t.Log("Fund trader (Alice) account with sufficient quote")
				var err error
				alice := sample.AccAddress()
				err = simapp.FundAccount(nibiruApp.BankKeeper, ctx, alice,
					sdk.NewCoins(sdk.NewInt64Coin("yyy", 60)))
				require.NoError(t, err)

				t.Log("Open long position with 10x leverage")
				side := types.Side_BUY
				quote := sdk.NewDec(60)
				leverage := sdk.NewDec(10)
				baseLimit := sdk.NewDec(150)
				err = nibiruApp.PerpKeeper.OpenPosition(
					ctx, pair, side, alice, quote, leverage, baseLimit)

				require.NoError(t, err)
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

func TestAddMargin_ShouldRaiseError(t *testing.T) {
	tests := []struct {
		name string
		test func()
	}{
		{
			name: "msg denom differs from pair quote asset",
			test: func() {
				nibiruApp, ctx := testutil.NewNibiruApp(true)

				tokenPair, err := common.NewTokenPairFromStr("atom:nusd")
				require.NoError(t, err)

				t.Log("Set vpool defined by pair on VpoolKeeper")
				vpoolKeeper := &nibiruApp.VpoolKeeper
				vpoolKeeper.CreatePool(
					ctx,
					tokenPair.String(),
					sdk.MustNewDecFromStr("0.9"), // 0.9 ratio
					sdk.NewDec(10_000_000),       //
					sdk.NewDec(5_000_000),        // 5 tokens
					sdk.MustNewDecFromStr("0.1"), // 0.9 ratio
				)
				require.True(t, vpoolKeeper.ExistsPool(ctx, tokenPair))

				t.Log("create msg for MsgAddMargin with invalid denom")
				traderAddr := sample.AccAddress()
				msg := &types.MsgAddMargin{
					Sender:    traderAddr.String(),
					TokenPair: tokenPair.String(),
					Margin:    sdk.NewCoin("notADenom", sdk.NewInt(400)),
				}

				goCtx := sdk.WrapSDKContext(ctx)
				_, err = nibiruApp.PerpKeeper.AddMargin(goCtx, msg)
				require.Error(t, err)
				require.ErrorContains(t, err, "invalid margin denom")
			},
		},
	}

	for _, testCase := range tests {
		tc := testCase
		t.Run(tc.name, func(t *testing.T) {
			tc.test()
		})
	}
}

func TestAddMargin_HappyPath(t *testing.T) {
	tests := []struct {
		name           string
		initialMargin  sdk.Dec
		addedMargin    sdk.Int
		expectedMargin sdk.Dec
	}{
		{
			name:           "add margin",
			initialMargin:  sdk.NewDec(100),
			addedMargin:    sdk.NewInt(100),
			expectedMargin: sdk.NewDec(200),
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			nibiruApp, ctx := testutil.NewNibiruApp(true)

			tokenPair, err := common.NewTokenPairFromStr("atom:nusd")
			require.NoError(t, err)

			t.Log("add margin funds (NUSD) to trader's account")
			traderAddr := sample.AccAddress()
			err = simapp.FundAccount(
				nibiruApp.BankKeeper,
				ctx,
				traderAddr,
				sdk.NewCoins(
					sdk.NewCoin(tokenPair.GetQuoteTokenDenom(), tc.addedMargin),
				),
			)
			require.NoErrorf(t, err, "fund account call should work")

			t.Log("Set vpool defined by pair on VpoolKeeper")
			vpoolKeeper := &nibiruApp.VpoolKeeper
			vpoolKeeper.CreatePool(
				ctx,
				tokenPair.String(),
				sdk.MustNewDecFromStr("0.9"), // 0.9 ratio
				sdk.NewDec(10_000_000),       //
				sdk.NewDec(5_000_000),        // 5 tokens
				sdk.MustNewDecFromStr("0.1"), // 0.9 ratio
			)
			require.True(t, vpoolKeeper.ExistsPool(ctx, tokenPair))

			t.Log("Set vpool defined by pair on PerpKeeper")
			perpKeeper := &nibiruApp.PerpKeeper
			perpKeeper.PairMetadata().Set(ctx, &types.PairMetadata{
				Pair:                       tokenPair.String(),
				CumulativePremiumFractions: []sdk.Dec{sdk.OneDec()},
			})

			t.Log("establish initial position")
			nibiruApp.PerpKeeper.SetPosition(
				ctx,
				tokenPair,
				traderAddr.String(),
				&types.Position{
					Address: traderAddr.String(),
					Pair:    tokenPair.String(),
					Size_:   sdk.NewDec(9999),
					Margin:  tc.initialMargin,
				},
			)

			goCtx := sdk.WrapSDKContext(ctx)
			msg := &types.MsgAddMargin{
				Sender: traderAddr.String(), TokenPair: tokenPair.String(),
				Margin: sdk.Coin{
					Denom:  tokenPair.GetQuoteTokenDenom(),
					Amount: tc.addedMargin}}
			_, err = nibiruApp.PerpKeeper.AddMargin(goCtx, msg)
			require.NoError(t, err)

			position, err := nibiruApp.PerpKeeper.GetPosition(
				ctx, tokenPair, traderAddr.String())
			require.NoError(t, err)
			require.Equal(t, tc.expectedMargin, position.Margin)
		})
	}
}

func TestRemoveMargin(t *testing.T) {
	testCases := []struct {
		name string
		test func()
	}{
		{
			name: "negative margin remove - fail",
			test: func() {
				removeAmt := sdk.NewInt(-5)

				nibiruApp, ctx := testutil.NewNibiruApp(true)
				alice := sample.AccAddress()
				pair := common.TokenPair("osmo:nusd")
				goCtx := sdk.WrapSDKContext(ctx)
				msg := &types.MsgRemoveMargin{
					Sender: alice.String(), TokenPair: pair.String(),
					Margin: sdk.Coin{Denom: common.StableDenom, Amount: removeAmt}}
				_, err := nibiruApp.PerpKeeper.RemoveMargin(goCtx, msg)
				require.Error(t, err)
				require.ErrorContains(t, err, "margin must be positive")
			},
		},
		{
			name: "zero margin remove - fail",
			test: func() {
				removeAmt := sdk.ZeroInt()

				nibiruApp, ctx := testutil.NewNibiruApp(true)
				alice := sample.AccAddress()
				pair := common.TokenPair("osmo:nusd")
				goCtx := sdk.WrapSDKContext(ctx)
				msg := &types.MsgRemoveMargin{
					Sender: alice.String(), TokenPair: pair.String(),
					Margin: sdk.Coin{Denom: common.StableDenom, Amount: removeAmt}}
				_, err := nibiruApp.PerpKeeper.RemoveMargin(goCtx, msg)
				require.Error(t, err)
				require.ErrorContains(t, err, "margin must be positive")
			},
		},
		{
			name: "vpool doesn't exit - fail",
			test: func() {
				removeAmt := sdk.NewInt(5)

				nibiruApp, ctx := testutil.NewNibiruApp(true)
				alice := sample.AccAddress()
				pair := common.TokenPair("osmo:nusd")
				goCtx := sdk.WrapSDKContext(ctx)
				msg := &types.MsgRemoveMargin{
					Sender: alice.String(), TokenPair: pair.String(),
					Margin: sdk.Coin{Denom: common.StableDenom, Amount: removeAmt}}
				_, err := nibiruApp.PerpKeeper.RemoveMargin(goCtx, msg)
				require.Error(t, err)
				require.ErrorContains(t, err, types.ErrPairNotFound.Error())
			},
		},
		{
			name: "pool exists but trader doesn't have position - fail",
			test: func() {
				t.Log("Setup Nibiru app, pair, and trader")
				nibiruApp, ctx := testutil.NewNibiruApp(true)
				alice := sample.AccAddress()
				pair := common.TokenPair("osmo:nusd")

				t.Log("Setup vpool defined by pair")
				vpoolKeeper := &nibiruApp.VpoolKeeper
				perpKeeper := &nibiruApp.PerpKeeper
				vpoolKeeper.CreatePool(
					ctx,
					pair.String(),
					sdk.MustNewDecFromStr("0.9"), // 0.9 ratio
					/* y */ sdk.NewDec(1_000_000), //
					/* x */ sdk.NewDec(1_000_000), //
					/* fluctLim */ sdk.MustNewDecFromStr("1.0"), // 100%
				)

				removeAmt := sdk.NewInt(5)
				goCtx := sdk.WrapSDKContext(ctx)
				msg := &types.MsgRemoveMargin{
					Sender: alice.String(), TokenPair: pair.String(),
					Margin: sdk.Coin{Denom: pair.GetQuoteTokenDenom(), Amount: removeAmt}}
				_, err := perpKeeper.RemoveMargin(
					goCtx, msg)
				require.Error(t, err)
				require.ErrorContains(t, err, types.ErrPositionNotFound.Error())
			},
		},
		{
			name: "remove margin from healthy position - fast integration test 1",
			test: func() {
				t.Log("Setup Nibiru app, pair, and trader")
				nibiruApp, ctx := testutil.NewNibiruApp(true)
				alice := sample.AccAddress()
				pair := common.TokenPair("xxx:yyy")

				t.Log("Set vpool defined by pair on VpoolKeeper")
				vpoolKeeper := &nibiruApp.VpoolKeeper
				quoteReserves := sdk.NewDec(1_000_000)
				baseReserves := sdk.NewDec(1_000_000)
				vpoolKeeper.CreatePool(
					ctx,
					pair.String(),
					/* tradeLimitRatio */ sdk.MustNewDecFromStr("0.9"), // 0.9 ratio
					/* y */ quoteReserves,
					/* x */ baseReserves,
					/* fluctLim */ sdk.MustNewDecFromStr("1.0"), // 0.9 ratio
				)
				require.True(t, vpoolKeeper.ExistsPool(ctx, pair))

				t.Log("Set vpool defined by pair on PerpKeeper")
				perpKeeper := &nibiruApp.PerpKeeper
				perpKeeper.PairMetadata().Set(ctx, &types.PairMetadata{
					Pair: pair.String(),
					CumulativePremiumFractions: []sdk.Dec{
						sdk.ZeroDec(),
						sdk.MustNewDecFromStr("0.1")},
				})

				t.Log("increment block height and time for twap calculation")
				ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 1).
					WithBlockTime(time.Now().Add(time.Minute))

				t.Log("Fund trader (Alice) account with sufficient quote")
				var err error
				err = simapp.FundAccount(nibiruApp.BankKeeper, ctx, alice,
					sdk.NewCoins(
						sdk.NewInt64Coin("yyy", 66),
					))
				require.NoError(t, err)

				t.Log("Open long position with 5x leverage")
				side := types.Side_BUY
				quote := sdk.NewInt(60)
				leverage := sdk.NewDec(5)
				baseLimit := sdk.NewInt(10)
				err = nibiruApp.PerpKeeper.OpenPosition(
					ctx, pair, side, alice, quote.ToDec(), leverage, baseLimit.ToDec())
				require.NoError(t, err)

				t.Log("Position should be accessible following 'OpenPosition'")
				_, err = nibiruApp.PerpKeeper.GetPosition(ctx, pair, alice.String())
				require.NoError(t, err)

				t.Log("Verify correct events emitted for 'OpenPosition'")
				expectedEvents := []sdk.Event{
					events.NewTransferEvent(
						/* coin */ sdk.NewCoin(pair.GetQuoteTokenDenom(), quote),
						/* from */ alice.String(),
						/* to */ nibiruApp.AccountKeeper.GetModuleAddress(
							types.VaultModuleAccount).String(),
					),
					// events.NewPositionChangeEvent(), TODO
				}
				for _, event := range expectedEvents {
					assert.Contains(t, ctx.EventManager().Events(), event)
				}

				t.Log("Attempt to remove 10% of the position")
				removeAmt := sdk.NewInt(6)
				goCtx := sdk.WrapSDKContext(ctx)
				msg := &types.MsgRemoveMargin{
					Sender: alice.String(), TokenPair: pair.String(),
					Margin: sdk.Coin{Denom: pair.GetQuoteTokenDenom(), Amount: removeAmt}}

				t.Log("'RemoveMargin' from the position")
				res, err := perpKeeper.RemoveMargin(goCtx, msg)
				require.NoError(t, err)
				assert.EqualValues(t, msg.Margin, res.MarginOut)
				assert.EqualValues(t, sdk.ZeroDec(), res.FundingPayment)

				t.Log("Verify correct events emitted for 'RemoveMargin'")
				expectedEvents = []sdk.Event{
					events.NewMarginChangeEvent(
						/* owner */ alice,
						/* vpool */ msg.TokenPair,
						/* marginAmt */ msg.Margin.Amount,
						/* fundingPayment */ res.FundingPayment,
					),
					events.NewTransferEvent(
						/* coin */ msg.Margin,
						/* from */ nibiruApp.AccountKeeper.GetModuleAddress(
							types.VaultModuleAccount).String(),
						/* to */ msg.Sender,
					),
				}
				for _, event := range expectedEvents {
					assert.Contains(t, ctx.EventManager().Events(), event)
				}
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
