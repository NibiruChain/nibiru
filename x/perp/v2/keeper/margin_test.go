package keeper_test

import (
	"testing"
	"time"

	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/common/denoms"
	"github.com/NibiruChain/nibiru/x/common/testutil"
	. "github.com/NibiruChain/nibiru/x/common/testutil/action"
	. "github.com/NibiruChain/nibiru/x/common/testutil/assertion"
	. "github.com/NibiruChain/nibiru/x/perp/v2/integration/action"
	. "github.com/NibiruChain/nibiru/x/perp/v2/integration/assertion"
	types "github.com/NibiruChain/nibiru/x/perp/v2/types"
)

func TestAddMargin(t *testing.T) {
	alice := testutil.AccAddress()
	pairBtcUsdc := asset.Registry.Pair(denoms.BTC, denoms.USDC)
	pairEthUsdc := asset.Registry.Pair(denoms.ETH, denoms.USDC)
	startBlockTime := time.Now()

	tc := TestCases{
		TC("existing long position, add margin").
			Given(
				CreateCustomMarket(pairBtcUsdc, WithEnabled(true)),
				SetBlockNumber(1),
				SetBlockTime(startBlockTime),
				FundAccount(alice, sdk.NewCoins(sdk.NewCoin(types.TestingCollateralDenomNUSD, sdk.NewInt(2000)))),
				MarketOrder(alice, pairBtcUsdc, types.Direction_LONG, sdk.NewInt(1000), sdk.NewDec(10), sdk.ZeroDec()),
			).
			When(
				MoveToNextBlock(),
				AddMargin(alice, pairBtcUsdc, sdk.NewInt(1000)),
			).
			Then(
				PositionShouldBeEqual(alice, pairBtcUsdc, Position_PositionShouldBeEqualTo(
					types.Position{
						Pair:                            pairBtcUsdc,
						TraderAddress:                   alice.String(),
						Size_:                           sdk.MustNewDecFromStr("9799.999903960000941192"),
						Margin:                          sdk.NewDec(1980),
						OpenNotional:                    sdk.NewDec(9800),
						LatestCumulativePremiumFraction: sdk.ZeroDec(),
						LastUpdatedBlockNumber:          2,
					},
				)),
				PositionChangedEventShouldBeEqual(&types.PositionChangedEvent{
					FinalPosition: types.Position{
						Pair:                            pairBtcUsdc,
						TraderAddress:                   alice.String(),
						Size_:                           sdk.MustNewDecFromStr("9799.999903960000941192"),
						Margin:                          sdk.NewDec(1980),
						OpenNotional:                    sdk.NewDec(9800),
						LatestCumulativePremiumFraction: sdk.ZeroDec(),
						LastUpdatedBlockNumber:          2,
					},
					PositionNotional:  sdk.NewDec(9800),
					RealizedPnl:       sdk.ZeroDec(),
					BadDebt:           sdk.NewCoin(types.TestingCollateralDenomNUSD, sdk.ZeroInt()),
					FundingPayment:    sdk.ZeroDec(),
					TransactionFee:    sdk.NewCoin(types.TestingCollateralDenomNUSD, sdk.ZeroInt()),
					BlockHeight:       2,
					MarginToUser:      sdk.NewInt(-1_000),
					ChangeReason:      types.ChangeReason_AddMargin,
					ExchangedNotional: sdk.MustNewDecFromStr("0"),
					ExchangedSize:     sdk.MustNewDecFromStr("0"),
				}),
				BalanceEqual(alice, types.TestingCollateralDenomNUSD, sdk.ZeroInt()),
				ModuleBalanceEqual(types.PerpEFModuleAccount, types.TestingCollateralDenomNUSD, sdk.NewInt(10)),
				ModuleBalanceEqual(types.FeePoolModuleAccount, types.TestingCollateralDenomNUSD, sdk.NewInt(10)),
			),

		TC("existing short position, add margin").
			Given(
				CreateCustomMarket(pairBtcUsdc, WithEnabled(true)),
				SetBlockNumber(1),
				SetBlockTime(startBlockTime),
				FundAccount(alice, sdk.NewCoins(sdk.NewCoin(types.TestingCollateralDenomNUSD, sdk.NewInt(2000)))),
				MarketOrder(alice, pairBtcUsdc, types.Direction_SHORT, sdk.NewInt(1000), sdk.NewDec(10), sdk.ZeroDec()),
			).
			When(
				MoveToNextBlock(),
				AddMargin(alice, pairBtcUsdc, sdk.NewInt(1000)),
			).
			Then(
				PositionShouldBeEqual(alice, pairBtcUsdc, Position_PositionShouldBeEqualTo(
					types.Position{
						Pair:                            pairBtcUsdc,
						TraderAddress:                   alice.String(),
						Size_:                           sdk.MustNewDecFromStr("-9800.000096040000941192"),
						Margin:                          sdk.NewDec(1980),
						OpenNotional:                    sdk.NewDec(9800),
						LatestCumulativePremiumFraction: sdk.ZeroDec(),
						LastUpdatedBlockNumber:          2,
					},
				)),
				PositionChangedEventShouldBeEqual(&types.PositionChangedEvent{
					FinalPosition: types.Position{
						Pair:                            pairBtcUsdc,
						TraderAddress:                   alice.String(),
						Size_:                           sdk.MustNewDecFromStr("-9800.000096040000941192"),
						Margin:                          sdk.NewDec(1980),
						OpenNotional:                    sdk.NewDec(9800),
						LatestCumulativePremiumFraction: sdk.ZeroDec(),
						LastUpdatedBlockNumber:          2,
					},
					PositionNotional:  sdk.NewDec(9800),
					RealizedPnl:       sdk.ZeroDec(),
					BadDebt:           sdk.NewCoin(types.TestingCollateralDenomNUSD, sdk.ZeroInt()),
					FundingPayment:    sdk.ZeroDec(),
					TransactionFee:    sdk.NewCoin(types.TestingCollateralDenomNUSD, sdk.ZeroInt()),
					BlockHeight:       2,
					MarginToUser:      sdk.NewInt(-1000),
					ChangeReason:      types.ChangeReason_AddMargin,
					ExchangedNotional: sdk.MustNewDecFromStr("0"),
					ExchangedSize:     sdk.MustNewDecFromStr("0"),
				}),
				BalanceEqual(alice, types.TestingCollateralDenomNUSD, sdk.ZeroInt()),
				ModuleBalanceEqual(types.PerpEFModuleAccount, types.TestingCollateralDenomNUSD, sdk.NewInt(10)),
				ModuleBalanceEqual(types.FeePoolModuleAccount, types.TestingCollateralDenomNUSD, sdk.NewInt(10)),
			),

		TC("Testing fails").
			Given(
				CreateCustomMarket(pairBtcUsdc, WithEnabled(true)),
				CreateCustomMarket(pairEthUsdc, WithEnabled(true)),

				SetBlockNumber(1),
				SetBlockTime(startBlockTime),
				FundAccount(alice, sdk.NewCoins(sdk.NewCoin(types.TestingCollateralDenomNUSD, sdk.NewInt(1020)))),
				MarketOrder(alice, pairBtcUsdc, types.Direction_LONG, sdk.NewInt(1000), sdk.NewDec(10), sdk.ZeroDec()),
			).
			When(
				MoveToNextBlock(),
				AddMarginFail(alice, asset.MustNewPair("luna:usdt"), sdk.NewInt(1000), types.ErrPairNotFound),
				AddMarginFail(alice, pairEthUsdc, sdk.NewInt(1000), types.ErrPositionNotFound),
				AddMarginFail(alice, pairBtcUsdc, sdk.NewInt(1000), sdkerrors.ErrInsufficientFunds),

				RemoveMarginFail(alice, asset.MustNewPair("luna:usdt"), sdk.NewInt(1000), types.ErrPairNotFound),
				RemoveMarginFail(alice, pairEthUsdc, sdk.NewInt(1000), types.ErrPositionNotFound),
				RemoveMarginFail(alice, pairBtcUsdc, sdk.NewInt(2000), types.ErrBadDebt),
				RemoveMarginFail(alice, pairBtcUsdc, sdk.NewInt(900), types.ErrMarginRatioTooLow),
			),
	}

	NewTestSuite(t).WithTestCases(tc...).Run()
}

func TestRemoveMargin(t *testing.T) {
	alice := testutil.AccAddress()
	pairBtcUsdc := asset.Registry.Pair(denoms.BTC, denoms.USDC)
	startBlockTime := time.Now()

	tc := TestCases{
		TC("existing long position, remove margin").
			Given(
				CreateCustomMarket(pairBtcUsdc, WithEnabled(true)),
				SetBlockNumber(1),
				SetBlockTime(startBlockTime),
				FundAccount(alice, sdk.NewCoins(sdk.NewCoin(types.TestingCollateralDenomNUSD, sdk.NewInt(1000)))),
				MarketOrder(alice, pairBtcUsdc, types.Direction_LONG, sdk.NewInt(1000), sdk.OneDec(), sdk.ZeroDec()),
			).
			When(
				MoveToNextBlock(),
				RemoveMargin(alice, pairBtcUsdc, sdk.NewInt(500)),
			).
			Then(
				PositionShouldBeEqual(alice, pairBtcUsdc, Position_PositionShouldBeEqualTo(types.Position{
					Pair:                            pairBtcUsdc,
					TraderAddress:                   alice.String(),
					Size_:                           sdk.MustNewDecFromStr("997.999999003996000994"),
					Margin:                          sdk.NewDec(498),
					OpenNotional:                    sdk.NewDec(998),
					LatestCumulativePremiumFraction: sdk.ZeroDec(),
					LastUpdatedBlockNumber:          2,
				})),
				PositionChangedEventShouldBeEqual(&types.PositionChangedEvent{
					FinalPosition: types.Position{
						Pair:                            pairBtcUsdc,
						TraderAddress:                   alice.String(),
						Size_:                           sdk.MustNewDecFromStr("997.999999003996000994"),
						Margin:                          sdk.NewDec(498),
						OpenNotional:                    sdk.NewDec(998),
						LatestCumulativePremiumFraction: sdk.ZeroDec(),
						LastUpdatedBlockNumber:          2,
					},
					PositionNotional:  sdk.NewDec(998),
					RealizedPnl:       sdk.ZeroDec(),
					BadDebt:           sdk.NewCoin(types.TestingCollateralDenomNUSD, sdk.ZeroInt()),
					FundingPayment:    sdk.ZeroDec(),
					TransactionFee:    sdk.NewCoin(types.TestingCollateralDenomNUSD, sdk.ZeroInt()),
					BlockHeight:       2,
					MarginToUser:      sdk.NewInt(500),
					ChangeReason:      types.ChangeReason_RemoveMargin,
					ExchangedNotional: sdk.MustNewDecFromStr("0"),
					ExchangedSize:     sdk.MustNewDecFromStr("0"),
				}),
				BalanceEqual(alice, types.TestingCollateralDenomNUSD, sdk.NewInt(500)),
				ModuleBalanceEqual(types.PerpEFModuleAccount, types.TestingCollateralDenomNUSD, sdk.OneInt()),
				ModuleBalanceEqual(types.FeePoolModuleAccount, types.TestingCollateralDenomNUSD, sdk.OneInt()),
			),

		TC("existing long position, remove almost all margin fails").
			Given(
				CreateCustomMarket(pairBtcUsdc, WithEnabled(true)),
				SetBlockNumber(1),
				SetBlockTime(startBlockTime),
				FundAccount(alice, sdk.NewCoins(sdk.NewCoin(types.TestingCollateralDenomNUSD, sdk.NewInt(1000)))),
				MarketOrder(alice, pairBtcUsdc, types.Direction_LONG, sdk.NewInt(1000), sdk.OneDec(), sdk.ZeroDec()),
				MoveToNextBlock(),
			).
			When(
				RemoveMarginFail(alice, pairBtcUsdc, sdk.NewInt(997), types.ErrMarginRatioTooLow),
			).
			Then(
				PositionShouldBeEqual(alice, pairBtcUsdc, Position_PositionShouldBeEqualTo(types.Position{
					Pair:                            pairBtcUsdc,
					TraderAddress:                   alice.String(),
					Size_:                           sdk.MustNewDecFromStr("997.999999003996000994"),
					Margin:                          sdk.NewDec(998),
					OpenNotional:                    sdk.NewDec(998),
					LatestCumulativePremiumFraction: sdk.ZeroDec(),
					LastUpdatedBlockNumber:          1,
				})),
				BalanceEqual(alice, types.TestingCollateralDenomNUSD, sdk.ZeroInt()),
				ModuleBalanceEqual(types.PerpEFModuleAccount, types.TestingCollateralDenomNUSD, sdk.OneInt()),
				ModuleBalanceEqual(types.FeePoolModuleAccount, types.TestingCollateralDenomNUSD, sdk.OneInt()),
			),

		TC("existing short position, remove margin").
			Given(
				CreateCustomMarket(pairBtcUsdc, WithEnabled(true)),
				SetBlockNumber(1),
				SetBlockTime(startBlockTime),
				FundAccount(alice, sdk.NewCoins(sdk.NewCoin(types.TestingCollateralDenomNUSD, sdk.NewInt(1000)))),
				MarketOrder(alice, pairBtcUsdc, types.Direction_SHORT, sdk.NewInt(1000), sdk.OneDec(), sdk.ZeroDec()),
			).
			When(
				MoveToNextBlock(),
				RemoveMargin(alice, pairBtcUsdc, sdk.NewInt(500)),
			).
			Then(
				PositionShouldBeEqual(alice, pairBtcUsdc, Position_PositionShouldBeEqualTo(types.Position{
					Pair:                            pairBtcUsdc,
					TraderAddress:                   alice.String(),
					Size_:                           sdk.MustNewDecFromStr("-998.000000996004000994"),
					Margin:                          sdk.NewDec(498),
					OpenNotional:                    sdk.NewDec(998),
					LatestCumulativePremiumFraction: sdk.ZeroDec(),
					LastUpdatedBlockNumber:          2,
				})),
				PositionChangedEventShouldBeEqual(&types.PositionChangedEvent{
					FinalPosition: types.Position{
						Pair:                            pairBtcUsdc,
						TraderAddress:                   alice.String(),
						Size_:                           sdk.MustNewDecFromStr("-998.000000996004000994"),
						Margin:                          sdk.NewDec(498),
						OpenNotional:                    sdk.NewDec(998),
						LatestCumulativePremiumFraction: sdk.ZeroDec(),
						LastUpdatedBlockNumber:          2,
					},
					PositionNotional:  sdk.NewDec(998),
					RealizedPnl:       sdk.ZeroDec(),
					BadDebt:           sdk.NewCoin(types.TestingCollateralDenomNUSD, sdk.ZeroInt()),
					FundingPayment:    sdk.ZeroDec(),
					TransactionFee:    sdk.NewCoin(types.TestingCollateralDenomNUSD, sdk.ZeroInt()),
					BlockHeight:       2,
					MarginToUser:      sdk.NewInt(500),
					ChangeReason:      types.ChangeReason_RemoveMargin,
					ExchangedNotional: sdk.MustNewDecFromStr("0"),
					ExchangedSize:     sdk.MustNewDecFromStr("0"),
				}),
				BalanceEqual(alice, types.TestingCollateralDenomNUSD, sdk.NewInt(500)),
				ModuleBalanceEqual(types.PerpEFModuleAccount, types.TestingCollateralDenomNUSD, sdk.OneInt()),
				ModuleBalanceEqual(types.FeePoolModuleAccount, types.TestingCollateralDenomNUSD, sdk.OneInt()),
			),

		TC("existing short position, remove almost all margin fails").
			Given(
				CreateCustomMarket(pairBtcUsdc, WithEnabled(true)),
				SetBlockNumber(1),
				SetBlockTime(startBlockTime),
				FundAccount(alice, sdk.NewCoins(sdk.NewCoin(types.TestingCollateralDenomNUSD, sdk.NewInt(1000)))),
				MarketOrder(alice, pairBtcUsdc, types.Direction_SHORT, sdk.NewInt(1000), sdk.OneDec(), sdk.ZeroDec()),
				MoveToNextBlock(),
			).
			When(
				RemoveMarginFail(alice, pairBtcUsdc, sdk.NewInt(997), types.ErrMarginRatioTooLow),
			).
			Then(
				PositionShouldBeEqual(alice, pairBtcUsdc, Position_PositionShouldBeEqualTo(types.Position{
					Pair:                            pairBtcUsdc,
					TraderAddress:                   alice.String(),
					Size_:                           sdk.MustNewDecFromStr("-998.000000996004000994"),
					Margin:                          sdk.NewDec(998),
					OpenNotional:                    sdk.NewDec(998),
					LatestCumulativePremiumFraction: sdk.ZeroDec(),
					LastUpdatedBlockNumber:          1,
				})),
				BalanceEqual(alice, types.TestingCollateralDenomNUSD, sdk.ZeroInt()),
				ModuleBalanceEqual(types.PerpEFModuleAccount, types.TestingCollateralDenomNUSD, sdk.OneInt()),
				ModuleBalanceEqual(types.FeePoolModuleAccount, types.TestingCollateralDenomNUSD, sdk.OneInt()),
			),
	}

	NewTestSuite(t).WithTestCases(tc...).Run()
}
