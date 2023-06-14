package keeper_test

import (
	"testing"
	"time"

	sdkmath "cosmossdk.io/math"

	"github.com/NibiruChain/nibiru/x/common/testutil/assertion"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/collections"

	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/common/denoms"
	"github.com/NibiruChain/nibiru/x/common/testutil"
	. "github.com/NibiruChain/nibiru/x/common/testutil/action"
	"github.com/NibiruChain/nibiru/x/common/testutil/mock"
	"github.com/NibiruChain/nibiru/x/common/testutil/testapp"
	. "github.com/NibiruChain/nibiru/x/oracle/integration/action"
	. "github.com/NibiruChain/nibiru/x/perp/v2/integration/action"
	. "github.com/NibiruChain/nibiru/x/perp/v2/integration/assertion"

	types "github.com/NibiruChain/nibiru/x/perp/v2/types"
)

func TestOpenPosition(t *testing.T) {
	alice := testutil.AccAddress()
	pairBtcUsdc := asset.Registry.Pair(denoms.BTC, denoms.NUSD)
	startBlockTime := time.Now()

	tc := TestCases{
		TC("new long position").
			Given(
				CreateCustomMarket(pairBtcUsdc),
				SetBlockTime(startBlockTime),
				SetBlockNumber(1),
				SetOraclePrice(pairBtcUsdc, sdk.NewDec(1)),
				FundAccount(alice, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(1020)))),
			).
			When(
				OpenPosition(alice, pairBtcUsdc, types.Direction_LONG, sdk.NewInt(1000), sdk.NewDec(10), sdk.ZeroDec(),
					OpenPositionResp_PositionShouldBeEqual(
						types.Position{
							Pair:                            pairBtcUsdc,
							TraderAddress:                   alice.String(),
							Margin:                          sdk.NewDec(1000),
							OpenNotional:                    sdk.NewDec(10_000),
							Size_:                           sdk.MustNewDecFromStr("9999.999900000001"),
							LastUpdatedBlockNumber:          1,
							LatestCumulativePremiumFraction: sdk.ZeroDec(),
						}),
					OpenPositionResp_ExchangeNotionalValueShouldBeEqual(sdk.NewDec(10_000)), // margin * leverage
					OpenPositionResp_ExchangedPositionSizeShouldBeEqual(sdk.MustNewDecFromStr("9999.999900000001")),
					OpenPositionResp_BadDebtShouldBeEqual(sdk.ZeroDec()),
					OpenPositionResp_FundingPaymentShouldBeEqual(sdk.ZeroDec()),
					OpenPositionResp_RealizedPnlShouldBeEqual(sdk.ZeroDec()),
					OpenPositionResp_UnrealizedPnlAfterShouldBeEqual(sdk.ZeroDec()),
					OpenPositionResp_MarginToVaultShouldBeEqual(sdk.NewDec(1000)),
					OpenPositionResp_PositionNotionalShouldBeEqual(sdk.NewDec(10_000)),
				),
			).
			Then(
				PositionShouldBeEqual(alice, pairBtcUsdc, Position_PositionShouldBeEqualTo(types.Position{
					Pair:                            pairBtcUsdc,
					TraderAddress:                   alice.String(),
					Margin:                          sdk.NewDec(1000),
					OpenNotional:                    sdk.NewDec(10_000),
					Size_:                           sdk.MustNewDecFromStr("9999.999900000001"),
					LastUpdatedBlockNumber:          1,
					LatestCumulativePremiumFraction: sdk.ZeroDec(),
				})),
				PositionChangedEventShouldBeEqual(&types.PositionChangedEvent{
					FinalPosition: types.Position{
						Pair:                            pairBtcUsdc,
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
				}),
			),

		TC("existing long position, go more long").
			Given(
				SetBlockNumber(1),
				SetBlockTime(startBlockTime),
				CreateCustomMarket(pairBtcUsdc),
				SetOraclePrice(pairBtcUsdc, sdk.NewDec(1)),
				FundAccount(alice, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(2040)))),
				OpenPosition(alice, pairBtcUsdc, types.Direction_LONG, sdk.NewInt(1000), sdk.NewDec(10), sdk.ZeroDec()),
			).
			When(
				MoveToNextBlock(),
				OpenPosition(alice, pairBtcUsdc, types.Direction_LONG, sdk.NewInt(1000), sdk.NewDec(10), sdk.ZeroDec(),
					OpenPositionResp_PositionShouldBeEqual(
						types.Position{
							Pair:                            pairBtcUsdc,
							TraderAddress:                   alice.String(),
							Margin:                          sdk.NewDec(2000),
							OpenNotional:                    sdk.NewDec(20_000),
							Size_:                           sdk.MustNewDecFromStr("19999.999600000008000000"),
							LastUpdatedBlockNumber:          2,
							LatestCumulativePremiumFraction: sdk.ZeroDec(),
						}),
					OpenPositionResp_ExchangeNotionalValueShouldBeEqual(sdk.NewDec(10_000)), // margin * leverage
					OpenPositionResp_ExchangedPositionSizeShouldBeEqual(sdk.MustNewDecFromStr("9999.999700000007000000")),
					OpenPositionResp_BadDebtShouldBeEqual(sdk.ZeroDec()),
					OpenPositionResp_FundingPaymentShouldBeEqual(sdk.ZeroDec()),
					OpenPositionResp_RealizedPnlShouldBeEqual(sdk.ZeroDec()),
					OpenPositionResp_UnrealizedPnlAfterShouldBeEqual(sdk.ZeroDec()),
					OpenPositionResp_MarginToVaultShouldBeEqual(sdk.NewDec(1000)),
					OpenPositionResp_PositionNotionalShouldBeEqual(sdk.NewDec(20_000)),
				),
			).
			Then(
				PositionChangedEventShouldBeEqual(&types.PositionChangedEvent{
					FinalPosition: types.Position{
						Pair:                            pairBtcUsdc,
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
				}),
			),

		TC("existing long position, decrease a bit").
			Given(
				CreateCustomMarket(pairBtcUsdc),
				SetBlockNumber(1),
				SetBlockTime(startBlockTime),
				SetOraclePrice(pairBtcUsdc, sdk.NewDec(1)),
				FundAccount(alice, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(1030)))),
				OpenPosition(alice, pairBtcUsdc, types.Direction_LONG, sdk.NewInt(1000), sdk.NewDec(10), sdk.ZeroDec()),
			).
			When(
				MoveToNextBlock(),
				OpenPosition(alice, pairBtcUsdc, types.Direction_SHORT, sdk.NewInt(500), sdk.NewDec(10), sdk.ZeroDec(),
					OpenPositionResp_PositionShouldBeEqual(
						types.Position{
							Pair:                            pairBtcUsdc,
							TraderAddress:                   alice.String(),
							Margin:                          sdk.NewDec(1000),
							OpenNotional:                    sdk.NewDec(5000),
							Size_:                           sdk.MustNewDecFromStr("4999.999975000000125000"),
							LastUpdatedBlockNumber:          2,
							LatestCumulativePremiumFraction: sdk.ZeroDec(),
						},
					),
					OpenPositionResp_ExchangeNotionalValueShouldBeEqual(sdk.NewDec(5000)), // margin * leverage
					OpenPositionResp_ExchangedPositionSizeShouldBeEqual(sdk.MustNewDecFromStr("-4999.999925000000875000")),
					OpenPositionResp_BadDebtShouldBeEqual(sdk.ZeroDec()),
					OpenPositionResp_FundingPaymentShouldBeEqual(sdk.ZeroDec()),
					OpenPositionResp_RealizedPnlShouldBeEqual(sdk.ZeroDec()),
					OpenPositionResp_UnrealizedPnlAfterShouldBeEqual(sdk.ZeroDec()),
					OpenPositionResp_MarginToVaultShouldBeEqual(sdk.ZeroDec()),
					OpenPositionResp_PositionNotionalShouldBeEqual(sdk.NewDec(5000)),
				),
			).
			Then(
				PositionChangedEventShouldBeEqual(&types.PositionChangedEvent{
					FinalPosition: types.Position{
						Pair:                            pairBtcUsdc,
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
				}),
			),

		TC("existing long position, decrease a lot").
			Given(
				CreateCustomMarket(pairBtcUsdc),
				SetBlockNumber(1),
				SetBlockTime(startBlockTime),
				SetOraclePrice(pairBtcUsdc, sdk.NewDec(1)),
				FundAccount(alice, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(4080)))),
				OpenPosition(alice, pairBtcUsdc, types.Direction_LONG, sdk.NewInt(1000), sdk.NewDec(10), sdk.ZeroDec()),
			).
			When(
				MoveToNextBlock(),
				OpenPosition(alice, pairBtcUsdc, types.Direction_SHORT, sdk.NewInt(3000), sdk.NewDec(10), sdk.ZeroDec(),
					OpenPositionResp_PositionShouldBeEqual(
						types.Position{
							Pair:                            pairBtcUsdc,
							TraderAddress:                   alice.String(),
							Margin:                          sdk.NewDec(2000),
							OpenNotional:                    sdk.NewDec(20000),
							Size_:                           sdk.MustNewDecFromStr("-20000.000400000008000000"),
							LastUpdatedBlockNumber:          2,
							LatestCumulativePremiumFraction: sdk.ZeroDec(),
						},
					),
					OpenPositionResp_ExchangeNotionalValueShouldBeEqual(sdk.NewDec(30000)), // margin * leverage
					OpenPositionResp_ExchangedPositionSizeShouldBeEqual(sdk.MustNewDecFromStr("-30000.000300000009000000")),
					OpenPositionResp_BadDebtShouldBeEqual(sdk.ZeroDec()),
					OpenPositionResp_FundingPaymentShouldBeEqual(sdk.ZeroDec()),
					OpenPositionResp_RealizedPnlShouldBeEqual(sdk.ZeroDec()),
					OpenPositionResp_UnrealizedPnlAfterShouldBeEqual(sdk.ZeroDec()),
					OpenPositionResp_MarginToVaultShouldBeEqual(sdk.NewDec(1000)),
					OpenPositionResp_PositionNotionalShouldBeEqual(sdk.NewDec(20000)),
				),
			).
			Then(
				PositionChangedEventShouldBeEqual(&types.PositionChangedEvent{
					FinalPosition: types.Position{
						Pair:                            pairBtcUsdc,
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
				}),
			),

		TC("new short position").
			Given(
				CreateCustomMarket(pairBtcUsdc),
				SetBlockTime(startBlockTime),
				SetBlockNumber(1),
				SetOraclePrice(pairBtcUsdc, sdk.NewDec(1)),
				FundAccount(alice, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(1020)))),
			).
			When(
				OpenPosition(alice, pairBtcUsdc, types.Direction_SHORT, sdk.NewInt(1000), sdk.NewDec(10), sdk.ZeroDec(),
					OpenPositionResp_PositionShouldBeEqual(
						types.Position{
							Pair:                            pairBtcUsdc,
							TraderAddress:                   alice.String(),
							Margin:                          sdk.NewDec(1000),
							OpenNotional:                    sdk.NewDec(10_000),
							Size_:                           sdk.MustNewDecFromStr("-10000.000100000001000000"),
							LastUpdatedBlockNumber:          1,
							LatestCumulativePremiumFraction: sdk.ZeroDec(),
						}),
					OpenPositionResp_ExchangeNotionalValueShouldBeEqual(sdk.NewDec(10_000)), // margin * leverage
					OpenPositionResp_ExchangedPositionSizeShouldBeEqual(sdk.MustNewDecFromStr("-10000.000100000001000000")),
					OpenPositionResp_BadDebtShouldBeEqual(sdk.ZeroDec()),
					OpenPositionResp_FundingPaymentShouldBeEqual(sdk.ZeroDec()),
					OpenPositionResp_RealizedPnlShouldBeEqual(sdk.ZeroDec()),
					OpenPositionResp_UnrealizedPnlAfterShouldBeEqual(sdk.ZeroDec()),
					OpenPositionResp_MarginToVaultShouldBeEqual(sdk.NewDec(1000)),
					OpenPositionResp_PositionNotionalShouldBeEqual(sdk.NewDec(10_000)),
				),
			).
			Then(
				PositionShouldBeEqual(alice, pairBtcUsdc,
					Position_PositionShouldBeEqualTo(types.Position{
						Pair:                            pairBtcUsdc,
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
						Pair:                            pairBtcUsdc,
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
				}),
			),

		TC("existing short position, go more short").
			Given(
				CreateCustomMarket(pairBtcUsdc),
				SetBlockNumber(1),
				SetBlockTime(startBlockTime),
				SetOraclePrice(pairBtcUsdc, sdk.NewDec(1)),
				FundAccount(alice, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(2040)))),
				OpenPosition(alice, pairBtcUsdc, types.Direction_SHORT, sdk.NewInt(1000), sdk.NewDec(10), sdk.ZeroDec()),
			).
			When(
				MoveToNextBlock(),
				OpenPosition(alice, pairBtcUsdc, types.Direction_SHORT, sdk.NewInt(1000), sdk.NewDec(10), sdk.ZeroDec(),
					OpenPositionResp_PositionShouldBeEqual(
						types.Position{
							Pair:                            pairBtcUsdc,
							TraderAddress:                   alice.String(),
							Margin:                          sdk.NewDec(2000),
							OpenNotional:                    sdk.NewDec(20_000),
							Size_:                           sdk.MustNewDecFromStr("-20000.000400000008000000"),
							LastUpdatedBlockNumber:          2,
							LatestCumulativePremiumFraction: sdk.ZeroDec(),
						},
					),
					OpenPositionResp_ExchangeNotionalValueShouldBeEqual(sdk.NewDec(10_000)), // margin * leverage
					OpenPositionResp_ExchangedPositionSizeShouldBeEqual(sdk.MustNewDecFromStr("-10000.000300000007000000")),
					OpenPositionResp_BadDebtShouldBeEqual(sdk.ZeroDec()),
					OpenPositionResp_FundingPaymentShouldBeEqual(sdk.ZeroDec()),
					OpenPositionResp_RealizedPnlShouldBeEqual(sdk.ZeroDec()),
					OpenPositionResp_UnrealizedPnlAfterShouldBeEqual(sdk.ZeroDec()),
					OpenPositionResp_MarginToVaultShouldBeEqual(sdk.NewDec(1000)),
					OpenPositionResp_PositionNotionalShouldBeEqual(sdk.NewDec(20_000)),
				),
			).
			Then(
				PositionChangedEventShouldBeEqual(&types.PositionChangedEvent{
					FinalPosition: types.Position{
						Pair:                            pairBtcUsdc,
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
				}),
			),

		TC("existing short position, decrease a bit").
			Given(
				CreateCustomMarket(pairBtcUsdc),
				SetBlockNumber(1),
				SetBlockTime(startBlockTime),
				SetOraclePrice(pairBtcUsdc, sdk.NewDec(1)),
				FundAccount(alice, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(1030)))),
				OpenPosition(alice, pairBtcUsdc, types.Direction_SHORT, sdk.NewInt(1000), sdk.NewDec(10), sdk.ZeroDec()),
			).
			When(
				MoveToNextBlock(),
				OpenPosition(alice, pairBtcUsdc, types.Direction_LONG, sdk.NewInt(500), sdk.NewDec(10), sdk.ZeroDec(),
					OpenPositionResp_PositionShouldBeEqual(
						types.Position{
							Pair:                            pairBtcUsdc,
							TraderAddress:                   alice.String(),
							Margin:                          sdk.NewDec(1000),
							OpenNotional:                    sdk.NewDec(5000),
							Size_:                           sdk.MustNewDecFromStr("-5000.000025000000125000"),
							LastUpdatedBlockNumber:          2,
							LatestCumulativePremiumFraction: sdk.ZeroDec(),
						},
					),
					OpenPositionResp_ExchangeNotionalValueShouldBeEqual(sdk.NewDec(5000)), // margin * leverage
					OpenPositionResp_ExchangedPositionSizeShouldBeEqual(sdk.MustNewDecFromStr("5000.000075000000875000")),
					OpenPositionResp_BadDebtShouldBeEqual(sdk.ZeroDec()),
					OpenPositionResp_FundingPaymentShouldBeEqual(sdk.ZeroDec()),
					OpenPositionResp_RealizedPnlShouldBeEqual(sdk.ZeroDec()),
					OpenPositionResp_UnrealizedPnlAfterShouldBeEqual(sdk.ZeroDec()),
					OpenPositionResp_MarginToVaultShouldBeEqual(sdk.ZeroDec()),
					OpenPositionResp_PositionNotionalShouldBeEqual(sdk.NewDec(5000)),
				),
			).
			Then(
				PositionChangedEventShouldBeEqual(&types.PositionChangedEvent{
					FinalPosition: types.Position{
						Pair:                            pairBtcUsdc,
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
				}),
			),

		TC("existing short position, decrease a lot").
			Given(
				CreateCustomMarket(pairBtcUsdc),
				SetBlockNumber(1),
				SetBlockTime(startBlockTime),
				SetOraclePrice(pairBtcUsdc, sdk.NewDec(1)),
				FundAccount(alice, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(4080)))),
				OpenPosition(alice, pairBtcUsdc, types.Direction_SHORT, sdk.NewInt(1000), sdk.NewDec(10), sdk.ZeroDec()),
			).
			When(
				MoveToNextBlock(),
				OpenPosition(alice, pairBtcUsdc, types.Direction_LONG, sdk.NewInt(3000), sdk.NewDec(10), sdk.ZeroDec(),
					OpenPositionResp_PositionShouldBeEqual(
						types.Position{
							Pair:                            pairBtcUsdc,
							TraderAddress:                   alice.String(),
							Margin:                          sdk.NewDec(2000),
							OpenNotional:                    sdk.NewDec(20000),
							Size_:                           sdk.MustNewDecFromStr("19999.999600000008000000"),
							LastUpdatedBlockNumber:          2,
							LatestCumulativePremiumFraction: sdk.ZeroDec(),
						},
					),
					OpenPositionResp_ExchangeNotionalValueShouldBeEqual(sdk.NewDec(30000)), // margin * leverage
					OpenPositionResp_ExchangedPositionSizeShouldBeEqual(sdk.MustNewDecFromStr("29999.999700000009000000")),
					OpenPositionResp_BadDebtShouldBeEqual(sdk.ZeroDec()),
					OpenPositionResp_FundingPaymentShouldBeEqual(sdk.ZeroDec()),
					OpenPositionResp_RealizedPnlShouldBeEqual(sdk.ZeroDec()),
					OpenPositionResp_UnrealizedPnlAfterShouldBeEqual(sdk.ZeroDec()),
					OpenPositionResp_MarginToVaultShouldBeEqual(sdk.NewDec(1000)),
					OpenPositionResp_PositionNotionalShouldBeEqual(sdk.NewDec(20000)),
				),
			).
			Then(
				PositionChangedEventShouldBeEqual(&types.PositionChangedEvent{
					FinalPosition: types.Position{
						Pair:                            pairBtcUsdc,
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
				}),
			),

		TC("user has insufficient funds").
			Given(
				CreateCustomMarket(pairBtcUsdc),
				SetBlockTime(startBlockTime),
				SetBlockNumber(1),
				SetOraclePrice(pairBtcUsdc, sdk.NewDec(1)),
				FundAccount(alice, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(99)))),
			).
			When(
				OpenPositionFails(
					alice, pairBtcUsdc, types.Direction_LONG, sdk.NewInt(100), sdk.OneDec(), sdk.ZeroDec(),
					sdkerrors.ErrInsufficientFunds),
			).
			Then(
				PositionShouldNotExist(alice, pairBtcUsdc),
			),

		TC("new long position, can close position after market is not enabled").
			Given(
				CreateCustomMarket(pairBtcUsdc),
				SetBlockTime(startBlockTime),
				SetBlockNumber(1),
				SetOraclePrice(pairBtcUsdc, sdk.NewDec(1)),
				FundAccount(alice, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(47_714_285_715)))),
				OpenPosition(alice, pairBtcUsdc, types.Direction_SHORT, sdk.NewInt(47_619_047_619), sdk.OneDec(), sdk.ZeroDec()),
				SetMarketEnabled(pairBtcUsdc, false),
			).
			When(
				ClosePosition(alice, pairBtcUsdc),
			).
			Then(
				PositionShouldNotExist(alice, pairBtcUsdc),
			),

		TC("new long position, can not open new position after market is not enabled").
			Given(
				CreateCustomMarket(pairBtcUsdc),
				SetBlockTime(startBlockTime),
				SetBlockNumber(1),
				SetOraclePrice(pairBtcUsdc, sdk.NewDec(1)),
				FundAccount(alice, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(47_714_285_715)))),
				SetMarketEnabled(pairBtcUsdc, false),
			).
			When(
				OpenPositionFails(alice, pairBtcUsdc, types.Direction_SHORT, sdk.NewInt(47_619_047_619), sdk.OneDec(), sdk.ZeroDec(),
					types.ErrMarketNotEnabled),
			).
			Then(
				PositionShouldNotExist(alice, pairBtcUsdc),
			),

		TC("existing long position, can not open new one but can close").
			Given(
				CreateCustomMarket(pairBtcUsdc),
				SetBlockTime(startBlockTime),
				SetBlockNumber(1),
				SetOraclePrice(pairBtcUsdc, sdk.NewDec(1)),
				FundAccount(alice, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(47_714_285_715)))),
				OpenPosition(alice, pairBtcUsdc, types.Direction_SHORT, sdk.NewInt(50_000), sdk.OneDec(), sdk.ZeroDec()),
				SetMarketEnabled(pairBtcUsdc, false),
			).
			When(
				OpenPositionFails(alice, pairBtcUsdc, types.Direction_SHORT, sdk.NewInt(47_619_047_619), sdk.OneDec(), sdk.ZeroDec(),
					types.ErrMarketNotEnabled),
				ClosePosition(alice, pairBtcUsdc),
			).
			Then(
				PositionShouldNotExist(alice, pairBtcUsdc),
			),

		TC("market doesn't exist").
			Given(
				SetBlockTime(startBlockTime),
				SetBlockNumber(1),
				SetOraclePrice(pairBtcUsdc, sdk.NewDec(1)),
				FundAccount(alice, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(47_714_285_715)))),
			).
			When(
				OpenPositionFails(alice, pairBtcUsdc, types.Direction_SHORT, sdk.NewInt(47_619_047_619), sdk.OneDec(), sdk.ZeroDec(),
					types.ErrPairNotFound),
			).
			Then(
				PositionShouldNotExist(alice, pairBtcUsdc),
			),

		TC("zero quote asset amount").
			Given(
				SetBlockTime(startBlockTime),
				SetBlockNumber(1),
				CreateCustomMarket(pairBtcUsdc),
				SetOraclePrice(pairBtcUsdc, sdk.NewDec(1)),
				FundAccount(alice, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(47_714_285_715)))),
			).
			When(
				OpenPositionFails(alice, pairBtcUsdc, types.Direction_LONG, sdk.ZeroInt(), sdk.OneDec(), sdk.ZeroDec(),
					types.ErrInputQuoteAmtNegative),
			).
			Then(
				PositionShouldNotExist(alice, pairBtcUsdc),
			),

		TC("zero leverage").
			Given(
				SetBlockTime(startBlockTime),
				SetBlockNumber(1),
				CreateCustomMarket(pairBtcUsdc),
				SetOraclePrice(pairBtcUsdc, sdk.NewDec(1)),
				FundAccount(alice, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(1e6)))),
			).
			When(
				OpenPositionFails(alice, pairBtcUsdc, types.Direction_LONG, sdk.OneInt(), sdk.ZeroDec(), sdk.ZeroDec(),
					types.ErrUserLeverageNegative),
			).
			Then(
				PositionShouldNotExist(alice, pairBtcUsdc),
			),

		TC("user leverage greater than market max leverage").
			Given(
				SetBlockTime(startBlockTime),
				SetBlockNumber(1),
				CreateCustomMarket(pairBtcUsdc),
				SetOraclePrice(pairBtcUsdc, sdk.NewDec(1)),
				FundAccount(alice, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(1e6)))),
			).
			When(
				OpenPositionFails(alice, pairBtcUsdc, types.Direction_LONG, sdk.OneInt(), sdk.NewDec(11), sdk.ZeroDec(),
					types.ErrLeverageIsTooHigh),
			).
			Then(
				PositionShouldNotExist(alice, pairBtcUsdc),
			),
	}

	NewTestSuite(t).WithTestCases(tc...).Run()
}

func TestOpenPositionError(t *testing.T) {
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
			margin:      sdk.NewInt(1),
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
			margin:          sdk.NewInt(0),
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
			leverage:        sdk.NewDec(0),
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
			baseLimit:       sdk.NewDec(0),
			expectedErr:     types.ErrLeverageIsTooHigh,
		},
	}

	for _, testCase := range testCases {
		tc := testCase
		t.Run(tc.name, func(t *testing.T) {
			app, ctx := testapp.NewNibiruTestAppAndContext(true)
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
			resp, err := app.PerpKeeperV2.OpenPosition(ctx, asset.Registry.Pair(denoms.BTC, denoms.NUSD), tc.side, traderAddr, tc.margin, tc.leverage, tc.baseLimit)
			require.ErrorContains(t, err, tc.expectedErr.Error())
			require.Nil(t, resp)
		})
	}
}

func TestClosePosition(t *testing.T) {
	tests := []struct {
		name string

		initialPosition    types.Position
		newPriceMultiplier sdk.Dec
		newLatestCPF       sdk.Dec

		expectedFundingPayment         sdk.Dec
		expectedBadDebt                sdk.Dec
		expectedRealizedPnl            sdk.Dec
		expectedMarginToVault          sdk.Dec
		expectedExchangedNotionalValue sdk.Dec
	}{
		{
			name: "long position, positive PnL",
			// user bought in at 100 BTC for 10 NUSD at 10x leverage (1 BTC = 1 NUSD)
			// notional value is 100 NUSD
			// BTC doubles in value, now its price is 1 BTC = 2 NUSD
			// user has position notional value of 200 NUSD and unrealized PnL of +100 NUSD
			// user closes position
			// user ends up with realized PnL of +100 NUSD, unrealized PnL after of 0 NUSD,
			//   position notional value of 0 NUSD
			initialPosition: types.Position{
				TraderAddress:                   testutil.AccAddress().String(),
				Pair:                            asset.Registry.Pair(denoms.BTC, denoms.NUSD),
				Size_:                           sdk.NewDec(100), // 100 BTC
				Margin:                          sdk.NewDec(10),  // 10 NUSD
				OpenNotional:                    sdk.NewDec(100), // 100 NUSD
				LatestCumulativePremiumFraction: sdk.ZeroDec(),
				LastUpdatedBlockNumber:          0,
			},
			newPriceMultiplier: sdk.NewDec(2),
			newLatestCPF:       sdk.MustNewDecFromStr("0.02"),

			expectedExchangedNotionalValue: sdk.MustNewDecFromStr("199.999999980000000002"),
			expectedBadDebt:                sdk.ZeroDec(),
			expectedFundingPayment:         sdk.NewDec(2),
			expectedRealizedPnl:            sdk.MustNewDecFromStr("99.999999980000000002"),
			expectedMarginToVault:          sdk.MustNewDecFromStr("-107.999999980000000002"),
		},
		{
			name: "close long position, negative PnL",
			// user bought in at 100 BTC for 10 NUSD at 10x leverage (1 BTC = 1 NUSD)
			//   position and open notional value is 100 NUSD
			// BTC drops in value, now its price is 1 BTC = 0.95 NUSD
			// user has position notional value of 195 NUSD and unrealized PnL of -5 NUSD
			// user closes position
			// user ends up with realized PnL of -5 NUSD, unrealized PnL of 0 NUSD,
			//   position notional value of 0 NUSD
			initialPosition: types.Position{
				TraderAddress:                   testutil.AccAddress().String(),
				Pair:                            asset.Registry.Pair(denoms.BTC, denoms.NUSD),
				Size_:                           sdk.NewDec(100),
				Margin:                          sdk.NewDec(10),
				OpenNotional:                    sdk.NewDec(100),
				LatestCumulativePremiumFraction: sdk.ZeroDec(),
				LastUpdatedBlockNumber:          0,
			},
			newPriceMultiplier: sdk.MustNewDecFromStr("0.95"),
			newLatestCPF:       sdk.MustNewDecFromStr("0.02"),

			expectedBadDebt:                sdk.ZeroDec(),
			expectedFundingPayment:         sdk.NewDec(2),
			expectedRealizedPnl:            sdk.MustNewDecFromStr("-5.000000009499999999"),
			expectedMarginToVault:          sdk.MustNewDecFromStr("-2.999999990500000001"), // 10(old) + (-5)(realized PnL) - (2)(funding payment)
			expectedExchangedNotionalValue: sdk.MustNewDecFromStr("94.999999990500000001"),
		},

		/*==========================SHORT POSITIONS===========================*/
		{
			name: "close short position, positive PnL",
			// user bought in at 100 BTC for 10 NUSD at 10x leverage (1 BTC = 1 NUSD)
			// position and open notional value is 100 NUSD
			// BTC drops in value, now its price is 1 BTC = 0.95 NUSD
			// user has position notional value of 95 NUSD and unrealized PnL of 5 NUSD
			// user closes position
			// user ends up with realized PnL of 5 NUSD, unrealized PnL of 0 NUSD,
			//   position notional value of 0 NUSD
			initialPosition: types.Position{
				TraderAddress:                   testutil.AccAddress().String(),
				Pair:                            asset.Registry.Pair(denoms.BTC, denoms.NUSD),
				Size_:                           sdk.NewDec(-100),
				Margin:                          sdk.NewDec(10),
				OpenNotional:                    sdk.NewDec(100),
				LatestCumulativePremiumFraction: sdk.ZeroDec(),
				LastUpdatedBlockNumber:          0,
			},
			newPriceMultiplier: sdk.MustNewDecFromStr("0.95"),
			newLatestCPF:       sdk.MustNewDecFromStr("0.02"),

			expectedBadDebt:                sdk.ZeroDec(),
			expectedFundingPayment:         sdk.NewDec(-2),
			expectedRealizedPnl:            sdk.MustNewDecFromStr("4.999999990499999999"),
			expectedMarginToVault:          sdk.MustNewDecFromStr("-16.999999990499999999"), // old(10) + (5)(realizedPnL) - (-2)(fundingPayment)
			expectedExchangedNotionalValue: sdk.MustNewDecFromStr("95.000000009500000001"),
		},
		{
			name: "decrease short position, negative PnL",
			// user bought in at 100 BTC for 10 NUSD at 10x leverage (1 BTC = 1 NUSD)
			// position and open notional value is 100 NUSD
			// BTC increases in value, now its price is 1 BTC = 1.05 NUSD
			// user has position notional value of 105 NUSD and unrealized PnL of -5 NUSD
			// user closes their position
			// user ends up with realized PnL of -5 NUSD, unrealized PnL of 0 NUSD
			//   position notional value of 0 NUSD
			initialPosition: types.Position{
				TraderAddress:                   testutil.AccAddress().String(),
				Pair:                            asset.Registry.Pair(denoms.BTC, denoms.NUSD),
				Size_:                           sdk.NewDec(-100), // -100 BTC
				Margin:                          sdk.NewDec(10),   // 10 NUSD
				OpenNotional:                    sdk.NewDec(100),  // 100 NUSD
				LatestCumulativePremiumFraction: sdk.ZeroDec(),
				LastUpdatedBlockNumber:          0,
			},
			newPriceMultiplier: sdk.MustNewDecFromStr("1.05"),
			newLatestCPF:       sdk.MustNewDecFromStr("0.02"),

			expectedBadDebt:                sdk.ZeroDec(),
			expectedFundingPayment:         sdk.NewDec(-2),
			expectedRealizedPnl:            sdk.MustNewDecFromStr("-5.000000010500000001"),
			expectedMarginToVault:          sdk.MustNewDecFromStr("-6.999999989499999999"), // old(10) + (-5)(realizedPnL) - (-2)(fundingPayment)
			expectedExchangedNotionalValue: sdk.MustNewDecFromStr("105.000000010500000001"),
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			app, ctx := testapp.NewNibiruTestAppAndContext(true)
			traderAddr := sdk.MustAccAddressFromBech32(tc.initialPosition.TraderAddress)

			market := mock.TestMarket().WithLatestCumulativePremiumFraction(tc.newLatestCPF)
			amm := mock.TestAMMDefault().WithPriceMultiplier(tc.newPriceMultiplier)
			app.PerpKeeperV2.Markets.Insert(ctx, tc.initialPosition.Pair, *market)
			app.PerpKeeperV2.AMMs.Insert(ctx, tc.initialPosition.Pair, *amm)
			app.PerpKeeperV2.ReserveSnapshots.Insert(ctx, collections.Join(tc.initialPosition.Pair, ctx.BlockTime()), types.ReserveSnapshot{
				Amm:         *amm,
				TimestampMs: ctx.BlockTime().UnixMilli(),
			})
			app.PerpKeeperV2.Positions.Insert(ctx, collections.Join(tc.initialPosition.Pair, traderAddr), tc.initialPosition)
			require.NoError(t, testapp.FundModuleAccount(app.BankKeeper, ctx, types.VaultModuleAccount, sdk.NewCoins(sdk.NewInt64Coin(denoms.NUSD, 1e18))))
			require.NoError(t, testapp.FundModuleAccount(app.BankKeeper, ctx, types.PerpEFModuleAccount, sdk.NewCoins(sdk.NewInt64Coin(denoms.NUSD, 1e18))))

			resp, err := app.PerpKeeperV2.ClosePosition(
				ctx,
				tc.initialPosition.Pair,
				traderAddr,
			)

			require.NoError(t, err)
			assert.Equal(t, types.PositionResp{
				Position: types.Position{
					TraderAddress:                   tc.initialPosition.TraderAddress,
					Pair:                            tc.initialPosition.Pair,
					Size_:                           sdk.ZeroDec(),
					Margin:                          sdk.ZeroDec(),
					OpenNotional:                    sdk.ZeroDec(),
					LatestCumulativePremiumFraction: tc.newLatestCPF,
					LastUpdatedBlockNumber:          ctx.BlockHeight(),
				},
				ExchangedNotionalValue: tc.expectedExchangedNotionalValue,
				ExchangedPositionSize:  tc.initialPosition.Size_.Neg(),
				BadDebt:                tc.expectedBadDebt,
				FundingPayment:         tc.expectedFundingPayment,
				RealizedPnl:            tc.expectedRealizedPnl,
				UnrealizedPnlAfter:     sdk.ZeroDec(),
				MarginToVault:          tc.expectedMarginToVault,
				PositionNotional:       sdk.ZeroDec(),
			}, *resp)

			testutil.RequireHasTypedEvent(t, ctx, &types.PositionChangedEvent{
				FinalPosition: types.Position{
					TraderAddress:                   tc.initialPosition.TraderAddress,
					Pair:                            tc.initialPosition.Pair,
					Size_:                           sdk.ZeroDec(),
					Margin:                          sdk.ZeroDec(),
					OpenNotional:                    sdk.ZeroDec(),
					LatestCumulativePremiumFraction: tc.newLatestCPF,
					LastUpdatedBlockNumber:          ctx.BlockHeight(),
				},
				PositionNotional: sdk.ZeroDec(),
				RealizedPnl:      tc.expectedRealizedPnl,
				BadDebt:          sdk.NewCoin(denoms.NUSD, sdk.ZeroInt()),
				FundingPayment:   tc.expectedFundingPayment,
				TransactionFee:   sdk.NewInt64Coin(denoms.NUSD, 0),
				BlockHeight:      ctx.BlockHeight(),
			})
		})
	}
}

func TestClosePositionWithBadDebt(t *testing.T) {
	pairBtcUsdc := asset.Registry.Pair(denoms.BTC, denoms.USDC)

	alice := testutil.AccAddress()
	startTime := time.Now()

	tc := TestCases{
		TC("close position with bad debt").
			Given(
				SetBlockNumber(1),
				SetBlockTime(startTime),
				CreateCustomMarket(pairBtcUsdc),

				InsertPosition(
					WithTrader(alice),
					WithPair(pairBtcUsdc),
					WithSize(sdk.NewDec(10000)),
					WithMargin(sdk.NewDec(750)),
					WithOpenNotional(sdk.NewDec(10800))),

				FundModule(types.VaultModuleAccount, sdk.NewCoins(sdk.NewInt64Coin(denoms.USDC, 1000))),
				FundModule(types.PerpEFModuleAccount, sdk.NewCoins(sdk.NewInt64Coin(denoms.USDC, 50))),
			).
			When(
				MoveToNextBlock(),
				ClosePosition(alice, pairBtcUsdc),
			).
			Then(
				PositionShouldNotExist(alice, pairBtcUsdc),
				assertion.ModuleBalanceEqual(types.VaultModuleAccount, denoms.USDC, sdk.NewInt(1050)), // 1000 + 50 from perp ef
				assertion.ModuleBalanceEqual(types.PerpEFModuleAccount, denoms.USDC, sdk.NewInt(0)),
			),
	}

	NewTestSuite(t).WithTestCases(tc...).Run()
}

func TestUpdateSwapInvariant(t *testing.T) {
	alice := testutil.AccAddress()
	bob := testutil.AccAddress()
	pairBtcUsdc := asset.Registry.Pair(denoms.BTC, denoms.NUSD)
	startBlockTime := time.Now()

	startingSwapInvariant := sdk.NewDec(1_000_000_000_000).Mul(sdk.NewDec(1_000_000_000_000))

	tc := TestCases{
		TC("only long position - no change to swap invariant").
			Given(
				CreateCustomMarket(pairBtcUsdc),
				SetBlockTime(startBlockTime),
				SetBlockNumber(1),
				SetOraclePrice(pairBtcUsdc, sdk.NewDec(1)),
				FundAccount(alice, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(10_200_000_000)))),
				FundModule(types.PerpEFModuleAccount, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(100000)))),
			).
			When(
				OpenPosition(alice, pairBtcUsdc, types.Direction_LONG, sdk.NewInt(10_000_000_000), sdk.NewDec(1), sdk.ZeroDec()),
				ClosePosition(alice, pairBtcUsdc),
			).
			Then(
				PositionShouldNotExist(alice, pairBtcUsdc),
			),
		TC("only short position - no change to swap invariant").
			Given(
				CreateCustomMarket(pairBtcUsdc),
				SetBlockTime(startBlockTime),
				SetBlockNumber(1),
				SetOraclePrice(pairBtcUsdc, sdk.NewDec(1)),
				FundAccount(alice, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(10_200_000_000)))),
				FundModule(types.PerpEFModuleAccount, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(100000)))),
			).
			When(
				OpenPosition(alice, pairBtcUsdc, types.Direction_SHORT, sdk.NewInt(10_000_000_000), sdk.NewDec(1), sdk.NewDec(10_000_000_000_000)),
				ClosePosition(alice, pairBtcUsdc),
			).
			Then(
				PositionShouldNotExist(alice, pairBtcUsdc),
			),
		TC("only long position - increasing k").
			Given(
				CreateCustomMarket(pairBtcUsdc),
				SetBlockTime(startBlockTime),
				SetBlockNumber(1),
				SetOraclePrice(pairBtcUsdc, sdk.NewDec(1)),
				FundAccount(alice, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(10_200_000_000)))),
				FundModule(types.PerpEFModuleAccount, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(100_000_000)))),
			).
			When(
				OpenPosition(alice, pairBtcUsdc, types.Direction_LONG, sdk.NewInt(10_000_000_000), sdk.NewDec(1), sdk.ZeroDec()),
				EditSwapInvariant(pairBtcUsdc, startingSwapInvariant.MulInt64(100)),
				AMMShouldBeEqual(
					pairBtcUsdc,
					AMM_SwapInvariantShouldBeEqual(sdk.MustNewDecFromStr("99999999999999999999999999.999999000000000000"))),
				ClosePosition(alice, pairBtcUsdc),
			).
			Then(
				PositionShouldNotExist(alice, pairBtcUsdc),
				ModuleBalanceShouldBeEqualTo(types.VaultModuleAccount, sdk.NewCoins()),
			),
		TC("only short position - increasing k").
			Given(
				CreateCustomMarket(pairBtcUsdc),
				SetBlockTime(startBlockTime),
				SetBlockNumber(1),
				SetOraclePrice(pairBtcUsdc, sdk.NewDec(1)),
				FundAccount(alice, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(10_200_000_000)))),
				FundModule(types.PerpEFModuleAccount, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(100_000_000)))),
			).
			When(
				OpenPosition(alice, pairBtcUsdc, types.Direction_SHORT, sdk.NewInt(10_000_000_000), sdk.NewDec(1), sdk.ZeroDec()),
				EditSwapInvariant(pairBtcUsdc, startingSwapInvariant.MulInt64(100)),
				AMMShouldBeEqual(
					pairBtcUsdc,
					AMM_SwapInvariantShouldBeEqual(sdk.MustNewDecFromStr("99999999999999999999999999.999999000000000000"))),
				ClosePosition(alice, pairBtcUsdc),
			).
			Then(
				PositionShouldNotExist(alice, pairBtcUsdc),
				ModuleBalanceShouldBeEqualTo(types.VaultModuleAccount, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.OneInt()))),
			),

		TC("only long position - decreasing k").
			Given(
				CreateCustomMarket(pairBtcUsdc),
				SetBlockTime(startBlockTime),
				SetBlockNumber(1),
				SetOraclePrice(pairBtcUsdc, sdk.NewDec(1)),
				FundAccount(alice, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(10_200_000_000)))),
				FundModule(types.PerpEFModuleAccount, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(100_000_000)))),
			).
			When(
				OpenPosition(alice, pairBtcUsdc, types.Direction_LONG, sdk.NewInt(10_000_000_000), sdk.NewDec(1), sdk.ZeroDec()),
				EditSwapInvariant(pairBtcUsdc, startingSwapInvariant.Mul(sdk.MustNewDecFromStr("0.1"))),
				AMMShouldBeEqual(
					pairBtcUsdc,
					AMM_SwapInvariantShouldBeEqual(sdk.MustNewDecFromStr("99999999999999999873578.871987715651277660"))),
				ClosePosition(alice, pairBtcUsdc),
			).
			Then(
				PositionShouldNotExist(alice, pairBtcUsdc),
				ModuleBalanceShouldBeEqualTo(types.VaultModuleAccount, sdk.NewCoins()),
			),
		TC("only short position - decreasing k").
			Given(
				CreateCustomMarket(pairBtcUsdc),
				SetBlockTime(startBlockTime),
				SetBlockNumber(1),
				SetOraclePrice(pairBtcUsdc, sdk.NewDec(1)),
				FundAccount(alice, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(10_200_000_000)))),
				FundModule(types.PerpEFModuleAccount, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(100_000_000)))),
			).
			When(
				OpenPosition(alice, pairBtcUsdc, types.Direction_SHORT, sdk.NewInt(10_000_000_000), sdk.NewDec(1), sdk.ZeroDec()),
				EditSwapInvariant(pairBtcUsdc, startingSwapInvariant.Mul(sdk.MustNewDecFromStr("0.1"))),
				AMMShouldBeEqual(
					pairBtcUsdc,
					AMM_SwapInvariantShouldBeEqual(sdk.MustNewDecFromStr("99999999999999999873578.871987801032774485"))),
				ClosePosition(alice, pairBtcUsdc),
			).
			Then(
				PositionShouldNotExist(alice, pairBtcUsdc),
				ModuleBalanceShouldBeEqualTo(types.VaultModuleAccount, sdk.NewCoins()),
			),

		TC("long and short position - increasing k").
			Given(
				CreateCustomMarket(pairBtcUsdc),
				SetBlockTime(startBlockTime),
				SetBlockNumber(1),
				SetOraclePrice(pairBtcUsdc, sdk.NewDec(1)),
				FundAccount(alice, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(10_200_000_000)))),
				FundAccount(bob, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(10_200_000_000)))),
			).
			When(
				OpenPosition(alice, pairBtcUsdc, types.Direction_LONG, sdk.NewInt(10_000_000_000), sdk.NewDec(1), sdk.ZeroDec()),
				OpenPosition(bob, pairBtcUsdc, types.Direction_SHORT, sdk.NewInt(10_000_000_000), sdk.NewDec(1), sdk.NewDec(10_000_000_000_000)),

				EditSwapInvariant(pairBtcUsdc, startingSwapInvariant.MulInt64(100)),
				AMMShouldBeEqual(
					pairBtcUsdc,
					AMM_BiasShouldBeEqual(sdk.ZeroDec()),
					AMM_SwapInvariantShouldBeEqual(sdk.MustNewDecFromStr("100000000000000000000000000.000000000000000000"))),

				ModuleBalanceShouldBeEqualTo(types.VaultModuleAccount, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(20_000_000_000)))),
				ModuleBalanceShouldBeEqualTo(types.FeePoolModuleAccount, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(20_000_000)))), // Fees are 10_000_000_000 * 0.0010 * 2

				ClosePosition(alice, pairBtcUsdc),
				ClosePosition(bob, pairBtcUsdc),
			).
			Then(
				PositionShouldNotExist(alice, pairBtcUsdc),
				PositionShouldNotExist(bob, pairBtcUsdc),

				ModuleBalanceShouldBeEqualTo(types.VaultModuleAccount, sdk.NewCoins()),
				ModuleBalanceShouldBeEqualTo(types.PerpEFModuleAccount, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(39_782_394)))),
			),
		TC("long and short position - reducing k").
			Given(
				CreateCustomMarket(pairBtcUsdc),
				SetBlockTime(startBlockTime),
				SetBlockNumber(1),
				SetOraclePrice(pairBtcUsdc, sdk.NewDec(1)),
				FundAccount(alice, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(10_200_000_000)))),
				FundAccount(bob, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(10_200_000_000)))),
			).
			When(
				OpenPosition(alice, pairBtcUsdc, types.Direction_LONG, sdk.NewInt(10_000_000_000), sdk.NewDec(1), sdk.ZeroDec()),
				OpenPosition(bob, pairBtcUsdc, types.Direction_SHORT, sdk.NewInt(10_000_000_000), sdk.NewDec(1), sdk.NewDec(10_000_000_000_000)),

				EditSwapInvariant(pairBtcUsdc, startingSwapInvariant.Mul(sdk.MustNewDecFromStr("0.1"))),
				AMMShouldBeEqual(
					pairBtcUsdc,
					AMM_BiasShouldBeEqual(sdk.ZeroDec()),
					AMM_SwapInvariantShouldBeEqual(sdk.MustNewDecFromStr("99999999999999999873578.871987712489000000"))),

				ModuleBalanceShouldBeEqualTo(types.VaultModuleAccount, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(20_000_000_000)))),
				ModuleBalanceShouldBeEqualTo(types.FeePoolModuleAccount, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(20_000_000)))), // Fees are 10_000_000_000 * 0.0010 * 2

				ClosePosition(alice, pairBtcUsdc),
				ClosePosition(bob, pairBtcUsdc),
			).
			Then(
				PositionShouldNotExist(alice, pairBtcUsdc),
				PositionShouldNotExist(bob, pairBtcUsdc),

				ModuleBalanceShouldBeEqualTo(types.VaultModuleAccount, sdk.NewCoins()),
				ModuleBalanceShouldBeEqualTo(types.PerpEFModuleAccount, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(39_200_810)))),
			),
	}

	NewTestSuite(t).WithTestCases(tc...).Run()
}
