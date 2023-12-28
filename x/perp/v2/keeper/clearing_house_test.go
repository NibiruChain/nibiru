package keeper_test

import (
	"testing"
	"time"

	sdkmath "cosmossdk.io/math"
	"github.com/NibiruChain/collections"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/common/denoms"
	"github.com/NibiruChain/nibiru/x/common/testutil"
	. "github.com/NibiruChain/nibiru/x/common/testutil/action"
	. "github.com/NibiruChain/nibiru/x/common/testutil/assertion"
	"github.com/NibiruChain/nibiru/x/common/testutil/mock"
	"github.com/NibiruChain/nibiru/x/common/testutil/testapp"
	. "github.com/NibiruChain/nibiru/x/perp/v2/integration/action"
	. "github.com/NibiruChain/nibiru/x/perp/v2/integration/assertion"

	types "github.com/NibiruChain/nibiru/x/perp/v2/types"
)

func TestMarketOrder(t *testing.T) {
	alice := testutil.AccAddress()
	bob := testutil.AccAddress()
	pairBtcNusd := asset.Registry.Pair(denoms.BTC, denoms.NUSD)
	startBlockTime := time.Now()

	tc := TestCases{
		TC("open big short position and then close after reducing swap invariant").
			Given(
				CreateCustomMarket(
					pairBtcNusd,
					WithEnabled(true),
					WithPricePeg(sdk.OneDec()),
					WithSqrtDepth(sdk.NewDec(100_000)),
				),
				SetBlockNumber(1),
				SetBlockTime(startBlockTime),

				FundAccount(alice, sdk.NewCoins(sdk.NewCoin(types.TestingCollateralDenomNUSD, sdk.NewInt(10_200)))),
				FundAccount(bob, sdk.NewCoins(sdk.NewCoin(types.TestingCollateralDenomNUSD, sdk.NewInt(10_200)))),
				FundModule(types.PerpFundModuleAccount, sdk.NewCoins(sdk.NewCoin(types.TestingCollateralDenomNUSD, sdk.NewInt(100_000_000)))),

				MarketOrder(alice, pairBtcNusd, types.Direction_SHORT, sdk.NewInt(10_000), sdk.OneDec(), sdk.ZeroDec()),
				MarketOrder(bob, pairBtcNusd, types.Direction_LONG, sdk.NewInt(10_000), sdk.OneDec(), sdk.ZeroDec()),

				ShiftSwapInvariant(pairBtcNusd, sdk.NewInt(1)),
			).
			When(
				PartialCloseFails(alice, pairBtcNusd, sdk.NewDec(5_000), types.ErrAmmNonpositiveReserves),
			).
			Then(
				ClosePosition(bob, pairBtcNusd),
				PartialClose(alice, pairBtcNusd, sdk.NewDec(5_000)),
			),

		TC("new long position").
			Given(
				CreateCustomMarket(pairBtcNusd,
					WithEnabled(true),
				),
				SetBlockTime(startBlockTime),
				SetBlockNumber(1),
				FundAccount(alice, sdk.NewCoins(sdk.NewCoin(types.TestingCollateralDenomNUSD, sdk.NewInt(1020)))),
			).
			When(
				MarketOrder(alice, pairBtcNusd, types.Direction_LONG, sdk.NewInt(1000), sdk.NewDec(10), sdk.ZeroDec(),
					MarketOrderResp_PositionShouldBeEqual(
						types.Position{
							Pair:                            pairBtcNusd,
							TraderAddress:                   alice.String(),
							Margin:                          sdk.NewDec(980),
							OpenNotional:                    sdk.NewDec(9_800),
							Size_:                           sdk.MustNewDecFromStr("9799.999903960000941192"),
							LastUpdatedBlockNumber:          1,
							LatestCumulativePremiumFraction: sdk.ZeroDec(),
						}),
					MarketOrderResp_ExchangeNotionalValueShouldBeEqual(sdk.NewDec(9_800)), // margin * leverage
					MarketOrderResp_ExchangedPositionSizeShouldBeEqual(sdk.MustNewDecFromStr("9799.999903960000941192")),
					MarketOrderResp_BadDebtShouldBeEqual(sdk.ZeroDec()),
					MarketOrderResp_FundingPaymentShouldBeEqual(sdk.ZeroDec()),
					MarketOrderResp_RealizedPnlShouldBeEqual(sdk.ZeroDec()),
					MarketOrderResp_UnrealizedPnlAfterShouldBeEqual(sdk.ZeroDec()),
					MarketOrderResp_MarginToVaultShouldBeEqual(sdk.NewDec(980)),
					MarketOrderResp_PositionNotionalShouldBeEqual(sdk.NewDec(9800)),
				),
			).
			Then(
				PositionShouldBeEqual(alice, pairBtcNusd, Position_PositionShouldBeEqualTo(types.Position{
					Pair:                            pairBtcNusd,
					TraderAddress:                   alice.String(),
					Margin:                          sdk.NewDec(980),
					OpenNotional:                    sdk.NewDec(9800),
					Size_:                           sdk.MustNewDecFromStr("9799.999903960000941192"),
					LastUpdatedBlockNumber:          1,
					LatestCumulativePremiumFraction: sdk.ZeroDec(),
				})),
				PositionChangedEventShouldBeEqual(&types.PositionChangedEvent{
					FinalPosition: types.Position{
						Pair:                            pairBtcNusd,
						TraderAddress:                   alice.String(),
						Margin:                          sdk.NewDec(980),
						OpenNotional:                    sdk.NewDec(9800),
						Size_:                           sdk.MustNewDecFromStr("9799.999903960000941192"),
						LastUpdatedBlockNumber:          1,
						LatestCumulativePremiumFraction: sdk.ZeroDec(),
					},
					PositionNotional: sdk.NewDec(9800),
					RealizedPnl:      sdk.ZeroDec(),
					BadDebt:          sdk.NewCoin(types.TestingCollateralDenomNUSD, sdk.ZeroInt()),
					FundingPayment:   sdk.ZeroDec(),
					TransactionFee:   sdk.NewCoin(types.TestingCollateralDenomNUSD, sdk.NewInt(20)),
					BlockHeight:      1,
					// exchangedMargin = - marginToVault - transferredFee
					MarginToUser:      sdk.NewInt(980 + 20).Neg(),
					ChangeReason:      types.ChangeReason_MarketOrder,
					ExchangedNotional: sdk.NewDec(9800),
					ExchangedSize:     sdk.MustNewDecFromStr("9799.999903960000941192"),
				}),
			),

		TC("existing long position, go more long").
			Given(
				SetBlockNumber(1),
				SetBlockTime(startBlockTime),
				CreateCustomMarket(pairBtcNusd, WithEnabled(true)),
				FundAccount(alice, sdk.NewCoins(sdk.NewCoin(types.TestingCollateralDenomNUSD, sdk.NewInt(2040)))),
				MarketOrder(alice, pairBtcNusd, types.Direction_LONG, sdk.NewInt(1000), sdk.NewDec(10), sdk.ZeroDec()),
			).
			When(
				MoveToNextBlock(),
				MarketOrder(alice, pairBtcNusd, types.Direction_LONG, sdk.NewInt(1000), sdk.NewDec(10), sdk.ZeroDec(),
					MarketOrderResp_PositionShouldBeEqual(
						types.Position{
							Pair:                            pairBtcNusd,
							TraderAddress:                   alice.String(),
							Margin:                          sdk.NewDec(1_960),
							OpenNotional:                    sdk.NewDec(19_600),
							Size_:                           sdk.MustNewDecFromStr("19599.999615840007529536"),
							LastUpdatedBlockNumber:          2,
							LatestCumulativePremiumFraction: sdk.ZeroDec(),
						}),
					MarketOrderResp_ExchangeNotionalValueShouldBeEqual(sdk.NewDec(9_800)), // margin * leverage
					MarketOrderResp_ExchangedPositionSizeShouldBeEqual(sdk.MustNewDecFromStr("9799.999711880006588344")),
					MarketOrderResp_BadDebtShouldBeEqual(sdk.ZeroDec()),
					MarketOrderResp_FundingPaymentShouldBeEqual(sdk.ZeroDec()),
					MarketOrderResp_RealizedPnlShouldBeEqual(sdk.ZeroDec()),
					MarketOrderResp_UnrealizedPnlAfterShouldBeEqual(sdk.ZeroDec()),
					MarketOrderResp_MarginToVaultShouldBeEqual(sdk.NewDec(980)),
					MarketOrderResp_PositionNotionalShouldBeEqual(sdk.NewDec(19_600)),
				),
			).
			Then(
				PositionChangedEventShouldBeEqual(&types.PositionChangedEvent{
					FinalPosition: types.Position{
						Pair:                            pairBtcNusd,
						TraderAddress:                   alice.String(),
						Margin:                          sdk.NewDec(1_960),
						OpenNotional:                    sdk.NewDec(19_600),
						Size_:                           sdk.MustNewDecFromStr("19599.999615840007529536"),
						LastUpdatedBlockNumber:          2,
						LatestCumulativePremiumFraction: sdk.ZeroDec(),
					},
					PositionNotional: sdk.NewDec(19_600),
					RealizedPnl:      sdk.ZeroDec(),
					BadDebt:          sdk.NewCoin(types.TestingCollateralDenomNUSD, sdk.ZeroInt()),
					FundingPayment:   sdk.ZeroDec(),
					TransactionFee:   sdk.NewCoin(types.TestingCollateralDenomNUSD, sdk.NewInt(20)), // 20 bps
					BlockHeight:      2,
					// exchangedMargin = - marginToVault - transferredFee
					MarginToUser:      sdk.NewInt(980 + 20).Neg(),
					ChangeReason:      types.ChangeReason_MarketOrder,
					ExchangedNotional: sdk.MustNewDecFromStr("9800.000000000000000000"),
					ExchangedSize:     sdk.MustNewDecFromStr("9799.999711880006588344"),
				}),
			),

		TC("existing long position, go more long but there's bad debt").
			Given(
				CreateCustomMarket(
					pairBtcNusd,
					WithEnabled(true),
					WithPricePeg(sdk.MustNewDecFromStr("0.89")),
					WithLatestMarketCPF(sdk.MustNewDecFromStr("0.0002")),
				),
				SetBlockNumber(1),
				SetBlockTime(startBlockTime),
				FundAccount(alice, sdk.NewCoins(sdk.NewCoin(types.TestingCollateralDenomNUSD, sdk.NewInt(18)))),
				InsertPosition(
					WithPair(pairBtcNusd),
					WithTrader(alice),
					WithMargin(sdk.NewDec(1_000)),
					WithSize(sdk.NewDec(10_000)),
					WithOpenNotional(sdk.NewDec(10_000)),
				),
			).
			When(
				MoveToNextBlock(),
				MarketOrderFails(alice, pairBtcNusd, types.Direction_LONG, sdk.OneInt(), sdk.OneDec(), sdk.ZeroDec(),
					types.ErrMarginRatioTooLow,
				),
			).
			Then(
				PositionShouldBeEqual(alice, pairBtcNusd, Position_PositionShouldBeEqualTo(types.Position{
					TraderAddress:                   alice.String(),
					Pair:                            pairBtcNusd,
					Size_:                           sdk.NewDec(10_000),
					Margin:                          sdk.NewDec(1_000),
					OpenNotional:                    sdk.NewDec(10_000),
					LatestCumulativePremiumFraction: sdk.ZeroDec(),
					LastUpdatedBlockNumber:          0,
				})),
			),

		TC("existing long position, close a bit but there's bad debt").
			Given(
				CreateCustomMarket(
					pairBtcNusd, WithEnabled(true),
					WithPricePeg(sdk.MustNewDecFromStr("0.89")),
					WithLatestMarketCPF(sdk.MustNewDecFromStr("0.0002")),
				),
				SetBlockNumber(1),
				SetBlockTime(startBlockTime),
				FundAccount(alice, sdk.NewCoins(sdk.NewCoin(types.TestingCollateralDenomNUSD, sdk.NewInt(18)))),
				InsertPosition(
					WithPair(pairBtcNusd),
					WithTrader(alice),
					WithMargin(sdk.NewDec(1_000)),
					WithSize(sdk.NewDec(10_000)),
					WithOpenNotional(sdk.NewDec(10_000)),
				),
			).
			When(
				MoveToNextBlock(),
				MarketOrderFails(alice, pairBtcNusd, types.Direction_SHORT, sdk.OneInt(), sdk.OneDec(), sdk.ZeroDec(),
					types.ErrMarginRatioTooLow,
				),
			).
			Then(
				PositionShouldBeEqual(alice, pairBtcNusd, Position_PositionShouldBeEqualTo(types.Position{
					TraderAddress:                   alice.String(),
					Pair:                            pairBtcNusd,
					Size_:                           sdk.NewDec(10_000),
					Margin:                          sdk.NewDec(1_000),
					OpenNotional:                    sdk.NewDec(10_000),
					LatestCumulativePremiumFraction: sdk.ZeroDec(),
					LastUpdatedBlockNumber:          0,
				})),
			),

		TC("open big long position and then close after reducing swap invariant").
			Given(
				CreateCustomMarket(
					pairBtcNusd, WithEnabled(true),
					WithPricePeg(sdk.OneDec()),
					WithSqrtDepth(sdk.NewDec(100_000)),
				),
				SetBlockNumber(1),
				SetBlockTime(startBlockTime),
				FundAccount(alice, sdk.NewCoins(sdk.NewCoin(types.TestingCollateralDenomNUSD, sdk.NewInt(10_000)))),
				MarketOrder(alice, pairBtcNusd, types.Direction_LONG, sdk.NewInt(9_000), sdk.NewDec(10), sdk.ZeroDec()),
				FundModule(types.PerpFundModuleAccount, sdk.NewCoins(sdk.NewCoin(types.TestingCollateralDenomNUSD, sdk.NewInt(100_000_000)))),
				ShiftSwapInvariant(pairBtcNusd, sdk.NewInt(1)),
			).
			When(
				ClosePosition(alice, pairBtcNusd),
			).
			Then(
				PositionShouldNotExist(alice, pairBtcNusd, 1),
			),

		TC("existing long position, decrease a bit").
			Given(
				CreateCustomMarket(pairBtcNusd, WithEnabled(true)),
				SetBlockNumber(1),
				SetBlockTime(startBlockTime),
				FundAccount(alice, sdk.NewCoins(sdk.NewCoin(types.TestingCollateralDenomNUSD, sdk.NewInt(1030)))),
				MarketOrder(alice, pairBtcNusd, types.Direction_LONG, sdk.NewInt(1000), sdk.NewDec(10), sdk.ZeroDec()),
			).
			When(
				MoveToNextBlock(),
				MarketOrder(alice, pairBtcNusd, types.Direction_SHORT, sdk.NewInt(500), sdk.NewDec(10), sdk.ZeroDec(),
					MarketOrderResp_PositionShouldBeEqual(
						types.Position{
							Pair:                            pairBtcNusd,
							TraderAddress:                   alice.String(),
							Margin:                          sdk.NewDec(980),
							OpenNotional:                    sdk.NewDec(4_900),
							Size_:                           sdk.MustNewDecFromStr("4899.999975990000117649"),
							LastUpdatedBlockNumber:          2,
							LatestCumulativePremiumFraction: sdk.ZeroDec(),
						},
					),
					MarketOrderResp_ExchangeNotionalValueShouldBeEqual(sdk.NewDec(4_900)), // (margin - fees) * leverage
					MarketOrderResp_ExchangedPositionSizeShouldBeEqual(sdk.MustNewDecFromStr("-4899.999927970000823543")),
					MarketOrderResp_BadDebtShouldBeEqual(sdk.ZeroDec()),
					MarketOrderResp_FundingPaymentShouldBeEqual(sdk.ZeroDec()),
					MarketOrderResp_RealizedPnlShouldBeEqual(sdk.ZeroDec()),
					MarketOrderResp_UnrealizedPnlAfterShouldBeEqual(sdk.ZeroDec()),
					MarketOrderResp_MarginToVaultShouldBeEqual(sdk.ZeroDec()),
					MarketOrderResp_PositionNotionalShouldBeEqual(sdk.NewDec(4_900)),
				),
			).
			Then(
				PositionChangedEventShouldBeEqual(&types.PositionChangedEvent{
					FinalPosition: types.Position{
						Pair:                            pairBtcNusd,
						TraderAddress:                   alice.String(),
						Margin:                          sdk.NewDec(980),
						OpenNotional:                    sdk.NewDec(4_900),
						Size_:                           sdk.MustNewDecFromStr("4899.999975990000117649"),
						LastUpdatedBlockNumber:          2,
						LatestCumulativePremiumFraction: sdk.ZeroDec(),
					},
					PositionNotional: sdk.NewDec(4_900),
					RealizedPnl:      sdk.ZeroDec(),
					BadDebt:          sdk.NewCoin(types.TestingCollateralDenomNUSD, sdk.ZeroInt()),
					FundingPayment:   sdk.ZeroDec(),
					TransactionFee:   sdk.NewCoin(types.TestingCollateralDenomNUSD, sdk.NewInt(10)), // 20 bps
					BlockHeight:      2,
					// exchangedMargin = - marginToVault - transferredFee
					MarginToUser:      sdk.NewInt(0 + 10).Neg(),
					ChangeReason:      types.ChangeReason_MarketOrder,
					ExchangedNotional: sdk.MustNewDecFromStr("-4900.000000000000000000"),
					ExchangedSize:     sdk.MustNewDecFromStr("-4899.999927970000823543"),
				}),
			),

		TC("existing long position, decrease a bit but there's bad debt").
			Given(
				CreateCustomMarket(
					pairBtcNusd,
					WithEnabled(true),
					WithPricePeg(sdk.MustNewDecFromStr("0.89")),
					WithLatestMarketCPF(sdk.MustNewDecFromStr("0.0002")),
				),
				SetBlockNumber(1),
				SetBlockTime(startBlockTime),
				FundAccount(alice, sdk.NewCoins(sdk.NewCoin(types.TestingCollateralDenomNUSD, sdk.NewInt(18)))),
				InsertPosition(
					WithPair(pairBtcNusd),
					WithTrader(alice),
					WithMargin(sdk.NewDec(1_000)),
					WithSize(sdk.NewDec(10_000)),
					WithOpenNotional(sdk.NewDec(10_000)),
				),
			).
			When(
				MoveToNextBlock(),
				MarketOrderFails(alice, pairBtcNusd, types.Direction_LONG, sdk.OneInt(), sdk.OneDec(), sdk.ZeroDec(),
					types.ErrMarginRatioTooLow,
				),
			).
			Then(
				PositionShouldBeEqual(alice, pairBtcNusd, Position_PositionShouldBeEqualTo(types.Position{
					TraderAddress:                   alice.String(),
					Pair:                            pairBtcNusd,
					Size_:                           sdk.NewDec(10_000),
					Margin:                          sdk.NewDec(1_000),
					OpenNotional:                    sdk.NewDec(10_000),
					LatestCumulativePremiumFraction: sdk.ZeroDec(),
					LastUpdatedBlockNumber:          0,
				})),
			),

		TC("existing long position, decrease a lot").
			Given(
				CreateCustomMarket(pairBtcNusd, WithEnabled(true)),
				SetBlockNumber(1),
				SetBlockTime(startBlockTime),
				FundAccount(alice, sdk.NewCoins(sdk.NewCoin(types.TestingCollateralDenomNUSD, sdk.NewInt(4080)))),
				MarketOrder(alice, pairBtcNusd, types.Direction_LONG, sdk.NewInt(1000), sdk.NewDec(10), sdk.ZeroDec()),
			).
			When(
				MoveToNextBlock(),
				MarketOrder(alice, pairBtcNusd, types.Direction_SHORT, sdk.NewInt(3000), sdk.NewDec(10), sdk.ZeroDec(),
					MarketOrderResp_PositionShouldBeEqual(
						types.Position{
							Pair:                            pairBtcNusd,
							TraderAddress:                   alice.String(),
							Margin:                          sdk.NewDec(1960),
							OpenNotional:                    sdk.NewDec(19_600),
							Size_:                           sdk.MustNewDecFromStr("-19600.000384160007529536"),
							LastUpdatedBlockNumber:          2,
							LatestCumulativePremiumFraction: sdk.ZeroDec(),
						},
					),
					MarketOrderResp_ExchangeNotionalValueShouldBeEqual(sdk.NewDec(29_400)), // margin * leverage
					MarketOrderResp_ExchangedPositionSizeShouldBeEqual(sdk.MustNewDecFromStr("-29400.000288120008470728")),
					MarketOrderResp_BadDebtShouldBeEqual(sdk.ZeroDec()),
					MarketOrderResp_FundingPaymentShouldBeEqual(sdk.ZeroDec()),
					MarketOrderResp_RealizedPnlShouldBeEqual(sdk.ZeroDec()),
					MarketOrderResp_UnrealizedPnlAfterShouldBeEqual(sdk.ZeroDec()),
					MarketOrderResp_MarginToVaultShouldBeEqual(sdk.NewDec(980)),
					MarketOrderResp_PositionNotionalShouldBeEqual(sdk.NewDec(19_600)),
				),
			).
			Then(
				PositionChangedEventShouldBeEqual(&types.PositionChangedEvent{
					FinalPosition: types.Position{
						Pair:                            pairBtcNusd,
						TraderAddress:                   alice.String(),
						Margin:                          sdk.NewDec(1960),
						OpenNotional:                    sdk.NewDec(19_600),
						Size_:                           sdk.MustNewDecFromStr("-19600.000384160007529536"),
						LastUpdatedBlockNumber:          2,
						LatestCumulativePremiumFraction: sdk.ZeroDec(),
					},
					PositionNotional: sdk.NewDec(19_600),
					RealizedPnl:      sdk.ZeroDec(),
					BadDebt:          sdk.NewCoin(types.TestingCollateralDenomNUSD, sdk.ZeroInt()),
					FundingPayment:   sdk.ZeroDec(),
					TransactionFee:   sdk.NewCoin(types.TestingCollateralDenomNUSD, sdk.NewInt(60)), // 20 bps
					BlockHeight:      2,
					// exchangedMargin = - marginToVault - transferredFee
					MarginToUser:      sdk.NewInt(980 + 60).Neg(),
					ChangeReason:      types.ChangeReason_MarketOrder,
					ExchangedNotional: sdk.MustNewDecFromStr("9800.000000000000000000"),
					ExchangedSize:     sdk.MustNewDecFromStr("-29400.000288120008470728"),
				}),
			),

		TC("existing long position, decrease a lot but there's bad debt").
			Given(
				CreateCustomMarket(
					pairBtcNusd, WithEnabled(true),
					WithPricePeg(sdk.MustNewDecFromStr("0.89")),
					WithLatestMarketCPF(sdk.MustNewDecFromStr("0.0002")),
				),
				SetBlockNumber(1),
				SetBlockTime(startBlockTime),
				FundAccount(alice, sdk.NewCoins(sdk.NewCoin(types.TestingCollateralDenomNUSD, sdk.NewInt(150)))),
				InsertPosition(
					WithPair(pairBtcNusd),
					WithTrader(alice),
					WithMargin(sdk.NewDec(1_000)),
					WithSize(sdk.NewDec(10_000)),
					WithOpenNotional(sdk.NewDec(10_000)),
				),
			).
			When(
				MoveToNextBlock(),
				MarketOrder(alice, pairBtcNusd, types.Direction_SHORT, sdk.NewInt(3000), sdk.NewDec(10), sdk.ZeroDec(),
					MarketOrderResp_PositionShouldBeEqual(
						types.Position{
							Pair:                            pairBtcNusd,
							TraderAddress:                   alice.String(),
							Margin:                          sdk.ZeroDec(),
							OpenNotional:                    sdk.ZeroDec(),
							Size_:                           sdk.ZeroDec(),
							LastUpdatedBlockNumber:          2,
							LatestCumulativePremiumFraction: sdk.MustNewDecFromStr("0.0002"),
						},
					),
					MarketOrderResp_ExchangeNotionalValueShouldBeEqual(sdk.MustNewDecFromStr("8899.999911000000890000")),
					MarketOrderResp_ExchangedPositionSizeShouldBeEqual(sdk.MustNewDecFromStr("-10000")),
					MarketOrderResp_BadDebtShouldBeEqual(sdk.MustNewDecFromStr("102.000088999999110000")),
					MarketOrderResp_FundingPaymentShouldBeEqual(sdk.NewDec(2)),
					MarketOrderResp_RealizedPnlShouldBeEqual(sdk.MustNewDecFromStr("-1100.000088999999110000")),
					MarketOrderResp_UnrealizedPnlAfterShouldBeEqual(sdk.ZeroDec()),
					MarketOrderResp_MarginToVaultShouldBeEqual(sdk.ZeroDec()),
					MarketOrderResp_PositionNotionalShouldBeEqual(sdk.ZeroDec()),
				),
			).
			Then(
				PositionChangedEventShouldBeEqual(&types.PositionChangedEvent{
					FinalPosition: types.Position{
						Pair:                            pairBtcNusd,
						TraderAddress:                   alice.String(),
						Margin:                          sdk.ZeroDec(),
						OpenNotional:                    sdk.ZeroDec(),
						Size_:                           sdk.ZeroDec(),
						LastUpdatedBlockNumber:          2,
						LatestCumulativePremiumFraction: sdk.MustNewDecFromStr("0.0002"),
					},
					PositionNotional:  sdk.ZeroDec(),
					RealizedPnl:       sdk.MustNewDecFromStr("-1100.000088999999110000"),
					BadDebt:           sdk.NewInt64Coin(types.TestingCollateralDenomNUSD, 102),
					FundingPayment:    sdk.NewDec(2),
					TransactionFee:    sdk.NewInt64Coin(types.TestingCollateralDenomNUSD, 60), // 20 bps
					BlockHeight:       2,
					MarginToUser:      sdk.NewInt(-60),
					ChangeReason:      types.ChangeReason_MarketOrder,
					ExchangedNotional: sdk.MustNewDecFromStr("-10000.000000000000000000"),
					ExchangedSize:     sdk.MustNewDecFromStr("-10000.000000000000000000"),
				}),
			),

		TC("new short position").
			Given(
				CreateCustomMarket(pairBtcNusd, WithEnabled(true)),
				SetBlockTime(startBlockTime),
				SetBlockNumber(1),
				FundAccount(alice, sdk.NewCoins(sdk.NewCoin(types.TestingCollateralDenomNUSD, sdk.NewInt(1020)))),
			).
			When(
				MarketOrder(alice, pairBtcNusd, types.Direction_SHORT, sdk.NewInt(1000), sdk.NewDec(10), sdk.ZeroDec(),
					MarketOrderResp_PositionShouldBeEqual(
						types.Position{
							Pair:                            pairBtcNusd,
							TraderAddress:                   alice.String(),
							Margin:                          sdk.NewDec(980),
							OpenNotional:                    sdk.NewDec(9_800),
							Size_:                           sdk.MustNewDecFromStr("-9800.000096040000941192"),
							LastUpdatedBlockNumber:          1,
							LatestCumulativePremiumFraction: sdk.ZeroDec(),
						}),
					MarketOrderResp_ExchangeNotionalValueShouldBeEqual(sdk.NewDec(9_800)), // margin * leverage
					MarketOrderResp_ExchangedPositionSizeShouldBeEqual(sdk.MustNewDecFromStr("-9800.000096040000941192")),
					MarketOrderResp_BadDebtShouldBeEqual(sdk.ZeroDec()),
					MarketOrderResp_FundingPaymentShouldBeEqual(sdk.ZeroDec()),
					MarketOrderResp_RealizedPnlShouldBeEqual(sdk.ZeroDec()),
					MarketOrderResp_UnrealizedPnlAfterShouldBeEqual(sdk.ZeroDec()),
					MarketOrderResp_MarginToVaultShouldBeEqual(sdk.NewDec(980)),
					MarketOrderResp_PositionNotionalShouldBeEqual(sdk.NewDec(9_800)),
				),
			).
			Then(
				PositionShouldBeEqual(alice, pairBtcNusd,
					Position_PositionShouldBeEqualTo(types.Position{
						Pair:                            pairBtcNusd,
						TraderAddress:                   alice.String(),
						Margin:                          sdk.NewDec(980),
						OpenNotional:                    sdk.NewDec(9_800),
						Size_:                           sdk.MustNewDecFromStr("-9800.000096040000941192"),
						LastUpdatedBlockNumber:          1,
						LatestCumulativePremiumFraction: sdk.ZeroDec(),
					}),
				),
				PositionChangedEventShouldBeEqual(&types.PositionChangedEvent{
					FinalPosition: types.Position{
						Pair:                            pairBtcNusd,
						TraderAddress:                   alice.String(),
						Margin:                          sdk.NewDec(980),
						OpenNotional:                    sdk.NewDec(9_800),
						Size_:                           sdk.MustNewDecFromStr("-9800.000096040000941192"),
						LastUpdatedBlockNumber:          1,
						LatestCumulativePremiumFraction: sdk.ZeroDec(),
					},
					PositionNotional: sdk.NewDec(9_800),
					RealizedPnl:      sdk.ZeroDec(),
					BadDebt:          sdk.NewCoin(types.TestingCollateralDenomNUSD, sdk.ZeroInt()),
					FundingPayment:   sdk.ZeroDec(),
					TransactionFee:   sdk.NewCoin(types.TestingCollateralDenomNUSD, sdk.NewInt(20)),
					BlockHeight:      1,
					// exchangedMargin = - marginToVault - transferredFee
					MarginToUser:      sdk.NewInt(980 + 20).Neg(),
					ChangeReason:      types.ChangeReason_MarketOrder,
					ExchangedNotional: sdk.MustNewDecFromStr("9800.000000000000000000"),
					ExchangedSize:     sdk.MustNewDecFromStr("-9800.000096040000941192"),
				}),
			),

		TC("existing short position, go more short").
			Given(
				CreateCustomMarket(pairBtcNusd, WithEnabled(true)),
				SetBlockNumber(1),
				SetBlockTime(startBlockTime),
				FundAccount(alice, sdk.NewCoins(sdk.NewCoin(types.TestingCollateralDenomNUSD, sdk.NewInt(2040)))),
				MarketOrder(alice, pairBtcNusd, types.Direction_SHORT, sdk.NewInt(1000), sdk.NewDec(10), sdk.ZeroDec()),
			).
			When(
				MoveToNextBlock(),
				MarketOrder(alice, pairBtcNusd, types.Direction_SHORT, sdk.NewInt(1000), sdk.NewDec(10), sdk.ZeroDec(),
					MarketOrderResp_PositionShouldBeEqual(
						types.Position{
							Pair:                            pairBtcNusd,
							TraderAddress:                   alice.String(),
							Margin:                          sdk.NewDec(1_960),
							OpenNotional:                    sdk.NewDec(19_600),
							Size_:                           sdk.MustNewDecFromStr("-19600.000384160007529536"),
							LastUpdatedBlockNumber:          2,
							LatestCumulativePremiumFraction: sdk.ZeroDec(),
						},
					),
					MarketOrderResp_ExchangeNotionalValueShouldBeEqual(sdk.NewDec(9_800)), // margin * leverage
					MarketOrderResp_ExchangedPositionSizeShouldBeEqual(sdk.MustNewDecFromStr("-9800.000288120006588344")),
					MarketOrderResp_BadDebtShouldBeEqual(sdk.ZeroDec()),
					MarketOrderResp_FundingPaymentShouldBeEqual(sdk.ZeroDec()),
					MarketOrderResp_RealizedPnlShouldBeEqual(sdk.ZeroDec()),
					MarketOrderResp_UnrealizedPnlAfterShouldBeEqual(sdk.ZeroDec()),
					MarketOrderResp_MarginToVaultShouldBeEqual(sdk.NewDec(980)),
					MarketOrderResp_PositionNotionalShouldBeEqual(sdk.NewDec(19_600)),
				),
			).
			Then(
				PositionChangedEventShouldBeEqual(&types.PositionChangedEvent{
					FinalPosition: types.Position{
						Pair:                            pairBtcNusd,
						TraderAddress:                   alice.String(),
						Margin:                          sdk.NewDec(1_960),
						OpenNotional:                    sdk.NewDec(19_600),
						Size_:                           sdk.MustNewDecFromStr("-19600.000384160007529536"),
						LastUpdatedBlockNumber:          2,
						LatestCumulativePremiumFraction: sdk.ZeroDec(),
					},
					PositionNotional: sdk.NewDec(19_600),
					RealizedPnl:      sdk.ZeroDec(),
					BadDebt:          sdk.NewCoin(types.TestingCollateralDenomNUSD, sdk.ZeroInt()),
					FundingPayment:   sdk.ZeroDec(),
					TransactionFee:   sdk.NewCoin(types.TestingCollateralDenomNUSD, sdk.NewInt(20)), // 20 bps
					BlockHeight:      2,
					// exchangedMargin = - marginToVault - transferredFee
					MarginToUser:      sdk.NewInt(980 + 20).Neg(),
					ChangeReason:      types.ChangeReason_MarketOrder,
					ExchangedNotional: sdk.MustNewDecFromStr("9800.000000000000000000"),
					ExchangedSize:     sdk.MustNewDecFromStr("-9800.000288120006588344"),
				}),
			),

		TC("existing short position, go more short but there's bad debt").
			Given(
				CreateCustomMarket(pairBtcNusd, WithEnabled(true),
					WithPricePeg(sdk.MustNewDecFromStr("1.11")),
					WithLatestMarketCPF(sdk.MustNewDecFromStr("0.0002")),
				),
				SetBlockNumber(1),
				SetBlockTime(startBlockTime),
				FundAccount(alice, sdk.NewCoins(sdk.NewCoin(types.TestingCollateralDenomNUSD, sdk.NewInt(22)))),
				InsertPosition(
					WithPair(pairBtcNusd),
					WithTrader(alice),
					WithMargin(sdk.NewDec(1_000)),
					WithSize(sdk.NewDec(-10_000)),
					WithOpenNotional(sdk.NewDec(10_000)),
				),
			).
			When(
				MoveToNextBlock(),
				MarketOrderFails(alice, pairBtcNusd, types.Direction_SHORT, sdk.OneInt(), sdk.OneDec(), sdk.ZeroDec(),
					types.ErrMarginRatioTooLow,
				),
			).
			Then(
				PositionShouldBeEqual(alice, pairBtcNusd, Position_PositionShouldBeEqualTo(types.Position{
					TraderAddress:                   alice.String(),
					Pair:                            pairBtcNusd,
					Size_:                           sdk.NewDec(-10_000),
					Margin:                          sdk.NewDec(1_000),
					OpenNotional:                    sdk.NewDec(10_000),
					LatestCumulativePremiumFraction: sdk.ZeroDec(),
					LastUpdatedBlockNumber:          0,
				})),
			),

		TC("existing short position, decrease a bit").
			Given(
				CreateCustomMarket(pairBtcNusd, WithEnabled(true)),
				SetBlockNumber(1),
				SetBlockTime(startBlockTime),
				FundAccount(alice, sdk.NewCoins(sdk.NewCoin(types.TestingCollateralDenomNUSD, sdk.NewInt(1030)))),
				MarketOrder(alice, pairBtcNusd, types.Direction_SHORT, sdk.NewInt(1000), sdk.NewDec(10), sdk.ZeroDec()),
			).
			When(
				MoveToNextBlock(),
				MarketOrder(alice, pairBtcNusd, types.Direction_LONG, sdk.NewInt(500), sdk.NewDec(10), sdk.ZeroDec(),
					MarketOrderResp_PositionShouldBeEqual(
						types.Position{
							Pair:                            pairBtcNusd,
							TraderAddress:                   alice.String(),
							Margin:                          sdk.NewDec(980),
							OpenNotional:                    sdk.NewDec(4_900),
							Size_:                           sdk.MustNewDecFromStr("-4900.000024010000117649"),
							LastUpdatedBlockNumber:          2,
							LatestCumulativePremiumFraction: sdk.ZeroDec(),
						},
					),
					MarketOrderResp_ExchangeNotionalValueShouldBeEqual(sdk.NewDec(4_900)), // margin * leverage
					MarketOrderResp_ExchangedPositionSizeShouldBeEqual(sdk.MustNewDecFromStr("4900.000072030000823543")),
					MarketOrderResp_BadDebtShouldBeEqual(sdk.ZeroDec()),
					MarketOrderResp_FundingPaymentShouldBeEqual(sdk.ZeroDec()),
					MarketOrderResp_RealizedPnlShouldBeEqual(sdk.ZeroDec()),
					MarketOrderResp_UnrealizedPnlAfterShouldBeEqual(sdk.ZeroDec()),
					MarketOrderResp_MarginToVaultShouldBeEqual(sdk.ZeroDec()),
					MarketOrderResp_PositionNotionalShouldBeEqual(sdk.NewDec(4900)),
				),
			).
			Then(
				PositionChangedEventShouldBeEqual(&types.PositionChangedEvent{
					FinalPosition: types.Position{
						Pair:                            pairBtcNusd,
						TraderAddress:                   alice.String(),
						Margin:                          sdk.NewDec(980),
						OpenNotional:                    sdk.NewDec(4_900),
						Size_:                           sdk.MustNewDecFromStr("-4900.000024010000117649"),
						LastUpdatedBlockNumber:          2,
						LatestCumulativePremiumFraction: sdk.ZeroDec(),
					},
					PositionNotional: sdk.NewDec(4_900),
					RealizedPnl:      sdk.ZeroDec(),
					BadDebt:          sdk.NewCoin(types.TestingCollateralDenomNUSD, sdk.ZeroInt()),
					FundingPayment:   sdk.ZeroDec(),
					TransactionFee:   sdk.NewCoin(types.TestingCollateralDenomNUSD, sdk.NewInt(10)), // 20 bps
					BlockHeight:      2,
					// exchangedMargin = - marginToVault - transferredFee
					MarginToUser:      sdk.NewInt(0 + 10).Neg(),
					ChangeReason:      types.ChangeReason_MarketOrder,
					ExchangedNotional: sdk.MustNewDecFromStr("-4900.000000000000000000"),
					ExchangedSize:     sdk.MustNewDecFromStr("4900.000072030000823543"),
				}),
			),

		TC("existing short position, decrease a bit but there's bad debt").
			Given(
				CreateCustomMarket(pairBtcNusd,
					WithEnabled(true),
					WithPricePeg(sdk.MustNewDecFromStr("1.11")),
					WithLatestMarketCPF(sdk.MustNewDecFromStr("0.0002")),
				),
				SetBlockNumber(1),
				SetBlockTime(startBlockTime),
				FundAccount(alice, sdk.NewCoins(sdk.NewCoin(types.TestingCollateralDenomNUSD, sdk.NewInt(22)))),
				InsertPosition(
					WithPair(pairBtcNusd),
					WithTrader(alice),
					WithMargin(sdk.NewDec(1_000)),
					WithSize(sdk.NewDec(-10_000)),
					WithOpenNotional(sdk.NewDec(10_000)),
				),
			).
			When(
				MoveToNextBlock(),
				MarketOrderFails(alice, pairBtcNusd, types.Direction_LONG, sdk.OneInt(), sdk.OneDec(), sdk.ZeroDec(),
					types.ErrMarginRatioTooLow,
				),
			).
			Then(
				PositionShouldBeEqual(alice, pairBtcNusd, Position_PositionShouldBeEqualTo(types.Position{
					TraderAddress:                   alice.String(),
					Pair:                            pairBtcNusd,
					Size_:                           sdk.NewDec(-10_000),
					Margin:                          sdk.NewDec(1_000),
					OpenNotional:                    sdk.NewDec(10_000),
					LatestCumulativePremiumFraction: sdk.ZeroDec(),
					LastUpdatedBlockNumber:          0,
				})),
			),

		TC("existing short position, decrease a lot").
			Given(
				CreateCustomMarket(pairBtcNusd, WithEnabled(true)),
				SetBlockNumber(1),
				SetBlockTime(startBlockTime),
				FundAccount(alice, sdk.NewCoins(sdk.NewCoin(types.TestingCollateralDenomNUSD, sdk.NewInt(4080)))),
				MarketOrder(alice, pairBtcNusd, types.Direction_SHORT, sdk.NewInt(1000), sdk.NewDec(10), sdk.ZeroDec()),
			).
			When(
				MoveToNextBlock(),
				MarketOrder(alice, pairBtcNusd, types.Direction_LONG, sdk.NewInt(3000), sdk.NewDec(10), sdk.ZeroDec(),
					MarketOrderResp_PositionShouldBeEqual(
						types.Position{
							Pair:                            pairBtcNusd,
							TraderAddress:                   alice.String(),
							Margin:                          sdk.NewDec(1960),
							OpenNotional:                    sdk.NewDec(19_600),
							Size_:                           sdk.MustNewDecFromStr("19599.999615840007529536"),
							LastUpdatedBlockNumber:          2,
							LatestCumulativePremiumFraction: sdk.ZeroDec(),
						},
					),
					MarketOrderResp_ExchangeNotionalValueShouldBeEqual(sdk.NewDec(29_400)), // margin * leverage
					MarketOrderResp_ExchangedPositionSizeShouldBeEqual(sdk.MustNewDecFromStr("29399.999711880008470728")),
					MarketOrderResp_BadDebtShouldBeEqual(sdk.ZeroDec()),
					MarketOrderResp_FundingPaymentShouldBeEqual(sdk.ZeroDec()),
					MarketOrderResp_RealizedPnlShouldBeEqual(sdk.ZeroDec()),
					MarketOrderResp_UnrealizedPnlAfterShouldBeEqual(sdk.ZeroDec()),
					MarketOrderResp_MarginToVaultShouldBeEqual(sdk.NewDec(980)),
					MarketOrderResp_PositionNotionalShouldBeEqual(sdk.NewDec(19_600)),
				),
			).
			Then(
				PositionChangedEventShouldBeEqual(&types.PositionChangedEvent{
					FinalPosition: types.Position{
						Pair:                            pairBtcNusd,
						TraderAddress:                   alice.String(),
						Margin:                          sdk.NewDec(1960),
						OpenNotional:                    sdk.NewDec(19_600),
						Size_:                           sdk.MustNewDecFromStr("19599.999615840007529536"),
						LastUpdatedBlockNumber:          2,
						LatestCumulativePremiumFraction: sdk.ZeroDec(),
					},
					PositionNotional: sdk.NewDec(19_600),
					RealizedPnl:      sdk.ZeroDec(),
					BadDebt:          sdk.NewCoin(types.TestingCollateralDenomNUSD, sdk.ZeroInt()),
					FundingPayment:   sdk.ZeroDec(),
					TransactionFee:   sdk.NewCoin(types.TestingCollateralDenomNUSD, sdk.NewInt(60)), // 20 bps
					BlockHeight:      2,
					// exchangedMargin = - marginToVault - transferredFee
					MarginToUser:      sdk.NewInt(980 + 60).Neg(),
					ChangeReason:      types.ChangeReason_MarketOrder,
					ExchangedNotional: sdk.MustNewDecFromStr("9800.000000000000000000"),
					ExchangedSize:     sdk.MustNewDecFromStr("29399.999711880008470728"),
				}),
			),

		TC("existing short position, decrease a lot but there's bad debt").
			Given(
				CreateCustomMarket(pairBtcNusd, WithEnabled(true),
					WithPricePeg(sdk.MustNewDecFromStr("1.11")),
					WithLatestMarketCPF(sdk.MustNewDecFromStr("0.0002")),
				),
				SetBlockNumber(1),
				SetBlockTime(startBlockTime),
				FundAccount(alice, sdk.NewCoins(sdk.NewCoin(types.TestingCollateralDenomNUSD, sdk.NewInt(150)))),
				InsertPosition(
					WithPair(pairBtcNusd),
					WithTrader(alice),
					WithMargin(sdk.NewDec(1_000)),
					WithSize(sdk.NewDec(-10_000)),
					WithOpenNotional(sdk.NewDec(10_000)),
				),
			).
			When(
				MoveToNextBlock(),
				MarketOrder(alice, pairBtcNusd, types.Direction_LONG, sdk.NewInt(3000), sdk.NewDec(10), sdk.ZeroDec(),
					MarketOrderResp_PositionShouldBeEqual(
						types.Position{
							Pair:                            pairBtcNusd,
							TraderAddress:                   alice.String(),
							Margin:                          sdk.ZeroDec(),
							OpenNotional:                    sdk.ZeroDec(),
							Size_:                           sdk.ZeroDec(),
							LastUpdatedBlockNumber:          2,
							LatestCumulativePremiumFraction: sdk.MustNewDecFromStr("0.0002"),
						},
					),
					MarketOrderResp_ExchangeNotionalValueShouldBeEqual(sdk.MustNewDecFromStr("11100.000111000001110000")),
					MarketOrderResp_ExchangedPositionSizeShouldBeEqual(sdk.MustNewDecFromStr("10000")),
					MarketOrderResp_BadDebtShouldBeEqual(sdk.MustNewDecFromStr("98.000111000001110000")),
					MarketOrderResp_FundingPaymentShouldBeEqual(sdk.NewDec(-2)),
					MarketOrderResp_RealizedPnlShouldBeEqual(sdk.MustNewDecFromStr("-1100.000111000001110000")),
					MarketOrderResp_UnrealizedPnlAfterShouldBeEqual(sdk.ZeroDec()),
					MarketOrderResp_MarginToVaultShouldBeEqual(sdk.ZeroDec()),
					MarketOrderResp_PositionNotionalShouldBeEqual(sdk.ZeroDec()),
				),
			).
			Then(
				PositionChangedEventShouldBeEqual(&types.PositionChangedEvent{
					FinalPosition: types.Position{
						Pair:                            pairBtcNusd,
						TraderAddress:                   alice.String(),
						Margin:                          sdk.ZeroDec(),
						OpenNotional:                    sdk.ZeroDec(),
						Size_:                           sdk.ZeroDec(),
						LastUpdatedBlockNumber:          2,
						LatestCumulativePremiumFraction: sdk.MustNewDecFromStr("0.0002"),
					},
					PositionNotional: sdk.ZeroDec(),
					RealizedPnl:      sdk.MustNewDecFromStr("-1100.000111000001110000"),
					BadDebt:          sdk.NewInt64Coin(types.TestingCollateralDenomNUSD, 98),
					FundingPayment:   sdk.NewDec(-2),
					TransactionFee:   sdk.NewInt64Coin(types.TestingCollateralDenomNUSD, 60), // 20 bps
					BlockHeight:      2,
					// exchangedMargin = - marginToVault - transferredFee
					MarginToUser:      sdk.NewInt(-60),
					ChangeReason:      types.ChangeReason_MarketOrder,
					ExchangedNotional: sdk.MustNewDecFromStr("-10000.000000000000000000"),
					ExchangedSize:     sdk.MustNewDecFromStr("10000.000000000000000000"),
				}),
			),

		TC("user has insufficient funds").
			Given(
				CreateCustomMarket(pairBtcNusd, WithEnabled(true)),
				SetBlockTime(startBlockTime),
				SetBlockNumber(1),
				FundAccount(alice, sdk.NewCoins(sdk.NewCoin(types.TestingCollateralDenomNUSD, sdk.NewInt(99)))),
			).
			When(
				MarketOrderFails(
					alice, pairBtcNusd, types.Direction_LONG, sdk.NewInt(100), sdk.OneDec(), sdk.ZeroDec(),
					sdkerrors.ErrInsufficientFunds),
			).
			Then(
				PositionShouldNotExist(alice, pairBtcNusd, 1),
			),

		TC("new long position, can not open new position after market is not enabled").
			Given(
				CreateCustomMarket(pairBtcNusd, WithEnabled(true)),
				SetBlockTime(startBlockTime),
				SetBlockNumber(1),
				FundAccount(alice, sdk.NewCoins(sdk.NewCoin(types.TestingCollateralDenomNUSD, sdk.NewInt(47_714_285_715)))),
				CloseMarket(pairBtcNusd),
			).
			When(
				MarketOrderFails(alice, pairBtcNusd, types.Direction_SHORT, sdk.NewInt(47_619_047_619), sdk.OneDec(), sdk.ZeroDec(),
					types.ErrMarketNotEnabled),
			).
			Then(
				PositionShouldNotExist(alice, pairBtcNusd, 1),
			),

		TC("market doesn't exist").
			Given(
				SetBlockTime(startBlockTime),
				SetBlockNumber(1),
				FundAccount(alice, sdk.NewCoins(sdk.NewCoin(types.TestingCollateralDenomNUSD, sdk.NewInt(47_714_285_715)))),
			).
			When(
				MarketOrderFails(alice, pairBtcNusd, types.Direction_SHORT, sdk.NewInt(47_619_047_619), sdk.OneDec(), sdk.ZeroDec(),
					types.ErrPairNotFound),
			).
			Then(
				PositionShouldNotExist(alice, pairBtcNusd, 1),
			),

		TC("zero quote asset amount").
			Given(
				SetBlockTime(startBlockTime),
				SetBlockNumber(1),
				CreateCustomMarket(pairBtcNusd, WithEnabled(true)),
				FundAccount(alice, sdk.NewCoins(sdk.NewCoin(types.TestingCollateralDenomNUSD, sdk.NewInt(47_714_285_715)))),
			).
			When(
				MarketOrderFails(alice, pairBtcNusd, types.Direction_LONG, sdk.ZeroInt(), sdk.OneDec(), sdk.ZeroDec(),
					types.ErrInputQuoteAmtNegative),
			).
			Then(
				PositionShouldNotExist(alice, pairBtcNusd, 1),
			),

		TC("zero leverage").
			Given(
				SetBlockTime(startBlockTime),
				SetBlockNumber(1),
				CreateCustomMarket(pairBtcNusd, WithEnabled(true)),
				FundAccount(alice, sdk.NewCoins(sdk.NewCoin(types.TestingCollateralDenomNUSD, sdk.NewInt(1e6)))),
			).
			When(
				MarketOrderFails(alice, pairBtcNusd, types.Direction_LONG, sdk.OneInt(), sdk.ZeroDec(), sdk.ZeroDec(),
					types.ErrUserLeverageNegative),
			).
			Then(
				PositionShouldNotExist(alice, pairBtcNusd, 1),
			),

		TC("user leverage greater than market max leverage").
			Given(
				SetBlockTime(startBlockTime),
				SetBlockNumber(1),
				CreateCustomMarket(pairBtcNusd, WithEnabled(true)),
				FundAccount(alice, sdk.NewCoins(sdk.NewCoin(types.TestingCollateralDenomNUSD, sdk.NewInt(1e6)))),
			).
			When(
				MarketOrderFails(alice, pairBtcNusd, types.Direction_LONG, sdk.OneInt(), sdk.NewDec(11), sdk.ZeroDec(),
					types.ErrLeverageIsTooHigh),
			).
			Then(
				PositionShouldNotExist(alice, pairBtcNusd, 1),
			),
	}

	NewTestSuite(t).WithTestCases(tc...).Run()
}

func TestMarketOrderError(t *testing.T) {
	testCases := []struct {
		name        string
		traderFunds sdk.Coins

		initialPosition *types.Position

		// position arguments
		side      types.Direction
		margin    sdkmath.Int
		leverage  sdk.Dec
		baseLimit sdk.Dec

		expectedErr error
	}{
		{
			name:            "not enough trader funds",
			traderFunds:     sdk.NewCoins(sdk.NewInt64Coin(types.TestingCollateralDenomNUSD, 999)),
			initialPosition: nil,
			side:            types.Direction_LONG,
			margin:          sdk.NewInt(1000),
			leverage:        sdk.NewDec(10),
			baseLimit:       sdk.ZeroDec(),
			expectedErr:     sdkerrors.ErrInsufficientFunds,
		},
		{
			name:        "position has bad debt",
			traderFunds: sdk.NewCoins(sdk.NewInt64Coin(types.TestingCollateralDenomNUSD, 999)),
			initialPosition: &types.Position{
				Pair:                            asset.Registry.Pair(denoms.BTC, denoms.NUSD),
				Size_:                           sdk.OneDec(),
				Margin:                          sdk.NewDec(1000),
				OpenNotional:                    sdk.NewDec(10_000),
				LatestCumulativePremiumFraction: sdk.ZeroDec(),
				LastUpdatedBlockNumber:          1,
			},
			side:        types.Direction_LONG,
			margin:      sdk.OneInt(),
			leverage:    sdk.OneDec(),
			baseLimit:   sdk.ZeroDec(),
			expectedErr: types.ErrMarginRatioTooLow,
		},
		{
			name:            "new long position not over base limit",
			traderFunds:     sdk.NewCoins(sdk.NewInt64Coin(types.TestingCollateralDenomNUSD, 1020)),
			initialPosition: nil,
			side:            types.Direction_LONG,
			margin:          sdk.NewInt(1000),
			leverage:        sdk.NewDec(10),
			baseLimit:       sdk.NewDec(10_000),
			expectedErr:     types.ErrAssetFailsUserLimit,
		},
		{
			name:            "new short position not under base limit",
			traderFunds:     sdk.NewCoins(sdk.NewInt64Coin(types.TestingCollateralDenomNUSD, 1020)),
			initialPosition: nil,
			side:            types.Direction_SHORT,
			margin:          sdk.NewInt(1000),
			leverage:        sdk.NewDec(10),
			baseLimit:       sdk.NewDec(9_800),
			expectedErr:     types.ErrAssetFailsUserLimit,
		},
		{
			name:            "quote asset amount is zero",
			traderFunds:     sdk.NewCoins(sdk.NewInt64Coin(types.TestingCollateralDenomNUSD, 1020)),
			initialPosition: nil,
			side:            types.Direction_SHORT,
			margin:          sdk.ZeroInt(),
			leverage:        sdk.NewDec(10),
			baseLimit:       sdk.NewDec(10_000),
			expectedErr:     types.ErrInputQuoteAmtNegative,
		},
		{
			name:            "leverage amount is zero",
			traderFunds:     sdk.NewCoins(sdk.NewInt64Coin(types.TestingCollateralDenomNUSD, 1020)),
			initialPosition: nil,
			side:            types.Direction_SHORT,
			margin:          sdk.NewInt(1000),
			leverage:        sdk.ZeroDec(),
			baseLimit:       sdk.NewDec(10_000),
			expectedErr:     types.ErrUserLeverageNegative,
		},
		{
			name:            "leverage amount is too high - SELL",
			traderFunds:     sdk.NewCoins(sdk.NewInt64Coin(types.TestingCollateralDenomNUSD, 1020)),
			initialPosition: nil,
			side:            types.Direction_SHORT,
			margin:          sdk.NewInt(100),
			leverage:        sdk.NewDec(100),
			baseLimit:       sdk.NewDec(11_000),
			expectedErr:     types.ErrLeverageIsTooHigh,
		},
		{
			name:            "leverage amount is too high - BUY",
			traderFunds:     sdk.NewCoins(sdk.NewInt64Coin(types.TestingCollateralDenomNUSD, 1020)),
			initialPosition: nil,
			side:            types.Direction_LONG,
			margin:          sdk.NewInt(100),
			leverage:        sdk.NewDec(16),
			baseLimit:       sdk.ZeroDec(),
			expectedErr:     types.ErrLeverageIsTooHigh,
		},
	}

	for _, testCase := range testCases {
		tc := testCase
		t.Run(tc.name, func(t *testing.T) {
			app, ctx := testapp.NewNibiruTestAppAndContext()
			traderAddr := testutil.AccAddress()

			market := mock.TestMarket()
			app.PerpKeeperV2.SaveMarket(ctx, *market)
			app.PerpKeeperV2.MarketLastVersion.Insert(ctx, asset.Registry.Pair(denoms.BTC, denoms.NUSD), types.MarketLastVersion{Version: market.Version})
			amm := mock.TestAMMDefault()
			app.PerpKeeperV2.SaveAMM(ctx, *amm)
			app.PerpKeeperV2.ReserveSnapshots.Insert(ctx, collections.Join(amm.Pair, ctx.BlockTime()), types.ReserveSnapshot{
				Amm:         *amm,
				TimestampMs: ctx.BlockTime().UnixMilli(),
			})
			require.NoError(t, testapp.FundAccount(app.BankKeeper, ctx, traderAddr, tc.traderFunds))

			if tc.initialPosition != nil {
				tc.initialPosition.TraderAddress = traderAddr.String()
				app.PerpKeeperV2.SavePosition(ctx, tc.initialPosition.Pair, 1, traderAddr, *tc.initialPosition)
			}

			ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 1).WithBlockTime(ctx.BlockTime().Add(time.Second * 5))
			resp, err := app.PerpKeeperV2.MarketOrder(ctx, asset.Registry.Pair(denoms.BTC, denoms.NUSD), tc.side, traderAddr, tc.margin, tc.leverage, tc.baseLimit)
			require.ErrorContains(t, err, tc.expectedErr.Error())
			require.Nil(t, resp)
		})
	}
}

func TestPartialClose(t *testing.T) {
	alice := testutil.AccAddress()
	pairBtcNusd := asset.Registry.Pair(denoms.BTC, denoms.NUSD)
	startBlockTime := time.Now()

	tc := TestCases{
		TC("partial close long position with positive PnL").
			Given(
				CreateCustomMarket(
					pairBtcNusd,
					WithEnabled(true),
					WithPricePeg(sdk.NewDec(2)),
					WithLatestMarketCPF(sdk.MustNewDecFromStr("0.0002")),
				),
				SetBlockTime(startBlockTime),
				SetBlockNumber(1),
				FundAccount(alice, sdk.NewCoins(sdk.NewCoin(types.TestingCollateralDenomNUSD, sdk.NewInt(1000)))),
				InsertPosition(
					WithPair(pairBtcNusd),
					WithTrader(alice),
					WithSize(sdk.NewDec(10_000)),
					WithMargin(sdk.NewDec(1_000)),
					WithOpenNotional(sdk.NewDec(10_000)),
				),
			).
			When(
				PartialClose(alice, pairBtcNusd, sdk.NewDec(2_500)),
			).
			Then(
				PositionShouldBeEqual(alice, pairBtcNusd, Position_PositionShouldBeEqualTo(types.Position{
					Pair:                            pairBtcNusd,
					TraderAddress:                   alice.String(),
					Margin:                          sdk.MustNewDecFromStr("3492.999950075025499499"),
					OpenNotional:                    sdk.MustNewDecFromStr("7504.999962575025468249"),
					Size_:                           sdk.MustNewDecFromStr("7505.000000024975000031"),
					LastUpdatedBlockNumber:          1,
					LatestCumulativePremiumFraction: sdk.MustNewDecFromStr("0.0002"),
				})),
				PositionChangedEventShouldBeEqual(&types.PositionChangedEvent{
					FinalPosition: types.Position{
						Pair:                            pairBtcNusd,
						TraderAddress:                   alice.String(),
						Margin:                          sdk.MustNewDecFromStr("3492.999950075025499499"),
						OpenNotional:                    sdk.MustNewDecFromStr("7504.999962575025468249"),
						Size_:                           sdk.MustNewDecFromStr("7505.000000024975000031"),
						LastUpdatedBlockNumber:          1,
						LatestCumulativePremiumFraction: sdk.MustNewDecFromStr("0.0002"),
					},
					PositionNotional:  sdk.MustNewDecFromStr("15009.999812500001968750"),
					RealizedPnl:       sdk.MustNewDecFromStr("2494.999950075025499499"),
					BadDebt:           sdk.NewCoin(types.TestingCollateralDenomNUSD, sdk.ZeroInt()),
					FundingPayment:    sdk.NewDec(2),
					TransactionFee:    sdk.NewCoin(types.TestingCollateralDenomNUSD, sdk.NewInt(10)),
					BlockHeight:       1,
					MarginToUser:      sdk.NewInt(-10),
					ChangeReason:      types.ChangeReason_PartialClose,
					ExchangedNotional: sdk.MustNewDecFromStr("5009.999812500001968750"),
					ExchangedSize:     sdk.MustNewDecFromStr("-2494.999999975024999969"),
				}),
			),

		TC("partial close long position with negative PnL").
			Given(
				CreateCustomMarket(
					pairBtcNusd,
					WithEnabled(true),
					WithPricePeg(sdk.MustNewDecFromStr("0.95")),
					WithLatestMarketCPF(sdk.MustNewDecFromStr("0.0002")),
				),
				SetBlockTime(startBlockTime),
				SetBlockNumber(1),
				FundAccount(alice, sdk.NewCoins(sdk.NewCoin(types.TestingCollateralDenomNUSD, sdk.NewInt(4)))),
				InsertPosition(
					WithPair(pairBtcNusd),
					WithTrader(alice),
					WithSize(sdk.NewDec(10_000)),
					WithMargin(sdk.NewDec(1_000)),
					WithOpenNotional(sdk.NewDec(10_000)),
				),
			).
			When(
				PartialClose(alice, pairBtcNusd, sdk.NewDec(2_500)),
			).
			Then(
				PositionShouldBeEqual(alice, pairBtcNusd, Position_PositionShouldBeEqualTo(types.Position{
					Pair:                            pairBtcNusd,
					TraderAddress:                   alice.String(),
					Margin:                          sdk.MustNewDecFromStr("873.210502606841456300"),
					OpenNotional:                    sdk.MustNewDecFromStr("7504.210508544341441456"),
					Size_:                           sdk.MustNewDecFromStr("7504.210526336824376757"),
					LastUpdatedBlockNumber:          1,
					LatestCumulativePremiumFraction: sdk.MustNewDecFromStr("0.0002"),
				})),
				PositionChangedEventShouldBeEqual(&types.PositionChangedEvent{
					FinalPosition: types.Position{
						Pair:                            pairBtcNusd,
						TraderAddress:                   alice.String(),
						Margin:                          sdk.MustNewDecFromStr("873.210502606841456300"),
						OpenNotional:                    sdk.MustNewDecFromStr("7504.210508544341441456"),
						Size_:                           sdk.MustNewDecFromStr("7504.210526336824376757"),
						LastUpdatedBlockNumber:          1,
						LatestCumulativePremiumFraction: sdk.MustNewDecFromStr("0.0002"),
					},
					PositionNotional:  sdk.MustNewDecFromStr("7128.999910937500935156"),
					RealizedPnl:       sdk.MustNewDecFromStr("-124.789497393158543700"),
					BadDebt:           sdk.NewCoin(types.TestingCollateralDenomNUSD, sdk.ZeroInt()),
					FundingPayment:    sdk.NewDec(2),
					TransactionFee:    sdk.NewCoin(types.TestingCollateralDenomNUSD, sdk.NewInt(4)),
					BlockHeight:       1,
					MarginToUser:      sdk.NewInt(-4),
					ChangeReason:      types.ChangeReason_PartialClose,
					ExchangedNotional: sdk.MustNewDecFromStr("-2871.000089062499064844"),
					ExchangedSize:     sdk.MustNewDecFromStr("-2495.789473663175623243"),
				}),
			),

		TC("partial close long position without bad debt but below maintenance margin ratio").
			Given(
				CreateCustomMarket(
					pairBtcNusd,
					WithEnabled(true),
					WithPricePeg(sdk.MustNewDecFromStr("0.94")),
					WithLatestMarketCPF(sdk.MustNewDecFromStr("0.0002")),
				),
				SetBlockTime(startBlockTime),
				SetBlockNumber(1),
				FundAccount(alice, sdk.NewCoins(sdk.NewCoin(types.TestingCollateralDenomNUSD, sdk.NewInt(4)))),
				InsertPosition(
					WithPair(pairBtcNusd),
					WithTrader(alice),
					WithSize(sdk.NewDec(10_000)),
					WithMargin(sdk.NewDec(1_000)),
					WithOpenNotional(sdk.NewDec(10_000)),
				),
			).
			When(
				PartialClose(alice, pairBtcNusd, sdk.NewDec(2_500)),
			).
			Then(
				PositionShouldBeEqual(alice, pairBtcNusd, Position_PositionShouldBeEqualTo(types.Position{
					Pair:                            pairBtcNusd,
					TraderAddress:                   alice.String(),
					Margin:                          sdk.MustNewDecFromStr("848.255295690211914400"),
					OpenNotional:                    sdk.MustNewDecFromStr("7504.255301565211899712"),
					Size_:                           sdk.MustNewDecFromStr("7504.255319170194658242"),
					LastUpdatedBlockNumber:          1,
					LatestCumulativePremiumFraction: sdk.MustNewDecFromStr("0.0002"),
				})),
				PositionChangedEventShouldBeEqual(&types.PositionChangedEvent{
					FinalPosition: types.Position{
						Pair:                            pairBtcNusd,
						TraderAddress:                   alice.String(),
						Margin:                          sdk.MustNewDecFromStr("848.255295690211914400"),
						OpenNotional:                    sdk.MustNewDecFromStr("7504.255301565211899712"),
						Size_:                           sdk.MustNewDecFromStr("7504.255319170194658242"),
						LastUpdatedBlockNumber:          1,
						LatestCumulativePremiumFraction: sdk.MustNewDecFromStr("0.0002"),
					},
					PositionNotional:  sdk.MustNewDecFromStr("7053.999911875000925312"),
					RealizedPnl:       sdk.MustNewDecFromStr("-149.744704309788085600"),
					BadDebt:           sdk.NewCoin(types.TestingCollateralDenomNUSD, sdk.ZeroInt()),
					FundingPayment:    sdk.NewDec(2),
					TransactionFee:    sdk.NewCoin(types.TestingCollateralDenomNUSD, sdk.NewInt(4)),
					BlockHeight:       1,
					MarginToUser:      sdk.NewInt(-4),
					ChangeReason:      types.ChangeReason_PartialClose,
					ExchangedNotional: sdk.MustNewDecFromStr("-2946.000088124999074688"),
					ExchangedSize:     sdk.MustNewDecFromStr("-2495.744680829805341758"),
				})),

		TC("partial close long position with bad debt").
			Given(
				CreateCustomMarket(
					pairBtcNusd,
					WithEnabled(true),
					WithPricePeg(sdk.MustNewDecFromStr("0.59")),
					WithLatestMarketCPF(sdk.MustNewDecFromStr("0.0002")),
				),
				SetBlockTime(startBlockTime),
				SetBlockNumber(1),
				FundAccount(alice, sdk.NewCoins(sdk.NewInt64Coin(types.TestingCollateralDenomNUSD, 2))),
				FundModule(types.PerpFundModuleAccount, sdk.NewCoins(sdk.NewInt64Coin(types.TestingCollateralDenomNUSD, 27))),
				InsertPosition(
					WithPair(pairBtcNusd),
					WithTrader(alice),
					WithSize(sdk.NewDec(10_000)),
					WithMargin(sdk.NewDec(1_000)),
					WithOpenNotional(sdk.NewDec(10_000)),
				),
			).
			When(
				PartialClose(alice, pairBtcNusd, sdk.NewDec(2_500)),
			).
			Then(
				PositionShouldBeEqual(alice, pairBtcNusd, Position_PositionShouldBeEqualTo(types.Position{
					Pair:                            pairBtcNusd,
					TraderAddress:                   alice.String(),
					Margin:                          sdk.ZeroDec(),
					OpenNotional:                    sdk.MustNewDecFromStr("7503.389819472919156581"),
					Size_:                           sdk.MustNewDecFromStr("7503.389830525412237884"),
					LastUpdatedBlockNumber:          1,
					LatestCumulativePremiumFraction: sdk.MustNewDecFromStr("0.0002"),
				})),
				PositionChangedEventShouldBeEqual(&types.PositionChangedEvent{
					FinalPosition: types.Position{
						Pair:                            pairBtcNusd,
						TraderAddress:                   alice.String(),
						Margin:                          sdk.ZeroDec(),
						OpenNotional:                    sdk.MustNewDecFromStr("7503.389819472919156581"),
						Size_:                           sdk.MustNewDecFromStr("7503.389830525412237884"),
						LastUpdatedBlockNumber:          1,
						LatestCumulativePremiumFraction: sdk.MustNewDecFromStr("0.0002"),
					},
					PositionNotional:  sdk.MustNewDecFromStr("4426.999944687500580781"),
					RealizedPnl:       sdk.MustNewDecFromStr("-1023.610184214580834200"),
					BadDebt:           sdk.NewInt64Coin(types.TestingCollateralDenomNUSD, 26),
					FundingPayment:    sdk.NewDec(2),
					TransactionFee:    sdk.NewInt64Coin(types.TestingCollateralDenomNUSD, 2),
					BlockHeight:       1,
					MarginToUser:      sdk.NewInt(-2),
					ChangeReason:      types.ChangeReason_PartialClose,
					ExchangedNotional: sdk.MustNewDecFromStr("-5573.000055312499419219"),
					ExchangedSize:     sdk.MustNewDecFromStr("-2496.610169474587762116"),
				})),

		TC("partial close short position with positive PnL").
			Given(
				CreateCustomMarket(
					pairBtcNusd,
					WithEnabled(true),
					WithPricePeg(sdk.MustNewDecFromStr("0.10")),
					WithLatestMarketCPF(sdk.MustNewDecFromStr("0.0002")),
				),
				SetBlockTime(startBlockTime),
				SetBlockNumber(1),
				FundAccount(alice, sdk.NewCoins(sdk.NewCoin(types.TestingCollateralDenomNUSD, sdk.NewInt(2)))),
				InsertPosition(
					WithPair(pairBtcNusd),
					WithTrader(alice),
					WithSize(sdk.NewDec(-10_000)),
					WithMargin(sdk.NewDec(1_000)),
					WithOpenNotional(sdk.NewDec(10_000)),
				),
			).
			When(
				PartialClose(alice, pairBtcNusd, sdk.NewDec(7_500)),
			).
			Then(
				PositionShouldBeEqual(alice, pairBtcNusd, Position_PositionShouldBeEqualTo(types.Position{
					Pair:                            pairBtcNusd,
					TraderAddress:                   alice.String(),
					Margin:                          sdk.MustNewDecFromStr("7733.999992789639924900"),
					OpenNotional:                    sdk.MustNewDecFromStr("2520.000001585360032912"),
					Size_:                           sdk.MustNewDecFromStr("-2519.999999700400001111"),
					LastUpdatedBlockNumber:          1,
					LatestCumulativePremiumFraction: sdk.MustNewDecFromStr("0.0002"),
				})),
				PositionChangedEventShouldBeEqual(&types.PositionChangedEvent{
					FinalPosition: types.Position{
						Pair:                            pairBtcNusd,
						TraderAddress:                   alice.String(),
						Margin:                          sdk.MustNewDecFromStr("7733.999992789639924900"),
						OpenNotional:                    sdk.MustNewDecFromStr("2520.000001585360032912"),
						Size_:                           sdk.MustNewDecFromStr("-2519.999999700400001111"),
						LastUpdatedBlockNumber:          1,
						LatestCumulativePremiumFraction: sdk.MustNewDecFromStr("0.0002"),
					},
					PositionNotional:  sdk.MustNewDecFromStr("252.000004375000057812"),
					RealizedPnl:       sdk.MustNewDecFromStr("6731.999992789639924900"),
					BadDebt:           sdk.NewCoin(types.TestingCollateralDenomNUSD, sdk.ZeroInt()),
					FundingPayment:    sdk.NewDec(-2),
					TransactionFee:    sdk.NewCoin(types.TestingCollateralDenomNUSD, sdk.NewInt(2)),
					BlockHeight:       1,
					MarginToUser:      sdk.NewInt(-2),
					ChangeReason:      types.ChangeReason_PartialClose,
					ExchangedNotional: sdk.MustNewDecFromStr("-9747.999995624999942188"),
					ExchangedSize:     sdk.MustNewDecFromStr("7480.000000299599998889"),
				}),
			),

		TC("partial close short position with negative PnL").
			Given(
				CreateCustomMarket(
					pairBtcNusd,
					WithEnabled(true),
					WithPricePeg(sdk.MustNewDecFromStr("1.05")),
					WithLatestMarketCPF(sdk.MustNewDecFromStr("0.0002")),
				),
				SetBlockTime(startBlockTime),
				SetBlockNumber(1),
				FundAccount(alice, sdk.NewCoins(sdk.NewCoin(types.TestingCollateralDenomNUSD, sdk.NewInt(16)))),
				InsertPosition(
					WithPair(pairBtcNusd),
					WithTrader(alice),
					WithSize(sdk.NewDec(-10_000)),
					WithMargin(sdk.NewDec(1_000)),
					WithOpenNotional(sdk.NewDec(10_000)),
				),
			).
			When(
				PartialClose(alice, pairBtcNusd, sdk.NewDec(7_500)),
			).
			Then(
				PositionShouldBeEqual(alice, pairBtcNusd, Position_PositionShouldBeEqualTo(types.Position{
					Pair:                            pairBtcNusd,
					TraderAddress:                   alice.String(),
					Margin:                          sdk.MustNewDecFromStr("627.761826160487012202"),
					OpenNotional:                    sdk.MustNewDecFromStr("2515.238114777012544829"),
					Size_:                           sdk.MustNewDecFromStr("-2515.238095009756009922"),
					LastUpdatedBlockNumber:          1,
					LatestCumulativePremiumFraction: sdk.MustNewDecFromStr("0.0002"),
				})),
				PositionChangedEventShouldBeEqual(&types.PositionChangedEvent{
					FinalPosition: types.Position{
						Pair:                            pairBtcNusd,
						TraderAddress:                   alice.String(),
						Margin:                          sdk.MustNewDecFromStr("627.761826160487012202"),
						OpenNotional:                    sdk.MustNewDecFromStr("2515.238114777012544829"),
						Size_:                           sdk.MustNewDecFromStr("-2515.238095009756009922"),
						LastUpdatedBlockNumber:          1,
						LatestCumulativePremiumFraction: sdk.MustNewDecFromStr("0.0002"),
					},
					PositionNotional:  sdk.MustNewDecFromStr("2641.000045937500607031"),
					RealizedPnl:       sdk.MustNewDecFromStr("-374.238173839512987798"),
					BadDebt:           sdk.NewCoin(types.TestingCollateralDenomNUSD, sdk.ZeroInt()),
					FundingPayment:    sdk.NewDec(-2),
					TransactionFee:    sdk.NewCoin(types.TestingCollateralDenomNUSD, sdk.NewInt(16)),
					BlockHeight:       1,
					MarginToUser:      sdk.NewInt(-16),
					ChangeReason:      types.ChangeReason_PartialClose,
					ExchangedNotional: sdk.MustNewDecFromStr("-7358.999954062499392969"),
					ExchangedSize:     sdk.MustNewDecFromStr("7484.761904990243990078"),
				}),
			),

		TC("partial close short position with no bad debt but below maintenance margin ratio").
			Given(
				CreateCustomMarket(
					pairBtcNusd,
					WithEnabled(true),
					WithPricePeg(sdk.MustNewDecFromStr("1.09")),
					WithLatestMarketCPF(sdk.MustNewDecFromStr("0.0002")),
				),
				SetBlockTime(startBlockTime),
				SetBlockNumber(1),
				FundAccount(alice, sdk.NewCoins(sdk.NewCoin(types.TestingCollateralDenomNUSD, sdk.NewInt(16)))),
				InsertPosition(
					WithPair(pairBtcNusd),
					WithTrader(alice),
					WithSize(sdk.NewDec(-10_000)),
					WithMargin(sdk.NewDec(1_000)),
					WithOpenNotional(sdk.NewDec(10_000)),
				),
			).
			When(
				PartialClose(alice, pairBtcNusd, sdk.NewDec(7_500)),
			).
			Then(
				PositionShouldBeEqual(alice, pairBtcNusd, Position_PositionShouldBeEqualTo(types.Position{
					Pair:                            pairBtcNusd,
					TraderAddress:                   alice.String(),
					Margin:                          sdk.MustNewDecFromStr("328.321019307633252802"),
					OpenNotional:                    sdk.MustNewDecFromStr("2514.678919379866287354"),
					Size_:                           sdk.MustNewDecFromStr("-2514.678898862600792000"),
					LastUpdatedBlockNumber:          1,
					LatestCumulativePremiumFraction: sdk.MustNewDecFromStr("0.0002"),
				})),
				PositionChangedEventShouldBeEqual(&types.PositionChangedEvent{
					FinalPosition: types.Position{
						Pair:                            pairBtcNusd,
						TraderAddress:                   alice.String(),
						Margin:                          sdk.MustNewDecFromStr("328.321019307633252802"),
						OpenNotional:                    sdk.MustNewDecFromStr("2514.678919379866287354"),
						Size_:                           sdk.MustNewDecFromStr("-2514.678898862600792000"),
						LastUpdatedBlockNumber:          1,
						LatestCumulativePremiumFraction: sdk.MustNewDecFromStr("0.0002"),
					},
					PositionNotional:  sdk.MustNewDecFromStr("2741.000047687500630156"),
					RealizedPnl:       sdk.MustNewDecFromStr("-673.678980692366747198"),
					BadDebt:           sdk.NewCoin(types.TestingCollateralDenomNUSD, sdk.ZeroInt()),
					FundingPayment:    sdk.NewDec(-2),
					TransactionFee:    sdk.NewCoin(types.TestingCollateralDenomNUSD, sdk.NewInt(16)),
					BlockHeight:       1,
					MarginToUser:      sdk.NewInt(-16),
					ChangeReason:      types.ChangeReason_PartialClose,
					ExchangedNotional: sdk.MustNewDecFromStr("-7258.999952312499369844"),
					ExchangedSize:     sdk.MustNewDecFromStr("7485.321101137399208000"),
				}),
			),

		TC("partial close short position with bad debt").
			Given(
				CreateCustomMarket(
					pairBtcNusd,
					WithEnabled(true),
					WithPricePeg(sdk.MustNewDecFromStr("1.14")),
					WithLatestMarketCPF(sdk.MustNewDecFromStr("0.0002")),
				),
				SetBlockTime(startBlockTime),
				SetBlockNumber(1),
				FundAccount(alice, sdk.NewCoins(sdk.NewCoin(types.TestingCollateralDenomNUSD, sdk.NewInt(18)))),
				FundModule(types.PerpFundModuleAccount, sdk.NewCoins(sdk.NewInt64Coin(types.TestingCollateralDenomNUSD, 48))),
				InsertPosition(
					WithPair(pairBtcNusd),
					WithTrader(alice),
					WithSize(sdk.NewDec(-10_000)),
					WithMargin(sdk.NewDec(1_000)),
					WithOpenNotional(sdk.NewDec(10_000)),
				),
			).
			When(
				PartialClose(alice, pairBtcNusd, sdk.NewDec(7_500)),
			).
			Then(
				PositionShouldBeEqual(alice, pairBtcNusd, Position_PositionShouldBeEqualTo(types.Position{
					Pair:                            pairBtcNusd,
					TraderAddress:                   alice.String(),
					Margin:                          sdk.ZeroDec(),
					OpenNotional:                    sdk.MustNewDecFromStr("2515.789494912333892759"),
					Size_:                           sdk.MustNewDecFromStr("-2515.789473447617729414"),
					LastUpdatedBlockNumber:          1,
					LatestCumulativePremiumFraction: sdk.MustNewDecFromStr("0.0002"),
				})),
				PositionChangedEventShouldBeEqual(&types.PositionChangedEvent{
					FinalPosition: types.Position{
						Pair:                            pairBtcNusd,
						TraderAddress:                   alice.String(),
						Margin:                          sdk.ZeroDec(),
						OpenNotional:                    sdk.MustNewDecFromStr("2515.789494912333892759"),
						Size_:                           sdk.MustNewDecFromStr("-2515.789473447617729414"),
						LastUpdatedBlockNumber:          1,
						LatestCumulativePremiumFraction: sdk.MustNewDecFromStr("0.0002"),
					},
					PositionNotional:  sdk.MustNewDecFromStr("2868.000049875000659062"),
					RealizedPnl:       sdk.MustNewDecFromStr("-1047.789559037334373697"),
					BadDebt:           sdk.NewInt64Coin(types.TestingCollateralDenomNUSD, 46),
					FundingPayment:    sdk.NewDec(-2),
					TransactionFee:    sdk.NewCoin(types.TestingCollateralDenomNUSD, sdk.NewInt(18)),
					BlockHeight:       1,
					MarginToUser:      sdk.NewInt(-18),
					ChangeReason:      types.ChangeReason_PartialClose,
					ExchangedNotional: sdk.MustNewDecFromStr("-7131.999950124999340938"),
					ExchangedSize:     sdk.MustNewDecFromStr("7484.210526552382270586"),
				}),
			),
		TC("test partial closes fail").
			Given(
				CreateCustomMarket(
					pairBtcNusd,
					WithEnabled(true),
					WithPricePeg(sdk.OneDec()),
					WithSqrtDepth(sdk.NewDec(10_000)),
				),
				SetBlockNumber(1),
				SetBlockTime(startBlockTime),

				FundAccount(alice, sdk.NewCoins(sdk.NewCoin(types.TestingCollateralDenomNUSD, sdk.NewInt(10_200)))),

				PartialCloseFails(alice, pairBtcNusd, sdk.NewDec(5_000), types.ErrPositionNotFound),
				MarketOrder(alice, pairBtcNusd, types.Direction_LONG, sdk.NewInt(9_000), sdk.OneDec(), sdk.ZeroDec()),
			).
			When(
				PartialCloseFails(alice, asset.MustNewPair("luna:usdt"), sdk.NewDec(5_000), types.ErrPairNotFound),
			),
	}

	NewTestSuite(t).WithTestCases(tc...).Run()
}

func TestClosePosition(t *testing.T) {
	alice := testutil.AccAddress()
	pairBtcNusd := asset.Registry.Pair(denoms.BTC, denoms.NUSD)
	startBlockTime := time.Now()

	tc := TestCases{
		TC("close long position with positive PnL").
			Given(
				CreateCustomMarket(pairBtcNusd,
					WithEnabled(true),
					WithPricePeg(sdk.NewDec(2)),
					WithLatestMarketCPF(sdk.MustNewDecFromStr("0.0002")),
				),
				SetBlockTime(startBlockTime),
				SetBlockNumber(1),
				FundAccount(alice, sdk.NewCoins(sdk.NewCoin(types.TestingCollateralDenomNUSD, sdk.NewInt(40)))),
				FundModule(types.VaultModuleAccount, sdk.NewCoins(sdk.NewInt64Coin(types.TestingCollateralDenomNUSD, 10_998))),
				InsertPosition(
					WithTrader(alice),
					WithPair(pairBtcNusd),
					WithSize(sdk.NewDec(10_000)),
					WithMargin(sdk.NewDec(1_000)),
					WithOpenNotional(sdk.NewDec(10_000)),
				),
			).
			When(
				ClosePosition(alice, pairBtcNusd),
			).
			Then(
				PositionShouldNotExist(alice, pairBtcNusd, 1),
				PositionChangedEventShouldBeEqual(&types.PositionChangedEvent{
					FinalPosition: types.Position{
						Pair:                            pairBtcNusd,
						TraderAddress:                   alice.String(),
						Margin:                          sdk.ZeroDec(),
						OpenNotional:                    sdk.ZeroDec(),
						Size_:                           sdk.ZeroDec(),
						LastUpdatedBlockNumber:          1,
						LatestCumulativePremiumFraction: sdk.MustNewDecFromStr("0.0002"),
					},
					PositionNotional:  sdk.ZeroDec(),
					RealizedPnl:       sdk.MustNewDecFromStr("9999.999800000002000000"),
					BadDebt:           sdk.NewCoin(types.TestingCollateralDenomNUSD, sdk.ZeroInt()),
					FundingPayment:    sdk.NewDec(2),
					TransactionFee:    sdk.NewCoin(types.TestingCollateralDenomNUSD, sdk.NewInt(22)),
					BlockHeight:       1,
					MarginToUser:      sdk.NewInt(10976),
					ChangeReason:      types.ChangeReason_ClosePosition,
					ExchangedNotional: sdk.MustNewDecFromStr("-10000.000000000000000000"),
					ExchangedSize:     sdk.MustNewDecFromStr("-10000.000000000000000000"),
				}),
			),

		TC("close long position with negative PnL").
			Given(
				CreateCustomMarket(pairBtcNusd,
					WithEnabled(true),
					WithPricePeg(sdk.MustNewDecFromStr("0.99")),
					WithLatestMarketCPF(sdk.MustNewDecFromStr("0.0002")),
				),
				SetBlockTime(startBlockTime),
				SetBlockNumber(1),
				FundAccount(alice, sdk.NewCoins(sdk.NewCoin(types.TestingCollateralDenomNUSD, sdk.NewInt(20)))),
				FundModule(types.VaultModuleAccount, sdk.NewCoins(sdk.NewInt64Coin(types.TestingCollateralDenomNUSD, 898))),
				InsertPosition(
					WithTrader(alice),
					WithPair(pairBtcNusd),
					WithSize(sdk.NewDec(10_000)),
					WithMargin(sdk.NewDec(1_000)),
					WithOpenNotional(sdk.NewDec(10_000)),
				),
			).
			When(
				ClosePosition(alice, pairBtcNusd),
			).
			Then(
				PositionShouldNotExist(alice, pairBtcNusd, 1),
				PositionChangedEventShouldBeEqual(&types.PositionChangedEvent{
					FinalPosition: types.Position{
						Pair:                            pairBtcNusd,
						TraderAddress:                   alice.String(),
						Margin:                          sdk.ZeroDec(),
						OpenNotional:                    sdk.ZeroDec(),
						Size_:                           sdk.ZeroDec(),
						LastUpdatedBlockNumber:          1,
						LatestCumulativePremiumFraction: sdk.MustNewDecFromStr("0.0002"),
					},
					PositionNotional:  sdk.ZeroDec(),
					RealizedPnl:       sdk.MustNewDecFromStr("-100.000098999999010000"),
					BadDebt:           sdk.NewCoin(types.TestingCollateralDenomNUSD, sdk.ZeroInt()),
					FundingPayment:    sdk.NewDec(2),
					TransactionFee:    sdk.NewCoin(types.TestingCollateralDenomNUSD, sdk.NewInt(2)),
					BlockHeight:       1,
					MarginToUser:      sdk.NewInt(896),
					ChangeReason:      types.ChangeReason_ClosePosition,
					ExchangedNotional: sdk.MustNewDecFromStr("-10000.000000000000000000"),
					ExchangedSize:     sdk.MustNewDecFromStr("-10000.000000000000000000"),
				}),
			),

		TC("close long position with bad debt").
			Given(
				CreateCustomMarket(pairBtcNusd,
					WithEnabled(true),
					WithPricePeg(sdk.MustNewDecFromStr("0.89")),
					WithLatestMarketCPF(sdk.MustNewDecFromStr("0.0002")),
				),
				SetBlockNumber(1),
				SetBlockTime(startBlockTime),
				InsertPosition(
					WithTrader(alice),
					WithPair(pairBtcNusd),
					WithSize(sdk.NewDec(10_000)),
					WithMargin(sdk.NewDec(1_000)),
					WithOpenNotional(sdk.NewDec(10_000)),
				),
				FundAccount(alice, sdk.NewCoins(sdk.NewInt64Coin(types.TestingCollateralDenomNUSD, 18))),
				FundModule(types.VaultModuleAccount, sdk.NewCoins(sdk.NewInt64Coin(types.TestingCollateralDenomNUSD, 1000))),
				FundModule(types.PerpFundModuleAccount, sdk.NewCoins(sdk.NewInt64Coin(types.TestingCollateralDenomNUSD, 102))),
			).
			When(
				MoveToNextBlock(),
				ClosePosition(alice, pairBtcNusd),
			).
			Then(
				PositionShouldNotExist(alice, pairBtcNusd, 1),
				PositionChangedEventShouldBeEqual(&types.PositionChangedEvent{
					FinalPosition: types.Position{
						TraderAddress:                   alice.String(),
						Pair:                            pairBtcNusd,
						Size_:                           sdk.ZeroDec(),
						Margin:                          sdk.ZeroDec(),
						OpenNotional:                    sdk.ZeroDec(),
						LatestCumulativePremiumFraction: sdk.MustNewDecFromStr("0.0002"),
						LastUpdatedBlockNumber:          2,
					},
					PositionNotional:  sdk.ZeroDec(),
					TransactionFee:    sdk.NewInt64Coin(types.TestingCollateralDenomNUSD, 0),
					RealizedPnl:       sdk.MustNewDecFromStr("-1100.000088999999110000"),
					BadDebt:           sdk.NewCoin(types.TestingCollateralDenomNUSD, sdk.NewInt(102)),
					FundingPayment:    sdk.NewDec(2),
					BlockHeight:       2,
					MarginToUser:      sdk.NewInt(0),
					ChangeReason:      types.ChangeReason_ClosePosition,
					ExchangedNotional: sdk.MustNewDecFromStr("-10000.000000000000000000"),
					ExchangedSize:     sdk.MustNewDecFromStr("-10000.000000000000000000"),
				}),
				ModuleBalanceEqual(types.VaultModuleAccount, types.TestingCollateralDenomNUSD, sdk.NewInt(1102)), // 1000 + 102 from perp ef
				ModuleBalanceEqual(types.PerpFundModuleAccount, types.TestingCollateralDenomNUSD, sdk.NewInt(0)),
			),

		TC("close short position with positive PnL").
			Given(
				CreateCustomMarket(pairBtcNusd,
					WithEnabled(true),
					WithPricePeg(sdk.MustNewDecFromStr("0.10")),
					WithLatestMarketCPF(sdk.MustNewDecFromStr("0.0002")),
				),
				SetBlockTime(startBlockTime),
				SetBlockNumber(1),
				FundAccount(alice, sdk.NewCoins(sdk.NewCoin(types.TestingCollateralDenomNUSD, sdk.NewInt(2)))),
				FundModule(types.VaultModuleAccount, sdk.NewCoins(sdk.NewInt64Coin(types.TestingCollateralDenomNUSD, 10_002))),
				InsertPosition(
					WithTrader(alice),
					WithPair(pairBtcNusd),
					WithSize(sdk.NewDec(-10_000)),
					WithMargin(sdk.NewDec(1_000)),
					WithOpenNotional(sdk.NewDec(10_000)),
				),
			).
			When(
				ClosePosition(alice, pairBtcNusd),
			).
			Then(
				PositionShouldNotExist(alice, pairBtcNusd, 1),
				PositionChangedEventShouldBeEqual(&types.PositionChangedEvent{
					FinalPosition: types.Position{
						Pair:                            pairBtcNusd,
						TraderAddress:                   alice.String(),
						Margin:                          sdk.ZeroDec(),
						OpenNotional:                    sdk.ZeroDec(),
						Size_:                           sdk.ZeroDec(),
						LastUpdatedBlockNumber:          1,
						LatestCumulativePremiumFraction: sdk.MustNewDecFromStr("0.0002"),
					},
					PositionNotional:  sdk.ZeroDec(),
					RealizedPnl:       sdk.MustNewDecFromStr("8999.999989999999900000"),
					BadDebt:           sdk.NewCoin(types.TestingCollateralDenomNUSD, sdk.ZeroInt()),
					FundingPayment:    sdk.NewDec(-2),
					TransactionFee:    sdk.NewCoin(types.TestingCollateralDenomNUSD, sdk.NewInt(20)),
					BlockHeight:       1,
					MarginToUser:      sdk.NewInt(9982),
					ChangeReason:      types.ChangeReason_ClosePosition,
					ExchangedNotional: sdk.MustNewDecFromStr("-10000.000000000000000000"),
					ExchangedSize:     sdk.MustNewDecFromStr("10000.000000000000000000"),
				}),
			),

		TC("close short position with negative PnL").
			Given(
				CreateCustomMarket(pairBtcNusd,
					WithEnabled(true),
					WithPricePeg(sdk.MustNewDecFromStr("1.01")),
					WithLatestMarketCPF(sdk.MustNewDecFromStr("0.0002")),
				),
				SetBlockTime(startBlockTime),
				SetBlockNumber(1),
				FundAccount(alice, sdk.NewCoins(sdk.NewCoin(types.TestingCollateralDenomNUSD, sdk.NewInt(20)))),
				FundModule(types.VaultModuleAccount, sdk.NewCoins(sdk.NewInt64Coin(types.TestingCollateralDenomNUSD, 902))),
				InsertPosition(
					WithTrader(alice),
					WithPair(pairBtcNusd),
					WithSize(sdk.NewDec(-10_000)),
					WithMargin(sdk.NewDec(1_000)),
					WithOpenNotional(sdk.NewDec(10_000)),
				),
			).
			When(
				ClosePosition(alice, pairBtcNusd),
			).
			Then(
				PositionShouldNotExist(alice, pairBtcNusd, 1),
				PositionChangedEventShouldBeEqual(&types.PositionChangedEvent{
					FinalPosition: types.Position{
						Pair:                            pairBtcNusd,
						TraderAddress:                   alice.String(),
						Margin:                          sdk.ZeroDec(),
						OpenNotional:                    sdk.ZeroDec(),
						Size_:                           sdk.ZeroDec(),
						LastUpdatedBlockNumber:          1,
						LatestCumulativePremiumFraction: sdk.MustNewDecFromStr("0.0002"),
					},
					PositionNotional:  sdk.ZeroDec(),
					RealizedPnl:       sdk.MustNewDecFromStr("-100.000101000001010000"),
					BadDebt:           sdk.NewCoin(types.TestingCollateralDenomNUSD, sdk.ZeroInt()),
					FundingPayment:    sdk.NewDec(-2),
					TransactionFee:    sdk.NewCoin(types.TestingCollateralDenomNUSD, sdk.NewInt(2)),
					BlockHeight:       1,
					MarginToUser:      sdk.NewInt(900),
					ChangeReason:      types.ChangeReason_ClosePosition,
					ExchangedNotional: sdk.MustNewDecFromStr("-10000.000000000000000000"),
					ExchangedSize:     sdk.MustNewDecFromStr("10000.000000000000000000"),
				}),
			),

		TC("close short position with bad debt").
			Given(
				CreateCustomMarket(pairBtcNusd,
					WithEnabled(true),
					WithPricePeg(sdk.MustNewDecFromStr("1.11")),
					WithLatestMarketCPF(sdk.MustNewDecFromStr("0.0002")),
				),
				SetBlockTime(startBlockTime),
				SetBlockNumber(1),
				InsertPosition(
					WithTrader(alice),
					WithPair(pairBtcNusd),
					WithSize(sdk.NewDec(-10_000)),
					WithMargin(sdk.NewDec(1_000)),
					WithOpenNotional(sdk.NewDec(10_000)),
				),
				FundAccount(alice, sdk.NewCoins(sdk.NewInt64Coin(types.TestingCollateralDenomNUSD, 22))),
				FundModule(types.VaultModuleAccount, sdk.NewCoins(sdk.NewInt64Coin(types.TestingCollateralDenomNUSD, 1000))),
				FundModule(types.PerpFundModuleAccount, sdk.NewCoins(sdk.NewInt64Coin(types.TestingCollateralDenomNUSD, 98))),
			).
			When(
				MoveToNextBlock(),
				ClosePosition(alice, pairBtcNusd),
			).
			Then(
				PositionShouldNotExist(alice, pairBtcNusd, 1),
				PositionChangedEventShouldBeEqual(&types.PositionChangedEvent{
					FinalPosition: types.Position{
						TraderAddress:                   alice.String(),
						Pair:                            pairBtcNusd,
						Size_:                           sdk.ZeroDec(),
						Margin:                          sdk.ZeroDec(),
						OpenNotional:                    sdk.ZeroDec(),
						LatestCumulativePremiumFraction: sdk.MustNewDecFromStr("0.0002"),
						LastUpdatedBlockNumber:          2,
					},
					PositionNotional:  sdk.ZeroDec(),
					TransactionFee:    sdk.NewInt64Coin(types.TestingCollateralDenomNUSD, 0),
					RealizedPnl:       sdk.MustNewDecFromStr("-1100.000111000001110000"),
					BadDebt:           sdk.NewInt64Coin(types.TestingCollateralDenomNUSD, 98),
					FundingPayment:    sdk.NewDec(-2),
					BlockHeight:       2,
					MarginToUser:      sdk.NewInt(0),
					ChangeReason:      types.ChangeReason_ClosePosition,
					ExchangedNotional: sdk.MustNewDecFromStr("-10000.000000000000000000"),
					ExchangedSize:     sdk.MustNewDecFromStr("10000.000000000000000000"),
				}),
				ModuleBalanceEqual(types.VaultModuleAccount, types.TestingCollateralDenomNUSD, sdk.NewInt(1098)), // 1000 + 98 from perp ef
				ModuleBalanceEqual(types.PerpFundModuleAccount, types.TestingCollateralDenomNUSD, sdk.NewInt(0)),
			),
	}

	NewTestSuite(t).WithTestCases(tc...).Run()
}

func TestUpdateSwapInvariant(t *testing.T) {
	alice := testutil.AccAddress()
	bob := testutil.AccAddress()
	pairBtcNusd := asset.Registry.Pair(denoms.BTC, denoms.NUSD)
	startBlockTime := time.Now()

	startingSwapInvariant := sdk.NewInt(1e12).Mul(sdk.NewInt(1e12))

	tc := TestCases{
		TC("only long position - no change to swap invariant").
			Given(
				CreateCustomMarket(pairBtcNusd, WithEnabled(true)),
				SetBlockTime(startBlockTime),
				SetBlockNumber(1),
				FundAccount(alice, sdk.NewCoins(sdk.NewCoin(types.TestingCollateralDenomNUSD, sdk.NewInt(10_200_000_000)))),
				FundModule(types.PerpFundModuleAccount, sdk.NewCoins(sdk.NewCoin(types.TestingCollateralDenomNUSD, sdk.NewInt(100000)))),
			).
			When(
				MarketOrder(alice, pairBtcNusd, types.Direction_LONG, sdk.NewInt(10_000_000_000), sdk.OneDec(), sdk.ZeroDec()),
				ClosePosition(alice, pairBtcNusd),
			).
			Then(
				PositionShouldNotExist(alice, pairBtcNusd, 1),
			),
		TC("only short position - no change to swap invariant").
			Given(
				CreateCustomMarket(pairBtcNusd, WithEnabled(true)),
				SetBlockTime(startBlockTime),
				SetBlockNumber(1),
				FundAccount(alice, sdk.NewCoins(sdk.NewCoin(types.TestingCollateralDenomNUSD, sdk.NewInt(10_200_000_000)))),
				FundModule(types.PerpFundModuleAccount, sdk.NewCoins(sdk.NewCoin(types.TestingCollateralDenomNUSD, sdk.NewInt(100000)))),
			).
			When(
				MarketOrder(alice, pairBtcNusd, types.Direction_SHORT, sdk.NewInt(10_000_000_000), sdk.OneDec(), sdk.NewDec(10_000_000_000_000)),
				ClosePosition(alice, pairBtcNusd),
			).
			Then(
				PositionShouldNotExist(alice, pairBtcNusd, 1),
			),
		TC("only long position - increasing k").
			Given(
				CreateCustomMarket(pairBtcNusd, WithEnabled(true)),
				SetBlockTime(startBlockTime),
				SetBlockNumber(1),
				FundAccount(alice, sdk.NewCoins(sdk.NewCoin(types.TestingCollateralDenomNUSD, sdk.NewInt(10_200_000_000)))),
				FundModule(types.PerpFundModuleAccount, sdk.NewCoins(sdk.NewCoin(types.TestingCollateralDenomNUSD, sdk.NewInt(100_000_000)))),
			).
			When(
				MarketOrder(alice, pairBtcNusd, types.Direction_LONG, sdk.NewInt(10_000_000_000), sdk.OneDec(), sdk.ZeroDec()),
				ShiftSwapInvariant(pairBtcNusd, startingSwapInvariant.MulRaw(100)),
				AMMShouldBeEqual(
					pairBtcNusd,
					AMM_SwapInvariantShouldBeEqual(sdk.MustNewDecFromStr("100000000000000000000000000.000005462000000000"))),
				ClosePosition(alice, pairBtcNusd),
			).
			Then(
				PositionShouldNotExist(alice, pairBtcNusd, 1),
				ModuleBalanceShouldBeEqualTo(types.VaultModuleAccount, sdk.NewCoins(sdk.NewCoin(types.TestingCollateralDenomNUSD, sdk.OneInt()))),
			),
		TC("only short position - increasing k").
			Given(
				CreateCustomMarket(pairBtcNusd, WithEnabled(true)),
				SetBlockTime(startBlockTime),
				SetBlockNumber(1),
				FundAccount(alice, sdk.NewCoins(sdk.NewCoin(types.TestingCollateralDenomNUSD, sdk.NewInt(10_200_000_000)))),
				FundModule(types.PerpFundModuleAccount, sdk.NewCoins(sdk.NewCoin(types.TestingCollateralDenomNUSD, sdk.NewInt(100_000_000)))),
			).
			When(
				MarketOrder(alice, pairBtcNusd, types.Direction_SHORT, sdk.NewInt(10_000_000_000), sdk.OneDec(), sdk.ZeroDec()),
				ShiftSwapInvariant(pairBtcNusd, startingSwapInvariant.MulRaw(100)),
				AMMShouldBeEqual(
					pairBtcNusd,
					AMM_SwapInvariantShouldBeEqual(sdk.MustNewDecFromStr("99999999999999999999999999.999996174000000000"))),
				ClosePosition(alice, pairBtcNusd),
			).
			Then(
				PositionShouldNotExist(alice, pairBtcNusd, 1),
				ModuleBalanceShouldBeEqualTo(types.VaultModuleAccount, sdk.NewCoins(sdk.NewCoin(types.TestingCollateralDenomNUSD, sdk.OneInt()))),
			),

		TC("only long position - decreasing k").
			Given(
				CreateCustomMarket(pairBtcNusd, WithEnabled(true)),
				SetBlockTime(startBlockTime),
				SetBlockNumber(1),
				FundAccount(alice, sdk.NewCoins(sdk.NewCoin(types.TestingCollateralDenomNUSD, sdk.NewInt(10_200_000_000)))),
				FundModule(types.PerpFundModuleAccount, sdk.NewCoins(sdk.NewCoin(types.TestingCollateralDenomNUSD, sdk.NewInt(100_000_000)))),
			).
			When(
				MarketOrder(alice, pairBtcNusd, types.Direction_LONG, sdk.NewInt(10_000_000_000), sdk.OneDec(), sdk.ZeroDec()),
				ShiftSwapInvariant(
					pairBtcNusd,
					startingSwapInvariant.QuoRaw(10)),
				AMMShouldBeEqual(
					pairBtcNusd,
					AMM_SwapInvariantShouldBeEqual(sdk.MustNewDecFromStr("99999999999999999873578.871987765741755797"))),
				ClosePosition(alice, pairBtcNusd),
			).
			Then(
				PositionShouldNotExist(alice, pairBtcNusd, 1),
				ModuleBalanceShouldBeEqualTo(types.VaultModuleAccount, sdk.NewCoins()),
			),
		TC("only short position - decreasing k").
			Given(
				CreateCustomMarket(pairBtcNusd, WithEnabled(true)),
				SetBlockTime(startBlockTime),
				SetBlockNumber(1),
				FundAccount(alice, sdk.NewCoins(sdk.NewCoin(types.TestingCollateralDenomNUSD, sdk.NewInt(10_200_000_000)))),
				FundModule(types.PerpFundModuleAccount, sdk.NewCoins(sdk.NewCoin(types.TestingCollateralDenomNUSD, sdk.NewInt(100_000_000)))),
			).
			When(
				MarketOrder(alice, pairBtcNusd, types.Direction_SHORT, sdk.NewInt(10_000_000_000), sdk.OneDec(), sdk.ZeroDec()),
				ShiftSwapInvariant(pairBtcNusd,
					startingSwapInvariant.QuoRaw(10)),
				AMMShouldBeEqual(
					pairBtcNusd,
					AMM_SwapInvariantShouldBeEqual(sdk.MustNewDecFromStr("99999999999999999873578.871987864651476452"))),
				ClosePosition(alice, pairBtcNusd),
			).
			Then(
				PositionShouldNotExist(alice, pairBtcNusd, 1),
				ModuleBalanceShouldBeEqualTo(types.VaultModuleAccount, sdk.NewCoins()),
			),

		TC("long and short position - increasing k").
			Given(
				CreateCustomMarket(pairBtcNusd, WithEnabled(true)),
				SetBlockTime(startBlockTime),
				SetBlockNumber(1),
				FundAccount(alice, sdk.NewCoins(sdk.NewCoin(types.TestingCollateralDenomNUSD, sdk.NewInt(10_200_000_000)))),
				FundAccount(bob, sdk.NewCoins(sdk.NewCoin(types.TestingCollateralDenomNUSD, sdk.NewInt(10_200_000_000)))),
			).
			When(
				MarketOrder(alice, pairBtcNusd, types.Direction_LONG, sdk.NewInt(10_000_000_000), sdk.OneDec(), sdk.ZeroDec()),
				MarketOrder(bob, pairBtcNusd, types.Direction_SHORT, sdk.NewInt(10_000_000_000), sdk.OneDec(), sdk.NewDec(10_000_000_000_000)),

				ShiftSwapInvariant(pairBtcNusd,
					startingSwapInvariant.MulRaw(100)),
				AMMShouldBeEqual(
					pairBtcNusd,
					AMM_BiasShouldBeEqual(sdk.ZeroDec()),
					AMM_SwapInvariantShouldBeEqual(sdk.MustNewDecFromStr("100000000000000000000000000.000000000000000000"))),

				ModuleBalanceShouldBeEqualTo(types.VaultModuleAccount, sdk.NewCoins(sdk.NewCoin(types.TestingCollateralDenomNUSD, sdk.NewInt(19_960_000_000)))),
				ModuleBalanceShouldBeEqualTo(types.FeePoolModuleAccount, sdk.NewCoins(sdk.NewCoin(types.TestingCollateralDenomNUSD, sdk.NewInt(20_000_000)))), // Fees are 10_000_000_000 * 0.0010 * 2

				ClosePosition(alice, pairBtcNusd),
				ClosePosition(bob, pairBtcNusd),
			).
			Then(
				PositionShouldNotExist(alice, pairBtcNusd, 1),
				PositionShouldNotExist(bob, pairBtcNusd, 1),

				ModuleBalanceShouldBeEqualTo(types.VaultModuleAccount, sdk.NewCoins()),
				ModuleBalanceShouldBeEqualTo(types.PerpFundModuleAccount, sdk.NewCoins(sdk.NewCoin(types.TestingCollateralDenomNUSD, sdk.NewInt(39_960_000)))),
			),
		TC("long and short position - reducing k").
			Given(
				CreateCustomMarket(pairBtcNusd, WithEnabled(true)),
				SetBlockTime(startBlockTime),
				SetBlockNumber(1),
				FundAccount(alice, sdk.NewCoins(sdk.NewCoin(types.TestingCollateralDenomNUSD, sdk.NewInt(10_200_000_000)))),
				FundAccount(bob, sdk.NewCoins(sdk.NewCoin(types.TestingCollateralDenomNUSD, sdk.NewInt(10_200_000_000)))),
			).
			When(
				MarketOrder(alice, pairBtcNusd, types.Direction_LONG, sdk.NewInt(10_000_000_000), sdk.OneDec(), sdk.ZeroDec()),
				MarketOrder(bob, pairBtcNusd, types.Direction_SHORT, sdk.NewInt(10_000_000_000), sdk.OneDec(), sdk.NewDec(10_000_000_000_000)),

				ShiftSwapInvariant(pairBtcNusd,
					startingSwapInvariant.QuoRaw(10)),
				AMMShouldBeEqual(
					pairBtcNusd,
					AMM_BiasShouldBeEqual(sdk.ZeroDec()),
					AMM_SwapInvariantShouldBeEqual(sdk.MustNewDecFromStr("99999999999999999873578.871987712489000000"))),

				ModuleBalanceShouldBeEqualTo(types.VaultModuleAccount, sdk.NewCoins(sdk.NewCoin(types.TestingCollateralDenomNUSD, sdk.NewInt(19_960_000_000)))),
				ModuleBalanceShouldBeEqualTo(types.FeePoolModuleAccount, sdk.NewCoins(sdk.NewCoin(types.TestingCollateralDenomNUSD, sdk.NewInt(20_000_000)))), // Fees are 10_000_000_000 * 0.0010 * 2

				ClosePosition(alice, pairBtcNusd),
				ClosePosition(bob, pairBtcNusd),
			).
			Then(
				PositionShouldNotExist(alice, pairBtcNusd, 1),
				PositionShouldNotExist(bob, pairBtcNusd, 1),

				ModuleBalanceShouldBeEqualTo(types.VaultModuleAccount, sdk.NewCoins()),
				ModuleBalanceShouldBeEqualTo(types.PerpFundModuleAccount, sdk.NewCoins(sdk.NewCoin(types.TestingCollateralDenomNUSD, sdk.NewInt(39_960_000)))),
			),
	}

	NewTestSuite(t).WithTestCases(tc...).Run()
}
