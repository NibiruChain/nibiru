package keeper

import (
	"fmt"
	"math"
	"testing"
	"time"

	"github.com/NibiruChain/collections"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/common/denoms"
	testutilevents "github.com/NibiruChain/nibiru/x/common/testutil"
	vpooltypes "github.com/NibiruChain/nibiru/x/perp/amm/types"
	"github.com/NibiruChain/nibiru/x/perp/types"
)

func TestRequireMoreMarginRatio(t *testing.T) {
	type test struct {
		marginRatio, threshold sdk.Dec
		largerThanEqualTo      bool
		wantErr                bool
	}

	cases := map[string]test{
		"ok - largeThanOrEqualTo true": {
			marginRatio:       sdk.NewDec(2),
			threshold:         sdk.NewDec(1),
			largerThanEqualTo: true,
			wantErr:           false,
		},
		"ok - largerThanOrEqualTo false": {
			marginRatio:       sdk.NewDec(1),
			threshold:         sdk.NewDec(2),
			largerThanEqualTo: false,
			wantErr:           false,
		},
		"fails - largerThanEqualTo true": {
			marginRatio:       sdk.NewDec(1),
			threshold:         sdk.NewDec(2),
			largerThanEqualTo: true,
			wantErr:           true,
		},
		"fails - largerThanEqualTo false": {
			marginRatio:       sdk.NewDec(2),
			threshold:         sdk.NewDec(1),
			largerThanEqualTo: false,
			wantErr:           true,
		},
	}

	for name, tc := range cases {
		tc := tc
		t.Run(name, func(t *testing.T) {
			err := validateMarginRatio(tc.marginRatio, tc.threshold, tc.largerThanEqualTo)
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

				vpool := vpooltypes.Vpool{Pair: asset.Registry.Pair(denoms.BTC, denoms.NUSD)}
				_, err := k.GetMarginRatio(
					ctx, vpool, pos, types.MarginCalculationPriceOption_MAX_PNL)
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
				TraderAddress:                   testutilevents.AccAddress().String(),
				Pair:                            asset.Registry.Pair(denoms.BTC, denoms.NUSD),
				Size_:                           sdk.NewDec(10),
				OpenNotional:                    sdk.NewDec(10),
				Margin:                          sdk.NewDec(1),
				LatestCumulativePremiumFraction: sdk.OneDec(),
			},
			newPrice:            sdk.MustNewDecFromStr("10"),
			expectedMarginRatio: sdk.MustNewDecFromStr("0.1"),
		},
		{
			name: "margin with price changes",
			position: types.Position{
				TraderAddress:                   testutilevents.AccAddress().String(),
				Pair:                            asset.Registry.Pair(denoms.BTC, denoms.NUSD),
				Size_:                           sdk.NewDec(10),
				OpenNotional:                    sdk.NewDec(10),
				Margin:                          sdk.NewDec(1),
				LatestCumulativePremiumFraction: sdk.OneDec(),
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
			vpool := vpooltypes.Vpool{Pair: asset.Registry.Pair(denoms.BTC, denoms.NUSD)}
			mocks.mockVpoolKeeper.EXPECT().
				GetBaseAssetPrice(
					vpool,
					vpooltypes.Direction_ADD_TO_POOL,
					tc.position.Size_.Abs(),
				).
				Return(tc.newPrice, nil)
			t.Log("Mock vpool twap")
			mocks.mockVpoolKeeper.EXPECT().
				GetBaseAssetTWAP(
					ctx,
					asset.Registry.Pair(denoms.BTC, denoms.NUSD),
					vpooltypes.Direction_ADD_TO_POOL,
					tc.position.Size_.Abs(),
					15*time.Minute,
				).
				Return(tc.newPrice, nil)

			SetPairMetadata(perpKeeper, ctx, types.PairMetadata{
				Pair:                            asset.Registry.Pair(denoms.BTC, denoms.NUSD),
				LatestCumulativePremiumFraction: sdk.OneDec(),
			})

			marginRatio, err := perpKeeper.GetMarginRatio(
				ctx, vpool, tc.position, types.MarginCalculationPriceOption_MAX_PNL,
			)

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
			name: "fail - request is too large",
			test: func() {
				perpKeeper, mocks, ctx := getKeeper(t)

				t.Log("Build msg that specifies an impossible margin removal (too high)")
				traderAddr := testutilevents.AccAddress()
				pair := asset.NewPair("osmo", "nusd")

				mocks.mockVpoolKeeper.EXPECT().GetPool(ctx, pair).Return(vpooltypes.Vpool{Pair: pair}, nil)

				t.Log("Set vpool defined by pair on PerpKeeper")
				SetPairMetadata(perpKeeper, ctx, types.PairMetadata{
					Pair:                            pair,
					LatestCumulativePremiumFraction: sdk.MustNewDecFromStr("0.1"),
				})

				t.Log("Set an underwater position, positive bad debt due to excessive margin request")
				SetPosition(perpKeeper, ctx, types.Position{
					TraderAddress:                   traderAddr.String(),
					Pair:                            pair,
					Size_:                           sdk.NewDec(1_000),
					OpenNotional:                    sdk.NewDec(1000),
					Margin:                          sdk.NewDec(500),
					LatestCumulativePremiumFraction: sdk.MustNewDecFromStr("0.1"),
					BlockNumber:                     ctx.BlockHeight(),
				})

				_, _, _, err := perpKeeper.RemoveMargin(ctx, pair, traderAddr, sdk.NewCoin(pair.QuoteDenom(), sdk.NewInt(600)))

				require.Error(t, err)
				require.ErrorContains(t, err, types.ErrFailedRemoveMarginCanCauseBadDebt.Error())
			},
		},
		{
			name: "fail - vault doesn't have enough funds",
			test: func() {
				perpKeeper, mocks, ctx := getKeeper(t)

				traderAddr := testutilevents.AccAddress()
				pair := asset.MustNewPair("osmo:nusd")
				marginToWithdraw := sdk.NewInt64Coin(pair.QuoteDenom(), 100)

				t.Log("mock vpool keeper")
				vpool := vpooltypes.Vpool{Pair: asset.Registry.Pair(denoms.BTC, denoms.NUSD)}
				mocks.mockVpoolKeeper.EXPECT().GetPool(ctx, pair).Return(vpool, nil)
				mocks.mockVpoolKeeper.EXPECT().GetMaintenanceMarginRatio(ctx, pair).
					Return(sdk.MustNewDecFromStr("0.0625"), nil)
				mocks.mockVpoolKeeper.EXPECT().GetMarkPrice(ctx, pair).Return(sdk.OneDec(), nil)
				mocks.mockVpoolKeeper.EXPECT().GetBaseAssetPrice(
					vpool,
					vpooltypes.Direction_ADD_TO_POOL,
					sdk.NewDec(1_000),
				).Return(sdk.NewDec(1000), nil).Times(2)
				mocks.mockVpoolKeeper.EXPECT().GetBaseAssetTWAP(
					ctx,
					pair,
					vpooltypes.Direction_ADD_TO_POOL,
					sdk.NewDec(1_000),
					15*time.Minute,
				).Return(sdk.NewDec(1000), nil)

				t.Log("mock account keeper")
				mocks.mockAccountKeeper.
					EXPECT().GetModuleAddress(types.VaultModuleAccount).
					Return(authtypes.NewModuleAddress(types.VaultModuleAccount))

				t.Log("mock bank keeper")
				expectedError := fmt.Errorf("not enough funds in vault module account")
				mocks.mockBankKeeper.EXPECT().SendCoinsFromModuleToModule(
					ctx, types.PerpEFModuleAccount, types.VaultModuleAccount, sdk.NewCoins(marginToWithdraw),
				).Return(expectedError)
				mocks.mockBankKeeper.EXPECT().GetBalance(
					ctx,
					authtypes.NewModuleAddress(types.VaultModuleAccount),
					pair.QuoteDenom(),
				).Return(sdk.NewInt64Coin(pair.QuoteDenom(), 0))

				t.Log("set pair metadata")
				SetPairMetadata(perpKeeper, ctx, types.PairMetadata{
					Pair:                            pair,
					LatestCumulativePremiumFraction: sdk.ZeroDec(),
				})

				t.Log("Set position a healthy position that has 0 unrealized funding")
				SetPosition(perpKeeper, ctx, types.Position{
					TraderAddress:                   traderAddr.String(),
					Pair:                            pair,
					Size_:                           sdk.NewDec(1_000),
					OpenNotional:                    sdk.NewDec(1_000),
					Margin:                          sdk.NewDec(500),
					LatestCumulativePremiumFraction: sdk.ZeroDec(),
					BlockNumber:                     ctx.BlockHeight(),
				})

				t.Log("Attempt to RemoveMargin when the vault lacks funds")
				_, _, _, err := perpKeeper.RemoveMargin(ctx, pair, traderAddr, marginToWithdraw)

				require.Error(t, err)
				require.ErrorContains(t, err, expectedError.Error())
			},
		},
		{
			name: "happy path - zero funding",
			test: func() {
				perpKeeper, mocks, ctx := getKeeper(t)

				traderAddr := testutilevents.AccAddress()
				pair := asset.MustNewPair("osmo:nusd")
				marginToWithdraw := sdk.NewInt64Coin(pair.QuoteDenom(), 100)

				t.Log("mock vpool keeper")
				vpool := vpooltypes.Vpool{Pair: asset.Registry.Pair(denoms.BTC, denoms.NUSD)}
				mocks.mockVpoolKeeper.EXPECT().GetPool(ctx, pair).Return(vpool, nil)
				mocks.mockVpoolKeeper.EXPECT().GetMaintenanceMarginRatio(ctx, pair).
					Return(sdk.MustNewDecFromStr("0.0625"), nil)

				mocks.mockVpoolKeeper.EXPECT().GetMarkPrice(ctx, pair).Return(sdk.OneDec(), nil)
				mocks.mockVpoolKeeper.EXPECT().GetBaseAssetPrice(
					vpool, vpooltypes.Direction_ADD_TO_POOL, sdk.NewDec(1_000)).
					Return(sdk.NewDec(1000), nil).Times(2)
				mocks.mockVpoolKeeper.EXPECT().GetBaseAssetTWAP(
					ctx, pair, vpooltypes.Direction_ADD_TO_POOL, sdk.NewDec(1_000),
					15*time.Minute,
				).Return(sdk.NewDec(1000), nil)

				t.Log("mock account keeper")
				mocks.mockAccountKeeper.
					EXPECT().GetModuleAddress(types.VaultModuleAccount).
					Return(authtypes.NewModuleAddress(types.VaultModuleAccount))

				t.Log("mock bank keeper")
				mocks.mockBankKeeper.
					EXPECT().GetBalance(ctx, authtypes.NewModuleAddress(types.VaultModuleAccount), pair.QuoteDenom()).
					Return(sdk.NewCoin(pair.QuoteDenom(), sdk.NewInt(math.MaxInt64)))
				mocks.mockBankKeeper.EXPECT().SendCoinsFromModuleToAccount(
					ctx, types.VaultModuleAccount, traderAddr, sdk.NewCoins(marginToWithdraw),
				).Return(nil)

				t.Log("set pair metadata")
				SetPairMetadata(perpKeeper, ctx, types.PairMetadata{
					Pair:                            pair,
					LatestCumulativePremiumFraction: sdk.ZeroDec(),
				})

				t.Log("Set position a healthy position that has 0 unrealized funding")
				SetPosition(perpKeeper, ctx, types.Position{
					TraderAddress:                   traderAddr.String(),
					Pair:                            pair,
					Size_:                           sdk.NewDec(1_000),
					OpenNotional:                    sdk.NewDec(1_000),
					Margin:                          sdk.NewDec(500),
					LatestCumulativePremiumFraction: sdk.ZeroDec(),
					BlockNumber:                     ctx.BlockHeight(),
				})

				t.Log("'RemoveMargin' from the position")
				marginOut, fundingPayment, position, err := perpKeeper.RemoveMargin(ctx, pair, traderAddr, marginToWithdraw)

				require.NoError(t, err)
				assert.EqualValues(t, marginToWithdraw, marginOut)
				assert.EqualValues(t, sdk.ZeroDec(), fundingPayment)
				assert.EqualValues(t, pair, position.Pair)
				assert.EqualValues(t, traderAddr.String(), position.TraderAddress)
				assert.EqualValues(t, sdk.NewDec(400), position.Margin)
				assert.EqualValues(t, sdk.NewDec(1000), position.OpenNotional)
				assert.EqualValues(t, sdk.NewDec(1000), position.Size_)
				assert.EqualValues(t, ctx.BlockHeight(), ctx.BlockHeight())
				assert.EqualValues(t, sdk.ZeroDec(), position.LatestCumulativePremiumFraction)

				t.Log("Verify correct events emitted for 'RemoveMargin'")
				testutilevents.RequireHasTypedEvent(t, ctx,
					&types.PositionChangedEvent{
						Pair:               pair,
						TraderAddress:      traderAddr.String(),
						Margin:             sdk.NewInt64Coin(pair.QuoteDenom(), 400),
						PositionNotional:   sdk.NewDec(1000),
						ExchangedNotional:  sdk.ZeroDec(),                                 // always zero when removing margin
						ExchangedSize:      sdk.ZeroDec(),                                 // always zero when removing margin
						TransactionFee:     sdk.NewCoin(pair.QuoteDenom(), sdk.ZeroInt()), // always zero when removing margin
						PositionSize:       sdk.NewDec(1000),
						RealizedPnl:        sdk.ZeroDec(), // always zero when removing margin
						UnrealizedPnlAfter: sdk.ZeroDec(),
						BadDebt:            sdk.NewCoin(pair.QuoteDenom(), sdk.ZeroInt()), // always zero when adding margin
						FundingPayment:     sdk.ZeroDec(),
						MarkPrice:          sdk.OneDec(),
						BlockHeight:        ctx.BlockHeight(),
						BlockTimeMs:        ctx.BlockTime().UnixMilli(),
					},
				)

				pos, err := perpKeeper.Positions.Get(ctx, collections.Join(pair, traderAddr))
				require.NoError(t, err)
				assert.EqualValues(t, sdk.NewDec(400).String(), pos.Margin.String())
				assert.EqualValues(t, sdk.NewDec(1000).String(), pos.Size_.String())
				assert.EqualValues(t, traderAddr.String(), pos.TraderAddress)
			},
		},
		{
			name: "happy path - massive funding payment",
			test: func() {
				perpKeeper, mocks, ctx := getKeeper(t)

				traderAddr := testutilevents.AccAddress()
				pair := asset.MustNewPair("osmo:nusd")
				marginToWithdraw := sdk.NewInt64Coin(pair.QuoteDenom(), 100)

				t.Log("mock vpool keeper")
				vpool := vpooltypes.Vpool{Pair: pair}
				mocks.mockVpoolKeeper.EXPECT().GetPool(ctx, pair).Return(vpool, nil)

				t.Log("set pair metadata")
				SetPairMetadata(perpKeeper, ctx, types.PairMetadata{
					Pair:                            pair,
					LatestCumulativePremiumFraction: sdk.OneDec(),
				})

				t.Log("Set position a healthy position that has 0 unrealized funding")
				SetPosition(perpKeeper, ctx, types.Position{
					TraderAddress:                   traderAddr.String(),
					Pair:                            pair,
					Size_:                           sdk.NewDec(500),
					OpenNotional:                    sdk.NewDec(500),
					Margin:                          sdk.NewDec(500),
					LatestCumulativePremiumFraction: sdk.ZeroDec(),
					BlockNumber:                     ctx.BlockHeight(),
				})

				t.Log("'RemoveMargin' from the position")
				_, _, _, err := perpKeeper.RemoveMargin(ctx, pair, traderAddr, marginToWithdraw)

				require.ErrorIs(t, err, types.ErrFailedRemoveMarginCanCauseBadDebt)

				pos, err := perpKeeper.Positions.Get(ctx, collections.Join(pair, traderAddr))
				require.NoError(t, err)
				assert.EqualValues(t, sdk.NewDec(500).String(), pos.Margin.String())
				assert.EqualValues(t, sdk.NewDec(500).String(), pos.Size_.String())
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
			name: "fail - user doesn't have enough funds",
			test: func() {
				perpKeeper, mocks, ctx := getKeeper(t)

				traderAddr := testutilevents.AccAddress()
				pair := asset.NewPair("uosmo", "unusd")
				margin := sdk.NewInt64Coin(pair.QuoteDenom(), 600)

				t.Log("set pair metadata")
				SetPairMetadata(perpKeeper, ctx, types.PairMetadata{
					Pair:                            pair,
					LatestCumulativePremiumFraction: sdk.ZeroDec(),
				})
				mocks.mockVpoolKeeper.EXPECT().GetPool(ctx, pair).Return(vpooltypes.Vpool{Pair: pair}, nil)

				t.Log("set a position")
				SetPosition(perpKeeper, ctx, types.Position{
					TraderAddress:                   traderAddr.String(),
					Pair:                            pair,
					Size_:                           sdk.NewDec(1_000),
					OpenNotional:                    sdk.NewDec(1_000),
					Margin:                          sdk.NewDec(500),
					LatestCumulativePremiumFraction: sdk.ZeroDec(),
					BlockNumber:                     ctx.BlockHeight(),
				})

				t.Log("mock bankkeeper not enough funds")
				expectedError := fmt.Errorf("not enough funds in vault module account")
				mocks.mockBankKeeper.EXPECT().SendCoinsFromAccountToModule(
					ctx, traderAddr, types.VaultModuleAccount, sdk.NewCoins(margin),
				).Return(expectedError)

				resp, err := perpKeeper.AddMargin(ctx, pair, traderAddr, margin)

				require.ErrorContains(t, err, expectedError.Error())
				require.Nil(t, resp)
			},
		},
		{
			name: "happy path - zero funding",
			test: func() {
				perpKeeper, mocks, ctx := getKeeper(t)
				pair := asset.MustNewPair("uosmo:unusd")
				traderAddr := testutilevents.AccAddress()
				margin := sdk.NewInt64Coin("unusd", 100)

				vpool := vpooltypes.Vpool{Pair: asset.Registry.Pair(denoms.BTC, denoms.NUSD)}
				mocks.mockVpoolKeeper.EXPECT().GetPool(ctx, pair).Return(vpool, nil)
				mocks.mockVpoolKeeper.EXPECT().GetBaseAssetPrice(vpool, vpooltypes.Direction_ADD_TO_POOL, sdk.NewDec(1000)).Return(sdk.NewDec(1000), nil)
				mocks.mockVpoolKeeper.EXPECT().GetMarkPrice(ctx, pair).Return(sdk.OneDec(), nil)

				t.Log("set pair metadata")
				SetPairMetadata(perpKeeper, ctx, types.PairMetadata{
					Pair:                            pair,
					LatestCumulativePremiumFraction: sdk.ZeroDec(),
				})

				t.Log("set position")
				SetPosition(perpKeeper, ctx, types.Position{
					TraderAddress:                   traderAddr.String(),
					Pair:                            pair,
					Size_:                           sdk.NewDec(1_000),
					OpenNotional:                    sdk.NewDec(1_000),
					Margin:                          sdk.NewDec(500),
					LatestCumulativePremiumFraction: sdk.ZeroDec(),
					BlockNumber:                     1,
				})

				t.Log("mock bankKeeper")
				mocks.mockBankKeeper.EXPECT().SendCoinsFromAccountToModule(
					ctx, traderAddr, types.VaultModuleAccount, sdk.NewCoins(margin),
				).Return(nil)

				t.Log("execute AddMargin")
				resp, err := perpKeeper.AddMargin(ctx, pair, traderAddr, margin)
				require.NoError(t, err)

				t.Log("assert correct response")
				assert.EqualValues(t, sdk.ZeroDec(), resp.FundingPayment)
				assert.EqualValues(t, sdk.NewDec(600), resp.Position.Margin)
				assert.EqualValues(t, sdk.NewDec(1_000), resp.Position.OpenNotional)
				assert.EqualValues(t, sdk.NewDec(1_000), resp.Position.Size_)
				assert.EqualValues(t, traderAddr.String(), resp.Position.TraderAddress)
				assert.EqualValues(t, pair, resp.Position.Pair)
				assert.EqualValues(t, sdk.ZeroDec(), resp.Position.LatestCumulativePremiumFraction)
				assert.EqualValues(t, ctx.BlockHeight(), resp.Position.BlockNumber)

				t.Log("Verify correct events emitted")
				testutilevents.RequireHasTypedEvent(t, ctx,
					&types.PositionChangedEvent{
						Pair:               pair,
						TraderAddress:      traderAddr.String(),
						Margin:             sdk.NewInt64Coin(pair.QuoteDenom(), 600),
						PositionNotional:   sdk.NewDec(1000),
						ExchangedNotional:  sdk.ZeroDec(),                                 // always zero when adding margin
						ExchangedSize:      sdk.ZeroDec(),                                 // always zero when adding margin
						TransactionFee:     sdk.NewCoin(pair.QuoteDenom(), sdk.ZeroInt()), // always zero when adding margin
						PositionSize:       sdk.NewDec(1000),
						RealizedPnl:        sdk.ZeroDec(), // always zero when adding margin
						UnrealizedPnlAfter: sdk.ZeroDec(),
						BadDebt:            sdk.NewCoin(pair.QuoteDenom(), sdk.ZeroInt()), // always zero when adding margin
						FundingPayment:     sdk.ZeroDec(),
						MarkPrice:          sdk.OneDec(),
						BlockHeight:        ctx.BlockHeight(),
						BlockTimeMs:        ctx.BlockTime().UnixMilli(),
					},
				)
			},
		},
		{
			name: "happy path - with funding payment",
			test: func() {
				perpKeeper, mocks, ctx := getKeeper(t)

				pair := asset.MustNewPair("uosmo:unusd")
				traderAddr := testutilevents.AccAddress()
				margin := sdk.NewInt64Coin("unusd", 100)

				vpool := vpooltypes.Vpool{Pair: asset.Registry.Pair(denoms.BTC, denoms.NUSD)}
				mocks.mockVpoolKeeper.EXPECT().GetPool(ctx, pair).Return(vpool, nil)
				mocks.mockVpoolKeeper.EXPECT().GetBaseAssetPrice(vpool, vpooltypes.Direction_ADD_TO_POOL, sdk.NewDec(1000)).Return(sdk.NewDec(1000), nil)
				mocks.mockVpoolKeeper.EXPECT().GetMarkPrice(ctx, pair).Return(sdk.OneDec(), nil)

				t.Log("set pair metadata")
				SetPairMetadata(perpKeeper, ctx, types.PairMetadata{
					Pair:                            pair,
					LatestCumulativePremiumFraction: sdk.MustNewDecFromStr("0.001"),
				})

				t.Log("set position")
				SetPosition(perpKeeper, ctx, types.Position{
					TraderAddress:                   traderAddr.String(),
					Pair:                            pair,
					Size_:                           sdk.NewDec(1_000),
					OpenNotional:                    sdk.NewDec(1_000),
					Margin:                          sdk.NewDec(500),
					LatestCumulativePremiumFraction: sdk.ZeroDec(),
					BlockNumber:                     1,
				})

				mocks.mockBankKeeper.EXPECT().SendCoinsFromAccountToModule(
					ctx, traderAddr, types.VaultModuleAccount, sdk.NewCoins(margin),
				).Return(nil)

				t.Log("execute AddMargin")
				resp, err := perpKeeper.AddMargin(ctx, pair, traderAddr, margin)
				require.NoError(t, err)

				t.Log("assert correct response")
				assert.EqualValues(t, sdk.NewDec(1), resp.FundingPayment)
				assert.EqualValues(t, sdk.NewDec(599), resp.Position.Margin)
				assert.EqualValues(t, sdk.NewDec(1_000), resp.Position.OpenNotional)
				assert.EqualValues(t, sdk.NewDec(1_000), resp.Position.Size_)
				assert.EqualValues(t, traderAddr.String(), resp.Position.TraderAddress)
				assert.EqualValues(t, pair, resp.Position.Pair)
				assert.EqualValues(t, sdk.MustNewDecFromStr("0.001"), resp.Position.LatestCumulativePremiumFraction)
				assert.EqualValues(t, ctx.BlockHeight(), resp.Position.BlockNumber)

				t.Log("Verify correct events emitted")
				testutilevents.RequireHasTypedEvent(t, ctx,
					&types.PositionChangedEvent{
						Pair:               pair,
						TraderAddress:      traderAddr.String(),
						Margin:             sdk.NewInt64Coin(pair.QuoteDenom(), 599),
						PositionNotional:   sdk.NewDec(1000),
						ExchangedNotional:  sdk.ZeroDec(),                                 // always zero when adding margin
						ExchangedSize:      sdk.ZeroDec(),                                 // always zero when adding margin
						TransactionFee:     sdk.NewCoin(pair.QuoteDenom(), sdk.ZeroInt()), // always zero when adding margin
						PositionSize:       sdk.NewDec(1000),
						RealizedPnl:        sdk.ZeroDec(), // always zero when adding margin
						UnrealizedPnlAfter: sdk.ZeroDec(),
						BadDebt:            sdk.NewCoin(pair.QuoteDenom(), sdk.ZeroInt()), // always zero when adding margin
						FundingPayment:     sdk.OneDec(),
						MarkPrice:          sdk.OneDec(),
						BlockHeight:        ctx.BlockHeight(),
						BlockTimeMs:        ctx.BlockTime().UnixMilli(),
					},
				)
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

func TestGetPositionNotionalAndUnrealizedPnl(t *testing.T) {
	tests := []struct {
		name                       string
		initialPosition            types.Position
		setMocks                   func(ctx sdk.Context, mocks mockedDependencies)
		pnlCalcOption              types.PnLCalcOption
		expectedPositionalNotional sdk.Dec
		expectedUnrealizedPnL      sdk.Dec
	}{
		{
			name: "long position; positive pnl; spot price calc",
			initialPosition: types.Position{
				TraderAddress: testutilevents.AccAddress().String(),
				Pair:          asset.Registry.Pair(denoms.BTC, denoms.NUSD),
				Size_:         sdk.NewDec(10),
				OpenNotional:  sdk.NewDec(10),
				Margin:        sdk.NewDec(1),
			},
			setMocks: func(ctx sdk.Context, mocks mockedDependencies) {
				vpool := vpooltypes.Vpool{Pair: asset.Registry.Pair(denoms.BTC, denoms.NUSD)}
				mocks.mockVpoolKeeper.EXPECT().
					GetBaseAssetPrice(
						vpool,
						vpooltypes.Direction_ADD_TO_POOL,
						sdk.NewDec(10),
					).
					Return(sdk.NewDec(20), nil)
			},
			pnlCalcOption:              types.PnLCalcOption_SPOT_PRICE,
			expectedPositionalNotional: sdk.NewDec(20),
			expectedUnrealizedPnL:      sdk.NewDec(10),
		},
		{
			name: "long position; negative pnl; spot price calc",
			initialPosition: types.Position{
				TraderAddress: testutilevents.AccAddress().String(),
				Pair:          asset.Registry.Pair(denoms.BTC, denoms.NUSD),
				Size_:         sdk.NewDec(10),
				OpenNotional:  sdk.NewDec(10),
				Margin:        sdk.NewDec(1),
			},
			setMocks: func(ctx sdk.Context, mocks mockedDependencies) {
				vpool := vpooltypes.Vpool{Pair: asset.Registry.Pair(denoms.BTC, denoms.NUSD)}
				mocks.mockVpoolKeeper.EXPECT().
					GetBaseAssetPrice(
						vpool,
						vpooltypes.Direction_ADD_TO_POOL,
						sdk.NewDec(10),
					).
					Return(sdk.NewDec(5), nil)
			},
			pnlCalcOption:              types.PnLCalcOption_SPOT_PRICE,
			expectedPositionalNotional: sdk.NewDec(5),
			expectedUnrealizedPnL:      sdk.NewDec(-5),
		},
		{
			name: "long position; positive pnl; twap calc",
			initialPosition: types.Position{
				TraderAddress: testutilevents.AccAddress().String(),
				Pair:          asset.Registry.Pair(denoms.BTC, denoms.NUSD),
				Size_:         sdk.NewDec(10),
				OpenNotional:  sdk.NewDec(10),
				Margin:        sdk.NewDec(1),
			},
			setMocks: func(ctx sdk.Context, mocks mockedDependencies) {
				mocks.mockVpoolKeeper.EXPECT().
					GetBaseAssetTWAP(
						ctx,
						asset.Registry.Pair(denoms.BTC, denoms.NUSD),
						vpooltypes.Direction_ADD_TO_POOL,
						sdk.NewDec(10),
						15*time.Minute,
					).
					Return(sdk.NewDec(20), nil)
			},
			pnlCalcOption:              types.PnLCalcOption_TWAP,
			expectedPositionalNotional: sdk.NewDec(20),
			expectedUnrealizedPnL:      sdk.NewDec(10),
		},
		{
			name: "long position; negative pnl; twap calc",
			initialPosition: types.Position{
				TraderAddress: testutilevents.AccAddress().String(),
				Pair:          asset.Registry.Pair(denoms.BTC, denoms.NUSD),
				Size_:         sdk.NewDec(10),
				OpenNotional:  sdk.NewDec(10),
				Margin:        sdk.NewDec(1),
			},
			setMocks: func(ctx sdk.Context, mocks mockedDependencies) {
				mocks.mockVpoolKeeper.EXPECT().
					GetBaseAssetTWAP(
						ctx,
						asset.Registry.Pair(denoms.BTC, denoms.NUSD),
						vpooltypes.Direction_ADD_TO_POOL,
						sdk.NewDec(10),
						15*time.Minute,
					).
					Return(sdk.NewDec(5), nil)
			},
			pnlCalcOption:              types.PnLCalcOption_TWAP,
			expectedPositionalNotional: sdk.NewDec(5),
			expectedUnrealizedPnL:      sdk.NewDec(-5),
		},
		{
			name: "long position; positive pnl; oracle calc",
			initialPosition: types.Position{
				TraderAddress: testutilevents.AccAddress().String(),
				Pair:          asset.Registry.Pair(denoms.BTC, denoms.NUSD),
				Size_:         sdk.NewDec(10),
				OpenNotional:  sdk.NewDec(10),
				Margin:        sdk.NewDec(1),
			},
			setMocks: func(ctx sdk.Context, mocks mockedDependencies) {
				mocks.mockOracleKeeper.EXPECT().
					GetExchangeRate(
						ctx,
						asset.Registry.Pair(denoms.BTC, denoms.NUSD),
					).
					Return(sdk.NewDec(2), nil)
			},
			pnlCalcOption:              types.PnLCalcOption_ORACLE,
			expectedPositionalNotional: sdk.NewDec(20),
			expectedUnrealizedPnL:      sdk.NewDec(10),
		},
		{
			name: "long position; negative pnl; oracle calc",
			initialPosition: types.Position{
				TraderAddress: testutilevents.AccAddress().String(),
				Pair:          asset.Registry.Pair(denoms.BTC, denoms.NUSD),
				Size_:         sdk.NewDec(10),
				OpenNotional:  sdk.NewDec(10),
				Margin:        sdk.NewDec(1),
			},
			setMocks: func(ctx sdk.Context, mocks mockedDependencies) {
				mocks.mockOracleKeeper.EXPECT().
					GetExchangeRate(
						ctx,
						asset.Registry.Pair(denoms.BTC, denoms.NUSD),
					).
					Return(sdk.MustNewDecFromStr("0.5"), nil)
			},
			pnlCalcOption:              types.PnLCalcOption_ORACLE,
			expectedPositionalNotional: sdk.NewDec(5),
			expectedUnrealizedPnL:      sdk.NewDec(-5),
		},
		{
			name: "short position; positive pnl; spot price calc",
			initialPosition: types.Position{
				TraderAddress: testutilevents.AccAddress().String(),
				Pair:          asset.Registry.Pair(denoms.BTC, denoms.NUSD),
				Size_:         sdk.NewDec(-10),
				OpenNotional:  sdk.NewDec(10),
				Margin:        sdk.NewDec(1),
			},
			setMocks: func(ctx sdk.Context, mocks mockedDependencies) {
				vpool := vpooltypes.Vpool{Pair: asset.Registry.Pair(denoms.BTC, denoms.NUSD)}
				mocks.mockVpoolKeeper.EXPECT().
					GetBaseAssetPrice(
						vpool,
						vpooltypes.Direction_REMOVE_FROM_POOL,
						sdk.NewDec(10),
					).
					Return(sdk.NewDec(5), nil)
			},
			pnlCalcOption:              types.PnLCalcOption_SPOT_PRICE,
			expectedPositionalNotional: sdk.NewDec(5),
			expectedUnrealizedPnL:      sdk.NewDec(5),
		},
		{
			name: "short position; negative pnl; spot price calc",
			initialPosition: types.Position{
				TraderAddress: testutilevents.AccAddress().String(),
				Pair:          asset.Registry.Pair(denoms.BTC, denoms.NUSD),
				Size_:         sdk.NewDec(-10),
				OpenNotional:  sdk.NewDec(10),
				Margin:        sdk.NewDec(1),
			},
			setMocks: func(ctx sdk.Context, mocks mockedDependencies) {
				vpool := vpooltypes.Vpool{Pair: asset.Registry.Pair(denoms.BTC, denoms.NUSD)}
				mocks.mockVpoolKeeper.EXPECT().
					GetBaseAssetPrice(
						vpool,
						vpooltypes.Direction_REMOVE_FROM_POOL,
						sdk.NewDec(10),
					).
					Return(sdk.NewDec(20), nil)
			},
			pnlCalcOption:              types.PnLCalcOption_SPOT_PRICE,
			expectedPositionalNotional: sdk.NewDec(20),
			expectedUnrealizedPnL:      sdk.NewDec(-10),
		},
		{
			name: "short position; positive pnl; twap calc",
			initialPosition: types.Position{
				TraderAddress: testutilevents.AccAddress().String(),
				Pair:          asset.Registry.Pair(denoms.BTC, denoms.NUSD),
				Size_:         sdk.NewDec(-10),
				OpenNotional:  sdk.NewDec(10),
				Margin:        sdk.NewDec(1),
			},
			setMocks: func(ctx sdk.Context, mocks mockedDependencies) {
				mocks.mockVpoolKeeper.EXPECT().
					GetBaseAssetTWAP(
						ctx,
						asset.Registry.Pair(denoms.BTC, denoms.NUSD),
						vpooltypes.Direction_REMOVE_FROM_POOL,
						sdk.NewDec(10),
						15*time.Minute,
					).
					Return(sdk.NewDec(5), nil)
			},
			pnlCalcOption:              types.PnLCalcOption_TWAP,
			expectedPositionalNotional: sdk.NewDec(5),
			expectedUnrealizedPnL:      sdk.NewDec(5),
		},
		{
			name: "short position; negative pnl; twap calc",
			initialPosition: types.Position{
				TraderAddress: testutilevents.AccAddress().String(),
				Pair:          asset.Registry.Pair(denoms.BTC, denoms.NUSD),
				Size_:         sdk.NewDec(-10),
				OpenNotional:  sdk.NewDec(10),
				Margin:        sdk.NewDec(1),
			},
			setMocks: func(ctx sdk.Context, mocks mockedDependencies) {
				mocks.mockVpoolKeeper.EXPECT().
					GetBaseAssetTWAP(
						ctx,
						asset.Registry.Pair(denoms.BTC, denoms.NUSD),
						vpooltypes.Direction_REMOVE_FROM_POOL,
						sdk.NewDec(10),
						15*time.Minute,
					).
					Return(sdk.NewDec(20), nil)
			},
			pnlCalcOption:              types.PnLCalcOption_TWAP,
			expectedPositionalNotional: sdk.NewDec(20),
			expectedUnrealizedPnL:      sdk.NewDec(-10),
		},
		{
			name: "short position; positive pnl; oracle calc",
			initialPosition: types.Position{
				TraderAddress: testutilevents.AccAddress().String(),
				Pair:          asset.Registry.Pair(denoms.BTC, denoms.NUSD),
				Size_:         sdk.NewDec(-10),
				OpenNotional:  sdk.NewDec(10),
				Margin:        sdk.NewDec(1),
			},
			setMocks: func(ctx sdk.Context, mocks mockedDependencies) {
				mocks.mockOracleKeeper.EXPECT().
					GetExchangeRate(
						ctx,
						asset.Registry.Pair(denoms.BTC, denoms.NUSD),
					).
					Return(sdk.MustNewDecFromStr("0.5"), nil)
			},
			pnlCalcOption:              types.PnLCalcOption_ORACLE,
			expectedPositionalNotional: sdk.NewDec(5),
			expectedUnrealizedPnL:      sdk.NewDec(5),
		},
		{
			name: "long position; negative pnl; oracle calc",
			initialPosition: types.Position{
				TraderAddress: testutilevents.AccAddress().String(),
				Pair:          asset.Registry.Pair(denoms.BTC, denoms.NUSD),
				Size_:         sdk.NewDec(-10),
				OpenNotional:  sdk.NewDec(10),
				Margin:        sdk.NewDec(1),
			},
			setMocks: func(ctx sdk.Context, mocks mockedDependencies) {
				mocks.mockOracleKeeper.EXPECT().
					GetExchangeRate(
						ctx,
						asset.Registry.Pair(denoms.BTC, denoms.NUSD),
					).
					Return(sdk.NewDec(2), nil)
			},
			pnlCalcOption:              types.PnLCalcOption_ORACLE,
			expectedPositionalNotional: sdk.NewDec(20),
			expectedUnrealizedPnL:      sdk.NewDec(-10),
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			perpKeeper, mocks, ctx := getKeeper(t)

			tc.setMocks(ctx, mocks)

			vpool := vpooltypes.Vpool{Pair: asset.Registry.Pair(denoms.BTC, denoms.NUSD)}
			positionalNotional, unrealizedPnl, err := perpKeeper.
				getPositionNotionalAndUnrealizedPnL(
					ctx,
					vpool,
					tc.initialPosition,
					tc.pnlCalcOption,
				)
			require.NoError(t, err)

			assert.EqualValues(t, tc.expectedPositionalNotional, positionalNotional)
			assert.EqualValues(t, tc.expectedUnrealizedPnL, unrealizedPnl)
		})
	}
}

func TestGetPreferencePositionNotionalAndUnrealizedPnL(t *testing.T) {
	// all tests are assumed long positions with positive pnl for ease of calculation
	// short positions and negative pnl are implicitly correct because of
	// TestGetPositionNotionalAndUnrealizedPnl
	testcases := []struct {
		name                       string
		initPosition               types.Position
		setMocks                   func(ctx sdk.Context, mocks mockedDependencies)
		pnlPreferenceOption        types.PnLPreferenceOption
		expectedPositionalNotional sdk.Dec
		expectedUnrealizedPnl      sdk.Dec
	}{
		{
			name: "max pnl, pick spot price",
			initPosition: types.Position{
				TraderAddress: testutilevents.AccAddress().String(),
				Pair:          asset.Registry.Pair(denoms.BTC, denoms.NUSD),
				Size_:         sdk.NewDec(10),
				OpenNotional:  sdk.NewDec(10),
				Margin:        sdk.NewDec(1),
			},
			setMocks: func(ctx sdk.Context, mocks mockedDependencies) {
				t.Log("Mock vpool spot price")
				vpool := vpooltypes.Vpool{Pair: asset.Registry.Pair(denoms.BTC, denoms.NUSD)}
				mocks.mockVpoolKeeper.EXPECT().
					GetBaseAssetPrice(
						vpool,
						vpooltypes.Direction_ADD_TO_POOL,
						sdk.NewDec(10),
					).
					Return(sdk.NewDec(20), nil)
				t.Log("Mock vpool twap")
				mocks.mockVpoolKeeper.EXPECT().
					GetBaseAssetTWAP(
						ctx,
						asset.Registry.Pair(denoms.BTC, denoms.NUSD),
						vpooltypes.Direction_ADD_TO_POOL,
						sdk.NewDec(10),
						15*time.Minute,
					).
					Return(sdk.NewDec(15), nil)
			},
			pnlPreferenceOption:        types.PnLPreferenceOption_MAX,
			expectedPositionalNotional: sdk.NewDec(20),
			expectedUnrealizedPnl:      sdk.NewDec(10),
		},
		{
			name: "max pnl, pick twap",
			initPosition: types.Position{
				TraderAddress: testutilevents.AccAddress().String(),
				Pair:          asset.Registry.Pair(denoms.BTC, denoms.NUSD),
				Size_:         sdk.NewDec(10),
				OpenNotional:  sdk.NewDec(10),
				Margin:        sdk.NewDec(1),
			},
			setMocks: func(ctx sdk.Context, mocks mockedDependencies) {
				t.Log("Mock vpool spot price")
				vpool := vpooltypes.Vpool{Pair: asset.Registry.Pair(denoms.BTC, denoms.NUSD)}
				mocks.mockVpoolKeeper.EXPECT().
					GetBaseAssetPrice(
						vpool,
						vpooltypes.Direction_ADD_TO_POOL,
						sdk.NewDec(10),
					).
					Return(sdk.NewDec(20), nil)
				t.Log("Mock vpool twap")
				mocks.mockVpoolKeeper.EXPECT().
					GetBaseAssetTWAP(
						ctx,
						asset.Registry.Pair(denoms.BTC, denoms.NUSD),
						vpooltypes.Direction_ADD_TO_POOL,
						sdk.NewDec(10),
						15*time.Minute,
					).
					Return(sdk.NewDec(30), nil)
			},
			pnlPreferenceOption:        types.PnLPreferenceOption_MAX,
			expectedPositionalNotional: sdk.NewDec(30),
			expectedUnrealizedPnl:      sdk.NewDec(20),
		},
		{
			name: "min pnl, pick spot price",
			initPosition: types.Position{
				TraderAddress: testutilevents.AccAddress().String(),
				Pair:          asset.Registry.Pair(denoms.BTC, denoms.NUSD),
				Size_:         sdk.NewDec(10),
				OpenNotional:  sdk.NewDec(10),
				Margin:        sdk.NewDec(1),
			},
			setMocks: func(ctx sdk.Context, mocks mockedDependencies) {
				t.Log("Mock vpool spot price")
				vpool := vpooltypes.Vpool{Pair: asset.Registry.Pair(denoms.BTC, denoms.NUSD)}
				mocks.mockVpoolKeeper.EXPECT().
					GetBaseAssetPrice(
						vpool,
						vpooltypes.Direction_ADD_TO_POOL,
						sdk.NewDec(10),
					).
					Return(sdk.NewDec(20), nil)
				t.Log("Mock vpool twap")
				mocks.mockVpoolKeeper.EXPECT().
					GetBaseAssetTWAP(
						ctx,
						asset.Registry.Pair(denoms.BTC, denoms.NUSD),
						vpooltypes.Direction_ADD_TO_POOL,
						sdk.NewDec(10),
						15*time.Minute,
					).
					Return(sdk.NewDec(30), nil)
			},
			pnlPreferenceOption:        types.PnLPreferenceOption_MIN,
			expectedPositionalNotional: sdk.NewDec(20),
			expectedUnrealizedPnl:      sdk.NewDec(10),
		},
		{
			name: "min pnl, pick twap",
			initPosition: types.Position{
				TraderAddress: testutilevents.AccAddress().String(),
				Pair:          asset.Registry.Pair(denoms.BTC, denoms.NUSD),
				Size_:         sdk.NewDec(10),
				OpenNotional:  sdk.NewDec(10),
				Margin:        sdk.NewDec(1),
			},
			setMocks: func(ctx sdk.Context, mocks mockedDependencies) {
				t.Log("Mock vpool spot price")
				vpool := vpooltypes.Vpool{Pair: asset.Registry.Pair(denoms.BTC, denoms.NUSD)}
				mocks.mockVpoolKeeper.EXPECT().
					GetBaseAssetPrice(
						vpool,
						vpooltypes.Direction_ADD_TO_POOL,
						sdk.NewDec(10),
					).
					Return(sdk.NewDec(20), nil)
				t.Log("Mock vpool twap")
				mocks.mockVpoolKeeper.EXPECT().
					GetBaseAssetTWAP(
						ctx,
						asset.Registry.Pair(denoms.BTC, denoms.NUSD),
						vpooltypes.Direction_ADD_TO_POOL,
						sdk.NewDec(10),
						15*time.Minute,
					).
					Return(sdk.NewDec(15), nil)
			},
			pnlPreferenceOption:        types.PnLPreferenceOption_MIN,
			expectedPositionalNotional: sdk.NewDec(15),
			expectedUnrealizedPnl:      sdk.NewDec(5),
		},
	}

	for _, tc := range testcases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			perpKeeper, mocks, ctx := getKeeper(t)

			tc.setMocks(ctx, mocks)

			vpool := vpooltypes.Vpool{Pair: asset.Registry.Pair(denoms.BTC, denoms.NUSD)}
			positionalNotional, unrealizedPnl, err := perpKeeper.
				GetPreferencePositionNotionalAndUnrealizedPnL(
					ctx,
					vpool,
					tc.initPosition,
					tc.pnlPreferenceOption,
				)

			require.NoError(t, err)
			assert.EqualValues(t, tc.expectedPositionalNotional, positionalNotional)
			assert.EqualValues(t, tc.expectedUnrealizedPnl, unrealizedPnl)
		})
	}
}
