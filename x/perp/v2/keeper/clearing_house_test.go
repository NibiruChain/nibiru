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
					WithPricePeg(sdk.OneDec()),
					WithSqrtDepth(sdk.NewDec(100_000)),
				),
				SetBlockNumber(1),
				SetBlockTime(startBlockTime),

				FundAccount(alice, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(10_200)))),
				FundAccount(bob, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(10_200)))),
				FundModule(types.PerpEFModuleAccount, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(100_000_000)))),

				MarketOrder(alice, pairBtcNusd, types.Direction_SHORT, sdk.NewInt(10_000), sdk.OneDec(), sdk.ZeroDec()),
				MarketOrder(bob, pairBtcNusd, types.Direction_LONG, sdk.NewInt(10_000), sdk.OneDec(), sdk.ZeroDec()),

				EditSwapInvariant(pairBtcNusd, sdk.OneDec()),
			).
			When(
				PartialCloseFails(alice, pairBtcNusd, sdk.NewDec(5_000), types.ErrBaseReserveAtZero),
			).
			Then(
				ClosePosition(bob, pairBtcNusd),
				PartialClose(alice, pairBtcNusd, sdk.NewDec(5_000)),
			),

		TC("new long position").
			Given(
				CreateCustomMarket(pairBtcNusd),
				SetBlockTime(startBlockTime),
				SetBlockNumber(1),
				FundAccount(alice, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(1020)))),
			).
			When(
				MarketOrder(alice, pairBtcNusd, types.Direction_LONG, sdk.NewInt(1000), sdk.NewDec(10), sdk.ZeroDec(),
					MarketOrderResp_PositionShouldBeEqual(
						types.Position{
							Pair:                            pairBtcNusd,
							TraderAddress:                   alice.String(),
							Margin:                          sdk.NewDec(1000),
							OpenNotional:                    sdk.NewDec(10_000),
							Size_:                           sdk.MustNewDecFromStr("9999.999900000001"),
							LastUpdatedBlockNumber:          1,
							LatestCumulativePremiumFraction: sdk.ZeroDec(),
						}),
					MarketOrderResp_ExchangeNotionalValueShouldBeEqual(sdk.NewDec(10_000)), // margin * leverage
					MarketOrderResp_ExchangedPositionSizeShouldBeEqual(sdk.MustNewDecFromStr("9999.999900000001")),
					MarketOrderResp_BadDebtShouldBeEqual(sdk.ZeroDec()),
					MarketOrderResp_FundingPaymentShouldBeEqual(sdk.ZeroDec()),
					MarketOrderResp_RealizedPnlShouldBeEqual(sdk.ZeroDec()),
					MarketOrderResp_UnrealizedPnlAfterShouldBeEqual(sdk.ZeroDec()),
					MarketOrderResp_MarginToVaultShouldBeEqual(sdk.NewDec(1000)),
					MarketOrderResp_PositionNotionalShouldBeEqual(sdk.NewDec(10_000)),
				),
			).
			Then(
				PositionShouldBeEqual(alice, pairBtcNusd, Position_PositionShouldBeEqualTo(types.Position{
					Pair:                            pairBtcNusd,
					TraderAddress:                   alice.String(),
					Margin:                          sdk.NewDec(1000),
					OpenNotional:                    sdk.NewDec(10_000),
					Size_:                           sdk.MustNewDecFromStr("9999.999900000001"),
					LastUpdatedBlockNumber:          1,
					LatestCumulativePremiumFraction: sdk.ZeroDec(),
				})),
				PositionChangedEventShouldBeEqual(&types.PositionChangedEvent{
					FinalPosition: types.Position{
						Pair:                            pairBtcNusd,
						TraderAddress:                   alice.String(),
						Margin:                          sdk.NewDec(1000),
						OpenNotional:                    sdk.NewDec(10_000),
						Size_:                           sdk.MustNewDecFromStr("9999.999900000001"),
						LastUpdatedBlockNumber:          1,
						LatestCumulativePremiumFraction: sdk.ZeroDec(),
					},
					PositionNotional: sdk.NewDec(10_000),
					RealizedPnl:      sdk.ZeroDec(),
					BadDebt:          sdk.NewCoin(denoms.NUSD, sdk.ZeroInt()),
					FundingPayment:   sdk.ZeroDec(),
					TransactionFee:   sdk.NewCoin(denoms.NUSD, sdk.NewInt(20)),
					BlockHeight:      1,
					// exchangedMargin = - marginToVault - transferredFee
					MarginToUser:      sdk.NewInt(1_000 + 20).Neg(),
					ChangeReason:      types.ChangeReason_MarketOrder,
					ExchangedNotional: sdk.MustNewDecFromStr("10000.000000000000000000"),
					ExchangedSize:     sdk.MustNewDecFromStr("9999.999900000001000000"),
				}),
			),

		TC("existing long position, go more long").
			Given(
				SetBlockNumber(1),
				SetBlockTime(startBlockTime),
				CreateCustomMarket(pairBtcNusd),
				FundAccount(alice, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(2040)))),
				MarketOrder(alice, pairBtcNusd, types.Direction_LONG, sdk.NewInt(1000), sdk.NewDec(10), sdk.ZeroDec()),
			).
			When(
				MoveToNextBlock(),
				MarketOrder(alice, pairBtcNusd, types.Direction_LONG, sdk.NewInt(1000), sdk.NewDec(10), sdk.ZeroDec(),
					MarketOrderResp_PositionShouldBeEqual(
						types.Position{
							Pair:                            pairBtcNusd,
							TraderAddress:                   alice.String(),
							Margin:                          sdk.NewDec(2000),
							OpenNotional:                    sdk.NewDec(20_000),
							Size_:                           sdk.MustNewDecFromStr("19999.999600000008000000"),
							LastUpdatedBlockNumber:          2,
							LatestCumulativePremiumFraction: sdk.ZeroDec(),
						}),
					MarketOrderResp_ExchangeNotionalValueShouldBeEqual(sdk.NewDec(10_000)), // margin * leverage
					MarketOrderResp_ExchangedPositionSizeShouldBeEqual(sdk.MustNewDecFromStr("9999.999700000007000000")),
					MarketOrderResp_BadDebtShouldBeEqual(sdk.ZeroDec()),
					MarketOrderResp_FundingPaymentShouldBeEqual(sdk.ZeroDec()),
					MarketOrderResp_RealizedPnlShouldBeEqual(sdk.ZeroDec()),
					MarketOrderResp_UnrealizedPnlAfterShouldBeEqual(sdk.ZeroDec()),
					MarketOrderResp_MarginToVaultShouldBeEqual(sdk.NewDec(1000)),
					MarketOrderResp_PositionNotionalShouldBeEqual(sdk.NewDec(20_000)),
				),
			).
			Then(
				PositionChangedEventShouldBeEqual(&types.PositionChangedEvent{
					FinalPosition: types.Position{
						Pair:                            pairBtcNusd,
						TraderAddress:                   alice.String(),
						Margin:                          sdk.NewDec(2000),
						OpenNotional:                    sdk.NewDec(20_000),
						Size_:                           sdk.MustNewDecFromStr("19999.999600000008000000"),
						LastUpdatedBlockNumber:          2,
						LatestCumulativePremiumFraction: sdk.ZeroDec(),
					},
					PositionNotional: sdk.NewDec(20_000),
					RealizedPnl:      sdk.ZeroDec(),
					BadDebt:          sdk.NewCoin(denoms.NUSD, sdk.ZeroInt()),
					FundingPayment:   sdk.ZeroDec(),
					TransactionFee:   sdk.NewCoin(denoms.NUSD, sdk.NewInt(20)), // 20 bps
					BlockHeight:      2,
					// exchangedMargin = - marginToVault - transferredFee
					MarginToUser:      sdk.NewInt(1_000 + 20).Neg(),
					ChangeReason:      types.ChangeReason_MarketOrder,
					ExchangedNotional: sdk.MustNewDecFromStr("10000.000000000000000000"),
					ExchangedSize:     sdk.MustNewDecFromStr("9999.999700000007000000"),
				}),
			),

		TC("existing long position, go more long but there's bad debt").
			Given(
				CreateCustomMarket(
					pairBtcNusd,
					WithPricePeg(sdk.MustNewDecFromStr("0.89")),
					WithLatestMarketCPF(sdk.MustNewDecFromStr("0.0002")),
				),
				SetBlockNumber(1),
				SetBlockTime(startBlockTime),
				FundAccount(alice, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(18)))),
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
					pairBtcNusd,
					WithPricePeg(sdk.MustNewDecFromStr("0.89")),
					WithLatestMarketCPF(sdk.MustNewDecFromStr("0.0002")),
				),
				SetBlockNumber(1),
				SetBlockTime(startBlockTime),
				FundAccount(alice, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(18)))),
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
					pairBtcNusd,
					WithPricePeg(sdk.OneDec()),
					WithSqrtDepth(sdk.NewDec(100_000)),
				),
				SetBlockNumber(1),
				SetBlockTime(startBlockTime),
				FundAccount(alice, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(10_000)))),
				MarketOrder(alice, pairBtcNusd, types.Direction_LONG, sdk.NewInt(9_000), sdk.NewDec(10), sdk.ZeroDec()),
				FundModule(types.PerpEFModuleAccount, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(100_000_000)))),
				EditSwapInvariant(pairBtcNusd, sdk.OneDec()),
			).
			When(
				ClosePosition(alice, pairBtcNusd),
			).
			Then(
				PositionShouldNotExist(alice, pairBtcNusd),
			),

		TC("existing long position, decrease a bit").
			Given(
				CreateCustomMarket(pairBtcNusd),
				SetBlockNumber(1),
				SetBlockTime(startBlockTime),
				FundAccount(alice, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(1030)))),
				MarketOrder(alice, pairBtcNusd, types.Direction_LONG, sdk.NewInt(1000), sdk.NewDec(10), sdk.ZeroDec()),
			).
			When(
				MoveToNextBlock(),
				MarketOrder(alice, pairBtcNusd, types.Direction_SHORT, sdk.NewInt(500), sdk.NewDec(10), sdk.ZeroDec(),
					MarketOrderResp_PositionShouldBeEqual(
						types.Position{
							Pair:                            pairBtcNusd,
							TraderAddress:                   alice.String(),
							Margin:                          sdk.NewDec(1000),
							OpenNotional:                    sdk.NewDec(5000),
							Size_:                           sdk.MustNewDecFromStr("4999.999975000000125000"),
							LastUpdatedBlockNumber:          2,
							LatestCumulativePremiumFraction: sdk.ZeroDec(),
						},
					),
					MarketOrderResp_ExchangeNotionalValueShouldBeEqual(sdk.NewDec(5000)), // margin * leverage
					MarketOrderResp_ExchangedPositionSizeShouldBeEqual(sdk.MustNewDecFromStr("-4999.999925000000875000")),
					MarketOrderResp_BadDebtShouldBeEqual(sdk.ZeroDec()),
					MarketOrderResp_FundingPaymentShouldBeEqual(sdk.ZeroDec()),
					MarketOrderResp_RealizedPnlShouldBeEqual(sdk.ZeroDec()),
					MarketOrderResp_UnrealizedPnlAfterShouldBeEqual(sdk.ZeroDec()),
					MarketOrderResp_MarginToVaultShouldBeEqual(sdk.ZeroDec()),
					MarketOrderResp_PositionNotionalShouldBeEqual(sdk.NewDec(5000)),
				),
			).
			Then(
				PositionChangedEventShouldBeEqual(&types.PositionChangedEvent{
					FinalPosition: types.Position{
						Pair:                            pairBtcNusd,
						TraderAddress:                   alice.String(),
						Margin:                          sdk.NewDec(1000),
						OpenNotional:                    sdk.NewDec(5000),
						Size_:                           sdk.MustNewDecFromStr("4999.999975000000125000"),
						LastUpdatedBlockNumber:          2,
						LatestCumulativePremiumFraction: sdk.ZeroDec(),
					},
					PositionNotional: sdk.NewDec(5000),
					RealizedPnl:      sdk.ZeroDec(),
					BadDebt:          sdk.NewCoin(denoms.NUSD, sdk.ZeroInt()),
					FundingPayment:   sdk.ZeroDec(),
					TransactionFee:   sdk.NewCoin(denoms.NUSD, sdk.NewInt(10)), // 20 bps
					BlockHeight:      2,
					// exchangedMargin = - marginToVault - transferredFee
					MarginToUser:      sdk.NewInt(0 + 10).Neg(),
					ChangeReason:      types.ChangeReason_MarketOrder,
					ExchangedNotional: sdk.MustNewDecFromStr("-5000.000000000000000000"),
					ExchangedSize:     sdk.MustNewDecFromStr("-4999.999925000000875000"),
				}),
			),

		TC("existing long position, decrease a bit but there's bad debt").
			Given(
				CreateCustomMarket(
					pairBtcNusd,
					WithPricePeg(sdk.MustNewDecFromStr("0.89")),
					WithLatestMarketCPF(sdk.MustNewDecFromStr("0.0002")),
				),
				SetBlockNumber(1),
				SetBlockTime(startBlockTime),
				FundAccount(alice, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(18)))),
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
				CreateCustomMarket(pairBtcNusd),
				SetBlockNumber(1),
				SetBlockTime(startBlockTime),
				FundAccount(alice, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(4080)))),
				MarketOrder(alice, pairBtcNusd, types.Direction_LONG, sdk.NewInt(1000), sdk.NewDec(10), sdk.ZeroDec()),
			).
			When(
				MoveToNextBlock(),
				MarketOrder(alice, pairBtcNusd, types.Direction_SHORT, sdk.NewInt(3000), sdk.NewDec(10), sdk.ZeroDec(),
					MarketOrderResp_PositionShouldBeEqual(
						types.Position{
							Pair:                            pairBtcNusd,
							TraderAddress:                   alice.String(),
							Margin:                          sdk.NewDec(2000),
							OpenNotional:                    sdk.NewDec(20000),
							Size_:                           sdk.MustNewDecFromStr("-20000.000400000008000000"),
							LastUpdatedBlockNumber:          2,
							LatestCumulativePremiumFraction: sdk.ZeroDec(),
						},
					),
					MarketOrderResp_ExchangeNotionalValueShouldBeEqual(sdk.NewDec(30000)), // margin * leverage
					MarketOrderResp_ExchangedPositionSizeShouldBeEqual(sdk.MustNewDecFromStr("-30000.000300000009000000")),
					MarketOrderResp_BadDebtShouldBeEqual(sdk.ZeroDec()),
					MarketOrderResp_FundingPaymentShouldBeEqual(sdk.ZeroDec()),
					MarketOrderResp_RealizedPnlShouldBeEqual(sdk.ZeroDec()),
					MarketOrderResp_UnrealizedPnlAfterShouldBeEqual(sdk.ZeroDec()),
					MarketOrderResp_MarginToVaultShouldBeEqual(sdk.NewDec(1000)),
					MarketOrderResp_PositionNotionalShouldBeEqual(sdk.NewDec(20000)),
				),
			).
			Then(
				PositionChangedEventShouldBeEqual(&types.PositionChangedEvent{
					FinalPosition: types.Position{
						Pair:                            pairBtcNusd,
						TraderAddress:                   alice.String(),
						Margin:                          sdk.NewDec(2000),
						OpenNotional:                    sdk.NewDec(20000),
						Size_:                           sdk.MustNewDecFromStr("-20000.000400000008000000"),
						LastUpdatedBlockNumber:          2,
						LatestCumulativePremiumFraction: sdk.ZeroDec(),
					},
					PositionNotional: sdk.NewDec(20000),
					RealizedPnl:      sdk.ZeroDec(),
					BadDebt:          sdk.NewCoin(denoms.NUSD, sdk.ZeroInt()),
					FundingPayment:   sdk.ZeroDec(),
					TransactionFee:   sdk.NewCoin(denoms.NUSD, sdk.NewInt(60)), // 20 bps
					BlockHeight:      2,
					// exchangedMargin = - marginToVault - transferredFee
					MarginToUser:      sdk.NewInt(1_000 + 60).Neg(),
					ChangeReason:      types.ChangeReason_MarketOrder,
					ExchangedNotional: sdk.MustNewDecFromStr("10000.000000000000000000"),
					ExchangedSize:     sdk.MustNewDecFromStr("-30000.000300000009000000"),
				}),
			),

		TC("existing long position, decrease a lot but there's bad debt").
			Given(
				CreateCustomMarket(
					pairBtcNusd,
					WithPricePeg(sdk.MustNewDecFromStr("0.89")),
					WithLatestMarketCPF(sdk.MustNewDecFromStr("0.0002")),
				),
				SetBlockNumber(1),
				SetBlockTime(startBlockTime),
				FundAccount(alice, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(18)))),
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
					BadDebt:           sdk.NewInt64Coin(denoms.NUSD, 102),
					FundingPayment:    sdk.NewDec(2),
					TransactionFee:    sdk.NewInt64Coin(denoms.NUSD, 18), // 20 bps
					BlockHeight:       2,
					MarginToUser:      sdk.NewInt(-18),
					ChangeReason:      types.ChangeReason_MarketOrder,
					ExchangedNotional: sdk.MustNewDecFromStr("-10000.000000000000000000"),
					ExchangedSize:     sdk.MustNewDecFromStr("-10000.000000000000000000"),
				}),
			),

		TC("new short position").
			Given(
				CreateCustomMarket(pairBtcNusd),
				SetBlockTime(startBlockTime),
				SetBlockNumber(1),
				FundAccount(alice, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(1020)))),
			).
			When(
				MarketOrder(alice, pairBtcNusd, types.Direction_SHORT, sdk.NewInt(1000), sdk.NewDec(10), sdk.ZeroDec(),
					MarketOrderResp_PositionShouldBeEqual(
						types.Position{
							Pair:                            pairBtcNusd,
							TraderAddress:                   alice.String(),
							Margin:                          sdk.NewDec(1000),
							OpenNotional:                    sdk.NewDec(10_000),
							Size_:                           sdk.MustNewDecFromStr("-10000.000100000001000000"),
							LastUpdatedBlockNumber:          1,
							LatestCumulativePremiumFraction: sdk.ZeroDec(),
						}),
					MarketOrderResp_ExchangeNotionalValueShouldBeEqual(sdk.NewDec(10_000)), // margin * leverage
					MarketOrderResp_ExchangedPositionSizeShouldBeEqual(sdk.MustNewDecFromStr("-10000.000100000001000000")),
					MarketOrderResp_BadDebtShouldBeEqual(sdk.ZeroDec()),
					MarketOrderResp_FundingPaymentShouldBeEqual(sdk.ZeroDec()),
					MarketOrderResp_RealizedPnlShouldBeEqual(sdk.ZeroDec()),
					MarketOrderResp_UnrealizedPnlAfterShouldBeEqual(sdk.ZeroDec()),
					MarketOrderResp_MarginToVaultShouldBeEqual(sdk.NewDec(1000)),
					MarketOrderResp_PositionNotionalShouldBeEqual(sdk.NewDec(10_000)),
				),
			).
			Then(
				PositionShouldBeEqual(alice, pairBtcNusd,
					Position_PositionShouldBeEqualTo(types.Position{
						Pair:                            pairBtcNusd,
						TraderAddress:                   alice.String(),
						Margin:                          sdk.NewDec(1000),
						OpenNotional:                    sdk.NewDec(10_000),
						Size_:                           sdk.MustNewDecFromStr("-10000.000100000001000000"),
						LastUpdatedBlockNumber:          1,
						LatestCumulativePremiumFraction: sdk.ZeroDec(),
					}),
				),
				PositionChangedEventShouldBeEqual(&types.PositionChangedEvent{
					FinalPosition: types.Position{
						Pair:                            pairBtcNusd,
						TraderAddress:                   alice.String(),
						Margin:                          sdk.NewDec(1000),
						OpenNotional:                    sdk.NewDec(10_000),
						Size_:                           sdk.MustNewDecFromStr("-10000.000100000001000000"),
						LastUpdatedBlockNumber:          1,
						LatestCumulativePremiumFraction: sdk.ZeroDec(),
					},
					PositionNotional: sdk.NewDec(10_000),
					RealizedPnl:      sdk.ZeroDec(),
					BadDebt:          sdk.NewCoin(denoms.NUSD, sdk.ZeroInt()),
					FundingPayment:   sdk.ZeroDec(),
					TransactionFee:   sdk.NewCoin(denoms.NUSD, sdk.NewInt(20)),
					BlockHeight:      1,
					// exchangedMargin = - marginToVault - transferredFee
					MarginToUser:      sdk.NewInt(1_000 + 20).Neg(),
					ChangeReason:      types.ChangeReason_MarketOrder,
					ExchangedNotional: sdk.MustNewDecFromStr("10000.000000000000000000"),
					ExchangedSize:     sdk.MustNewDecFromStr("-10000.000100000001000000"),
				}),
			),

		TC("existing short position, go more short").
			Given(
				CreateCustomMarket(pairBtcNusd),
				SetBlockNumber(1),
				SetBlockTime(startBlockTime),
				FundAccount(alice, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(2040)))),
				MarketOrder(alice, pairBtcNusd, types.Direction_SHORT, sdk.NewInt(1000), sdk.NewDec(10), sdk.ZeroDec()),
			).
			When(
				MoveToNextBlock(),
				MarketOrder(alice, pairBtcNusd, types.Direction_SHORT, sdk.NewInt(1000), sdk.NewDec(10), sdk.ZeroDec(),
					MarketOrderResp_PositionShouldBeEqual(
						types.Position{
							Pair:                            pairBtcNusd,
							TraderAddress:                   alice.String(),
							Margin:                          sdk.NewDec(2000),
							OpenNotional:                    sdk.NewDec(20_000),
							Size_:                           sdk.MustNewDecFromStr("-20000.000400000008000000"),
							LastUpdatedBlockNumber:          2,
							LatestCumulativePremiumFraction: sdk.ZeroDec(),
						},
					),
					MarketOrderResp_ExchangeNotionalValueShouldBeEqual(sdk.NewDec(10_000)), // margin * leverage
					MarketOrderResp_ExchangedPositionSizeShouldBeEqual(sdk.MustNewDecFromStr("-10000.000300000007000000")),
					MarketOrderResp_BadDebtShouldBeEqual(sdk.ZeroDec()),
					MarketOrderResp_FundingPaymentShouldBeEqual(sdk.ZeroDec()),
					MarketOrderResp_RealizedPnlShouldBeEqual(sdk.ZeroDec()),
					MarketOrderResp_UnrealizedPnlAfterShouldBeEqual(sdk.ZeroDec()),
					MarketOrderResp_MarginToVaultShouldBeEqual(sdk.NewDec(1000)),
					MarketOrderResp_PositionNotionalShouldBeEqual(sdk.NewDec(20_000)),
				),
			).
			Then(
				PositionChangedEventShouldBeEqual(&types.PositionChangedEvent{
					FinalPosition: types.Position{
						Pair:                            pairBtcNusd,
						TraderAddress:                   alice.String(),
						Margin:                          sdk.NewDec(2000),
						OpenNotional:                    sdk.NewDec(20_000),
						Size_:                           sdk.MustNewDecFromStr("-20000.000400000008000000"),
						LastUpdatedBlockNumber:          2,
						LatestCumulativePremiumFraction: sdk.ZeroDec(),
					},
					PositionNotional: sdk.NewDec(20_000),
					RealizedPnl:      sdk.ZeroDec(),
					BadDebt:          sdk.NewCoin(denoms.NUSD, sdk.ZeroInt()),
					FundingPayment:   sdk.ZeroDec(),
					TransactionFee:   sdk.NewCoin(denoms.NUSD, sdk.NewInt(20)), // 20 bps
					BlockHeight:      2,
					// exchangedMargin = - marginToVault - transferredFee
					MarginToUser:      sdk.NewInt(1_000 + 20).Neg(),
					ChangeReason:      types.ChangeReason_MarketOrder,
					ExchangedNotional: sdk.MustNewDecFromStr("10000.000000000000000000"),
					ExchangedSize:     sdk.MustNewDecFromStr("-10000.000300000007000000"),
				}),
			),

		TC("existing short position, go more short but there's bad debt").
			Given(
				CreateCustomMarket(pairBtcNusd,
					WithPricePeg(sdk.MustNewDecFromStr("1.11")),
					WithLatestMarketCPF(sdk.MustNewDecFromStr("0.0002")),
				),
				SetBlockNumber(1),
				SetBlockTime(startBlockTime),
				FundAccount(alice, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(22)))),
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
				CreateCustomMarket(pairBtcNusd),
				SetBlockNumber(1),
				SetBlockTime(startBlockTime),
				FundAccount(alice, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(1030)))),
				MarketOrder(alice, pairBtcNusd, types.Direction_SHORT, sdk.NewInt(1000), sdk.NewDec(10), sdk.ZeroDec()),
			).
			When(
				MoveToNextBlock(),
				MarketOrder(alice, pairBtcNusd, types.Direction_LONG, sdk.NewInt(500), sdk.NewDec(10), sdk.ZeroDec(),
					MarketOrderResp_PositionShouldBeEqual(
						types.Position{
							Pair:                            pairBtcNusd,
							TraderAddress:                   alice.String(),
							Margin:                          sdk.NewDec(1000),
							OpenNotional:                    sdk.NewDec(5000),
							Size_:                           sdk.MustNewDecFromStr("-5000.000025000000125000"),
							LastUpdatedBlockNumber:          2,
							LatestCumulativePremiumFraction: sdk.ZeroDec(),
						},
					),
					MarketOrderResp_ExchangeNotionalValueShouldBeEqual(sdk.NewDec(5000)), // margin * leverage
					MarketOrderResp_ExchangedPositionSizeShouldBeEqual(sdk.MustNewDecFromStr("5000.000075000000875000")),
					MarketOrderResp_BadDebtShouldBeEqual(sdk.ZeroDec()),
					MarketOrderResp_FundingPaymentShouldBeEqual(sdk.ZeroDec()),
					MarketOrderResp_RealizedPnlShouldBeEqual(sdk.ZeroDec()),
					MarketOrderResp_UnrealizedPnlAfterShouldBeEqual(sdk.ZeroDec()),
					MarketOrderResp_MarginToVaultShouldBeEqual(sdk.ZeroDec()),
					MarketOrderResp_PositionNotionalShouldBeEqual(sdk.NewDec(5000)),
				),
			).
			Then(
				PositionChangedEventShouldBeEqual(&types.PositionChangedEvent{
					FinalPosition: types.Position{
						Pair:                            pairBtcNusd,
						TraderAddress:                   alice.String(),
						Margin:                          sdk.NewDec(1000),
						OpenNotional:                    sdk.NewDec(5000),
						Size_:                           sdk.MustNewDecFromStr("-5000.000025000000125000"),
						LastUpdatedBlockNumber:          2,
						LatestCumulativePremiumFraction: sdk.ZeroDec(),
					},
					PositionNotional: sdk.NewDec(5000),
					RealizedPnl:      sdk.ZeroDec(),
					BadDebt:          sdk.NewCoin(denoms.NUSD, sdk.ZeroInt()),
					FundingPayment:   sdk.ZeroDec(),
					TransactionFee:   sdk.NewCoin(denoms.NUSD, sdk.NewInt(10)), // 20 bps
					BlockHeight:      2,
					// exchangedMargin = - marginToVault - transferredFee
					MarginToUser:      sdk.NewInt(0 + 10).Neg(),
					ChangeReason:      types.ChangeReason_MarketOrder,
					ExchangedNotional: sdk.MustNewDecFromStr("-5000.000000000000000000"),
					ExchangedSize:     sdk.MustNewDecFromStr("5000.000075000000875000"),
				}),
			),

		TC("existing short position, decrease a bit but there's bad debt").
			Given(
				CreateCustomMarket(pairBtcNusd,
					WithPricePeg(sdk.MustNewDecFromStr("1.11")),
					WithLatestMarketCPF(sdk.MustNewDecFromStr("0.0002")),
				),
				SetBlockNumber(1),
				SetBlockTime(startBlockTime),
				FundAccount(alice, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(22)))),
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
				CreateCustomMarket(pairBtcNusd),
				SetBlockNumber(1),
				SetBlockTime(startBlockTime),
				FundAccount(alice, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(4080)))),
				MarketOrder(alice, pairBtcNusd, types.Direction_SHORT, sdk.NewInt(1000), sdk.NewDec(10), sdk.ZeroDec()),
			).
			When(
				MoveToNextBlock(),
				MarketOrder(alice, pairBtcNusd, types.Direction_LONG, sdk.NewInt(3000), sdk.NewDec(10), sdk.ZeroDec(),
					MarketOrderResp_PositionShouldBeEqual(
						types.Position{
							Pair:                            pairBtcNusd,
							TraderAddress:                   alice.String(),
							Margin:                          sdk.NewDec(2000),
							OpenNotional:                    sdk.NewDec(20000),
							Size_:                           sdk.MustNewDecFromStr("19999.999600000008000000"),
							LastUpdatedBlockNumber:          2,
							LatestCumulativePremiumFraction: sdk.ZeroDec(),
						},
					),
					MarketOrderResp_ExchangeNotionalValueShouldBeEqual(sdk.NewDec(30000)), // margin * leverage
					MarketOrderResp_ExchangedPositionSizeShouldBeEqual(sdk.MustNewDecFromStr("29999.999700000009000000")),
					MarketOrderResp_BadDebtShouldBeEqual(sdk.ZeroDec()),
					MarketOrderResp_FundingPaymentShouldBeEqual(sdk.ZeroDec()),
					MarketOrderResp_RealizedPnlShouldBeEqual(sdk.ZeroDec()),
					MarketOrderResp_UnrealizedPnlAfterShouldBeEqual(sdk.ZeroDec()),
					MarketOrderResp_MarginToVaultShouldBeEqual(sdk.NewDec(1000)),
					MarketOrderResp_PositionNotionalShouldBeEqual(sdk.NewDec(20000)),
				),
			).
			Then(
				PositionChangedEventShouldBeEqual(&types.PositionChangedEvent{
					FinalPosition: types.Position{
						Pair:                            pairBtcNusd,
						TraderAddress:                   alice.String(),
						Margin:                          sdk.NewDec(2000),
						OpenNotional:                    sdk.NewDec(20000),
						Size_:                           sdk.MustNewDecFromStr("19999.999600000008000000"),
						LastUpdatedBlockNumber:          2,
						LatestCumulativePremiumFraction: sdk.ZeroDec(),
					},
					PositionNotional: sdk.NewDec(20000),
					RealizedPnl:      sdk.ZeroDec(),
					BadDebt:          sdk.NewCoin(denoms.NUSD, sdk.ZeroInt()),
					FundingPayment:   sdk.ZeroDec(),
					TransactionFee:   sdk.NewCoin(denoms.NUSD, sdk.NewInt(60)), // 20 bps
					BlockHeight:      2,
					// exchangedMargin = - marginToVault - transferredFee
					MarginToUser:      sdk.NewInt(1_000 + 60).Neg(),
					ChangeReason:      types.ChangeReason_MarketOrder,
					ExchangedNotional: sdk.MustNewDecFromStr("10000.000000000000000000"),
					ExchangedSize:     sdk.MustNewDecFromStr("29999.999700000009000000"),
				}),
			),

		TC("existing short position, decrease a lot but there's bad debt").
			Given(
				CreateCustomMarket(pairBtcNusd,
					WithPricePeg(sdk.MustNewDecFromStr("1.11")),
					WithLatestMarketCPF(sdk.MustNewDecFromStr("0.0002")),
				),
				SetBlockNumber(1),
				SetBlockTime(startBlockTime),
				FundAccount(alice, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(22)))),
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
					BadDebt:          sdk.NewInt64Coin(denoms.NUSD, 98),
					FundingPayment:   sdk.NewDec(-2),
					TransactionFee:   sdk.NewInt64Coin(denoms.NUSD, 22), // 20 bps
					BlockHeight:      2,
					// exchangedMargin = - marginToVault - transferredFee
					MarginToUser:      sdk.NewInt(-22),
					ChangeReason:      types.ChangeReason_MarketOrder,
					ExchangedNotional: sdk.MustNewDecFromStr("-10000.000000000000000000"),
					ExchangedSize:     sdk.MustNewDecFromStr("10000.000000000000000000"),
				}),
			),

		TC("user has insufficient funds").
			Given(
				CreateCustomMarket(pairBtcNusd),
				SetBlockTime(startBlockTime),
				SetBlockNumber(1),
				FundAccount(alice, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(99)))),
			).
			When(
				MarketOrderFails(
					alice, pairBtcNusd, types.Direction_LONG, sdk.NewInt(100), sdk.OneDec(), sdk.ZeroDec(),
					sdkerrors.ErrInsufficientFunds),
			).
			Then(
				PositionShouldNotExist(alice, pairBtcNusd),
			),

		TC("new long position, can close position after market is not enabled").
			Given(
				CreateCustomMarket(pairBtcNusd),
				SetBlockTime(startBlockTime),
				SetBlockNumber(1),
				FundAccount(alice, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(47_714_285_715)))),
				MarketOrder(alice, pairBtcNusd, types.Direction_SHORT, sdk.NewInt(47_619_047_619), sdk.OneDec(), sdk.ZeroDec()),
				SetMarketEnabled(pairBtcNusd, false),
			).
			When(
				ClosePosition(alice, pairBtcNusd),
			).
			Then(
				PositionShouldNotExist(alice, pairBtcNusd),
			),

		TC("new long position, can not open new position after market is not enabled").
			Given(
				CreateCustomMarket(pairBtcNusd),
				SetBlockTime(startBlockTime),
				SetBlockNumber(1),
				FundAccount(alice, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(47_714_285_715)))),
				SetMarketEnabled(pairBtcNusd, false),
			).
			When(
				MarketOrderFails(alice, pairBtcNusd, types.Direction_SHORT, sdk.NewInt(47_619_047_619), sdk.OneDec(), sdk.ZeroDec(),
					types.ErrMarketNotEnabled),
			).
			Then(
				PositionShouldNotExist(alice, pairBtcNusd),
			),

		TC("existing long position, can not open new one but can close").
			Given(
				CreateCustomMarket(pairBtcNusd),
				SetBlockTime(startBlockTime),
				SetBlockNumber(1),
				FundAccount(alice, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(47_714_285_715)))),
				MarketOrder(alice, pairBtcNusd, types.Direction_SHORT, sdk.NewInt(50_000), sdk.OneDec(), sdk.ZeroDec()),
				SetMarketEnabled(pairBtcNusd, false),
			).
			When(
				MarketOrderFails(alice, pairBtcNusd, types.Direction_SHORT, sdk.NewInt(47_619_047_619), sdk.OneDec(), sdk.ZeroDec(),
					types.ErrMarketNotEnabled),
				ClosePosition(alice, pairBtcNusd),
			).
			Then(
				PositionShouldNotExist(alice, pairBtcNusd),
			),

		TC("market doesn't exist").
			Given(
				SetBlockTime(startBlockTime),
				SetBlockNumber(1),
				FundAccount(alice, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(47_714_285_715)))),
			).
			When(
				MarketOrderFails(alice, pairBtcNusd, types.Direction_SHORT, sdk.NewInt(47_619_047_619), sdk.OneDec(), sdk.ZeroDec(),
					types.ErrPairNotFound),
			).
			Then(
				PositionShouldNotExist(alice, pairBtcNusd),
			),

		TC("zero quote asset amount").
			Given(
				SetBlockTime(startBlockTime),
				SetBlockNumber(1),
				CreateCustomMarket(pairBtcNusd),
				FundAccount(alice, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(47_714_285_715)))),
			).
			When(
				MarketOrderFails(alice, pairBtcNusd, types.Direction_LONG, sdk.ZeroInt(), sdk.OneDec(), sdk.ZeroDec(),
					types.ErrInputQuoteAmtNegative),
			).
			Then(
				PositionShouldNotExist(alice, pairBtcNusd),
			),

		TC("zero leverage").
			Given(
				SetBlockTime(startBlockTime),
				SetBlockNumber(1),
				CreateCustomMarket(pairBtcNusd),
				FundAccount(alice, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(1e6)))),
			).
			When(
				MarketOrderFails(alice, pairBtcNusd, types.Direction_LONG, sdk.OneInt(), sdk.ZeroDec(), sdk.ZeroDec(),
					types.ErrUserLeverageNegative),
			).
			Then(
				PositionShouldNotExist(alice, pairBtcNusd),
			),

		TC("user leverage greater than market max leverage").
			Given(
				SetBlockTime(startBlockTime),
				SetBlockNumber(1),
				CreateCustomMarket(pairBtcNusd),
				FundAccount(alice, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(1e6)))),
			).
			When(
				MarketOrderFails(alice, pairBtcNusd, types.Direction_LONG, sdk.OneInt(), sdk.NewDec(11), sdk.ZeroDec(),
					types.ErrLeverageIsTooHigh),
			).
			Then(
				PositionShouldNotExist(alice, pairBtcNusd),
			),

		TC("position should not exist after opening a closing manually").
			Given(
				SetBlockTime(startBlockTime),
				SetBlockNumber(1),
				CreateCustomMarket(pairBtcNusd, WithPricePeg(sdk.MustNewDecFromStr("25001.0112"))),
				FundAccount(alice, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(20_000_000_000+20_000_000)))),
			).
			When(
				MarketOrder(alice, pairBtcNusd, types.Direction_SHORT, sdk.NewInt(10_000_000_000), sdk.OneDec(), sdk.ZeroDec()),
				MarketOrder(alice, pairBtcNusd, types.Direction_LONG, sdk.NewInt(10_000_000_000), sdk.OneDec(), sdk.ZeroDec()),
			).
			Then(
				PositionShouldNotExist(alice, pairBtcNusd),
			),

		TC("position should not exist after opening a closing manually - reverse with leverage").
			Given(
				SetBlockTime(startBlockTime),
				SetBlockNumber(1),
				CreateCustomMarket(pairBtcNusd, WithPricePeg(sdk.MustNewDecFromStr("25001.0112"))),
				FundAccount(alice, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(1e6)))),
			).
			When(
				MarketOrder(alice, pairBtcNusd, types.Direction_SHORT, sdk.NewInt(100_000), sdk.OneDec(), sdk.ZeroDec()),
				MarketOrder(alice, pairBtcNusd, types.Direction_LONG, sdk.NewInt(50_000), sdk.NewDec(2), sdk.ZeroDec()),
			).
			Then(
				PositionShouldNotExist(alice, pairBtcNusd),
			),
		TC("position should not exist after opening a closing manually - open with leverage").
			Given(
				SetBlockTime(startBlockTime),
				SetBlockNumber(1),
				CreateCustomMarket(pairBtcNusd, WithPricePeg(sdk.MustNewDecFromStr("25001.0112"))),
				FundAccount(alice, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(1e6)))),
			).
			When(
				MarketOrder(alice, pairBtcNusd, types.Direction_LONG, sdk.NewInt(50_000), sdk.NewDec(2), sdk.ZeroDec()),
				MarketOrder(alice, pairBtcNusd, types.Direction_SHORT, sdk.NewInt(100_000), sdk.OneDec(), sdk.ZeroDec()),
			).
			Then(
				PositionShouldNotExist(alice, pairBtcNusd),
			),

		TC("position should not exist after opening a closing manually - reverse with leverage").
			Given(
				SetBlockTime(startBlockTime),
				SetBlockNumber(1),
				CreateCustomMarket(pairBtcNusd, WithPricePeg(sdk.MustNewDecFromStr("25001.0112"))),
				FundAccount(alice, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(1e6)))),
			).
			When(
				MarketOrder(alice, pairBtcNusd, types.Direction_SHORT, sdk.NewInt(100_000), sdk.OneDec(), sdk.ZeroDec()),
				MarketOrder(alice, pairBtcNusd, types.Direction_LONG, sdk.NewInt(50_000), sdk.NewDec(2), sdk.ZeroDec()),
			).
			Then(
				PositionShouldNotExist(alice, pairBtcNusd),
			),

		TC("position should not exist after opening a closing manually - reverse with leverage - more steps").
			Given(
				SetBlockTime(startBlockTime),
				SetBlockNumber(1),
				CreateCustomMarket(pairBtcNusd, WithPricePeg(sdk.MustNewDecFromStr("25000"))),
				FundAccount(alice, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(1e6)))),
			).
			When(
				MarketOrder(alice, pairBtcNusd, types.Direction_SHORT, sdk.NewInt(100_000), sdk.OneDec(), sdk.ZeroDec()),
				MarketOrder(alice, pairBtcNusd, types.Direction_LONG, sdk.NewInt(50_000), sdk.NewDec(4), sdk.ZeroDec()),
				MarketOrder(alice, pairBtcNusd, types.Direction_SHORT, sdk.NewInt(50_000), sdk.NewDec(2), sdk.ZeroDec()),
			).
			Then(
				PositionShouldNotExist(alice, pairBtcNusd),
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
			traderFunds:     sdk.NewCoins(sdk.NewInt64Coin(denoms.NUSD, 999)),
			initialPosition: nil,
			side:            types.Direction_LONG,
			margin:          sdk.NewInt(1000),
			leverage:        sdk.NewDec(10),
			baseLimit:       sdk.ZeroDec(),
			expectedErr:     sdkerrors.ErrInsufficientFunds,
		},
		{
			name:        "position has bad debt",
			traderFunds: sdk.NewCoins(sdk.NewInt64Coin(denoms.NUSD, 999)),
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
			traderFunds:     sdk.NewCoins(sdk.NewInt64Coin(denoms.NUSD, 1020)),
			initialPosition: nil,
			side:            types.Direction_LONG,
			margin:          sdk.NewInt(1000),
			leverage:        sdk.NewDec(10),
			baseLimit:       sdk.NewDec(10_000),
			expectedErr:     types.ErrAssetFailsUserLimit,
		},
		{
			name:            "new short position not under base limit",
			traderFunds:     sdk.NewCoins(sdk.NewInt64Coin(denoms.NUSD, 1020)),
			initialPosition: nil,
			side:            types.Direction_SHORT,
			margin:          sdk.NewInt(1000),
			leverage:        sdk.NewDec(10),
			baseLimit:       sdk.NewDec(10_000),
			expectedErr:     types.ErrAssetFailsUserLimit,
		},
		{
			name:            "quote asset amount is zero",
			traderFunds:     sdk.NewCoins(sdk.NewInt64Coin(denoms.NUSD, 1020)),
			initialPosition: nil,
			side:            types.Direction_SHORT,
			margin:          sdk.ZeroInt(),
			leverage:        sdk.NewDec(10),
			baseLimit:       sdk.NewDec(10_000),
			expectedErr:     types.ErrInputQuoteAmtNegative,
		},
		{
			name:            "leverage amount is zero",
			traderFunds:     sdk.NewCoins(sdk.NewInt64Coin(denoms.NUSD, 1020)),
			initialPosition: nil,
			side:            types.Direction_SHORT,
			margin:          sdk.NewInt(1000),
			leverage:        sdk.ZeroDec(),
			baseLimit:       sdk.NewDec(10_000),
			expectedErr:     types.ErrUserLeverageNegative,
		},
		{
			name:            "leverage amount is too high - SELL",
			traderFunds:     sdk.NewCoins(sdk.NewInt64Coin(denoms.NUSD, 1020)),
			initialPosition: nil,
			side:            types.Direction_SHORT,
			margin:          sdk.NewInt(100),
			leverage:        sdk.NewDec(100),
			baseLimit:       sdk.NewDec(11_000),
			expectedErr:     types.ErrLeverageIsTooHigh,
		},
		{
			name:            "leverage amount is too high - BUY",
			traderFunds:     sdk.NewCoins(sdk.NewInt64Coin(denoms.NUSD, 1020)),
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
				app.PerpKeeperV2.Positions.Insert(ctx, collections.Join(tc.initialPosition.Pair, traderAddr), *tc.initialPosition)
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
					WithPricePeg(sdk.NewDec(2)),
					WithLatestMarketCPF(sdk.MustNewDecFromStr("0.0002")),
				),
				SetBlockTime(startBlockTime),
				SetBlockNumber(1),
				FundAccount(alice, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(10)))),
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
					Margin:                          sdk.MustNewDecFromStr("3497.999950000000500000"),
					OpenNotional:                    sdk.MustNewDecFromStr("7499.999962500000468750"),
					Size_:                           sdk.MustNewDecFromStr("7500.000000000000000000"),
					LastUpdatedBlockNumber:          1,
					LatestCumulativePremiumFraction: sdk.MustNewDecFromStr("0.0002"),
				})),
				PositionChangedEventShouldBeEqual(&types.PositionChangedEvent{
					FinalPosition: types.Position{
						Pair:                            pairBtcNusd,
						TraderAddress:                   alice.String(),
						Margin:                          sdk.MustNewDecFromStr("3497.999950000000500000"),
						OpenNotional:                    sdk.MustNewDecFromStr("7499.999962500000468750"),
						Size_:                           sdk.MustNewDecFromStr("7500.000000000000000000"),
						LastUpdatedBlockNumber:          1,
						LatestCumulativePremiumFraction: sdk.MustNewDecFromStr("0.0002"),
					},
					PositionNotional:  sdk.MustNewDecFromStr("14999.999812500001968750"),
					RealizedPnl:       sdk.MustNewDecFromStr("2499.999950000000500000"),
					BadDebt:           sdk.NewCoin(denoms.NUSD, sdk.ZeroInt()),
					FundingPayment:    sdk.NewDec(2),
					TransactionFee:    sdk.NewCoin(denoms.NUSD, sdk.NewInt(10)),
					BlockHeight:       1,
					MarginToUser:      sdk.NewInt(-10),
					ChangeReason:      types.ChangeReason_PartialClose,
					ExchangedNotional: sdk.MustNewDecFromStr("4999.999812500001968750"),
					ExchangedSize:     sdk.MustNewDecFromStr("-2500.000000000000000000"),
				}),
			),

		TC("partial close long position with negative PnL").
			Given(
				CreateCustomMarket(
					pairBtcNusd,
					WithPricePeg(sdk.MustNewDecFromStr("0.95")),
					WithLatestMarketCPF(sdk.MustNewDecFromStr("0.0002")),
				),
				SetBlockTime(startBlockTime),
				SetBlockNumber(1),
				FundAccount(alice, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(4)))),
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
					Margin:                          sdk.MustNewDecFromStr("872.999976250000237500"),
					OpenNotional:                    sdk.MustNewDecFromStr("7499.999982187500222656"),
					Size_:                           sdk.MustNewDecFromStr("7500.000000000000000000"),
					LastUpdatedBlockNumber:          1,
					LatestCumulativePremiumFraction: sdk.MustNewDecFromStr("0.0002"),
				})),
				PositionChangedEventShouldBeEqual(&types.PositionChangedEvent{
					FinalPosition: types.Position{
						Pair:                            pairBtcNusd,
						TraderAddress:                   alice.String(),
						Margin:                          sdk.MustNewDecFromStr("872.999976250000237500"),
						OpenNotional:                    sdk.MustNewDecFromStr("7499.999982187500222656"),
						Size_:                           sdk.MustNewDecFromStr("7500.000000000000000000"),
						LastUpdatedBlockNumber:          1,
						LatestCumulativePremiumFraction: sdk.MustNewDecFromStr("0.0002"),
					},
					PositionNotional:  sdk.MustNewDecFromStr("7124.999910937500935156"),
					RealizedPnl:       sdk.MustNewDecFromStr("-125.000023749999762500"),
					BadDebt:           sdk.NewCoin(denoms.NUSD, sdk.ZeroInt()),
					FundingPayment:    sdk.NewDec(2),
					TransactionFee:    sdk.NewCoin(denoms.NUSD, sdk.NewInt(4)),
					BlockHeight:       1,
					MarginToUser:      sdk.NewInt(-4),
					ChangeReason:      types.ChangeReason_PartialClose,
					ExchangedNotional: sdk.MustNewDecFromStr("-2875.000089062499064844"),
					ExchangedSize:     sdk.MustNewDecFromStr("-2500.000000000000000000"),
				}),
			),

		TC("partial close long position without bad debt but below maintenance margin ratio").
			Given(
				CreateCustomMarket(
					pairBtcNusd,
					WithPricePeg(sdk.MustNewDecFromStr("0.94")),
					WithLatestMarketCPF(sdk.MustNewDecFromStr("0.0002")),
				),
				SetBlockTime(startBlockTime),
				SetBlockNumber(1),
				FundAccount(alice, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(4)))),
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
					Margin:                          sdk.MustNewDecFromStr("847.999976500000235000"),
					OpenNotional:                    sdk.MustNewDecFromStr("7499.999982375000220312"),
					Size_:                           sdk.MustNewDecFromStr("7499.999999999999999999"),
					LastUpdatedBlockNumber:          1,
					LatestCumulativePremiumFraction: sdk.MustNewDecFromStr("0.0002"),
				})),
				PositionChangedEventShouldBeEqual(&types.PositionChangedEvent{
					FinalPosition: types.Position{
						Pair:                            pairBtcNusd,
						TraderAddress:                   alice.String(),
						Margin:                          sdk.MustNewDecFromStr("847.999976500000235000"),
						OpenNotional:                    sdk.MustNewDecFromStr("7499.999982375000220312"),
						Size_:                           sdk.MustNewDecFromStr("7499.999999999999999999"),
						LastUpdatedBlockNumber:          1,
						LatestCumulativePremiumFraction: sdk.MustNewDecFromStr("0.0002"),
					},
					PositionNotional:  sdk.MustNewDecFromStr("7049.999911875000925312"),
					RealizedPnl:       sdk.MustNewDecFromStr("-150.000023499999765000"),
					BadDebt:           sdk.NewCoin(denoms.NUSD, sdk.ZeroInt()),
					FundingPayment:    sdk.NewDec(2),
					TransactionFee:    sdk.NewCoin(denoms.NUSD, sdk.NewInt(4)),
					BlockHeight:       1,
					MarginToUser:      sdk.NewInt(-4),
					ChangeReason:      types.ChangeReason_PartialClose,
					ExchangedNotional: sdk.MustNewDecFromStr("-2950.000088124999074688"),
					ExchangedSize:     sdk.MustNewDecFromStr("-2500.000000000000000001"),
				})),

		TC("partial close long position with bad debt").
			Given(
				CreateCustomMarket(
					pairBtcNusd,
					WithPricePeg(sdk.MustNewDecFromStr("0.59")),
					WithLatestMarketCPF(sdk.MustNewDecFromStr("0.0002")),
				),
				SetBlockTime(startBlockTime),
				SetBlockNumber(1),
				FundAccount(alice, sdk.NewCoins(sdk.NewInt64Coin(denoms.NUSD, 2))),
				FundModule(types.PerpEFModuleAccount, sdk.NewCoins(sdk.NewInt64Coin(denoms.NUSD, 27))),
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
					OpenNotional:                    sdk.MustNewDecFromStr("7499.999988937500138281"),
					Size_:                           sdk.MustNewDecFromStr("7500.000000000000000000"),
					LastUpdatedBlockNumber:          1,
					LatestCumulativePremiumFraction: sdk.MustNewDecFromStr("0.0002"),
				})),
				PositionChangedEventShouldBeEqual(&types.PositionChangedEvent{
					FinalPosition: types.Position{
						Pair:                            pairBtcNusd,
						TraderAddress:                   alice.String(),
						Margin:                          sdk.ZeroDec(),
						OpenNotional:                    sdk.MustNewDecFromStr("7499.999988937500138281"),
						Size_:                           sdk.MustNewDecFromStr("7500.000000000000000000"),
						LastUpdatedBlockNumber:          1,
						LatestCumulativePremiumFraction: sdk.MustNewDecFromStr("0.0002"),
					},
					PositionNotional:  sdk.MustNewDecFromStr("4424.999944687500580781"),
					RealizedPnl:       sdk.MustNewDecFromStr("-1025.000014749999852500"),
					BadDebt:           sdk.NewInt64Coin(denoms.NUSD, 27),
					FundingPayment:    sdk.NewDec(2),
					TransactionFee:    sdk.NewInt64Coin(denoms.NUSD, 2),
					BlockHeight:       1,
					MarginToUser:      sdk.NewInt(-2),
					ChangeReason:      types.ChangeReason_PartialClose,
					ExchangedNotional: sdk.MustNewDecFromStr("-5575.000055312499419219"),
					ExchangedSize:     sdk.MustNewDecFromStr("-2500.000000000000000000"),
				})),

		TC("partial close short position with positive PnL").
			Given(
				CreateCustomMarket(
					pairBtcNusd,
					WithPricePeg(sdk.MustNewDecFromStr("0.10")),
					WithLatestMarketCPF(sdk.MustNewDecFromStr("0.0002")),
				),
				SetBlockTime(startBlockTime),
				SetBlockNumber(1),
				FundAccount(alice, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(2)))),
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
					Margin:                          sdk.MustNewDecFromStr("7751.999992499999925000"),
					OpenNotional:                    sdk.MustNewDecFromStr("2500.000001875000032812"),
					Size_:                           sdk.MustNewDecFromStr("-2499.999999999999999995"),
					LastUpdatedBlockNumber:          1,
					LatestCumulativePremiumFraction: sdk.MustNewDecFromStr("0.0002"),
				})),
				PositionChangedEventShouldBeEqual(&types.PositionChangedEvent{
					FinalPosition: types.Position{
						Pair:                            pairBtcNusd,
						TraderAddress:                   alice.String(),
						Margin:                          sdk.MustNewDecFromStr("7751.999992499999925000"),
						OpenNotional:                    sdk.MustNewDecFromStr("2500.000001875000032812"),
						Size_:                           sdk.MustNewDecFromStr("-2499.999999999999999995"),
						LastUpdatedBlockNumber:          1,
						LatestCumulativePremiumFraction: sdk.MustNewDecFromStr("0.0002"),
					},
					PositionNotional:  sdk.MustNewDecFromStr("250.000004375000057812"),
					RealizedPnl:       sdk.MustNewDecFromStr("6749.999992499999925000"),
					BadDebt:           sdk.NewCoin(denoms.NUSD, sdk.ZeroInt()),
					FundingPayment:    sdk.NewDec(-2),
					TransactionFee:    sdk.NewCoin(denoms.NUSD, sdk.NewInt(2)),
					BlockHeight:       1,
					MarginToUser:      sdk.NewInt(-2),
					ChangeReason:      types.ChangeReason_PartialClose,
					ExchangedNotional: sdk.MustNewDecFromStr("-9749.999995624999942188"),
					ExchangedSize:     sdk.MustNewDecFromStr("7500.000000000000000005"),
				}),
			),

		TC("partial close short position with negative PnL").
			Given(
				CreateCustomMarket(
					pairBtcNusd,
					WithPricePeg(sdk.MustNewDecFromStr("1.05")),
					WithLatestMarketCPF(sdk.MustNewDecFromStr("0.0002")),
				),
				SetBlockTime(startBlockTime),
				SetBlockNumber(1),
				FundAccount(alice, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(16)))),
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
					Margin:                          sdk.MustNewDecFromStr("626.999921249999212500"),
					OpenNotional:                    sdk.MustNewDecFromStr("2500.000019687500344531"),
					Size_:                           sdk.MustNewDecFromStr("-2500.000000000000000000"),
					LastUpdatedBlockNumber:          1,
					LatestCumulativePremiumFraction: sdk.MustNewDecFromStr("0.0002"),
				})),
				PositionChangedEventShouldBeEqual(&types.PositionChangedEvent{
					FinalPosition: types.Position{
						Pair:                            pairBtcNusd,
						TraderAddress:                   alice.String(),
						Margin:                          sdk.MustNewDecFromStr("626.999921249999212500"),
						OpenNotional:                    sdk.MustNewDecFromStr("2500.000019687500344531"),
						Size_:                           sdk.MustNewDecFromStr("-2500.000000000000000000"),
						LastUpdatedBlockNumber:          1,
						LatestCumulativePremiumFraction: sdk.MustNewDecFromStr("0.0002"),
					},
					PositionNotional:  sdk.MustNewDecFromStr("2625.000045937500607031"),
					RealizedPnl:       sdk.MustNewDecFromStr("-375.000078750000787500"),
					BadDebt:           sdk.NewCoin(denoms.NUSD, sdk.ZeroInt()),
					FundingPayment:    sdk.NewDec(-2),
					TransactionFee:    sdk.NewCoin(denoms.NUSD, sdk.NewInt(16)),
					BlockHeight:       1,
					MarginToUser:      sdk.NewInt(-16),
					ChangeReason:      types.ChangeReason_PartialClose,
					ExchangedNotional: sdk.MustNewDecFromStr("-7374.999954062499392969"),
					ExchangedSize:     sdk.MustNewDecFromStr("7500.000000000000000000"),
				}),
			),

		TC("partial close short position with no bad debt but below maintenance margin ratio").
			Given(
				CreateCustomMarket(
					pairBtcNusd,
					WithPricePeg(sdk.MustNewDecFromStr("1.09")),
					WithLatestMarketCPF(sdk.MustNewDecFromStr("0.0002")),
				),
				SetBlockTime(startBlockTime),
				SetBlockNumber(1),
				FundAccount(alice, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(16)))),
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
					Margin:                          sdk.MustNewDecFromStr("326.999918249999182500"),
					OpenNotional:                    sdk.MustNewDecFromStr("2500.000020437500357656"),
					Size_:                           sdk.MustNewDecFromStr("-2500.000000000000000000"),
					LastUpdatedBlockNumber:          1,
					LatestCumulativePremiumFraction: sdk.MustNewDecFromStr("0.0002"),
				})),
				PositionChangedEventShouldBeEqual(&types.PositionChangedEvent{
					FinalPosition: types.Position{
						Pair:                            pairBtcNusd,
						TraderAddress:                   alice.String(),
						Margin:                          sdk.MustNewDecFromStr("326.999918249999182500"),
						OpenNotional:                    sdk.MustNewDecFromStr("2500.000020437500357656"),
						Size_:                           sdk.MustNewDecFromStr("-2500.000000000000000000"),
						LastUpdatedBlockNumber:          1,
						LatestCumulativePremiumFraction: sdk.MustNewDecFromStr("0.0002"),
					},
					PositionNotional:  sdk.MustNewDecFromStr("2725.000047687500630156"),
					RealizedPnl:       sdk.MustNewDecFromStr("-675.000081750000817500"),
					BadDebt:           sdk.NewCoin(denoms.NUSD, sdk.ZeroInt()),
					FundingPayment:    sdk.NewDec(-2),
					TransactionFee:    sdk.NewCoin(denoms.NUSD, sdk.NewInt(16)),
					BlockHeight:       1,
					MarginToUser:      sdk.NewInt(-16),
					ChangeReason:      types.ChangeReason_PartialClose,
					ExchangedNotional: sdk.MustNewDecFromStr("-7274.999952312499369844"),
					ExchangedSize:     sdk.MustNewDecFromStr("7500.000000000000000000"),
				}),
			),

		TC("partial close short position with bad debt").
			Given(
				CreateCustomMarket(
					pairBtcNusd,
					WithPricePeg(sdk.MustNewDecFromStr("1.14")),
					WithLatestMarketCPF(sdk.MustNewDecFromStr("0.0002")),
				),
				SetBlockTime(startBlockTime),
				SetBlockNumber(1),
				FundAccount(alice, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(18)))),
				FundModule(types.PerpEFModuleAccount, sdk.NewCoins(sdk.NewInt64Coin(denoms.NUSD, 48))),
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
					OpenNotional:                    sdk.MustNewDecFromStr("2500.000021375000374062"),
					Size_:                           sdk.MustNewDecFromStr("-2500.000000000000000000"),
					LastUpdatedBlockNumber:          1,
					LatestCumulativePremiumFraction: sdk.MustNewDecFromStr("0.0002"),
				})),
				PositionChangedEventShouldBeEqual(&types.PositionChangedEvent{
					FinalPosition: types.Position{
						Pair:                            pairBtcNusd,
						TraderAddress:                   alice.String(),
						Margin:                          sdk.ZeroDec(),
						OpenNotional:                    sdk.MustNewDecFromStr("2500.000021375000374062"),
						Size_:                           sdk.MustNewDecFromStr("-2500.000000000000000000"),
						LastUpdatedBlockNumber:          1,
						LatestCumulativePremiumFraction: sdk.MustNewDecFromStr("0.0002"),
					},
					PositionNotional:  sdk.MustNewDecFromStr("2850.000049875000659062"),
					RealizedPnl:       sdk.MustNewDecFromStr("-1050.000085500000855000"),
					BadDebt:           sdk.NewInt64Coin(denoms.NUSD, 48),
					FundingPayment:    sdk.NewDec(-2),
					TransactionFee:    sdk.NewCoin(denoms.NUSD, sdk.NewInt(18)),
					BlockHeight:       1,
					MarginToUser:      sdk.NewInt(-18),
					ChangeReason:      types.ChangeReason_PartialClose,
					ExchangedNotional: sdk.MustNewDecFromStr("-7149.999950124999340938"),
					ExchangedSize:     sdk.MustNewDecFromStr("7500.000000000000000000"),
				}),
			),
		TC("test partial closes fail").
			Given(
				CreateCustomMarket(
					pairBtcNusd,
					WithPricePeg(sdk.OneDec()),
					WithSqrtDepth(sdk.NewDec(10_000)),
				),
				SetBlockNumber(1),
				SetBlockTime(startBlockTime),

				FundAccount(alice, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(10_200)))),

				PartialCloseFails(alice, pairBtcNusd, sdk.NewDec(5_000), collections.ErrNotFound),
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
					WithPricePeg(sdk.NewDec(2)),
					WithLatestMarketCPF(sdk.MustNewDecFromStr("0.0002")),
				),
				SetBlockTime(startBlockTime),
				SetBlockNumber(1),
				FundAccount(alice, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(40)))),
				FundModule(types.VaultModuleAccount, sdk.NewCoins(sdk.NewInt64Coin(denoms.NUSD, 10_998))),
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
				PositionShouldNotExist(alice, pairBtcNusd),
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
					BadDebt:           sdk.NewCoin(denoms.NUSD, sdk.ZeroInt()),
					FundingPayment:    sdk.NewDec(2),
					TransactionFee:    sdk.NewCoin(denoms.NUSD, sdk.NewInt(40)),
					BlockHeight:       1,
					MarginToUser:      sdk.NewInt(10_958),
					ChangeReason:      types.ChangeReason_ClosePosition,
					ExchangedNotional: sdk.MustNewDecFromStr("-10000.000000000000000000"),
					ExchangedSize:     sdk.MustNewDecFromStr("-10000.000000000000000000"),
				}),
			),

		TC("close long position with negative PnL").
			Given(
				CreateCustomMarket(pairBtcNusd,
					WithPricePeg(sdk.MustNewDecFromStr("0.99")),
					WithLatestMarketCPF(sdk.MustNewDecFromStr("0.0002")),
				),
				SetBlockTime(startBlockTime),
				SetBlockNumber(1),
				FundAccount(alice, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(20)))),
				FundModule(types.VaultModuleAccount, sdk.NewCoins(sdk.NewInt64Coin(denoms.NUSD, 898))),
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
				PositionShouldNotExist(alice, pairBtcNusd),
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
					BadDebt:           sdk.NewCoin(denoms.NUSD, sdk.ZeroInt()),
					FundingPayment:    sdk.NewDec(2),
					TransactionFee:    sdk.NewCoin(denoms.NUSD, sdk.NewInt(20)),
					BlockHeight:       1,
					MarginToUser:      sdk.NewInt(878),
					ChangeReason:      types.ChangeReason_ClosePosition,
					ExchangedNotional: sdk.MustNewDecFromStr("-10000.000000000000000000"),
					ExchangedSize:     sdk.MustNewDecFromStr("-10000.000000000000000000"),
				}),
			),

		TC("close long position with bad debt").
			Given(
				CreateCustomMarket(pairBtcNusd,
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
				FundAccount(alice, sdk.NewCoins(sdk.NewInt64Coin(denoms.NUSD, 18))),
				FundModule(types.VaultModuleAccount, sdk.NewCoins(sdk.NewInt64Coin(denoms.NUSD, 1000))),
				FundModule(types.PerpEFModuleAccount, sdk.NewCoins(sdk.NewInt64Coin(denoms.NUSD, 102))),
			).
			When(
				MoveToNextBlock(),
				ClosePosition(alice, pairBtcNusd),
			).
			Then(
				PositionShouldNotExist(alice, pairBtcNusd),
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
					TransactionFee:    sdk.NewInt64Coin(denoms.NUSD, 18),
					RealizedPnl:       sdk.MustNewDecFromStr("-1100.000088999999110000"),
					BadDebt:           sdk.NewCoin(denoms.NUSD, sdk.NewInt(102)),
					FundingPayment:    sdk.NewDec(2),
					BlockHeight:       2,
					MarginToUser:      sdk.NewInt(-18),
					ChangeReason:      types.ChangeReason_ClosePosition,
					ExchangedNotional: sdk.MustNewDecFromStr("-10000.000000000000000000"),
					ExchangedSize:     sdk.MustNewDecFromStr("-10000.000000000000000000"),
				}),
				ModuleBalanceEqual(types.VaultModuleAccount, denoms.NUSD, sdk.NewInt(1102)), // 1000 + 102 from perp ef
				ModuleBalanceEqual(types.PerpEFModuleAccount, denoms.NUSD, sdk.NewInt(9)),
			),

		TC("close short position with positive PnL").
			Given(
				CreateCustomMarket(pairBtcNusd,
					WithPricePeg(sdk.MustNewDecFromStr("0.10")),
					WithLatestMarketCPF(sdk.MustNewDecFromStr("0.0002")),
				),
				SetBlockTime(startBlockTime),
				SetBlockNumber(1),
				FundAccount(alice, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(2)))),
				FundModule(types.VaultModuleAccount, sdk.NewCoins(sdk.NewInt64Coin(denoms.NUSD, 10_002))),
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
				PositionShouldNotExist(alice, pairBtcNusd),
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
					BadDebt:           sdk.NewCoin(denoms.NUSD, sdk.ZeroInt()),
					FundingPayment:    sdk.NewDec(-2),
					TransactionFee:    sdk.NewCoin(denoms.NUSD, sdk.NewInt(2)),
					BlockHeight:       1,
					MarginToUser:      sdk.NewInt(10_000),
					ChangeReason:      types.ChangeReason_ClosePosition,
					ExchangedNotional: sdk.MustNewDecFromStr("-10000.000000000000000000"),
					ExchangedSize:     sdk.MustNewDecFromStr("10000.000000000000000000"),
				}),
			),

		TC("close short position with negative PnL").
			Given(
				CreateCustomMarket(pairBtcNusd,
					WithPricePeg(sdk.MustNewDecFromStr("1.01")),
					WithLatestMarketCPF(sdk.MustNewDecFromStr("0.0002")),
				),
				SetBlockTime(startBlockTime),
				SetBlockNumber(1),
				FundAccount(alice, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(20)))),
				FundModule(types.VaultModuleAccount, sdk.NewCoins(sdk.NewInt64Coin(denoms.NUSD, 902))),
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
				PositionShouldNotExist(alice, pairBtcNusd),
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
					BadDebt:           sdk.NewCoin(denoms.NUSD, sdk.ZeroInt()),
					FundingPayment:    sdk.NewDec(-2),
					TransactionFee:    sdk.NewCoin(denoms.NUSD, sdk.NewInt(20)),
					BlockHeight:       1,
					MarginToUser:      sdk.NewInt(882),
					ChangeReason:      types.ChangeReason_ClosePosition,
					ExchangedNotional: sdk.MustNewDecFromStr("-10000.000000000000000000"),
					ExchangedSize:     sdk.MustNewDecFromStr("10000.000000000000000000"),
				}),
			),

		TC("close short position with bad debt").
			Given(
				CreateCustomMarket(pairBtcNusd,
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
				FundAccount(alice, sdk.NewCoins(sdk.NewInt64Coin(denoms.NUSD, 22))),
				FundModule(types.VaultModuleAccount, sdk.NewCoins(sdk.NewInt64Coin(denoms.NUSD, 1000))),
				FundModule(types.PerpEFModuleAccount, sdk.NewCoins(sdk.NewInt64Coin(denoms.NUSD, 98))),
			).
			When(
				MoveToNextBlock(),
				ClosePosition(alice, pairBtcNusd),
			).
			Then(
				PositionShouldNotExist(alice, pairBtcNusd),
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
					TransactionFee:    sdk.NewInt64Coin(denoms.NUSD, 22),
					RealizedPnl:       sdk.MustNewDecFromStr("-1100.000111000001110000"),
					BadDebt:           sdk.NewInt64Coin(denoms.NUSD, 98),
					FundingPayment:    sdk.NewDec(-2),
					BlockHeight:       2,
					MarginToUser:      sdk.NewInt(-22),
					ChangeReason:      types.ChangeReason_ClosePosition,
					ExchangedNotional: sdk.MustNewDecFromStr("-10000.000000000000000000"),
					ExchangedSize:     sdk.MustNewDecFromStr("10000.000000000000000000"),
				}),
				ModuleBalanceEqual(types.VaultModuleAccount, denoms.NUSD, sdk.NewInt(1098)), // 1000 + 98 from perp ef
				ModuleBalanceEqual(types.PerpEFModuleAccount, denoms.NUSD, sdk.NewInt(11)),
			),
	}

	NewTestSuite(t).WithTestCases(tc...).Run()
}

func TestUpdateSwapInvariant(t *testing.T) {
	alice := testutil.AccAddress()
	bob := testutil.AccAddress()
	pairBtcNusd := asset.Registry.Pair(denoms.BTC, denoms.NUSD)
	startBlockTime := time.Now()

	startingSwapInvariant := sdk.NewDec(1_000_000_000_000).Mul(sdk.NewDec(1_000_000_000_000))

	tc := TestCases{
		TC("only long position - no change to swap invariant").
			Given(
				CreateCustomMarket(pairBtcNusd),
				SetBlockTime(startBlockTime),
				SetBlockNumber(1),
				FundAccount(alice, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(10_200_000_000)))),
				FundModule(types.PerpEFModuleAccount, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(100000)))),
			).
			When(
				MarketOrder(alice, pairBtcNusd, types.Direction_LONG, sdk.NewInt(10_000_000_000), sdk.OneDec(), sdk.ZeroDec()),
				ClosePosition(alice, pairBtcNusd),
			).
			Then(
				PositionShouldNotExist(alice, pairBtcNusd),
			),
		TC("only short position - no change to swap invariant").
			Given(
				CreateCustomMarket(pairBtcNusd),
				SetBlockTime(startBlockTime),
				SetBlockNumber(1),
				FundAccount(alice, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(10_200_000_000)))),
				FundModule(types.PerpEFModuleAccount, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(100000)))),
			).
			When(
				MarketOrder(alice, pairBtcNusd, types.Direction_SHORT, sdk.NewInt(10_000_000_000), sdk.OneDec(), sdk.NewDec(10_000_000_000_000)),
				ClosePosition(alice, pairBtcNusd),
			).
			Then(
				PositionShouldNotExist(alice, pairBtcNusd),
			),
		TC("only long position - increasing k").
			Given(
				CreateCustomMarket(pairBtcNusd),
				SetBlockTime(startBlockTime),
				SetBlockNumber(1),
				FundAccount(alice, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(10_200_000_000)))),
				FundModule(types.PerpEFModuleAccount, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(100_000_000)))),
			).
			When(
				MarketOrder(alice, pairBtcNusd, types.Direction_LONG, sdk.NewInt(10_000_000_000), sdk.OneDec(), sdk.ZeroDec()),
				EditSwapInvariant(pairBtcNusd, startingSwapInvariant.MulInt64(100)),
				AMMShouldBeEqual(
					pairBtcNusd,
					AMM_SwapInvariantShouldBeEqual(sdk.MustNewDecFromStr("99999999999999999999999999.999999000000000000"))),
				ClosePosition(alice, pairBtcNusd),
			).
			Then(
				PositionShouldNotExist(alice, pairBtcNusd),
				ModuleBalanceShouldBeEqualTo(types.VaultModuleAccount, sdk.NewCoins()),
			),
		TC("only short position - increasing k").
			Given(
				CreateCustomMarket(pairBtcNusd),
				SetBlockTime(startBlockTime),
				SetBlockNumber(1),
				FundAccount(alice, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(10_200_000_000)))),
				FundModule(types.PerpEFModuleAccount, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(100_000_000)))),
			).
			When(
				MarketOrder(alice, pairBtcNusd, types.Direction_SHORT, sdk.NewInt(10_000_000_000), sdk.OneDec(), sdk.ZeroDec()),
				EditSwapInvariant(pairBtcNusd, startingSwapInvariant.MulInt64(100)),
				AMMShouldBeEqual(
					pairBtcNusd,
					AMM_SwapInvariantShouldBeEqual(sdk.MustNewDecFromStr("99999999999999999999999999.999999000000000000"))),
				ClosePosition(alice, pairBtcNusd),
			).
			Then(
				PositionShouldNotExist(alice, pairBtcNusd),
				ModuleBalanceShouldBeEqualTo(types.VaultModuleAccount, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.OneInt()))),
			),

		TC("only long position - decreasing k").
			Given(
				CreateCustomMarket(pairBtcNusd),
				SetBlockTime(startBlockTime),
				SetBlockNumber(1),
				FundAccount(alice, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(10_200_000_000)))),
				FundModule(types.PerpEFModuleAccount, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(100_000_000)))),
			).
			When(
				MarketOrder(alice, pairBtcNusd, types.Direction_LONG, sdk.NewInt(10_000_000_000), sdk.OneDec(), sdk.ZeroDec()),
				EditSwapInvariant(pairBtcNusd, startingSwapInvariant.Mul(sdk.MustNewDecFromStr("0.1"))),
				AMMShouldBeEqual(
					pairBtcNusd,
					AMM_SwapInvariantShouldBeEqual(sdk.MustNewDecFromStr("99999999999999999873578.871987715651277660"))),
				ClosePosition(alice, pairBtcNusd),
			).
			Then(
				PositionShouldNotExist(alice, pairBtcNusd),
				ModuleBalanceShouldBeEqualTo(types.VaultModuleAccount, sdk.NewCoins()),
			),
		TC("only short position - decreasing k").
			Given(
				CreateCustomMarket(pairBtcNusd),
				SetBlockTime(startBlockTime),
				SetBlockNumber(1),
				FundAccount(alice, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(10_200_000_000)))),
				FundModule(types.PerpEFModuleAccount, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(100_000_000)))),
			).
			When(
				MarketOrder(alice, pairBtcNusd, types.Direction_SHORT, sdk.NewInt(10_000_000_000), sdk.OneDec(), sdk.ZeroDec()),
				EditSwapInvariant(pairBtcNusd, startingSwapInvariant.Mul(sdk.MustNewDecFromStr("0.1"))),
				AMMShouldBeEqual(
					pairBtcNusd,
					AMM_SwapInvariantShouldBeEqual(sdk.MustNewDecFromStr("99999999999999999873578.871987801032774485"))),
				ClosePosition(alice, pairBtcNusd),
			).
			Then(
				PositionShouldNotExist(alice, pairBtcNusd),
				ModuleBalanceShouldBeEqualTo(types.VaultModuleAccount, sdk.NewCoins()),
			),

		TC("long and short position - increasing k").
			Given(
				CreateCustomMarket(pairBtcNusd),
				SetBlockTime(startBlockTime),
				SetBlockNumber(1),
				FundAccount(alice, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(10_200_000_000)))),
				FundAccount(bob, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(10_200_000_000)))),
			).
			When(
				MarketOrder(alice, pairBtcNusd, types.Direction_LONG, sdk.NewInt(10_000_000_000), sdk.OneDec(), sdk.ZeroDec()),
				MarketOrder(bob, pairBtcNusd, types.Direction_SHORT, sdk.NewInt(10_000_000_000), sdk.OneDec(), sdk.NewDec(10_000_000_000_000)),

				EditSwapInvariant(pairBtcNusd, startingSwapInvariant.MulInt64(100)),
				AMMShouldBeEqual(
					pairBtcNusd,
					AMM_BiasShouldBeEqual(sdk.ZeroDec()),
					AMM_SwapInvariantShouldBeEqual(sdk.MustNewDecFromStr("100000000000000000000000000.000000000000000000"))),

				ModuleBalanceShouldBeEqualTo(types.VaultModuleAccount, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(20_000_000_000)))),
				ModuleBalanceShouldBeEqualTo(types.FeePoolModuleAccount, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(20_000_000)))), // Fees are 10_000_000_000 * 0.0010 * 2

				ClosePosition(alice, pairBtcNusd),
				ClosePosition(bob, pairBtcNusd),
			).
			Then(
				PositionShouldNotExist(alice, pairBtcNusd),
				PositionShouldNotExist(bob, pairBtcNusd),

				ModuleBalanceShouldBeEqualTo(types.VaultModuleAccount, sdk.NewCoins()),
				ModuleBalanceShouldBeEqualTo(types.PerpEFModuleAccount, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(39_782_394)))),
			),
		TC("long and short position - reducing k").
			Given(
				CreateCustomMarket(pairBtcNusd),
				SetBlockTime(startBlockTime),
				SetBlockNumber(1),
				FundAccount(alice, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(10_200_000_000)))),
				FundAccount(bob, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(10_200_000_000)))),
			).
			When(
				MarketOrder(alice, pairBtcNusd, types.Direction_LONG, sdk.NewInt(10_000_000_000), sdk.OneDec(), sdk.ZeroDec()),
				MarketOrder(bob, pairBtcNusd, types.Direction_SHORT, sdk.NewInt(10_000_000_000), sdk.OneDec(), sdk.NewDec(10_000_000_000_000)),

				EditSwapInvariant(pairBtcNusd, startingSwapInvariant.Mul(sdk.MustNewDecFromStr("0.1"))),
				AMMShouldBeEqual(
					pairBtcNusd,
					AMM_BiasShouldBeEqual(sdk.ZeroDec()),
					AMM_SwapInvariantShouldBeEqual(sdk.MustNewDecFromStr("99999999999999999873578.871987712489000000"))),

				ModuleBalanceShouldBeEqualTo(types.VaultModuleAccount, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(20_000_000_000)))),
				ModuleBalanceShouldBeEqualTo(types.FeePoolModuleAccount, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(20_000_000)))), // Fees are 10_000_000_000 * 0.0010 * 2

				ClosePosition(alice, pairBtcNusd),
				ClosePosition(bob, pairBtcNusd),
			).
			Then(
				PositionShouldNotExist(alice, pairBtcNusd),
				PositionShouldNotExist(bob, pairBtcNusd),

				ModuleBalanceShouldBeEqualTo(types.VaultModuleAccount, sdk.NewCoins()),
				ModuleBalanceShouldBeEqualTo(types.PerpEFModuleAccount, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(39_200_810)))),
			),
	}

	NewTestSuite(t).WithTestCases(tc...).Run()
}
