package keeper

import (
	"fmt"
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/perp/events"
	"github.com/NibiruChain/nibiru/x/perp/types"
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

func TestGetMarginRatio_Unit(t *testing.T) {
	tests := []struct {
		name                string
		position            types.Position
		newPrice            sdk.Dec
		expectedMarginRatio sdk.Dec
	}{
		{
			name: "margin without price changes",
			position: types.Position{
				Address:                             sample.AccAddress().String(),
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
				Address:                             sample.AccAddress().String(),
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
			k, deps, ctx := getKeeper(t)

			t.Log("Mock vpool spot price")
			deps.mockVpoolKeeper.EXPECT().
				GetBaseAssetPrice(
					ctx,
					common.TokenPair("BTC:NUSD"),
					vpooltypes.Direction_ADD_TO_POOL,
					sdk.NewDec(10),
				).
				Return(tc.newPrice, nil)
			t.Log("Mock vpool twap")
			deps.mockVpoolKeeper.EXPECT().
				GetBaseAssetTWAP(
					ctx,
					common.TokenPair("BTC:NUSD"),
					vpooltypes.Direction_ADD_TO_POOL,
					sdk.NewDec(10),
					15*time.Minute,
				).
				Return(sdk.NewDec(10), nil)

			k.PairMetadata().Set(ctx, &types.PairMetadata{
				Pair:                       "BTC:NUSD",
				CumulativePremiumFractions: []sdk.Dec{sdk.OneDec()},
			})

			marginRatio, err := k.GetMarginRatio(
				ctx, tc.position, types.MarginCalculationPriceOption_MAX_PNL)
			require.NoError(t, err)
			require.Equal(t, tc.expectedMarginRatio, marginRatio)
		})
	}
}

func TestRemoveMargin_Unit(t *testing.T) {
	tests := []struct {
		name string
		test func()
	}{
		{
			name: "fail - invalid sender",
			test: func() {
				k, _, ctx := getKeeper(t)
				goCtx := sdk.WrapSDKContext(ctx)

				invalidSender := "notABech32"
				msg := &types.MsgRemoveMargin{Sender: invalidSender}
				_, err := k.RemoveMargin(goCtx, msg)
				require.Error(t, err)
				require.ErrorContains(t, err, "decoding bech32 failed")
			},
		},
		{
			name: "fail - invalid token pair",
			test: func() {
				k, _, ctx := getKeeper(t)
				goCtx := sdk.WrapSDKContext(ctx)

				alice := sample.AccAddress()
				the3pool := "dai:usdc:usdt"
				msg := &types.MsgRemoveMargin{Sender: alice.String(),
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
				k, mocks, ctx := getKeeper(t)
				goCtx := sdk.WrapSDKContext(ctx)

				t.Log("Build msg that specifies an impossible margin removal (too high)")
				alice := sample.AccAddress()
				pair := common.TokenPair("osmo:nusd")
				msg := &types.MsgRemoveMargin{Sender: alice.String(),
					TokenPair: pair.String(),
					Margin:    sdk.NewCoin(pair.GetQuoteTokenDenom(), sdk.NewInt(600)),
				}

				mocks.mockVpoolKeeper.EXPECT().ExistsPool(ctx, pair).Return(true)

				t.Log("Set vpool defined by pair on PerpKeeper")
				k.PairMetadata().Set(ctx, &types.PairMetadata{
					Pair: pair.String(),
					CumulativePremiumFractions: []sdk.Dec{
						sdk.ZeroDec(),
						sdk.MustNewDecFromStr("0.1")},
				})

				t.Log("Set an underwater position, positive bad debt due to excessive margin request")
				k.SetPosition(ctx, pair, alice.String(), &types.Position{
					Address:                             alice.String(),
					Pair:                                pair.String(),
					Size_:                               sdk.NewDec(1_000),
					OpenNotional:                        sdk.NewDec(1000),
					Margin:                              sdk.NewDec(500),
					LastUpdateCumulativePremiumFraction: sdk.MustNewDecFromStr("0.1"),
					BlockNumber:                         ctx.BlockHeight(),
				})
				_, err := k.RemoveMargin(goCtx, msg)
				require.Error(t, err)
				require.ErrorContains(t, err, "position has bad debt")
			},
		},
		{
			name: "fail - vault doesn't have enough funds",
			test: func() {
				k, mocks, ctx := getKeeper(t)
				goCtx := sdk.WrapSDKContext(ctx)

				alice := sample.AccAddress()
				msg := &types.MsgRemoveMargin{Sender: alice.String(),
					TokenPair: "osmo:nusd",
					Margin:    sdk.NewCoin("nusd", sdk.NewInt(100)),
				}

				pair := common.TokenPair(msg.TokenPair)
				mocks.mockVpoolKeeper.EXPECT().ExistsPool(ctx, pair).
					AnyTimes().Return(true)

				t.Log("Set vpool defined by pair on PerpKeeper")
				k.PairMetadata().Set(ctx, &types.PairMetadata{
					Pair: pair.String(),
					CumulativePremiumFractions: []sdk.Dec{
						sdk.ZeroDec(),
						sdk.MustNewDecFromStr("0.1")},
				})

				t.Log("Set position a healthy position that has 0 unrealized funding")
				k.SetPosition(ctx, pair, alice.String(), &types.Position{
					Address:                             alice.String(),
					Pair:                                pair.String(),
					Size_:                               sdk.NewDec(1_000),
					OpenNotional:                        sdk.NewDec(1000),
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

				t.Log("Attempt to RemoveMargin when the vault lacks funds")
				expectedError := fmt.Errorf("not enough funds in vault module account")
				mocks.mockBankKeeper.EXPECT().SendCoinsFromModuleToAccount(
					ctx, types.VaultModuleAccount, alice, sdk.NewCoins(msg.Margin),
				).Return(expectedError)

				_, err := k.RemoveMargin(goCtx, msg)
				require.Error(t, err)
				require.ErrorContains(t, err, expectedError.Error())
			},
		},
		{
			name: "happy path - zero funding",
			test: func() {
				k, mocks, ctx := getKeeper(t)
				goCtx := sdk.WrapSDKContext(ctx)

				alice := sample.AccAddress()
				msg := &types.MsgRemoveMargin{Sender: alice.String(),
					TokenPair: "osmo:nusd",
					Margin:    sdk.NewCoin("nusd", sdk.NewInt(100)),
				}

				pair := common.TokenPair(msg.TokenPair)
				mocks.mockVpoolKeeper.EXPECT().ExistsPool(ctx, pair).
					AnyTimes().Return(true)

				t.Log("Set vpool defined by pair on PerpKeeper")
				k.PairMetadata().Set(ctx, &types.PairMetadata{
					Pair: pair.String(),
					CumulativePremiumFractions: []sdk.Dec{
						sdk.ZeroDec(),
						sdk.MustNewDecFromStr("0.1")},
				})

				t.Log("Set position a healthy position that has 0 unrealized funding")
				k.SetPosition(ctx, pair, alice.String(), &types.Position{
					Address:                             alice.String(),
					Pair:                                pair.String(),
					Size_:                               sdk.NewDec(1_000),
					OpenNotional:                        sdk.NewDec(1000),
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

				t.Log("'RemoveMargin' from the position")
				vaultAddr := authtypes.NewModuleAddress(types.VaultModuleAccount)
				mocks.mockAccountKeeper.EXPECT().
					GetModuleAddress(types.VaultModuleAccount).
					Return(vaultAddr)
				mocks.mockBankKeeper.EXPECT().SendCoinsFromModuleToAccount(
					ctx, types.VaultModuleAccount, alice, sdk.NewCoins(msg.Margin),
				).Return(nil)

				res, err := k.RemoveMargin(goCtx, msg)
				require.NoError(t, err)
				assert.EqualValues(t, msg.Margin, res.MarginOut)
				assert.EqualValues(t, sdk.ZeroDec(), res.FundingPayment)

				t.Log("Verify correct events emitted for 'RemoveMargin'")
				expectedEvents := []sdk.Event{
					events.NewMarginChangeEvent(
						/* owner */ alice,
						/* vpool */ msg.TokenPair,
						/* marginAmt */ msg.Margin.Amount,
						/* fundingPayment */ res.FundingPayment,
					),
					events.NewTransferEvent(
						/* coin */ msg.Margin,
						/* from */ vaultAddr.String(),
						/* to */ msg.Sender,
					),
				}
				for _, event := range expectedEvents {
					assert.Contains(t, ctx.EventManager().Events(), event)
				}

				pos, err := k.GetPosition(ctx, pair, alice.String())
				require.NoError(t, err)
				assert.EqualValues(t, sdk.NewDec(400).String(), pos.Margin.String())
				assert.EqualValues(t, sdk.NewDec(1000).String(), pos.Size_.String())
				assert.EqualValues(t, alice.String(), pos.Address)
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
