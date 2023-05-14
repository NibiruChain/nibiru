package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/common/denoms"
	"github.com/NibiruChain/nibiru/x/common/testutil"
	. "github.com/NibiruChain/nibiru/x/common/testutil/action"
	. "github.com/NibiruChain/nibiru/x/common/testutil/assertion"
	. "github.com/NibiruChain/nibiru/x/perp/integration/action/v2"
	. "github.com/NibiruChain/nibiru/x/perp/integration/assertion/v2"
	v2types "github.com/NibiruChain/nibiru/x/perp/types/v2"
)

// func TestMsgServerAddMargin(t *testing.T) {
// 	tests := []struct {
// 		name string

// 		traderFunds     sdk.Coins
// 		initialPosition *v2types.Position
// 		margin          sdk.Coin

// 		expectedErr error
// 	}{
// 		{
// 			name:        "trader not enough funds",
// 			traderFunds: sdk.NewCoins(sdk.NewInt64Coin(denoms.NUSD, 999)),
// 			initialPosition: &v2types.Position{
// 				Pair:                            asset.Registry.Pair(denoms.BTC, denoms.NUSD),
// 				Size_:                           sdk.OneDec(),
// 				Margin:                          sdk.OneDec(),
// 				OpenNotional:                    sdk.OneDec(),
// 				LatestCumulativePremiumFraction: sdk.ZeroDec(),
// 				LastUpdatedBlockNumber:          1,
// 			},
// 			margin:      sdk.NewInt64Coin(denoms.NUSD, 1000),
// 			expectedErr: sdkerrors.ErrInsufficientFunds,
// 		},
// 		{
// 			name:            "no initial position",
// 			traderFunds:     sdk.NewCoins(sdk.NewInt64Coin(denoms.NUSD, 1000)),
// 			initialPosition: nil,
// 			margin:          sdk.NewInt64Coin(denoms.NUSD, 1000),
// 			expectedErr:     collections.ErrNotFound,
// 		},
// 		{
// 			name:        "success",
// 			traderFunds: sdk.NewCoins(sdk.NewInt64Coin(denoms.NUSD, 1000)),
// 			initialPosition: &v2types.Position{
// 				Pair:                            asset.Registry.Pair(denoms.BTC, denoms.NUSD),
// 				Size_:                           sdk.OneDec(),
// 				Margin:                          sdk.OneDec(),
// 				OpenNotional:                    sdk.OneDec(),
// 				LatestCumulativePremiumFraction: sdk.ZeroDec(),
// 				LastUpdatedBlockNumber:          1,
// 			},
// 			margin:      sdk.NewInt64Coin(denoms.NUSD, 1000),
// 			expectedErr: nil,
// 		},
// 	}

// 	for _, tc := range tests {
// 		tc := tc
// 		t.Run(tc.name, func(t *testing.T) {
// 			app, ctx := testapp.NewNibiruTestAppAndContext(true)
// 			msgServer := keeper.NewMsgServerImpl(app.PerpKeeperV2)
// 			traderAddr := testutil.AccAddress()

// 			t.Log("create market")
// 			assert.NoError(t, app.PerpAmmKeeper.CreatePool(
// 				ctx,
// 				asset.Registry.Pair(denoms.BTC, denoms.NUSD),
// 				/* quoteReserve */ sdk.NewDec(1*common.TO_MICRO),
// 				/* baseReserve */ sdk.NewDec(1*common.TO_MICRO),
// 				v2types.MarketConfig{
// 					TradeLimitRatio:        sdk.OneDec(),
// 					FluctuationLimitRatio:  sdk.OneDec(),
// 					MaxOracleSpreadRatio:   sdk.OneDec(),
// 					MaintenanceMarginRatio: sdk.MustNewDecFromStr("0.0625"),
// 					MaxLeverage:            sdk.MustNewDecFromStr("15"),
// 				},
// 				sdk.OneDec(),
// 			))
// 			keeper.SetPairMetadata(app.PerpKeeperV2, ctx, types.PairMetadata{
// 				Pair:                            asset.Registry.Pair(denoms.BTC, denoms.NUSD),
// 				LatestCumulativePremiumFraction: sdk.ZeroDec(),
// 			})

// 			t.Log("fund trader")
// 			require.NoError(t, testapp.FundAccount(app.BankKeeper, ctx, traderAddr, tc.traderFunds))

// 			if tc.initialPosition != nil {
// 				t.Log("create position")
// 				tc.initialPosition.TraderAddress = traderAddr.String()
// 				keeper.SetPosition(app.PerpKeeperV2, ctx, *tc.initialPosition)
// 			}

// 			resp, err := msgServer.AddMargin(sdk.WrapSDKContext(ctx), &v2types.MsgAddMargin{
// 				Sender: traderAddr.String(),
// 				Pair:   asset.Registry.Pair(denoms.BTC, denoms.NUSD),
// 				Margin: tc.margin,
// 			})

// 			if tc.expectedErr != nil {
// 				require.ErrorContains(t, err, tc.expectedErr.Error())
// 				require.Nil(t, resp)
// 			} else {
// 				require.NoError(t, err)
// 				require.NotNil(t, resp)
// 				assert.EqualValues(t, resp.FundingPayment, sdk.ZeroDec())
// 				assert.EqualValues(t, tc.initialPosition.Pair, resp.Position.Pair)
// 				assert.EqualValues(t, tc.initialPosition.TraderAddress, resp.Position.TraderAddress)
// 				assert.EqualValues(t, tc.initialPosition.Margin.Add(tc.margin.Amount.ToDec()), resp.Position.Margin)
// 				assert.EqualValues(t, tc.initialPosition.OpenNotional, resp.Position.OpenNotional)
// 				assert.EqualValues(t, tc.initialPosition.Size_, resp.Position.Size_)
// 				assert.EqualValues(t, ctx.BlockHeight(), resp.Position.LastUpdatedBlockNumber)
// 				assert.EqualValues(t, sdk.ZeroDec(), resp.Position.LatestCumulativePremiumFraction)
// 			}
// 		})
// 	}
// }

// func TestMsgServerRemoveMargin(t *testing.T) {
// 	tests := []struct {
// 		name string

// 		vaultFunds      sdk.Coins
// 		initialPosition *v2types.Position
// 		marginToRemove  sdk.Coin

// 		expectedErr error
// 	}{
// 		{
// 			name:       "position not enough margin",
// 			vaultFunds: sdk.NewCoins(sdk.NewInt64Coin(denoms.NUSD, 1000)),
// 			initialPosition: &v2types.Position{
// 				Pair:                            asset.Registry.Pair(denoms.BTC, denoms.NUSD),
// 				Size_:                           sdk.OneDec(),
// 				Margin:                          sdk.OneDec(),
// 				OpenNotional:                    sdk.OneDec(),
// 				LatestCumulativePremiumFraction: sdk.ZeroDec(),
// 				LastUpdatedBlockNumber:          1,
// 			},
// 			marginToRemove: sdk.NewInt64Coin(denoms.NUSD, 1000),
// 			expectedErr:    types.ErrFailedRemoveMarginCanCauseBadDebt,
// 		},
// 		{
// 			name:            "no initial position",
// 			vaultFunds:      sdk.NewCoins(sdk.NewInt64Coin(denoms.NUSD, 0)),
// 			initialPosition: nil,
// 			marginToRemove:  sdk.NewInt64Coin(denoms.NUSD, 1000),
// 			expectedErr:     collections.ErrNotFound,
// 		},
// 		{
// 			name:       "vault insufficient funds",
// 			vaultFunds: sdk.NewCoins(sdk.NewInt64Coin(denoms.NUSD, 999)),
// 			initialPosition: &v2types.Position{
// 				Pair:                            asset.Registry.Pair(denoms.BTC, denoms.NUSD),
// 				Size_:                           sdk.OneDec(),
// 				Margin:                          sdk.NewDec(1e6),
// 				OpenNotional:                    sdk.OneDec(),
// 				LatestCumulativePremiumFraction: sdk.ZeroDec(),
// 				LastUpdatedBlockNumber:          1,
// 			},
// 			marginToRemove: sdk.NewInt64Coin(denoms.NUSD, 1000),
// 			expectedErr:    sdkerrors.ErrInsufficientFunds,
// 		},
// 		{
// 			name:       "success",
// 			vaultFunds: sdk.NewCoins(sdk.NewInt64Coin(denoms.NUSD, 1000)),
// 			initialPosition: &v2types.Position{
// 				Pair:                            asset.Registry.Pair(denoms.BTC, denoms.NUSD),
// 				Size_:                           sdk.OneDec(),
// 				Margin:                          sdk.NewDec(1e6),
// 				OpenNotional:                    sdk.OneDec(),
// 				LatestCumulativePremiumFraction: sdk.ZeroDec(),
// 				LastUpdatedBlockNumber:          1,
// 			},
// 			marginToRemove: sdk.NewInt64Coin(denoms.NUSD, 1000),
// 			expectedErr:    nil,
// 		},
// 	}

// 	for _, tc := range tests {
// 		tc := tc
// 		t.Run(tc.name, func(t *testing.T) {
// 			app, ctx := testapp.NewNibiruTestAppAndContext(true)
// 			msgServer := keeper.NewMsgServerImpl(app.PerpKeeperV2)
// 			traderAddr := testutil.AccAddress()

// 			t.Log("create market")
// 			assert.NoError(t, app.PerpAmmKeeper.CreatePool(
// 				ctx,
// 				asset.Registry.Pair(denoms.BTC, denoms.NUSD),
// 				/* quoteReserve */ sdk.NewDec(1*common.TO_MICRO),
// 				/* baseReserve */ sdk.NewDec(1*common.TO_MICRO),
// 				v2types.MarketConfig{
// 					TradeLimitRatio:        sdk.OneDec(),
// 					FluctuationLimitRatio:  sdk.OneDec(),
// 					MaxOracleSpreadRatio:   sdk.OneDec(),
// 					MaintenanceMarginRatio: sdk.MustNewDecFromStr("0.0625"),
// 					MaxLeverage:            sdk.MustNewDecFromStr("15"),
// 				},
// 				sdk.OneDec(),
// 			))
// 			keeper.SetPairMetadata(app.PerpKeeperV2, ctx, types.PairMetadata{
// 				Pair:                            asset.Registry.Pair(denoms.BTC, denoms.NUSD),
// 				LatestCumulativePremiumFraction: sdk.ZeroDec(),
// 			})

// 			t.Log("fund vault")
// 			require.NoError(t, testapp.FundModuleAccount(app.BankKeeper, ctx, v2types.VaultModuleAccount, tc.vaultFunds))

// 			if tc.initialPosition != nil {
// 				t.Log("create position")
// 				tc.initialPosition.TraderAddress = traderAddr.String()
// 				keeper.SetPosition(app.PerpKeeperV2, ctx, *tc.initialPosition)
// 			}

// 			ctx = ctx.WithBlockTime(ctx.BlockTime().Add(time.Second * 5)).WithBlockHeight(ctx.BlockHeight() + 1)

// 			resp, err := msgServer.RemoveMargin(sdk.WrapSDKContext(ctx), &v2types.MsgRemoveMargin{
// 				Sender: traderAddr.String(),
// 				Pair:   asset.Registry.Pair(denoms.BTC, denoms.NUSD),
// 				Margin: tc.marginToRemove,
// 			})

// 			if tc.expectedErr != nil {
// 				require.ErrorContains(t, err, tc.expectedErr.Error())
// 				require.Nil(t, resp)
// 			} else {
// 				require.NoError(t, err)
// 				require.NotNil(t, resp)
// 				assert.EqualValues(t, tc.marginToRemove, resp.MarginOut)
// 				assert.EqualValues(t, resp.FundingPayment, sdk.ZeroDec())
// 				assert.EqualValues(t, tc.initialPosition.Pair, resp.Position.Pair)
// 				assert.EqualValues(t, tc.initialPosition.TraderAddress, resp.Position.TraderAddress)
// 				assert.EqualValues(t, tc.initialPosition.Margin.Sub(tc.marginToRemove.Amount.ToDec()), resp.Position.Margin)
// 				assert.EqualValues(t, tc.initialPosition.OpenNotional, resp.Position.OpenNotional)
// 				assert.EqualValues(t, tc.initialPosition.Size_, resp.Position.Size_)
// 				assert.EqualValues(t, ctx.BlockHeight(), resp.Position.LastUpdatedBlockNumber)
// 				assert.EqualValues(t, sdk.ZeroDec(), resp.Position.LatestCumulativePremiumFraction)
// 			}
// 		})
// 	}
// }

// func TestMsgServerMultiLiquidate(t *testing.T) {
// 	app, ctx := testapp.NewNibiruTestAppAndContext(true)
// 	ctx = ctx.WithBlockTime(time.Now())
// 	msgServer := keeper.NewMsgServerImpl(app.PerpKeeperV2)

// 	pair := asset.Registry.Pair(denoms.BTC, denoms.NUSD)
// 	liquidator := testutil.AccAddress()

// 	atRiskTrader1 := testutil.AccAddress()
// 	notAtRiskTrader := testutil.AccAddress()
// 	atRiskTrader2 := testutil.AccAddress()

// 	t.Log("create market")
// 	assert.NoError(t, app.PerpAmmKeeper.CreatePool(
// 		/* ctx */ ctx,
// 		/* pair */ pair,
// 		/* quoteReserve */ sdk.NewDec(1*common.TO_MICRO),
// 		/* baseReserve */ sdk.NewDec(1*common.TO_MICRO),
// 		v2types.MarketConfig{
// 			TradeLimitRatio:        sdk.OneDec(),
// 			FluctuationLimitRatio:  sdk.OneDec(),
// 			MaxOracleSpreadRatio:   sdk.OneDec(),
// 			MaintenanceMarginRatio: sdk.MustNewDecFromStr("0.0625"),
// 			MaxLeverage:            sdk.MustNewDecFromStr("15"),
// 		},
// 		sdk.OneDec(),
// 	))
// 	keeper.SetPairMetadata(app.PerpKeeperV2, ctx, types.PairMetadata{
// 		Pair:                            pair,
// 		LatestCumulativePremiumFraction: sdk.ZeroDec(),
// 	})
// 	ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 1).WithBlockTime(time.Now().Add(time.Minute))

// 	t.Log("set oracle price")
// 	app.OracleKeeper.SetPrice(ctx, asset.Registry.Pair(denoms.BTC, denoms.NUSD), sdk.OneDec())

// 	t.Log("create positions")
// 	atRiskPosition1 := v2types.Position{
// 		TraderAddress:                   atRiskTrader1.String(),
// 		Pair:                            pair,
// 		Size_:                           sdk.OneDec(),
// 		Margin:                          sdk.OneDec(),
// 		OpenNotional:                    sdk.NewDec(2),
// 		LatestCumulativePremiumFraction: sdk.ZeroDec(),
// 	}
// 	atRiskPosition2 := v2types.Position{
// 		TraderAddress:                   atRiskTrader2.String(),
// 		Pair:                            pair,
// 		Size_:                           sdk.OneDec(),
// 		Margin:                          sdk.OneDec(),
// 		OpenNotional:                    sdk.NewDec(2), // new spot price is 1, so position can be liquidated
// 		LatestCumulativePremiumFraction: sdk.ZeroDec(),
// 		LastUpdatedBlockNumber:          1,
// 	}
// 	notAtRiskPosition := v2types.Position{
// 		TraderAddress:                   notAtRiskTrader.String(),
// 		Pair:                            pair,
// 		Size_:                           sdk.OneDec(),
// 		Margin:                          sdk.OneDec(),
// 		OpenNotional:                    sdk.MustNewDecFromStr("0.1"), // open price is lower than current price so no way trader gets liquidated
// 		LatestCumulativePremiumFraction: sdk.ZeroDec(),
// 		LastUpdatedBlockNumber:          1,
// 	}
// 	keeper.SetPosition(app.PerpKeeperV2, ctx, atRiskPosition1)
// 	keeper.SetPosition(app.PerpKeeperV2, ctx, notAtRiskPosition)
// 	keeper.SetPosition(app.PerpKeeperV2, ctx, atRiskPosition2)

// 	require.NoError(t, testapp.FundModuleAccount(app.BankKeeper, ctx, v2types.VaultModuleAccount, sdk.NewCoins(sdk.NewInt64Coin(pair.QuoteDenom(), 2))))

// 	setLiquidator(ctx, app.PerpKeeperV2, liquidator)
// 	resp, err := msgServer.MultiLiquidate(sdk.WrapSDKContext(ctx), &v2types.MsgMultiLiquidate{
// 		Sender: liquidator.String(),
// 		Liquidations: []*v2types.MsgMultiLiquidate_Liquidation{
// 			{
// 				Pair:   pair,
// 				Trader: atRiskTrader1.String(),
// 			},
// 			{
// 				Pair:   pair,
// 				Trader: notAtRiskTrader.String(),
// 			},
// 			{
// 				Pair:   pair,
// 				Trader: atRiskTrader2.String(),
// 			},
// 		},
// 	})
// 	require.NoError(t, err)

// 	assert.True(t, resp.Liquidations[0].Success)
// 	assert.False(t, resp.Liquidations[1].Success)
// 	assert.True(t, resp.Liquidations[2].Success)

// 	// NOTE: we don't care about checking if liquidations math is correct. This is the duty of keeper.Liquidate
// 	// what we care about is that the first and third liquidations made some modifications at state
// 	// and events levels, whilst the second (which failed) didn't.

// 	assertNotLiquidated := func(old v2types.Position) {
// 		position, err := app.PerpKeeperV2.Positions.Get(ctx, collections.Join(old.Pair, sdk.MustAccAddressFromBech32(old.TraderAddress)))
// 		require.NoError(t, err)
// 		assert.Equal(t, old, position)
// 	}

// 	assertLiquidated := func(old v2types.Position) {
// 		_, err := app.PerpKeeperV2.Positions.Get(ctx, collections.Join(old.Pair, sdk.MustAccAddressFromBech32(old.TraderAddress)))
// 		require.ErrorIs(t, err, collections.ErrNotFound)
// 		// NOTE(mercilex): does not cover partial liquidation
// 	}
// 	assertNotLiquidated(notAtRiskPosition)
// 	assertLiquidated(atRiskPosition1)
// 	assertLiquidated(atRiskPosition2)
// }

// func TestMsgServerDonateToEcosystemFund(t *testing.T) {
// 	tests := []struct {
// 		name string

// 		sender       sdk.AccAddress
// 		initialFunds sdk.Coins
// 		donation     sdk.Coin

// 		expectedErr error
// 	}{
// 		{
// 			name:         "not enough funds",
// 			sender:       testutil.AccAddress(),
// 			initialFunds: sdk.NewCoins(),
// 			donation:     sdk.NewInt64Coin(denoms.NUSD, 100),
// 			expectedErr:  fmt.Errorf("insufficient funds"),
// 		},
// 		{
// 			name:         "success",
// 			sender:       testutil.AccAddress(),
// 			initialFunds: sdk.NewCoins(sdk.NewInt64Coin(denoms.NUSD, 1e6)),
// 			donation:     sdk.NewInt64Coin(denoms.NUSD, 100),
// 			expectedErr:  nil,
// 		},
// 	}

// 	for _, tc := range tests {
// 		tc := tc
// 		t.Run(tc.name, func(t *testing.T) {
// 			app, ctx := testapp.NewNibiruTestAppAndContext(true)
// 			msgServer := keeper.NewMsgServerImpl(app.PerpKeeperV2)
// 			require.NoError(t, testapp.FundAccount(app.BankKeeper, ctx, tc.sender, tc.initialFunds))

// 			resp, err := msgServer.DonateToEcosystemFund(sdk.WrapSDKContext(ctx), &v2types.MsgDonateToEcosystemFund{
// 				Sender:   tc.sender.String(),
// 				Donation: tc.donation,
// 			})

// 			if tc.expectedErr != nil {
// 				require.ErrorContains(t, err, tc.expectedErr.Error())
// 				require.Nil(t, resp)
// 			} else {
// 				require.NoError(t, err)
// 				require.NotNil(t, resp)
// 				assert.EqualValues(t,
// 					tc.donation,
// 					app.BankKeeper.GetBalance(
// 						ctx,
// 						app.AccountKeeper.GetModuleAddress(v2types.PerpEFModuleAccount),
// 						denoms.NUSD,
// 					),
// 				)
// 			}
// 		})
// 	}
// }

func TestMsgServerOpenPosition(t *testing.T) {
	pair := asset.Registry.Pair(denoms.BTC, denoms.NUSD)
	alice := testutil.AccAddress()

	tests := TestCases{
		TC("open long position").
			Given(
				CreateCustomMarket(pair),
				FundAccount(alice, sdk.NewCoins(sdk.NewInt64Coin(denoms.NUSD, 100))),
			).
			When(
				MsgServerOpenPosition(alice, pair, v2types.Direction_LONG, sdk.NewInt(1), sdk.NewDec(1), sdk.ZeroInt()),
			).
			Then(
				PositionShouldBeEqual(alice, pair,
					Position_PositionShouldBeEqualTo(v2types.Position{
						TraderAddress:                   alice.String(),
						Pair:                            pair,
						Size_:                           sdk.MustNewDecFromStr("0.999999999999"),
						Margin:                          sdk.NewDec(1),
						OpenNotional:                    sdk.NewDec(1),
						LatestCumulativePremiumFraction: sdk.ZeroDec(),
						LastUpdatedBlockNumber:          1,
					}),
				),
				BalanceEqual(alice, denoms.NUSD, sdk.NewInt(98)),
			),

		TC("open short position").
			Given(
				CreateCustomMarket(pair),
				FundAccount(alice, sdk.NewCoins(sdk.NewInt64Coin(denoms.NUSD, 100))),
			).
			When(
				MsgServerOpenPosition(alice, pair, v2types.Direction_SHORT, sdk.NewInt(1), sdk.NewDec(1), sdk.ZeroInt()),
			).
			Then(
				PositionShouldBeEqual(alice, pair,
					Position_PositionShouldBeEqualTo(v2types.Position{
						TraderAddress:                   alice.String(),
						Pair:                            pair,
						Size_:                           sdk.MustNewDecFromStr("-1.000000000001"),
						Margin:                          sdk.NewDec(1),
						OpenNotional:                    sdk.NewDec(1),
						LatestCumulativePremiumFraction: sdk.ZeroDec(),
						LastUpdatedBlockNumber:          1,
					}),
				),
				BalanceEqual(alice, denoms.NUSD, sdk.NewInt(98)),
			),
	}

	NewTestSuite(t).WithTestCases(tests...).Run()
}

func TestMsgServerClosePosition(t *testing.T) {
	pair := asset.Registry.Pair(denoms.BTC, denoms.NUSD)
	alice := testutil.AccAddress()

	tests := TestCases{
		TC("close long position").
			Given(
				CreateCustomMarket(pair),
				FundAccount(alice, sdk.NewCoins(sdk.NewInt64Coin(denoms.NUSD, 100))),
				OpenPosition(alice, pair, v2types.Direction_LONG, sdk.NewInt(1), sdk.NewDec(1), sdk.ZeroDec()),
				MoveToNextBlock(),
			).
			When(
				MsgServerClosePosition(alice, pair),
			).
			Then(
				PositionShouldNotExist(alice, pair),
				BalanceEqual(alice, denoms.NUSD, sdk.NewInt(99)),
			),

		TC("close short position").
			Given(
				CreateCustomMarket(pair),
				FundAccount(alice, sdk.NewCoins(sdk.NewInt64Coin(denoms.NUSD, 100))),
				OpenPosition(alice, pair, v2types.Direction_LONG, sdk.NewInt(1), sdk.NewDec(1), sdk.ZeroDec()),
				MoveToNextBlock(),
			).
			When(
				MsgServerClosePosition(alice, pair),
			).
			Then(
				PositionShouldNotExist(alice, pair),
				BalanceEqual(alice, denoms.NUSD, sdk.NewInt(99)),
			),
	}

	NewTestSuite(t).WithTestCases(tests...).Run()
}

func TestMsgServerAddMargin(t *testing.T) {
	pair := asset.Registry.Pair(denoms.BTC, denoms.NUSD)
	alice := testutil.AccAddress()

	tests := TestCases{
		TC("add margin").
			Given(
				CreateCustomMarket(pair),
				FundAccount(alice, sdk.NewCoins(sdk.NewInt64Coin(denoms.NUSD, 100))),
				OpenPosition(alice, pair, v2types.Direction_LONG, sdk.NewInt(1), sdk.NewDec(1), sdk.ZeroDec()),
				MoveToNextBlock(),
			).
			When(
				MsgServerAddMargin(alice, pair, sdk.NewInt(1)),
			).
			Then(
				PositionShouldBeEqual(alice, pair,
					Position_PositionShouldBeEqualTo(v2types.Position{
						TraderAddress:                   alice.String(),
						Pair:                            pair,
						Size_:                           sdk.MustNewDecFromStr("0.999999999999"),
						Margin:                          sdk.NewDec(2),
						OpenNotional:                    sdk.NewDec(1),
						LatestCumulativePremiumFraction: sdk.ZeroDec(),
						LastUpdatedBlockNumber:          2,
					}),
				),
				BalanceEqual(alice, denoms.NUSD, sdk.NewInt(97)),
			),
	}

	NewTestSuite(t).WithTestCases(tests...).Run()
}

func TestMsgServerRemoveMargin(t *testing.T) {
	pair := asset.Registry.Pair(denoms.BTC, denoms.NUSD)
	alice := testutil.AccAddress()

	tests := TestCases{
		TC("add margin").
			Given(
				CreateCustomMarket(pair),
				FundAccount(alice, sdk.NewCoins(sdk.NewInt64Coin(denoms.NUSD, 100))),
				OpenPosition(alice, pair, v2types.Direction_LONG, sdk.NewInt(2), sdk.NewDec(1), sdk.ZeroDec()),
				MoveToNextBlock(),
			).
			When(
				MsgServerRemoveMargin(alice, pair, sdk.NewInt(1)),
			).
			Then(
				PositionShouldBeEqual(alice, pair,
					Position_PositionShouldBeEqualTo(v2types.Position{
						TraderAddress:                   alice.String(),
						Pair:                            pair,
						Size_:                           sdk.MustNewDecFromStr("1.999999999996"),
						Margin:                          sdk.NewDec(1),
						OpenNotional:                    sdk.NewDec(2),
						LatestCumulativePremiumFraction: sdk.ZeroDec(),
						LastUpdatedBlockNumber:          2,
					}),
				),
				BalanceEqual(alice, denoms.NUSD, sdk.NewInt(98)),
			),
	}

	NewTestSuite(t).WithTestCases(tests...).Run()
}
