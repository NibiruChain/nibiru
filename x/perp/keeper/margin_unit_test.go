package keeper

import (
	"fmt"
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/perp/types"
	testutilevents "github.com/NibiruChain/nibiru/x/testutil/events"
	"github.com/NibiruChain/nibiru/x/testutil/sample"
	vpooltypes "github.com/NibiruChain/nibiru/x/vpool/types"
)

func Test_requireMoreMarginRatio(t *testing.T) {
	type test struct {
		marginRatio, baseMarginRatio sdk.Dec
		largerThanEqualTo            bool
		wantErr                      bool
	}

	cases := map[string]test{
		"ok - largeThanOrEqualTo true": {
			marginRatio:       sdk.NewDec(2),
			baseMarginRatio:   sdk.NewDec(1),
			largerThanEqualTo: true,
			wantErr:           false,
		},
		"ok - largerThanOrEqualTo false": {
			marginRatio:       sdk.NewDec(1),
			baseMarginRatio:   sdk.NewDec(2),
			largerThanEqualTo: false,
			wantErr:           false,
		},
		"fails - largerThanEqualTo true": {
			marginRatio:       sdk.NewDec(1),
			baseMarginRatio:   sdk.NewDec(2),
			largerThanEqualTo: true,
			wantErr:           true,
		},
		"fails - largerThanEqualTo false": {
			marginRatio:       sdk.NewDec(2),
			baseMarginRatio:   sdk.NewDec(1),
			largerThanEqualTo: false,
			wantErr:           true,
		},
	}

	for name, tc := range cases {
		tc := tc
		t.Run(name, func(t *testing.T) {
			err := requireMoreMarginRatio(tc.marginRatio, tc.baseMarginRatio, tc.largerThanEqualTo)
			switch {
			case tc.wantErr:
				if err == nil {
					t.Fatalf("expected error")
				}
			case !tc.wantErr:
				if err != nil {
					t.Fatalf("unexpected error: %s", err)
				}
			}
		})
	}
}

func TestGetMarginRatio_Errors(t *testing.T) {
	tests := []struct {
		name string
		test func()
	}{
		{
			name: "empty size position",
			test: func() {
				k, _, ctx := getKeeper(t)

				pos := types.Position{
					Size_: sdk.ZeroDec(),
				}

				_, err := k.GetMarginRatio(
					ctx, pos, types.MarginCalculationPriceOption_MAX_PNL)
				assert.EqualError(t, err, types.ErrPositionZero.Error())
			},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			tc.test()
		})
	}
}

func TestGetMarginRatio(t *testing.T) {
	tests := []struct {
		name                string
		position            types.Position
		newPrice            sdk.Dec
		expectedMarginRatio sdk.Dec
	}{
		{
			name: "margin without price changes",
			position: types.Position{
				TraderAddress:                       sample.AccAddress().String(),
				Pair:                                "BTC:NUSD",
				Size_:                               sdk.NewDec(10),
				OpenNotional:                        sdk.NewDec(10),
				Margin:                              sdk.NewDec(1),
				LastUpdateCumulativePremiumFraction: sdk.OneDec(),
			},
			newPrice:            sdk.MustNewDecFromStr("10"),
			expectedMarginRatio: sdk.MustNewDecFromStr("0.1"),
		},
		{
			name: "margin with price changes",
			position: types.Position{
				TraderAddress:                       sample.AccAddress().String(),
				Pair:                                "BTC:NUSD",
				Size_:                               sdk.NewDec(10),
				OpenNotional:                        sdk.NewDec(10),
				Margin:                              sdk.NewDec(1),
				LastUpdateCumulativePremiumFraction: sdk.OneDec(),
			},
			newPrice:            sdk.MustNewDecFromStr("12"),
			expectedMarginRatio: sdk.MustNewDecFromStr("0.25"),
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			perpKeeper, mocks, ctx := getKeeper(t)

			t.Log("Mock vpool spot price")
			mocks.mockVpoolKeeper.EXPECT().
				GetBaseAssetPrice(
					ctx,
					BtcNusdPair,
					vpooltypes.Direction_ADD_TO_POOL,
					tc.position.Size_.Abs(),
				).
				Return(tc.newPrice, nil)
			t.Log("Mock vpool twap")
			mocks.mockVpoolKeeper.EXPECT().
				GetBaseAssetTWAP(
					ctx,
					BtcNusdPair,
					vpooltypes.Direction_ADD_TO_POOL,
					tc.position.Size_.Abs(),
					15*time.Minute,
				).
				Return(tc.newPrice, nil)

			perpKeeper.PairMetadata().Set(ctx, &types.PairMetadata{
				Pair:                       "BTC:NUSD",
				CumulativePremiumFractions: []sdk.Dec{sdk.OneDec()},
			})

			marginRatio, err := perpKeeper.GetMarginRatio(
				ctx, tc.position, types.MarginCalculationPriceOption_MAX_PNL)

			require.NoError(t, err)
			require.Equal(t, tc.expectedMarginRatio, marginRatio)
		})
	}
}

func TestRemoveMargin(t *testing.T) {
	tests := []struct {
		name string
		test func()
	}{
		{
			name: "fail - invalid sender",
			test: func() {
				k, _, ctx := getKeeper(t)
				goCtx := sdk.WrapSDKContext(ctx)

				msg := &types.MsgRemoveMargin{Sender: ""}
				_, err := k.RemoveMargin(goCtx, msg)
				require.Error(t, err)
			},
		},
		{
			name: "fail - invalid token pair",
			test: func() {
				k, _, ctx := getKeeper(t)
				goCtx := sdk.WrapSDKContext(ctx)

				trader := sample.AccAddress()
				the3pool := "dai:usdc:usdt"
				msg := &types.MsgRemoveMargin{
					Sender:    trader.String(),
					TokenPair: the3pool,
					Margin:    sdk.NewCoin(common.StableDenom, sdk.NewInt(5))}
				_, err := k.RemoveMargin(goCtx, msg)
				require.Error(t, err)
				require.ErrorContains(t, err, common.ErrInvalidTokenPair.Error())
			},
		},
		{
			name: "fail - request is too large",
			test: func() {
				perpKeeper, mocks, ctx := getKeeper(t)
				goCtx := sdk.WrapSDKContext(ctx)

				t.Log("Build msg that specifies an impossible margin removal (too high)")
				trader := sample.AccAddress()
				pair := common.AssetPair{
					Token0: "osmo",
					Token1: "nusd",
				}
				msg := &types.MsgRemoveMargin{
					Sender:    trader.String(),
					TokenPair: pair.String(),
					Margin:    sdk.NewCoin(pair.GetQuoteTokenDenom(), sdk.NewInt(600)),
				}

				mocks.mockVpoolKeeper.EXPECT().ExistsPool(ctx, pair).Return(true)

				t.Log("Set vpool defined by pair on PerpKeeper")
				perpKeeper.PairMetadata().Set(ctx, &types.PairMetadata{
					Pair: pair.String(),
					CumulativePremiumFractions: []sdk.Dec{
						sdk.ZeroDec(),
						sdk.MustNewDecFromStr("0.1")},
				})

				t.Log("Set an underwater position, positive bad debt due to excessive margin request")
				perpKeeper.SetPosition(ctx, pair, trader, &types.Position{
					TraderAddress:                       trader.String(),
					Pair:                                pair.String(),
					Size_:                               sdk.NewDec(1_000),
					OpenNotional:                        sdk.NewDec(1000),
					Margin:                              sdk.NewDec(500),
					LastUpdateCumulativePremiumFraction: sdk.MustNewDecFromStr("0.1"),
					BlockNumber:                         ctx.BlockHeight(),
				})

				_, err := perpKeeper.RemoveMargin(goCtx, msg)

				require.Error(t, err)
				require.ErrorContains(t, err, "position has bad debt")
			},
		},
		{
			name: "fail - vault doesn't have enough funds",
			test: func() {
				perpKeeper, mocks, ctx := getKeeper(t)
				goCtx := sdk.WrapSDKContext(ctx)

				traderAddr := sample.AccAddress()
				msg := &types.MsgRemoveMargin{
					Sender:    traderAddr.String(),
					TokenPair: "osmo:nusd",
					Margin:    sdk.NewCoin("nusd", sdk.NewInt(100)),
				}

				pair, err := common.NewAssetPairFromStr(msg.TokenPair)
				require.NoError(t, err)
				mocks.mockVpoolKeeper.EXPECT().ExistsPool(ctx, pair).
					AnyTimes().Return(true)

				t.Log("Set vpool defined by pair on PerpKeeper")
				perpKeeper.PairMetadata().Set(ctx, &types.PairMetadata{
					Pair: pair.String(),
					CumulativePremiumFractions: []sdk.Dec{
						sdk.ZeroDec(),
						sdk.MustNewDecFromStr("0.1")},
				})

				t.Log("Set position a healthy position that has 0 unrealized funding")
				perpKeeper.SetPosition(ctx, pair, traderAddr, &types.Position{
					TraderAddress:                       traderAddr.String(),
					Pair:                                pair.String(),
					Size_:                               sdk.NewDec(1_000),
					OpenNotional:                        sdk.NewDec(1_000),
					Margin:                              sdk.NewDec(500),
					LastUpdateCumulativePremiumFraction: sdk.MustNewDecFromStr("0.1"),
					BlockNumber:                         ctx.BlockHeight(),
				})

				mocks.mockVpoolKeeper.EXPECT().GetBaseAssetPrice(
					ctx,
					pair,
					vpooltypes.Direction_ADD_TO_POOL,
					sdk.NewDec(1_000),
				).Return(sdk.NewDec(100), nil)
				mocks.mockVpoolKeeper.EXPECT().GetBaseAssetTWAP(
					ctx,
					pair,
					vpooltypes.Direction_ADD_TO_POOL,
					sdk.NewDec(1_000),
					15*time.Minute,
				).Return(sdk.NewDec(100), nil)

				t.Log("Attempt to RemoveMargin when the vault lacks funds")
				expectedError := fmt.Errorf("not enough funds in vault module account")
				mocks.mockBankKeeper.EXPECT().SendCoinsFromModuleToAccount(
					ctx, types.VaultModuleAccount, traderAddr, sdk.NewCoins(msg.Margin),
				).Return(expectedError)

				_, err = perpKeeper.RemoveMargin(goCtx, msg)

				require.Error(t, err)
				require.ErrorContains(t, err, expectedError.Error())
			},
		},
		{
			name: "happy path - zero funding",
			test: func() {
				perpKeeper, mocks, ctx := getKeeper(t)
				goCtx := sdk.WrapSDKContext(ctx)

				traderAddr := sample.AccAddress()
				msg := &types.MsgRemoveMargin{
					Sender:    traderAddr.String(),
					TokenPair: "osmo:nusd",
					Margin:    sdk.NewCoin("nusd", sdk.NewInt(100)),
				}

				pair, err := common.NewAssetPairFromStr(msg.TokenPair)
				require.NoError(t, err)
				mocks.mockVpoolKeeper.EXPECT().ExistsPool(ctx, pair).
					AnyTimes().Return(true)

				t.Log("Set vpool defined by pair on PerpKeeper")
				perpKeeper.PairMetadata().Set(ctx, &types.PairMetadata{
					Pair: pair.String(),
					CumulativePremiumFractions: []sdk.Dec{
						sdk.ZeroDec(),
						sdk.MustNewDecFromStr("0.1")},
				})

				t.Log("Set position a healthy position that has 0 unrealized funding")
				perpKeeper.SetPosition(ctx, pair, traderAddr, &types.Position{
					TraderAddress:                       traderAddr.String(),
					Pair:                                pair.String(),
					Size_:                               sdk.NewDec(1_000),
					OpenNotional:                        sdk.NewDec(1_000),
					Margin:                              sdk.NewDec(500),
					LastUpdateCumulativePremiumFraction: sdk.MustNewDecFromStr("0.1"),
					BlockNumber:                         ctx.BlockHeight(),
				})

				mocks.mockVpoolKeeper.EXPECT().GetBaseAssetPrice(
					ctx, pair, vpooltypes.Direction_ADD_TO_POOL, sdk.NewDec(1_000)).
					Return(sdk.NewDec(100), nil)
				mocks.mockVpoolKeeper.EXPECT().GetBaseAssetTWAP(
					ctx, pair, vpooltypes.Direction_ADD_TO_POOL, sdk.NewDec(1_000),
					15*time.Minute,
				).Return(sdk.NewDec(100), nil)

				mocks.mockBankKeeper.EXPECT().SendCoinsFromModuleToAccount(
					ctx, types.VaultModuleAccount, traderAddr, sdk.NewCoins(msg.Margin),
				).Return(nil)

				t.Log("'RemoveMargin' from the position")
				res, err := perpKeeper.RemoveMargin(goCtx, msg)

				require.NoError(t, err)
				assert.EqualValues(t, msg.Margin, res.MarginOut)
				assert.EqualValues(t, sdk.ZeroDec(), res.FundingPayment)

				t.Log("Verify correct events emitted for 'RemoveMargin'")
				testutilevents.RequireHasTypedEvent(t, ctx, &types.MarginChangedEvent{
					Pair:           msg.TokenPair,
					TraderAddress:  traderAddr,
					MarginAmount:   msg.Margin.Amount,
					FundingPayment: res.FundingPayment,
				})

				pos, err := perpKeeper.GetPosition(ctx, pair, traderAddr)
				require.NoError(t, err)
				assert.EqualValues(t, sdk.NewDec(400).String(), pos.Margin.String())
				assert.EqualValues(t, sdk.NewDec(1000).String(), pos.Size_.String())
				assert.EqualValues(t, traderAddr.String(), pos.TraderAddress)
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

func TestAddMargin(t *testing.T) {
	tests := []struct {
		name string
		test func()
	}{
		{
			name: "fail - invalid sender",
			test: func() {
				k, _, ctx := getKeeper(t)
				goCtx := sdk.WrapSDKContext(ctx)

				msg := &types.MsgAddMargin{Sender: ""}
				_, err := k.AddMargin(goCtx, msg)
				require.Error(t, err)
			},
		},
		{
			name: "fail - invalid token pair",
			test: func() {
				k, _, ctx := getKeeper(t)
				goCtx := sdk.WrapSDKContext(ctx)

				trader := sample.AccAddress()
				the3pool := "dai:usdc:usdt"
				msg := &types.MsgAddMargin{
					Sender:    trader.String(),
					TokenPair: the3pool,
					Margin:    sdk.NewInt64Coin(common.StableDenom, 5),
				}
				_, err := k.AddMargin(goCtx, msg)
				require.ErrorContains(t, err, common.ErrInvalidTokenPair.Error())
			},
		},
		{
			name: "fail - user doesn't have enough funds",
			test: func() {
				perpKeeper, mocks, ctx := getKeeper(t)
				goCtx := sdk.WrapSDKContext(ctx)

				t.Log("build msg")
				traderAddr := sample.AccAddress()
				assetPair := common.AssetPair{
					Token0: "osmo",
					Token1: "nusd",
				}
				msg := &types.MsgAddMargin{
					Sender:    traderAddr.String(),
					TokenPair: assetPair.String(),
					Margin:    sdk.NewInt64Coin(assetPair.GetQuoteTokenDenom(), 600),
				}

				mocks.mockVpoolKeeper.EXPECT().ExistsPool(ctx, assetPair).Return(true)

				t.Log("set a position")
				perpKeeper.SetPosition(ctx, assetPair, traderAddr, &types.Position{
					TraderAddress:                       traderAddr.String(),
					Pair:                                assetPair.String(),
					Size_:                               sdk.NewDec(1_000),
					OpenNotional:                        sdk.NewDec(1_000),
					Margin:                              sdk.NewDec(500),
					LastUpdateCumulativePremiumFraction: sdk.ZeroDec(),
					BlockNumber:                         ctx.BlockHeight(),
				})

				t.Log("mock bankkeeper not enough funds")
				expectedError := fmt.Errorf("not enough funds in vault module account")
				mocks.mockBankKeeper.EXPECT().SendCoinsFromAccountToModule(
					ctx, traderAddr, types.VaultModuleAccount, sdk.NewCoins(msg.Margin),
				).Return(expectedError)

				_, err := perpKeeper.AddMargin(goCtx, msg)

				require.Error(t, err)
				require.ErrorContains(t, err, expectedError.Error())
			},
		},
		{
			name: "happy path - zero funding",
			test: func() {
				perpKeeper, mocks, ctx := getKeeper(t)
				goCtx := sdk.WrapSDKContext(ctx)

				assetPair, err := common.NewAssetPairFromStr("uosmo:unusd")
				require.NoError(t, err)

				traderAddr := sample.AccAddress()
				msg := &types.MsgAddMargin{
					Sender:    traderAddr.String(),
					TokenPair: assetPair.String(),
					Margin:    sdk.NewCoin("unusd", sdk.NewInt(100)),
				}

				mocks.mockVpoolKeeper.EXPECT().ExistsPool(ctx, assetPair).
					AnyTimes().Return(true)

				t.Log("set pair metadata")
				perpKeeper.PairMetadata().Set(ctx, &types.PairMetadata{
					Pair: assetPair.String(),
					CumulativePremiumFractions: []sdk.Dec{
						sdk.ZeroDec(),
					},
				})

				t.Log("set position")
				perpKeeper.SetPosition(ctx, assetPair, traderAddr, &types.Position{
					TraderAddress:                       traderAddr.String(),
					Pair:                                assetPair.String(),
					Size_:                               sdk.NewDec(1_000),
					OpenNotional:                        sdk.NewDec(1_000),
					Margin:                              sdk.NewDec(500),
					LastUpdateCumulativePremiumFraction: sdk.ZeroDec(),
					BlockNumber:                         ctx.BlockHeight(),
				})

				mocks.mockBankKeeper.EXPECT().SendCoinsFromAccountToModule(
					ctx, traderAddr, types.VaultModuleAccount, sdk.NewCoins(msg.Margin),
				).Return(nil)

				t.Log("execute AddMargin")
				_, err = perpKeeper.AddMargin(goCtx, msg)

				require.NoError(t, err)

				t.Log("Verify correct events emitted")
				testutilevents.RequireHasTypedEvent(t, ctx, &types.MarginChangedEvent{
					Pair:           msg.TokenPair,
					TraderAddress:  traderAddr,
					MarginAmount:   msg.Margin.Amount,
					FundingPayment: sdk.ZeroDec(),
				})

				pos, err := perpKeeper.GetPosition(ctx, assetPair, traderAddr)
				require.NoError(t, err)

				assert.EqualValues(t, sdk.NewDec(600).String(), pos.Margin.String())
				assert.EqualValues(t, sdk.NewDec(1000).String(), pos.Size_.String())
				assert.EqualValues(t, traderAddr.String(), pos.TraderAddress)
			},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			tc.test()
		})
	}
}
