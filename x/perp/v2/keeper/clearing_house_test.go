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
	tutilaction "github.com/NibiruChain/nibiru/x/common/testutil/action"
	tutilassert "github.com/NibiruChain/nibiru/x/common/testutil/assertion"
	"github.com/NibiruChain/nibiru/x/common/testutil/mock"
	"github.com/NibiruChain/nibiru/x/common/testutil/testapp"
	perpaction "github.com/NibiruChain/nibiru/x/perp/v2/integration/action"
	perpassert "github.com/NibiruChain/nibiru/x/perp/v2/integration/assertion"

	types "github.com/NibiruChain/nibiru/x/perp/v2/types"
)

func TestMarketOrder(t *testing.T) {
	alice := testutil.AccAddress()
	bob := testutil.AccAddress()
	pairBtcNusd := asset.Registry.Pair(denoms.BTC, denoms.NUSD)
	startBlockTime := time.Now()

	tc := tutilaction.TestCases{
		tutilaction.TC("open big short position and then close after reducing swap invariant").
			Given(
				perpaction.CreateCustomMarket(
					pairBtcNusd,
					perpaction.WithPricePeg(sdk.OneDec()),
					perpaction.WithSqrtDepth(sdk.NewDec(100_000)),
				),
				tutilaction.SetBlockNumber(1),
				tutilaction.SetBlockTime(startBlockTime),

				tutilaction.FundAccount(alice, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(10_200)))),
				tutilaction.FundAccount(bob, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(10_200)))),
				tutilaction.FundModule(types.PerpEFModuleAccount, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(100_000_000)))),

				perpaction.MarketOrder(alice, pairBtcNusd, types.Direction_SHORT, sdk.NewInt(10_000), sdk.OneDec(), sdk.ZeroDec()),
				perpaction.MarketOrder(bob, pairBtcNusd, types.Direction_LONG, sdk.NewInt(10_000), sdk.OneDec(), sdk.ZeroDec()),

				perpaction.EditSwapInvariant(pairBtcNusd, sdk.OneDec()),
			).
			When(
				perpaction.PartialCloseFails(alice, pairBtcNusd, sdk.NewDec(5_000), types.ErrBaseReserveAtZero),
			).
			Then(
				perpaction.ClosePosition(bob, pairBtcNusd),
				perpaction.PartialClose(alice, pairBtcNusd, sdk.NewDec(5_000)),
			),

		tutilaction.TC("new long position").
			Given(
				perpaction.CreateCustomMarket(pairBtcNusd),
				tutilaction.SetBlockTime(startBlockTime),
				tutilaction.SetBlockNumber(1),
				tutilaction.FundAccount(alice, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(1020)))),
			).
			When(
				perpaction.MarketOrder(alice, pairBtcNusd, types.Direction_LONG, sdk.NewInt(1000), sdk.NewDec(10), sdk.ZeroDec(),
					perpaction.MarketOrderResp_PositionShouldBeEqual(
						types.Position{
							Pair:                            pairBtcNusd,
							TraderAddress:                   alice.String(),
							Margin:                          sdk.NewDec(1000),
							OpenNotional:                    sdk.NewDec(10_000),
							Size_:                           sdk.MustNewDecFromStr("9999.999900000001"),
							LastUpdatedBlockNumber:          1,
							LatestCumulativePremiumFraction: sdk.ZeroDec(),
						}),
					perpaction.MarketOrderResp_ExchangeNotionalValueShouldBeEqual(sdk.NewDec(10_000)), // margin * leverage
					perpaction.MarketOrderResp_ExchangedPositionSizeShouldBeEqual(sdk.MustNewDecFromStr("9999.999900000001")),
					perpaction.MarketOrderResp_BadDebtShouldBeEqual(sdk.ZeroDec()),
					perpaction.MarketOrderResp_FundingPaymentShouldBeEqual(sdk.ZeroDec()),
					perpaction.MarketOrderResp_RealizedPnlShouldBeEqual(sdk.ZeroDec()),
					perpaction.MarketOrderResp_UnrealizedPnlAfterShouldBeEqual(sdk.ZeroDec()),
					perpaction.MarketOrderResp_MarginToVaultShouldBeEqual(sdk.NewDec(1000)),
					perpaction.MarketOrderResp_PositionNotionalShouldBeEqual(sdk.NewDec(10_000)),
				),
			).
			Then(
				perpassert.PositionShouldBeEqual(alice, pairBtcNusd, perpassert.Position_PositionShouldBeEqualTo(types.Position{
					Pair:                            pairBtcNusd,
					TraderAddress:                   alice.String(),
					Margin:                          sdk.NewDec(1000),
					OpenNotional:                    sdk.NewDec(10_000),
					Size_:                           sdk.MustNewDecFromStr("9999.999900000001"),
					LastUpdatedBlockNumber:          1,
					LatestCumulativePremiumFraction: sdk.ZeroDec(),
				})),
				perpassert.PositionChangedEventShouldBeEqual(&types.PositionChangedEvent{
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

		tutilaction.TC("existing long position, go more long").
			Given(
				tutilaction.SetBlockNumber(1),
				tutilaction.SetBlockTime(startBlockTime),
				perpaction.CreateCustomMarket(pairBtcNusd),
				tutilaction.FundAccount(alice, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(2040)))),
				perpaction.MarketOrder(alice, pairBtcNusd, types.Direction_LONG, sdk.NewInt(1000), sdk.NewDec(10), sdk.ZeroDec()),
			).
			When(
				tutilaction.MoveToNextBlock(),
				perpaction.MarketOrder(alice, pairBtcNusd, types.Direction_LONG, sdk.NewInt(1000), sdk.NewDec(10), sdk.ZeroDec(),
					perpaction.MarketOrderResp_PositionShouldBeEqual(
						types.Position{
							Pair:                            pairBtcNusd,
							TraderAddress:                   alice.String(),
							Margin:                          sdk.NewDec(2000),
							OpenNotional:                    sdk.NewDec(20_000),
							Size_:                           sdk.MustNewDecFromStr("19999.999600000008000000"),
							LastUpdatedBlockNumber:          2,
							LatestCumulativePremiumFraction: sdk.ZeroDec(),
						}),
					perpaction.MarketOrderResp_ExchangeNotionalValueShouldBeEqual(sdk.NewDec(10_000)), // margin * leverage
					perpaction.MarketOrderResp_ExchangedPositionSizeShouldBeEqual(sdk.MustNewDecFromStr("9999.999700000007000000")),
					perpaction.MarketOrderResp_BadDebtShouldBeEqual(sdk.ZeroDec()),
					perpaction.MarketOrderResp_FundingPaymentShouldBeEqual(sdk.ZeroDec()),
					perpaction.MarketOrderResp_RealizedPnlShouldBeEqual(sdk.ZeroDec()),
					perpaction.MarketOrderResp_UnrealizedPnlAfterShouldBeEqual(sdk.ZeroDec()),
					perpaction.MarketOrderResp_MarginToVaultShouldBeEqual(sdk.NewDec(1000)),
					perpaction.MarketOrderResp_PositionNotionalShouldBeEqual(sdk.NewDec(20_000)),
				),
			).
			Then(
				perpassert.PositionChangedEventShouldBeEqual(&types.PositionChangedEvent{
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

		tutilaction.TC("existing long position, go more long but there's bad debt").
			Given(
				perpaction.CreateCustomMarket(
					pairBtcNusd,
					perpaction.WithPricePeg(sdk.MustNewDecFromStr("0.89")),
					perpaction.WithLatestMarketCPF(sdk.MustNewDecFromStr("0.0002")),
				),
				tutilaction.SetBlockNumber(1),
				tutilaction.SetBlockTime(startBlockTime),
				tutilaction.FundAccount(alice, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(18)))),
				perpaction.InsertPosition(
					perpaction.WithPair(pairBtcNusd),
					perpaction.WithTrader(alice),
					perpaction.WithMargin(sdk.NewDec(1_000)),
					perpaction.WithSize(sdk.NewDec(10_000)),
					perpaction.WithOpenNotional(sdk.NewDec(10_000)),
				),
			).
			When(
				tutilaction.MoveToNextBlock(),
				perpaction.MarketOrderFails(alice, pairBtcNusd, types.Direction_LONG, sdk.OneInt(), sdk.OneDec(), sdk.ZeroDec(),
					types.ErrMarginRatioTooLow,
				),
			).
			Then(
				perpassert.PositionShouldBeEqual(alice, pairBtcNusd, perpassert.Position_PositionShouldBeEqualTo(types.Position{
					TraderAddress:                   alice.String(),
					Pair:                            pairBtcNusd,
					Size_:                           sdk.NewDec(10_000),
					Margin:                          sdk.NewDec(1_000),
					OpenNotional:                    sdk.NewDec(10_000),
					LatestCumulativePremiumFraction: sdk.ZeroDec(),
					LastUpdatedBlockNumber:          0,
				})),
			),

		tutilaction.TC("existing long position, close a bit but there's bad debt").
			Given(
				perpaction.CreateCustomMarket(
					pairBtcNusd,
					perpaction.WithPricePeg(sdk.MustNewDecFromStr("0.89")),
					perpaction.WithLatestMarketCPF(sdk.MustNewDecFromStr("0.0002")),
				),
				tutilaction.SetBlockNumber(1),
				tutilaction.SetBlockTime(startBlockTime),
				tutilaction.FundAccount(alice, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(18)))),
				perpaction.InsertPosition(
					perpaction.WithPair(pairBtcNusd),
					perpaction.WithTrader(alice),
					perpaction.WithMargin(sdk.NewDec(1_000)),
					perpaction.WithSize(sdk.NewDec(10_000)),
					perpaction.WithOpenNotional(sdk.NewDec(10_000)),
				),
			).
			When(
				tutilaction.MoveToNextBlock(),
				perpaction.MarketOrderFails(alice, pairBtcNusd, types.Direction_SHORT, sdk.OneInt(), sdk.OneDec(), sdk.ZeroDec(),
					types.ErrMarginRatioTooLow,
				),
			).
			Then(
				perpassert.PositionShouldBeEqual(alice, pairBtcNusd, perpassert.Position_PositionShouldBeEqualTo(types.Position{
					TraderAddress:                   alice.String(),
					Pair:                            pairBtcNusd,
					Size_:                           sdk.NewDec(10_000),
					Margin:                          sdk.NewDec(1_000),
					OpenNotional:                    sdk.NewDec(10_000),
					LatestCumulativePremiumFraction: sdk.ZeroDec(),
					LastUpdatedBlockNumber:          0,
				})),
			),

		tutilaction.TC("open big long position and then close after reducing swap invariant").
			Given(
				perpaction.CreateCustomMarket(
					pairBtcNusd,
					perpaction.WithPricePeg(sdk.OneDec()),
					perpaction.WithSqrtDepth(sdk.NewDec(100_000)),
				),
				tutilaction.SetBlockNumber(1),
				tutilaction.SetBlockTime(startBlockTime),
				tutilaction.FundAccount(alice, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(10_000)))),
				perpaction.MarketOrder(alice, pairBtcNusd, types.Direction_LONG, sdk.NewInt(9_000), sdk.NewDec(10), sdk.ZeroDec()),
				tutilaction.FundModule(types.PerpEFModuleAccount, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(100_000_000)))),
				perpaction.EditSwapInvariant(pairBtcNusd, sdk.OneDec()),
			).
			When(
				perpaction.ClosePosition(alice, pairBtcNusd),
			).
			Then(
				perpassert.PositionShouldNotExist(alice, pairBtcNusd),
			),

		tutilaction.TC("existing long position, decrease a bit").
			Given(
				perpaction.CreateCustomMarket(pairBtcNusd),
				tutilaction.SetBlockNumber(1),
				tutilaction.SetBlockTime(startBlockTime),
				tutilaction.FundAccount(alice, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(1030)))),
				perpaction.MarketOrder(alice, pairBtcNusd, types.Direction_LONG, sdk.NewInt(1000), sdk.NewDec(10), sdk.ZeroDec()),
			).
			When(
				tutilaction.MoveToNextBlock(),
				perpaction.MarketOrder(alice, pairBtcNusd, types.Direction_SHORT, sdk.NewInt(500), sdk.NewDec(10), sdk.ZeroDec(),
					perpaction.MarketOrderResp_PositionShouldBeEqual(
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
					perpaction.MarketOrderResp_ExchangeNotionalValueShouldBeEqual(sdk.NewDec(5000)), // margin * leverage
					perpaction.MarketOrderResp_ExchangedPositionSizeShouldBeEqual(sdk.MustNewDecFromStr("-4999.999925000000875000")),
					perpaction.MarketOrderResp_BadDebtShouldBeEqual(sdk.ZeroDec()),
					perpaction.MarketOrderResp_FundingPaymentShouldBeEqual(sdk.ZeroDec()),
					perpaction.MarketOrderResp_RealizedPnlShouldBeEqual(sdk.ZeroDec()),
					perpaction.MarketOrderResp_UnrealizedPnlAfterShouldBeEqual(sdk.ZeroDec()),
					perpaction.MarketOrderResp_MarginToVaultShouldBeEqual(sdk.ZeroDec()),
					perpaction.MarketOrderResp_PositionNotionalShouldBeEqual(sdk.NewDec(5000)),
				),
			).
			Then(
				perpassert.PositionChangedEventShouldBeEqual(&types.PositionChangedEvent{
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

		tutilaction.TC("existing long position, decrease a bit but there's bad debt").
			Given(
				perpaction.CreateCustomMarket(
					pairBtcNusd,
					perpaction.WithPricePeg(sdk.MustNewDecFromStr("0.89")),
					perpaction.WithLatestMarketCPF(sdk.MustNewDecFromStr("0.0002")),
				),
				tutilaction.SetBlockNumber(1),
				tutilaction.SetBlockTime(startBlockTime),
				tutilaction.FundAccount(alice, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(18)))),
				perpaction.InsertPosition(
					perpaction.WithPair(pairBtcNusd),
					perpaction.WithTrader(alice),
					perpaction.WithMargin(sdk.NewDec(1_000)),
					perpaction.WithSize(sdk.NewDec(10_000)),
					perpaction.WithOpenNotional(sdk.NewDec(10_000)),
				),
			).
			When(
				tutilaction.MoveToNextBlock(),
				perpaction.MarketOrderFails(alice, pairBtcNusd, types.Direction_LONG, sdk.OneInt(), sdk.OneDec(), sdk.ZeroDec(),
					types.ErrMarginRatioTooLow,
				),
			).
			Then(
				perpassert.PositionShouldBeEqual(alice, pairBtcNusd, perpassert.Position_PositionShouldBeEqualTo(types.Position{
					TraderAddress:                   alice.String(),
					Pair:                            pairBtcNusd,
					Size_:                           sdk.NewDec(10_000),
					Margin:                          sdk.NewDec(1_000),
					OpenNotional:                    sdk.NewDec(10_000),
					LatestCumulativePremiumFraction: sdk.ZeroDec(),
					LastUpdatedBlockNumber:          0,
				})),
			),

		tutilaction.TC("existing long position, decrease a lot").
			Given(
				perpaction.CreateCustomMarket(pairBtcNusd),
				tutilaction.SetBlockNumber(1),
				tutilaction.SetBlockTime(startBlockTime),
				tutilaction.FundAccount(alice, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(4080)))),
				perpaction.MarketOrder(alice, pairBtcNusd, types.Direction_LONG, sdk.NewInt(1000), sdk.NewDec(10), sdk.ZeroDec()),
			).
			When(
				tutilaction.MoveToNextBlock(),
				perpaction.MarketOrder(alice, pairBtcNusd, types.Direction_SHORT, sdk.NewInt(3000), sdk.NewDec(10), sdk.ZeroDec(),
					perpaction.MarketOrderResp_PositionShouldBeEqual(
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
					perpaction.MarketOrderResp_ExchangeNotionalValueShouldBeEqual(sdk.NewDec(30000)), // margin * leverage
					perpaction.MarketOrderResp_ExchangedPositionSizeShouldBeEqual(sdk.MustNewDecFromStr("-30000.000300000009000000")),
					perpaction.MarketOrderResp_BadDebtShouldBeEqual(sdk.ZeroDec()),
					perpaction.MarketOrderResp_FundingPaymentShouldBeEqual(sdk.ZeroDec()),
					perpaction.MarketOrderResp_RealizedPnlShouldBeEqual(sdk.ZeroDec()),
					perpaction.MarketOrderResp_UnrealizedPnlAfterShouldBeEqual(sdk.ZeroDec()),
					perpaction.MarketOrderResp_MarginToVaultShouldBeEqual(sdk.NewDec(1000)),
					perpaction.MarketOrderResp_PositionNotionalShouldBeEqual(sdk.NewDec(20000)),
				),
			).
			Then(
				perpassert.PositionChangedEventShouldBeEqual(&types.PositionChangedEvent{
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

		tutilaction.TC("existing long position, decrease a lot but there's bad debt").
			Given(
				perpaction.CreateCustomMarket(
					pairBtcNusd,
					perpaction.WithPricePeg(sdk.MustNewDecFromStr("0.89")),
					perpaction.WithLatestMarketCPF(sdk.MustNewDecFromStr("0.0002")),
				),
				tutilaction.SetBlockNumber(1),
				tutilaction.SetBlockTime(startBlockTime),
				tutilaction.FundAccount(alice, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(18)))),
				perpaction.InsertPosition(
					perpaction.WithPair(pairBtcNusd),
					perpaction.WithTrader(alice),
					perpaction.WithMargin(sdk.NewDec(1_000)),
					perpaction.WithSize(sdk.NewDec(10_000)),
					perpaction.WithOpenNotional(sdk.NewDec(10_000)),
				),
			).
			When(
				tutilaction.MoveToNextBlock(),
				perpaction.MarketOrder(alice, pairBtcNusd, types.Direction_SHORT, sdk.NewInt(3000), sdk.NewDec(10), sdk.ZeroDec(),
					perpaction.MarketOrderResp_PositionShouldBeEqual(
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
					perpaction.MarketOrderResp_ExchangeNotionalValueShouldBeEqual(sdk.MustNewDecFromStr("8899.999911000000890000")),
					perpaction.MarketOrderResp_ExchangedPositionSizeShouldBeEqual(sdk.MustNewDecFromStr("-10000")),
					perpaction.MarketOrderResp_BadDebtShouldBeEqual(sdk.MustNewDecFromStr("102.000088999999110000")),
					perpaction.MarketOrderResp_FundingPaymentShouldBeEqual(sdk.NewDec(2)),
					perpaction.MarketOrderResp_RealizedPnlShouldBeEqual(sdk.MustNewDecFromStr("-1100.000088999999110000")),
					perpaction.MarketOrderResp_UnrealizedPnlAfterShouldBeEqual(sdk.ZeroDec()),
					perpaction.MarketOrderResp_MarginToVaultShouldBeEqual(sdk.ZeroDec()),
					perpaction.MarketOrderResp_PositionNotionalShouldBeEqual(sdk.ZeroDec()),
				),
			).
			Then(
				perpassert.PositionChangedEventShouldBeEqual(&types.PositionChangedEvent{
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

		tutilaction.TC("new short position").
			Given(
				perpaction.CreateCustomMarket(pairBtcNusd),
				tutilaction.SetBlockTime(startBlockTime),
				tutilaction.SetBlockNumber(1),
				tutilaction.FundAccount(alice, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(1020)))),
			).
			When(
				perpaction.MarketOrder(alice, pairBtcNusd, types.Direction_SHORT, sdk.NewInt(1000), sdk.NewDec(10), sdk.ZeroDec(),
					perpaction.MarketOrderResp_PositionShouldBeEqual(
						types.Position{
							Pair:                            pairBtcNusd,
							TraderAddress:                   alice.String(),
							Margin:                          sdk.NewDec(1000),
							OpenNotional:                    sdk.NewDec(10_000),
							Size_:                           sdk.MustNewDecFromStr("-10000.000100000001000000"),
							LastUpdatedBlockNumber:          1,
							LatestCumulativePremiumFraction: sdk.ZeroDec(),
						}),
					perpaction.MarketOrderResp_ExchangeNotionalValueShouldBeEqual(sdk.NewDec(10_000)), // margin * leverage
					perpaction.MarketOrderResp_ExchangedPositionSizeShouldBeEqual(sdk.MustNewDecFromStr("-10000.000100000001000000")),
					perpaction.MarketOrderResp_BadDebtShouldBeEqual(sdk.ZeroDec()),
					perpaction.MarketOrderResp_FundingPaymentShouldBeEqual(sdk.ZeroDec()),
					perpaction.MarketOrderResp_RealizedPnlShouldBeEqual(sdk.ZeroDec()),
					perpaction.MarketOrderResp_UnrealizedPnlAfterShouldBeEqual(sdk.ZeroDec()),
					perpaction.MarketOrderResp_MarginToVaultShouldBeEqual(sdk.NewDec(1000)),
					perpaction.MarketOrderResp_PositionNotionalShouldBeEqual(sdk.NewDec(10_000)),
				),
			).
			Then(
				perpassert.PositionShouldBeEqual(alice, pairBtcNusd,
					perpassert.Position_PositionShouldBeEqualTo(types.Position{
						Pair:                            pairBtcNusd,
						TraderAddress:                   alice.String(),
						Margin:                          sdk.NewDec(1000),
						OpenNotional:                    sdk.NewDec(10_000),
						Size_:                           sdk.MustNewDecFromStr("-10000.000100000001000000"),
						LastUpdatedBlockNumber:          1,
						LatestCumulativePremiumFraction: sdk.ZeroDec(),
					}),
				),
				perpassert.PositionChangedEventShouldBeEqual(&types.PositionChangedEvent{
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

		tutilaction.TC("existing short position, go more short").
			Given(
				perpaction.CreateCustomMarket(pairBtcNusd),
				tutilaction.SetBlockNumber(1),
				tutilaction.SetBlockTime(startBlockTime),
				tutilaction.FundAccount(alice, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(2040)))),
				perpaction.MarketOrder(alice, pairBtcNusd, types.Direction_SHORT, sdk.NewInt(1000), sdk.NewDec(10), sdk.ZeroDec()),
			).
			When(
				tutilaction.MoveToNextBlock(),
				perpaction.MarketOrder(alice, pairBtcNusd, types.Direction_SHORT, sdk.NewInt(1000), sdk.NewDec(10), sdk.ZeroDec(),
					perpaction.MarketOrderResp_PositionShouldBeEqual(
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
					perpaction.MarketOrderResp_ExchangeNotionalValueShouldBeEqual(sdk.NewDec(10_000)), // margin * leverage
					perpaction.MarketOrderResp_ExchangedPositionSizeShouldBeEqual(sdk.MustNewDecFromStr("-10000.000300000007000000")),
					perpaction.MarketOrderResp_BadDebtShouldBeEqual(sdk.ZeroDec()),
					perpaction.MarketOrderResp_FundingPaymentShouldBeEqual(sdk.ZeroDec()),
					perpaction.MarketOrderResp_RealizedPnlShouldBeEqual(sdk.ZeroDec()),
					perpaction.MarketOrderResp_UnrealizedPnlAfterShouldBeEqual(sdk.ZeroDec()),
					perpaction.MarketOrderResp_MarginToVaultShouldBeEqual(sdk.NewDec(1000)),
					perpaction.MarketOrderResp_PositionNotionalShouldBeEqual(sdk.NewDec(20_000)),
				),
			).
			Then(
				perpassert.PositionChangedEventShouldBeEqual(&types.PositionChangedEvent{
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

		tutilaction.TC("existing short position, go more short but there's bad debt").
			Given(
				perpaction.CreateCustomMarket(pairBtcNusd,
					perpaction.WithPricePeg(sdk.MustNewDecFromStr("1.11")),
					perpaction.WithLatestMarketCPF(sdk.MustNewDecFromStr("0.0002")),
				),
				tutilaction.SetBlockNumber(1),
				tutilaction.SetBlockTime(startBlockTime),
				tutilaction.FundAccount(alice, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(22)))),
				perpaction.InsertPosition(
					perpaction.WithPair(pairBtcNusd),
					perpaction.WithTrader(alice),
					perpaction.WithMargin(sdk.NewDec(1_000)),
					perpaction.WithSize(sdk.NewDec(-10_000)),
					perpaction.WithOpenNotional(sdk.NewDec(10_000)),
				),
			).
			When(
				tutilaction.MoveToNextBlock(),
				perpaction.MarketOrderFails(alice, pairBtcNusd, types.Direction_SHORT, sdk.OneInt(), sdk.OneDec(), sdk.ZeroDec(),
					types.ErrMarginRatioTooLow,
				),
			).
			Then(
				perpassert.PositionShouldBeEqual(alice, pairBtcNusd, perpassert.Position_PositionShouldBeEqualTo(types.Position{
					TraderAddress:                   alice.String(),
					Pair:                            pairBtcNusd,
					Size_:                           sdk.NewDec(-10_000),
					Margin:                          sdk.NewDec(1_000),
					OpenNotional:                    sdk.NewDec(10_000),
					LatestCumulativePremiumFraction: sdk.ZeroDec(),
					LastUpdatedBlockNumber:          0,
				})),
			),

		tutilaction.TC("existing short position, decrease a bit").
			Given(
				perpaction.CreateCustomMarket(pairBtcNusd),
				tutilaction.SetBlockNumber(1),
				tutilaction.SetBlockTime(startBlockTime),
				tutilaction.FundAccount(alice, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(1030)))),
				perpaction.MarketOrder(alice, pairBtcNusd, types.Direction_SHORT, sdk.NewInt(1000), sdk.NewDec(10), sdk.ZeroDec()),
			).
			When(
				tutilaction.MoveToNextBlock(),
				perpaction.MarketOrder(alice, pairBtcNusd, types.Direction_LONG, sdk.NewInt(500), sdk.NewDec(10), sdk.ZeroDec(),
					perpaction.MarketOrderResp_PositionShouldBeEqual(
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
					perpaction.MarketOrderResp_ExchangeNotionalValueShouldBeEqual(sdk.NewDec(5000)), // margin * leverage
					perpaction.MarketOrderResp_ExchangedPositionSizeShouldBeEqual(sdk.MustNewDecFromStr("5000.000075000000875000")),
					perpaction.MarketOrderResp_BadDebtShouldBeEqual(sdk.ZeroDec()),
					perpaction.MarketOrderResp_FundingPaymentShouldBeEqual(sdk.ZeroDec()),
					perpaction.MarketOrderResp_RealizedPnlShouldBeEqual(sdk.ZeroDec()),
					perpaction.MarketOrderResp_UnrealizedPnlAfterShouldBeEqual(sdk.ZeroDec()),
					perpaction.MarketOrderResp_MarginToVaultShouldBeEqual(sdk.ZeroDec()),
					perpaction.MarketOrderResp_PositionNotionalShouldBeEqual(sdk.NewDec(5000)),
				),
			).
			Then(
				perpassert.PositionChangedEventShouldBeEqual(&types.PositionChangedEvent{
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

		tutilaction.TC("existing short position, decrease a bit but there's bad debt").
			Given(
				perpaction.CreateCustomMarket(pairBtcNusd,
					perpaction.WithPricePeg(sdk.MustNewDecFromStr("1.11")),
					perpaction.WithLatestMarketCPF(sdk.MustNewDecFromStr("0.0002")),
				),
				tutilaction.SetBlockNumber(1),
				tutilaction.SetBlockTime(startBlockTime),
				tutilaction.FundAccount(alice, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(22)))),
				perpaction.InsertPosition(
					perpaction.WithPair(pairBtcNusd),
					perpaction.WithTrader(alice),
					perpaction.WithMargin(sdk.NewDec(1_000)),
					perpaction.WithSize(sdk.NewDec(-10_000)),
					perpaction.WithOpenNotional(sdk.NewDec(10_000)),
				),
			).
			When(
				tutilaction.MoveToNextBlock(),
				perpaction.MarketOrderFails(alice, pairBtcNusd, types.Direction_LONG, sdk.OneInt(), sdk.OneDec(), sdk.ZeroDec(),
					types.ErrMarginRatioTooLow,
				),
			).
			Then(
				perpassert.PositionShouldBeEqual(alice, pairBtcNusd, perpassert.Position_PositionShouldBeEqualTo(types.Position{
					TraderAddress:                   alice.String(),
					Pair:                            pairBtcNusd,
					Size_:                           sdk.NewDec(-10_000),
					Margin:                          sdk.NewDec(1_000),
					OpenNotional:                    sdk.NewDec(10_000),
					LatestCumulativePremiumFraction: sdk.ZeroDec(),
					LastUpdatedBlockNumber:          0,
				})),
			),

		tutilaction.TC("existing short position, decrease a lot").
			Given(
				perpaction.CreateCustomMarket(pairBtcNusd),
				tutilaction.SetBlockNumber(1),
				tutilaction.SetBlockTime(startBlockTime),
				tutilaction.FundAccount(alice, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(4080)))),
				perpaction.MarketOrder(alice, pairBtcNusd, types.Direction_SHORT, sdk.NewInt(1000), sdk.NewDec(10), sdk.ZeroDec()),
			).
			When(
				tutilaction.MoveToNextBlock(),
				perpaction.MarketOrder(alice, pairBtcNusd, types.Direction_LONG, sdk.NewInt(3000), sdk.NewDec(10), sdk.ZeroDec(),
					perpaction.MarketOrderResp_PositionShouldBeEqual(
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
					perpaction.MarketOrderResp_ExchangeNotionalValueShouldBeEqual(sdk.NewDec(30000)), // margin * leverage
					perpaction.MarketOrderResp_ExchangedPositionSizeShouldBeEqual(sdk.MustNewDecFromStr("29999.999700000009000000")),
					perpaction.MarketOrderResp_BadDebtShouldBeEqual(sdk.ZeroDec()),
					perpaction.MarketOrderResp_FundingPaymentShouldBeEqual(sdk.ZeroDec()),
					perpaction.MarketOrderResp_RealizedPnlShouldBeEqual(sdk.ZeroDec()),
					perpaction.MarketOrderResp_UnrealizedPnlAfterShouldBeEqual(sdk.ZeroDec()),
					perpaction.MarketOrderResp_MarginToVaultShouldBeEqual(sdk.NewDec(1000)),
					perpaction.MarketOrderResp_PositionNotionalShouldBeEqual(sdk.NewDec(20000)),
				),
			).
			Then(
				perpassert.PositionChangedEventShouldBeEqual(&types.PositionChangedEvent{
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

		tutilaction.TC("existing short position, decrease a lot but there's bad debt").
			Given(
				perpaction.CreateCustomMarket(pairBtcNusd,
					perpaction.WithPricePeg(sdk.MustNewDecFromStr("1.11")),
					perpaction.WithLatestMarketCPF(sdk.MustNewDecFromStr("0.0002")),
				),
				tutilaction.SetBlockNumber(1),
				tutilaction.SetBlockTime(startBlockTime),
				tutilaction.FundAccount(alice, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(22)))),
				perpaction.InsertPosition(
					perpaction.WithPair(pairBtcNusd),
					perpaction.WithTrader(alice),
					perpaction.WithMargin(sdk.NewDec(1_000)),
					perpaction.WithSize(sdk.NewDec(-10_000)),
					perpaction.WithOpenNotional(sdk.NewDec(10_000)),
				),
			).
			When(
				tutilaction.MoveToNextBlock(),
				perpaction.MarketOrder(alice, pairBtcNusd, types.Direction_LONG, sdk.NewInt(3000), sdk.NewDec(10), sdk.ZeroDec(),
					perpaction.MarketOrderResp_PositionShouldBeEqual(
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
					perpaction.MarketOrderResp_ExchangeNotionalValueShouldBeEqual(sdk.MustNewDecFromStr("11100.000111000001110000")),
					perpaction.MarketOrderResp_ExchangedPositionSizeShouldBeEqual(sdk.MustNewDecFromStr("10000")),
					perpaction.MarketOrderResp_BadDebtShouldBeEqual(sdk.MustNewDecFromStr("98.000111000001110000")),
					perpaction.MarketOrderResp_FundingPaymentShouldBeEqual(sdk.NewDec(-2)),
					perpaction.MarketOrderResp_RealizedPnlShouldBeEqual(sdk.MustNewDecFromStr("-1100.000111000001110000")),
					perpaction.MarketOrderResp_UnrealizedPnlAfterShouldBeEqual(sdk.ZeroDec()),
					perpaction.MarketOrderResp_MarginToVaultShouldBeEqual(sdk.ZeroDec()),
					perpaction.MarketOrderResp_PositionNotionalShouldBeEqual(sdk.ZeroDec()),
				),
			).
			Then(
				perpassert.PositionChangedEventShouldBeEqual(&types.PositionChangedEvent{
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

		tutilaction.TC("user has insufficient funds").
			Given(
				perpaction.CreateCustomMarket(pairBtcNusd),
				tutilaction.SetBlockTime(startBlockTime),
				tutilaction.SetBlockNumber(1),
				tutilaction.FundAccount(alice, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(99)))),
			).
			When(
				perpaction.MarketOrderFails(
					alice, pairBtcNusd, types.Direction_LONG, sdk.NewInt(100), sdk.OneDec(), sdk.ZeroDec(),
					sdkerrors.ErrInsufficientFunds),
			).
			Then(
				perpassert.PositionShouldNotExist(alice, pairBtcNusd),
			),

		tutilaction.TC("new long position, can close position after market is not enabled").
			Given(
				perpaction.CreateCustomMarket(pairBtcNusd),
				tutilaction.SetBlockTime(startBlockTime),
				tutilaction.SetBlockNumber(1),
				tutilaction.FundAccount(alice, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(47_714_285_715)))),
				perpaction.MarketOrder(alice, pairBtcNusd, types.Direction_SHORT, sdk.NewInt(47_619_047_619), sdk.OneDec(), sdk.ZeroDec()),
				perpaction.SetMarketEnabled(pairBtcNusd, false),
			).
			When(
				perpaction.ClosePosition(alice, pairBtcNusd),
			).
			Then(
				perpassert.PositionShouldNotExist(alice, pairBtcNusd),
			),

		tutilaction.TC("new long position, can not open new position after market is not enabled").
			Given(
				perpaction.CreateCustomMarket(pairBtcNusd),
				tutilaction.SetBlockTime(startBlockTime),
				tutilaction.SetBlockNumber(1),
				tutilaction.FundAccount(alice, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(47_714_285_715)))),
				perpaction.SetMarketEnabled(pairBtcNusd, false),
			).
			When(
				perpaction.MarketOrderFails(alice, pairBtcNusd, types.Direction_SHORT, sdk.NewInt(47_619_047_619), sdk.OneDec(), sdk.ZeroDec(),
					types.ErrMarketNotEnabled),
			).
			Then(
				perpassert.PositionShouldNotExist(alice, pairBtcNusd),
			),

		tutilaction.TC("existing long position, can not open new one but can close").
			Given(
				perpaction.CreateCustomMarket(pairBtcNusd),
				tutilaction.SetBlockTime(startBlockTime),
				tutilaction.SetBlockNumber(1),
				tutilaction.FundAccount(alice, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(47_714_285_715)))),
				perpaction.MarketOrder(alice, pairBtcNusd, types.Direction_SHORT, sdk.NewInt(50_000), sdk.OneDec(), sdk.ZeroDec()),
				perpaction.SetMarketEnabled(pairBtcNusd, false),
			).
			When(
				perpaction.MarketOrderFails(alice, pairBtcNusd, types.Direction_SHORT, sdk.NewInt(47_619_047_619), sdk.OneDec(), sdk.ZeroDec(),
					types.ErrMarketNotEnabled),
				perpaction.ClosePosition(alice, pairBtcNusd),
			).
			Then(
				perpassert.PositionShouldNotExist(alice, pairBtcNusd),
			),

		tutilaction.TC("market doesn't exist").
			Given(
				tutilaction.SetBlockTime(startBlockTime),
				tutilaction.SetBlockNumber(1),
				tutilaction.FundAccount(alice, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(47_714_285_715)))),
			).
			When(
				perpaction.MarketOrderFails(alice, pairBtcNusd, types.Direction_SHORT, sdk.NewInt(47_619_047_619), sdk.OneDec(), sdk.ZeroDec(),
					types.ErrPairNotFound),
			).
			Then(
				perpassert.PositionShouldNotExist(alice, pairBtcNusd),
			),

		tutilaction.TC("zero quote asset amount").
			Given(
				tutilaction.SetBlockTime(startBlockTime),
				tutilaction.SetBlockNumber(1),
				perpaction.CreateCustomMarket(pairBtcNusd),
				tutilaction.FundAccount(alice, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(47_714_285_715)))),
			).
			When(
				perpaction.MarketOrderFails(alice, pairBtcNusd, types.Direction_LONG, sdk.ZeroInt(), sdk.OneDec(), sdk.ZeroDec(),
					types.ErrInputQuoteAmtNegative),
			).
			Then(
				perpassert.PositionShouldNotExist(alice, pairBtcNusd),
			),

		tutilaction.TC("zero leverage").
			Given(
				tutilaction.SetBlockTime(startBlockTime),
				tutilaction.SetBlockNumber(1),
				perpaction.CreateCustomMarket(pairBtcNusd),
				tutilaction.FundAccount(alice, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(1e6)))),
			).
			When(
				perpaction.MarketOrderFails(alice, pairBtcNusd, types.Direction_LONG, sdk.OneInt(), sdk.ZeroDec(), sdk.ZeroDec(),
					types.ErrUserLeverageNegative),
			).
			Then(
				perpassert.PositionShouldNotExist(alice, pairBtcNusd),
			),

		tutilaction.TC("user leverage greater than market max leverage").
			Given(
				tutilaction.SetBlockTime(startBlockTime),
				tutilaction.SetBlockNumber(1),
				perpaction.CreateCustomMarket(pairBtcNusd),
				tutilaction.FundAccount(alice, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(1e6)))),
			).
			When(
				perpaction.MarketOrderFails(alice, pairBtcNusd, types.Direction_LONG, sdk.OneInt(), sdk.NewDec(11), sdk.ZeroDec(),
					types.ErrLeverageIsTooHigh),
			).
			Then(
				perpassert.PositionShouldNotExist(alice, pairBtcNusd),
			),

		tutilaction.TC("position should not exist after opening a closing manually").
			Given(
				tutilaction.SetBlockTime(startBlockTime),
				tutilaction.SetBlockNumber(1),
				perpaction.CreateCustomMarket(pairBtcNusd, perpaction.WithPricePeg(sdk.MustNewDecFromStr("25001.0112"))),
				tutilaction.FundAccount(alice, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(20_000_000_000+20_000_000)))),
			).
			When(
				perpaction.MarketOrder(alice, pairBtcNusd, types.Direction_SHORT, sdk.NewInt(10_000_000_000), sdk.OneDec(), sdk.ZeroDec()),
				perpaction.MarketOrder(alice, pairBtcNusd, types.Direction_LONG, sdk.NewInt(10_000_000_000), sdk.OneDec(), sdk.ZeroDec()),
			).
			Then(
				perpassert.PositionShouldNotExist(alice, pairBtcNusd),
			),

		tutilaction.TC("position should not exist after opening a closing manually - reverse with leverage").
			Given(
				tutilaction.SetBlockTime(startBlockTime),
				tutilaction.SetBlockNumber(1),
				perpaction.CreateCustomMarket(pairBtcNusd, perpaction.WithPricePeg(sdk.MustNewDecFromStr("25001.0112"))),
				tutilaction.FundAccount(alice, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(1e6)))),
			).
			When(
				perpaction.MarketOrder(alice, pairBtcNusd, types.Direction_SHORT, sdk.NewInt(100_000), sdk.OneDec(), sdk.ZeroDec()),
				perpaction.MarketOrder(alice, pairBtcNusd, types.Direction_LONG, sdk.NewInt(50_000), sdk.NewDec(2), sdk.ZeroDec()),
			).
			Then(
				perpassert.PositionShouldNotExist(alice, pairBtcNusd),
			),
		tutilaction.TC("position should not exist after opening a closing manually - open with leverage").
			Given(
				tutilaction.SetBlockTime(startBlockTime),
				tutilaction.SetBlockNumber(1),
				perpaction.CreateCustomMarket(pairBtcNusd, perpaction.WithPricePeg(sdk.MustNewDecFromStr("25001.0112"))),
				tutilaction.FundAccount(alice, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(1e6)))),
			).
			When(
				perpaction.MarketOrder(alice, pairBtcNusd, types.Direction_LONG, sdk.NewInt(50_000), sdk.NewDec(2), sdk.ZeroDec()),
				perpaction.MarketOrder(alice, pairBtcNusd, types.Direction_SHORT, sdk.NewInt(100_000), sdk.OneDec(), sdk.ZeroDec()),
			).
			Then(
				perpassert.PositionShouldNotExist(alice, pairBtcNusd),
			),

		tutilaction.TC("position should not exist after opening a closing manually - reverse with leverage").
			Given(
				tutilaction.SetBlockTime(startBlockTime),
				tutilaction.SetBlockNumber(1),
				perpaction.CreateCustomMarket(pairBtcNusd, perpaction.WithPricePeg(sdk.MustNewDecFromStr("25001.0112"))),
				tutilaction.FundAccount(alice, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(1e6)))),
			).
			When(
				perpaction.MarketOrder(alice, pairBtcNusd, types.Direction_SHORT, sdk.NewInt(100_000), sdk.OneDec(), sdk.ZeroDec()),
				perpaction.MarketOrder(alice, pairBtcNusd, types.Direction_LONG, sdk.NewInt(50_000), sdk.NewDec(2), sdk.ZeroDec()),
			).
			Then(
				perpassert.PositionShouldNotExist(alice, pairBtcNusd),
			),

		tutilaction.TC("position should not exist after opening a closing manually - reverse with leverage - more steps").
			Given(
				tutilaction.SetBlockTime(startBlockTime),
				tutilaction.SetBlockNumber(1),
				perpaction.CreateCustomMarket(pairBtcNusd, perpaction.WithPricePeg(sdk.MustNewDecFromStr("25000"))),
				tutilaction.FundAccount(alice, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(1e6)))),
			).
			When(
				perpaction.MarketOrder(alice, pairBtcNusd, types.Direction_SHORT, sdk.NewInt(100_000), sdk.OneDec(), sdk.ZeroDec()),
				perpaction.MarketOrder(alice, pairBtcNusd, types.Direction_LONG, sdk.NewInt(50_000), sdk.NewDec(4), sdk.ZeroDec()),
				perpaction.MarketOrder(alice, pairBtcNusd, types.Direction_SHORT, sdk.NewInt(50_000), sdk.NewDec(2), sdk.ZeroDec()),
			).
			Then(
				perpassert.PositionShouldNotExist(alice, pairBtcNusd),
			),
	}

	tutilaction.NewTestSuite(t).WithTestCases(tc...).Run()
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
			app.PerpKeeperV2.Markets.Insert(ctx, asset.Registry.Pair(denoms.BTC, denoms.NUSD), *market)
			amm := mock.TestAMMDefault()
			app.PerpKeeperV2.AMMs.Insert(ctx, amm.Pair, *amm)
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

	tc := tutilaction.TestCases{
		tutilaction.TC("partial close long position with positive PnL").
			Given(
				perpaction.CreateCustomMarket(
					pairBtcNusd,
					perpaction.WithPricePeg(sdk.NewDec(2)),
					perpaction.WithLatestMarketCPF(sdk.MustNewDecFromStr("0.0002")),
				),
				tutilaction.SetBlockTime(startBlockTime),
				tutilaction.SetBlockNumber(1),
				tutilaction.FundAccount(alice, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(10)))),
				perpaction.InsertPosition(
					perpaction.WithPair(pairBtcNusd),
					perpaction.WithTrader(alice),
					perpaction.WithSize(sdk.NewDec(10_000)),
					perpaction.WithMargin(sdk.NewDec(1_000)),
					perpaction.WithOpenNotional(sdk.NewDec(10_000)),
				),
			).
			When(
				perpaction.PartialClose(alice, pairBtcNusd, sdk.NewDec(2_500)),
			).
			Then(
				perpassert.PositionShouldBeEqual(alice, pairBtcNusd, perpassert.Position_PositionShouldBeEqualTo(types.Position{
					Pair:                            pairBtcNusd,
					TraderAddress:                   alice.String(),
					Margin:                          sdk.MustNewDecFromStr("3497.999950000000500000"),
					OpenNotional:                    sdk.MustNewDecFromStr("7499.999962500000468750"),
					Size_:                           sdk.MustNewDecFromStr("7500.000000000000000000"),
					LastUpdatedBlockNumber:          1,
					LatestCumulativePremiumFraction: sdk.MustNewDecFromStr("0.0002"),
				})),
				perpassert.PositionChangedEventShouldBeEqual(&types.PositionChangedEvent{
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

		tutilaction.TC("partial close long position with negative PnL").
			Given(
				perpaction.CreateCustomMarket(
					pairBtcNusd,
					perpaction.WithPricePeg(sdk.MustNewDecFromStr("0.95")),
					perpaction.WithLatestMarketCPF(sdk.MustNewDecFromStr("0.0002")),
				),
				tutilaction.SetBlockTime(startBlockTime),
				tutilaction.SetBlockNumber(1),
				tutilaction.FundAccount(alice, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(4)))),
				perpaction.InsertPosition(
					perpaction.WithPair(pairBtcNusd),
					perpaction.WithTrader(alice),
					perpaction.WithSize(sdk.NewDec(10_000)),
					perpaction.WithMargin(sdk.NewDec(1_000)),
					perpaction.WithOpenNotional(sdk.NewDec(10_000)),
				),
			).
			When(
				perpaction.PartialClose(alice, pairBtcNusd, sdk.NewDec(2_500)),
			).
			Then(
				perpassert.PositionShouldBeEqual(alice, pairBtcNusd, perpassert.Position_PositionShouldBeEqualTo(types.Position{
					Pair:                            pairBtcNusd,
					TraderAddress:                   alice.String(),
					Margin:                          sdk.MustNewDecFromStr("872.999976250000237500"),
					OpenNotional:                    sdk.MustNewDecFromStr("7499.999982187500222656"),
					Size_:                           sdk.MustNewDecFromStr("7500.000000000000000000"),
					LastUpdatedBlockNumber:          1,
					LatestCumulativePremiumFraction: sdk.MustNewDecFromStr("0.0002"),
				})),
				perpassert.PositionChangedEventShouldBeEqual(&types.PositionChangedEvent{
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

		tutilaction.TC("partial close long position without bad debt but below maintenance margin ratio").
			Given(
				perpaction.CreateCustomMarket(
					pairBtcNusd,
					perpaction.WithPricePeg(sdk.MustNewDecFromStr("0.94")),
					perpaction.WithLatestMarketCPF(sdk.MustNewDecFromStr("0.0002")),
				),
				tutilaction.SetBlockTime(startBlockTime),
				tutilaction.SetBlockNumber(1),
				tutilaction.FundAccount(alice, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(4)))),
				perpaction.InsertPosition(
					perpaction.WithPair(pairBtcNusd),
					perpaction.WithTrader(alice),
					perpaction.WithSize(sdk.NewDec(10_000)),
					perpaction.WithMargin(sdk.NewDec(1_000)),
					perpaction.WithOpenNotional(sdk.NewDec(10_000)),
				),
			).
			When(
				perpaction.PartialClose(alice, pairBtcNusd, sdk.NewDec(2_500)),
			).
			Then(
				perpassert.PositionShouldBeEqual(alice, pairBtcNusd, perpassert.Position_PositionShouldBeEqualTo(types.Position{
					Pair:                            pairBtcNusd,
					TraderAddress:                   alice.String(),
					Margin:                          sdk.MustNewDecFromStr("847.999976500000235000"),
					OpenNotional:                    sdk.MustNewDecFromStr("7499.999982375000220312"),
					Size_:                           sdk.MustNewDecFromStr("7499.999999999999999999"),
					LastUpdatedBlockNumber:          1,
					LatestCumulativePremiumFraction: sdk.MustNewDecFromStr("0.0002"),
				})),
				perpassert.PositionChangedEventShouldBeEqual(&types.PositionChangedEvent{
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

		tutilaction.TC("partial close long position with bad debt").
			Given(
				perpaction.CreateCustomMarket(
					pairBtcNusd,
					perpaction.WithPricePeg(sdk.MustNewDecFromStr("0.59")),
					perpaction.WithLatestMarketCPF(sdk.MustNewDecFromStr("0.0002")),
				),
				tutilaction.SetBlockTime(startBlockTime),
				tutilaction.SetBlockNumber(1),
				tutilaction.FundAccount(alice, sdk.NewCoins(sdk.NewInt64Coin(denoms.NUSD, 2))),
				tutilaction.FundModule(types.PerpEFModuleAccount, sdk.NewCoins(sdk.NewInt64Coin(denoms.NUSD, 27))),
				perpaction.InsertPosition(
					perpaction.WithPair(pairBtcNusd),
					perpaction.WithTrader(alice),
					perpaction.WithSize(sdk.NewDec(10_000)),
					perpaction.WithMargin(sdk.NewDec(1_000)),
					perpaction.WithOpenNotional(sdk.NewDec(10_000)),
				),
			).
			When(
				perpaction.PartialClose(alice, pairBtcNusd, sdk.NewDec(2_500)),
			).
			Then(
				perpassert.PositionShouldBeEqual(alice, pairBtcNusd, perpassert.Position_PositionShouldBeEqualTo(types.Position{
					Pair:                            pairBtcNusd,
					TraderAddress:                   alice.String(),
					Margin:                          sdk.ZeroDec(),
					OpenNotional:                    sdk.MustNewDecFromStr("7499.999988937500138281"),
					Size_:                           sdk.MustNewDecFromStr("7500.000000000000000000"),
					LastUpdatedBlockNumber:          1,
					LatestCumulativePremiumFraction: sdk.MustNewDecFromStr("0.0002"),
				})),
				perpassert.PositionChangedEventShouldBeEqual(&types.PositionChangedEvent{
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

		tutilaction.TC("partial close short position with positive PnL").
			Given(
				perpaction.CreateCustomMarket(
					pairBtcNusd,
					perpaction.WithPricePeg(sdk.MustNewDecFromStr("0.10")),
					perpaction.WithLatestMarketCPF(sdk.MustNewDecFromStr("0.0002")),
				),
				tutilaction.SetBlockTime(startBlockTime),
				tutilaction.SetBlockNumber(1),
				tutilaction.FundAccount(alice, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(2)))),
				perpaction.InsertPosition(
					perpaction.WithPair(pairBtcNusd),
					perpaction.WithTrader(alice),
					perpaction.WithSize(sdk.NewDec(-10_000)),
					perpaction.WithMargin(sdk.NewDec(1_000)),
					perpaction.WithOpenNotional(sdk.NewDec(10_000)),
				),
			).
			When(
				perpaction.PartialClose(alice, pairBtcNusd, sdk.NewDec(7_500)),
			).
			Then(
				perpassert.PositionShouldBeEqual(alice, pairBtcNusd, perpassert.Position_PositionShouldBeEqualTo(types.Position{
					Pair:                            pairBtcNusd,
					TraderAddress:                   alice.String(),
					Margin:                          sdk.MustNewDecFromStr("7751.999992499999925000"),
					OpenNotional:                    sdk.MustNewDecFromStr("2500.000001875000032812"),
					Size_:                           sdk.MustNewDecFromStr("-2499.999999999999999995"),
					LastUpdatedBlockNumber:          1,
					LatestCumulativePremiumFraction: sdk.MustNewDecFromStr("0.0002"),
				})),
				perpassert.PositionChangedEventShouldBeEqual(&types.PositionChangedEvent{
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

		tutilaction.TC("partial close short position with negative PnL").
			Given(
				perpaction.CreateCustomMarket(
					pairBtcNusd,
					perpaction.WithPricePeg(sdk.MustNewDecFromStr("1.05")),
					perpaction.WithLatestMarketCPF(sdk.MustNewDecFromStr("0.0002")),
				),
				tutilaction.SetBlockTime(startBlockTime),
				tutilaction.SetBlockNumber(1),
				tutilaction.FundAccount(alice, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(16)))),
				perpaction.InsertPosition(
					perpaction.WithPair(pairBtcNusd),
					perpaction.WithTrader(alice),
					perpaction.WithSize(sdk.NewDec(-10_000)),
					perpaction.WithMargin(sdk.NewDec(1_000)),
					perpaction.WithOpenNotional(sdk.NewDec(10_000)),
				),
			).
			When(
				perpaction.PartialClose(alice, pairBtcNusd, sdk.NewDec(7_500)),
			).
			Then(
				perpassert.PositionShouldBeEqual(alice, pairBtcNusd, perpassert.Position_PositionShouldBeEqualTo(types.Position{
					Pair:                            pairBtcNusd,
					TraderAddress:                   alice.String(),
					Margin:                          sdk.MustNewDecFromStr("626.999921249999212500"),
					OpenNotional:                    sdk.MustNewDecFromStr("2500.000019687500344531"),
					Size_:                           sdk.MustNewDecFromStr("-2500.000000000000000000"),
					LastUpdatedBlockNumber:          1,
					LatestCumulativePremiumFraction: sdk.MustNewDecFromStr("0.0002"),
				})),
				perpassert.PositionChangedEventShouldBeEqual(&types.PositionChangedEvent{
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

		tutilaction.TC("partial close short position with no bad debt but below maintenance margin ratio").
			Given(
				perpaction.CreateCustomMarket(
					pairBtcNusd,
					perpaction.WithPricePeg(sdk.MustNewDecFromStr("1.09")),
					perpaction.WithLatestMarketCPF(sdk.MustNewDecFromStr("0.0002")),
				),
				tutilaction.SetBlockTime(startBlockTime),
				tutilaction.SetBlockNumber(1),
				tutilaction.FundAccount(alice, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(16)))),
				perpaction.InsertPosition(
					perpaction.WithPair(pairBtcNusd),
					perpaction.WithTrader(alice),
					perpaction.WithSize(sdk.NewDec(-10_000)),
					perpaction.WithMargin(sdk.NewDec(1_000)),
					perpaction.WithOpenNotional(sdk.NewDec(10_000)),
				),
			).
			When(
				perpaction.PartialClose(alice, pairBtcNusd, sdk.NewDec(7_500)),
			).
			Then(
				perpassert.PositionShouldBeEqual(alice, pairBtcNusd, perpassert.Position_PositionShouldBeEqualTo(types.Position{
					Pair:                            pairBtcNusd,
					TraderAddress:                   alice.String(),
					Margin:                          sdk.MustNewDecFromStr("326.999918249999182500"),
					OpenNotional:                    sdk.MustNewDecFromStr("2500.000020437500357656"),
					Size_:                           sdk.MustNewDecFromStr("-2500.000000000000000000"),
					LastUpdatedBlockNumber:          1,
					LatestCumulativePremiumFraction: sdk.MustNewDecFromStr("0.0002"),
				})),
				perpassert.PositionChangedEventShouldBeEqual(&types.PositionChangedEvent{
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

		tutilaction.TC("partial close short position with bad debt").
			Given(
				perpaction.CreateCustomMarket(
					pairBtcNusd,
					perpaction.WithPricePeg(sdk.MustNewDecFromStr("1.14")),
					perpaction.WithLatestMarketCPF(sdk.MustNewDecFromStr("0.0002")),
				),
				tutilaction.SetBlockTime(startBlockTime),
				tutilaction.SetBlockNumber(1),
				tutilaction.FundAccount(alice, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(18)))),
				tutilaction.FundModule(types.PerpEFModuleAccount, sdk.NewCoins(sdk.NewInt64Coin(denoms.NUSD, 48))),
				perpaction.InsertPosition(
					perpaction.WithPair(pairBtcNusd),
					perpaction.WithTrader(alice),
					perpaction.WithSize(sdk.NewDec(-10_000)),
					perpaction.WithMargin(sdk.NewDec(1_000)),
					perpaction.WithOpenNotional(sdk.NewDec(10_000)),
				),
			).
			When(
				perpaction.PartialClose(alice, pairBtcNusd, sdk.NewDec(7_500)),
			).
			Then(
				perpassert.PositionShouldBeEqual(alice, pairBtcNusd, perpassert.Position_PositionShouldBeEqualTo(types.Position{
					Pair:                            pairBtcNusd,
					TraderAddress:                   alice.String(),
					Margin:                          sdk.ZeroDec(),
					OpenNotional:                    sdk.MustNewDecFromStr("2500.000021375000374062"),
					Size_:                           sdk.MustNewDecFromStr("-2500.000000000000000000"),
					LastUpdatedBlockNumber:          1,
					LatestCumulativePremiumFraction: sdk.MustNewDecFromStr("0.0002"),
				})),
				perpassert.PositionChangedEventShouldBeEqual(&types.PositionChangedEvent{
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
		tutilaction.TC("test partial closes fail").
			Given(
				perpaction.CreateCustomMarket(
					pairBtcNusd,
					perpaction.WithPricePeg(sdk.OneDec()),
					perpaction.WithSqrtDepth(sdk.NewDec(10_000)),
				),
				tutilaction.SetBlockNumber(1),
				tutilaction.SetBlockTime(startBlockTime),

				tutilaction.FundAccount(alice, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(10_200)))),

				perpaction.PartialCloseFails(alice, pairBtcNusd, sdk.NewDec(5_000), collections.ErrNotFound),
				perpaction.MarketOrder(alice, pairBtcNusd, types.Direction_LONG, sdk.NewInt(9_000), sdk.OneDec(), sdk.ZeroDec()),
			).
			When(
				perpaction.PartialCloseFails(alice, asset.MustNewPair("luna:usdt"), sdk.NewDec(5_000), types.ErrPairNotFound),
			),
	}

	tutilaction.NewTestSuite(t).WithTestCases(tc...).Run()
}

func TestClosePosition(t *testing.T) {
	alice := testutil.AccAddress()
	pairBtcNusd := asset.Registry.Pair(denoms.BTC, denoms.NUSD)
	startBlockTime := time.Now()

	tc := tutilaction.TestCases{
		tutilaction.TC("close long position with positive PnL").
			Given(
				perpaction.CreateCustomMarket(pairBtcNusd,
					perpaction.WithPricePeg(sdk.NewDec(2)),
					perpaction.WithLatestMarketCPF(sdk.MustNewDecFromStr("0.0002")),
				),
				tutilaction.SetBlockTime(startBlockTime),
				tutilaction.SetBlockNumber(1),
				tutilaction.FundAccount(alice, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(40)))),
				tutilaction.FundModule(types.VaultModuleAccount, sdk.NewCoins(sdk.NewInt64Coin(denoms.NUSD, 10_998))),
				perpaction.InsertPosition(
					perpaction.WithTrader(alice),
					perpaction.WithPair(pairBtcNusd),
					perpaction.WithSize(sdk.NewDec(10_000)),
					perpaction.WithMargin(sdk.NewDec(1_000)),
					perpaction.WithOpenNotional(sdk.NewDec(10_000)),
				),
			).
			When(
				perpaction.ClosePosition(alice, pairBtcNusd),
			).
			Then(
				perpassert.PositionShouldNotExist(alice, pairBtcNusd),
				perpassert.PositionChangedEventShouldBeEqual(&types.PositionChangedEvent{
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

		tutilaction.TC("close long position with negative PnL").
			Given(
				perpaction.CreateCustomMarket(pairBtcNusd,
					perpaction.WithPricePeg(sdk.MustNewDecFromStr("0.99")),
					perpaction.WithLatestMarketCPF(sdk.MustNewDecFromStr("0.0002")),
				),
				tutilaction.SetBlockTime(startBlockTime),
				tutilaction.SetBlockNumber(1),
				tutilaction.FundAccount(alice, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(20)))),
				tutilaction.FundModule(types.VaultModuleAccount, sdk.NewCoins(sdk.NewInt64Coin(denoms.NUSD, 898))),
				perpaction.InsertPosition(
					perpaction.WithTrader(alice),
					perpaction.WithPair(pairBtcNusd),
					perpaction.WithSize(sdk.NewDec(10_000)),
					perpaction.WithMargin(sdk.NewDec(1_000)),
					perpaction.WithOpenNotional(sdk.NewDec(10_000)),
				),
			).
			When(
				perpaction.ClosePosition(alice, pairBtcNusd),
			).
			Then(
				perpassert.PositionShouldNotExist(alice, pairBtcNusd),
				perpassert.PositionChangedEventShouldBeEqual(&types.PositionChangedEvent{
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

		tutilaction.TC("close long position with bad debt").
			Given(
				perpaction.CreateCustomMarket(pairBtcNusd,
					perpaction.WithPricePeg(sdk.MustNewDecFromStr("0.89")),
					perpaction.WithLatestMarketCPF(sdk.MustNewDecFromStr("0.0002")),
				),
				tutilaction.SetBlockNumber(1),
				tutilaction.SetBlockTime(startBlockTime),
				perpaction.InsertPosition(
					perpaction.WithTrader(alice),
					perpaction.WithPair(pairBtcNusd),
					perpaction.WithSize(sdk.NewDec(10_000)),
					perpaction.WithMargin(sdk.NewDec(1_000)),
					perpaction.WithOpenNotional(sdk.NewDec(10_000)),
				),
				tutilaction.FundAccount(alice, sdk.NewCoins(sdk.NewInt64Coin(denoms.NUSD, 18))),
				tutilaction.FundModule(types.VaultModuleAccount, sdk.NewCoins(sdk.NewInt64Coin(denoms.NUSD, 1000))),
				tutilaction.FundModule(types.PerpEFModuleAccount, sdk.NewCoins(sdk.NewInt64Coin(denoms.NUSD, 102))),
			).
			When(
				tutilaction.MoveToNextBlock(),
				perpaction.ClosePosition(alice, pairBtcNusd),
			).
			Then(
				perpassert.PositionShouldNotExist(alice, pairBtcNusd),
				perpassert.PositionChangedEventShouldBeEqual(&types.PositionChangedEvent{
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
				tutilassert.ModuleBalanceEqual(types.VaultModuleAccount, denoms.NUSD, sdk.NewInt(1102)), // 1000 + 102 from perp ef
				tutilassert.ModuleBalanceEqual(types.PerpEFModuleAccount, denoms.NUSD, sdk.NewInt(9)),
			),

		tutilaction.TC("close short position with positive PnL").
			Given(
				perpaction.CreateCustomMarket(pairBtcNusd,
					perpaction.WithPricePeg(sdk.MustNewDecFromStr("0.10")),
					perpaction.WithLatestMarketCPF(sdk.MustNewDecFromStr("0.0002")),
				),
				tutilaction.SetBlockTime(startBlockTime),
				tutilaction.SetBlockNumber(1),
				tutilaction.FundAccount(alice, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(2)))),
				tutilaction.FundModule(types.VaultModuleAccount, sdk.NewCoins(sdk.NewInt64Coin(denoms.NUSD, 10_002))),
				perpaction.InsertPosition(
					perpaction.WithTrader(alice),
					perpaction.WithPair(pairBtcNusd),
					perpaction.WithSize(sdk.NewDec(-10_000)),
					perpaction.WithMargin(sdk.NewDec(1_000)),
					perpaction.WithOpenNotional(sdk.NewDec(10_000)),
				),
			).
			When(
				perpaction.ClosePosition(alice, pairBtcNusd),
			).
			Then(
				perpassert.PositionShouldNotExist(alice, pairBtcNusd),
				perpassert.PositionChangedEventShouldBeEqual(&types.PositionChangedEvent{
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

		tutilaction.TC("close short position with negative PnL").
			Given(
				perpaction.CreateCustomMarket(pairBtcNusd,
					perpaction.WithPricePeg(sdk.MustNewDecFromStr("1.01")),
					perpaction.WithLatestMarketCPF(sdk.MustNewDecFromStr("0.0002")),
				),
				tutilaction.SetBlockTime(startBlockTime),
				tutilaction.SetBlockNumber(1),
				tutilaction.FundAccount(alice, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(20)))),
				tutilaction.FundModule(types.VaultModuleAccount, sdk.NewCoins(sdk.NewInt64Coin(denoms.NUSD, 902))),
				perpaction.InsertPosition(
					perpaction.WithTrader(alice),
					perpaction.WithPair(pairBtcNusd),
					perpaction.WithSize(sdk.NewDec(-10_000)),
					perpaction.WithMargin(sdk.NewDec(1_000)),
					perpaction.WithOpenNotional(sdk.NewDec(10_000)),
				),
			).
			When(
				perpaction.ClosePosition(alice, pairBtcNusd),
			).
			Then(
				perpassert.PositionShouldNotExist(alice, pairBtcNusd),
				perpassert.PositionChangedEventShouldBeEqual(&types.PositionChangedEvent{
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

		tutilaction.TC("close short position with bad debt").
			Given(
				perpaction.CreateCustomMarket(pairBtcNusd,
					perpaction.WithPricePeg(sdk.MustNewDecFromStr("1.11")),
					perpaction.WithLatestMarketCPF(sdk.MustNewDecFromStr("0.0002")),
				),
				tutilaction.SetBlockTime(startBlockTime),
				tutilaction.SetBlockNumber(1),
				perpaction.InsertPosition(
					perpaction.WithTrader(alice),
					perpaction.WithPair(pairBtcNusd),
					perpaction.WithSize(sdk.NewDec(-10_000)),
					perpaction.WithMargin(sdk.NewDec(1_000)),
					perpaction.WithOpenNotional(sdk.NewDec(10_000)),
				),
				tutilaction.FundAccount(alice, sdk.NewCoins(sdk.NewInt64Coin(denoms.NUSD, 22))),
				tutilaction.FundModule(types.VaultModuleAccount, sdk.NewCoins(sdk.NewInt64Coin(denoms.NUSD, 1000))),
				tutilaction.FundModule(types.PerpEFModuleAccount, sdk.NewCoins(sdk.NewInt64Coin(denoms.NUSD, 98))),
			).
			When(
				tutilaction.MoveToNextBlock(),
				perpaction.ClosePosition(alice, pairBtcNusd),
			).
			Then(
				perpassert.PositionShouldNotExist(alice, pairBtcNusd),
				perpassert.PositionChangedEventShouldBeEqual(&types.PositionChangedEvent{
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
				tutilassert.ModuleBalanceEqual(types.VaultModuleAccount, denoms.NUSD, sdk.NewInt(1098)), // 1000 + 98 from perp ef
				tutilassert.ModuleBalanceEqual(types.PerpEFModuleAccount, denoms.NUSD, sdk.NewInt(11)),
			),
	}

	tutilaction.NewTestSuite(t).WithTestCases(tc...).Run()
}

func TestUpdateSwapInvariant(t *testing.T) {
	alice := testutil.AccAddress()
	bob := testutil.AccAddress()
	pairBtcNusd := asset.Registry.Pair(denoms.BTC, denoms.NUSD)
	startBlockTime := time.Now()

	startingSwapInvariant := sdk.NewDec(1_000_000_000_000).Mul(sdk.NewDec(1_000_000_000_000))

	tc := tutilaction.TestCases{
		tutilaction.TC("only long position - no change to swap invariant").
			Given(
				perpaction.CreateCustomMarket(pairBtcNusd),
				tutilaction.SetBlockTime(startBlockTime),
				tutilaction.SetBlockNumber(1),
				tutilaction.FundAccount(alice, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(10_200_000_000)))),
				tutilaction.FundModule(types.PerpEFModuleAccount, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(100000)))),
			).
			When(
				perpaction.MarketOrder(alice, pairBtcNusd, types.Direction_LONG, sdk.NewInt(10_000_000_000), sdk.OneDec(), sdk.ZeroDec()),
				perpaction.ClosePosition(alice, pairBtcNusd),
			).
			Then(
				perpassert.PositionShouldNotExist(alice, pairBtcNusd),
			),
		tutilaction.TC("only short position - no change to swap invariant").
			Given(
				perpaction.CreateCustomMarket(pairBtcNusd),
				tutilaction.SetBlockTime(startBlockTime),
				tutilaction.SetBlockNumber(1),
				tutilaction.FundAccount(alice, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(10_200_000_000)))),
				tutilaction.FundModule(types.PerpEFModuleAccount, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(100000)))),
			).
			When(
				perpaction.MarketOrder(alice, pairBtcNusd, types.Direction_SHORT, sdk.NewInt(10_000_000_000), sdk.OneDec(), sdk.NewDec(10_000_000_000_000)),
				perpaction.ClosePosition(alice, pairBtcNusd),
			).
			Then(
				perpassert.PositionShouldNotExist(alice, pairBtcNusd),
			),
		tutilaction.TC("only long position - increasing k").
			Given(
				perpaction.CreateCustomMarket(pairBtcNusd),
				tutilaction.SetBlockTime(startBlockTime),
				tutilaction.SetBlockNumber(1),
				tutilaction.FundAccount(alice, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(10_200_000_000)))),
				tutilaction.FundModule(types.PerpEFModuleAccount, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(100_000_000)))),
			).
			When(
				perpaction.MarketOrder(alice, pairBtcNusd, types.Direction_LONG, sdk.NewInt(10_000_000_000), sdk.OneDec(), sdk.ZeroDec()),
				perpaction.EditSwapInvariant(pairBtcNusd, startingSwapInvariant.MulInt64(100)),
				perpassert.AMMShouldBeEqual(
					pairBtcNusd,
					perpassert.AMM_SwapInvariantShouldBeEqual(sdk.MustNewDecFromStr("99999999999999999999999999.999999000000000000"))),
				perpaction.ClosePosition(alice, pairBtcNusd),
			).
			Then(
				perpassert.PositionShouldNotExist(alice, pairBtcNusd),
				perpassert.ModuleBalanceShouldBeEqualTo(types.VaultModuleAccount, sdk.NewCoins()),
			),
		tutilaction.TC("only short position - increasing k").
			Given(
				perpaction.CreateCustomMarket(pairBtcNusd),
				tutilaction.SetBlockTime(startBlockTime),
				tutilaction.SetBlockNumber(1),
				tutilaction.FundAccount(alice, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(10_200_000_000)))),
				tutilaction.FundModule(types.PerpEFModuleAccount, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(100_000_000)))),
			).
			When(
				perpaction.MarketOrder(alice, pairBtcNusd, types.Direction_SHORT, sdk.NewInt(10_000_000_000), sdk.OneDec(), sdk.ZeroDec()),
				perpaction.EditSwapInvariant(pairBtcNusd, startingSwapInvariant.MulInt64(100)),
				perpassert.AMMShouldBeEqual(
					pairBtcNusd,
					perpassert.AMM_SwapInvariantShouldBeEqual(sdk.MustNewDecFromStr("99999999999999999999999999.999999000000000000"))),
				perpaction.ClosePosition(alice, pairBtcNusd),
			).
			Then(
				perpassert.PositionShouldNotExist(alice, pairBtcNusd),
				perpassert.ModuleBalanceShouldBeEqualTo(types.VaultModuleAccount, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.OneInt()))),
			),

		tutilaction.TC("only long position - decreasing k").
			Given(
				perpaction.CreateCustomMarket(pairBtcNusd),
				tutilaction.SetBlockTime(startBlockTime),
				tutilaction.SetBlockNumber(1),
				tutilaction.FundAccount(alice, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(10_200_000_000)))),
				tutilaction.FundModule(types.PerpEFModuleAccount, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(100_000_000)))),
			).
			When(
				perpaction.MarketOrder(alice, pairBtcNusd, types.Direction_LONG, sdk.NewInt(10_000_000_000), sdk.OneDec(), sdk.ZeroDec()),
				perpaction.EditSwapInvariant(pairBtcNusd, startingSwapInvariant.Mul(sdk.MustNewDecFromStr("0.1"))),
				perpassert.AMMShouldBeEqual(
					pairBtcNusd,
					perpassert.AMM_SwapInvariantShouldBeEqual(sdk.MustNewDecFromStr("99999999999999999873578.871987715651277660"))),
				perpaction.ClosePosition(alice, pairBtcNusd),
			).
			Then(
				perpassert.PositionShouldNotExist(alice, pairBtcNusd),
				perpassert.ModuleBalanceShouldBeEqualTo(types.VaultModuleAccount, sdk.NewCoins()),
			),
		tutilaction.TC("only short position - decreasing k").
			Given(
				perpaction.CreateCustomMarket(pairBtcNusd),
				tutilaction.SetBlockTime(startBlockTime),
				tutilaction.SetBlockNumber(1),
				tutilaction.FundAccount(alice, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(10_200_000_000)))),
				tutilaction.FundModule(types.PerpEFModuleAccount, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(100_000_000)))),
			).
			When(
				perpaction.MarketOrder(alice, pairBtcNusd, types.Direction_SHORT, sdk.NewInt(10_000_000_000), sdk.OneDec(), sdk.ZeroDec()),
				perpaction.EditSwapInvariant(pairBtcNusd, startingSwapInvariant.Mul(sdk.MustNewDecFromStr("0.1"))),
				perpassert.AMMShouldBeEqual(
					pairBtcNusd,
					perpassert.AMM_SwapInvariantShouldBeEqual(sdk.MustNewDecFromStr("99999999999999999873578.871987801032774485"))),
				perpaction.ClosePosition(alice, pairBtcNusd),
			).
			Then(
				perpassert.PositionShouldNotExist(alice, pairBtcNusd),
				perpassert.ModuleBalanceShouldBeEqualTo(types.VaultModuleAccount, sdk.NewCoins()),
			),

		tutilaction.TC("long and short position - increasing k").
			Given(
				perpaction.CreateCustomMarket(pairBtcNusd),
				tutilaction.SetBlockTime(startBlockTime),
				tutilaction.SetBlockNumber(1),
				tutilaction.FundAccount(alice, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(10_200_000_000)))),
				tutilaction.FundAccount(bob, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(10_200_000_000)))),
			).
			When(
				perpaction.MarketOrder(alice, pairBtcNusd, types.Direction_LONG, sdk.NewInt(10_000_000_000), sdk.OneDec(), sdk.ZeroDec()),
				perpaction.MarketOrder(bob, pairBtcNusd, types.Direction_SHORT, sdk.NewInt(10_000_000_000), sdk.OneDec(), sdk.NewDec(10_000_000_000_000)),

				perpaction.EditSwapInvariant(pairBtcNusd, startingSwapInvariant.MulInt64(100)),
				perpassert.AMMShouldBeEqual(
					pairBtcNusd,
					perpassert.AMM_BiasShouldBeEqual(sdk.ZeroDec()),
					perpassert.AMM_SwapInvariantShouldBeEqual(sdk.MustNewDecFromStr("100000000000000000000000000.000000000000000000"))),

				perpassert.ModuleBalanceShouldBeEqualTo(types.VaultModuleAccount, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(20_000_000_000)))),
				perpassert.ModuleBalanceShouldBeEqualTo(types.FeePoolModuleAccount, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(20_000_000)))), // Fees are 10_000_000_000 * 0.0010 * 2

				perpaction.ClosePosition(alice, pairBtcNusd),
				perpaction.ClosePosition(bob, pairBtcNusd),
			).
			Then(
				perpassert.PositionShouldNotExist(alice, pairBtcNusd),
				perpassert.PositionShouldNotExist(bob, pairBtcNusd),

				perpassert.ModuleBalanceShouldBeEqualTo(types.VaultModuleAccount, sdk.NewCoins()),
				perpassert.ModuleBalanceShouldBeEqualTo(types.PerpEFModuleAccount, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(39_782_394)))),
			),
		tutilaction.TC("long and short position - reducing k").
			Given(
				perpaction.CreateCustomMarket(pairBtcNusd),
				tutilaction.SetBlockTime(startBlockTime),
				tutilaction.SetBlockNumber(1),
				tutilaction.FundAccount(alice, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(10_200_000_000)))),
				tutilaction.FundAccount(bob, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(10_200_000_000)))),
			).
			When(
				perpaction.MarketOrder(alice, pairBtcNusd, types.Direction_LONG, sdk.NewInt(10_000_000_000), sdk.OneDec(), sdk.ZeroDec()),
				perpaction.MarketOrder(bob, pairBtcNusd, types.Direction_SHORT, sdk.NewInt(10_000_000_000), sdk.OneDec(), sdk.NewDec(10_000_000_000_000)),

				perpaction.EditSwapInvariant(pairBtcNusd, startingSwapInvariant.Mul(sdk.MustNewDecFromStr("0.1"))),
				perpassert.AMMShouldBeEqual(
					pairBtcNusd,
					perpassert.AMM_BiasShouldBeEqual(sdk.ZeroDec()),
					perpassert.AMM_SwapInvariantShouldBeEqual(sdk.MustNewDecFromStr("99999999999999999873578.871987712489000000"))),

				perpassert.ModuleBalanceShouldBeEqualTo(types.VaultModuleAccount, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(20_000_000_000)))),
				perpassert.ModuleBalanceShouldBeEqualTo(types.FeePoolModuleAccount, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(20_000_000)))), // Fees are 10_000_000_000 * 0.0010 * 2

				perpaction.ClosePosition(alice, pairBtcNusd),
				perpaction.ClosePosition(bob, pairBtcNusd),
			).
			Then(
				perpassert.PositionShouldNotExist(alice, pairBtcNusd),
				perpassert.PositionShouldNotExist(bob, pairBtcNusd),

				perpassert.ModuleBalanceShouldBeEqualTo(types.VaultModuleAccount, sdk.NewCoins()),
				perpassert.ModuleBalanceShouldBeEqualTo(types.PerpEFModuleAccount, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(39_200_810)))),
			),
	}

	tutilaction.NewTestSuite(t).WithTestCases(tc...).Run()
}
