package keeper_test

import (
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	"github.com/NibiruChain/collections"

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
				CreateCustomMarket(pairBtcUsdc),
				SetBlockNumber(1),
				SetBlockTime(startBlockTime),
				FundAccount(alice, sdk.NewCoins(sdk.NewCoin(denoms.USDC, sdk.NewInt(2020)))),
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
						Size_:                           sdk.MustNewDecFromStr("9999.999900000001000000"),
						Margin:                          sdk.NewDec(2000),
						OpenNotional:                    sdk.NewDec(10000),
						LatestCumulativePremiumFraction: sdk.ZeroDec(),
						LastUpdatedBlockNumber:          2,
					},
				)),
				PositionChangedEventShouldBeEqual(&types.PositionChangedEvent{
					FinalPosition: types.Position{
						Pair:                            pairBtcUsdc,
						TraderAddress:                   alice.String(),
						Size_:                           sdk.MustNewDecFromStr("9999.999900000001000000"),
						Margin:                          sdk.NewDec(2000),
						OpenNotional:                    sdk.NewDec(10000),
						LatestCumulativePremiumFraction: sdk.ZeroDec(),
						LastUpdatedBlockNumber:          2,
					},
					PositionNotional: sdk.NewDec(10_000),
					RealizedPnl:      sdk.ZeroDec(),
					BadDebt:          sdk.NewCoin(denoms.USDC, sdk.ZeroInt()),
					FundingPayment:   sdk.ZeroDec(),
					TransactionFee:   sdk.NewCoin(denoms.USDC, sdk.ZeroInt()),
					BlockHeight:      2,
					MarginToUser:     sdk.NewInt(-1_000),
					ChangeReason:     types.ChangeReason_AddMargin,
				}),
				BalanceEqual(alice, denoms.USDC, sdk.ZeroInt()),
				ModuleBalanceEqual(types.PerpEFModuleAccount, denoms.USDC, sdk.NewInt(10)),
				ModuleBalanceEqual(types.FeePoolModuleAccount, denoms.USDC, sdk.NewInt(10)),
			),

		TC("existing short position, add margin").
			Given(
				CreateCustomMarket(pairBtcUsdc),
				SetBlockNumber(1),
				SetBlockTime(startBlockTime),
				FundAccount(alice, sdk.NewCoins(sdk.NewCoin(denoms.USDC, sdk.NewInt(2020)))),
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
						Size_:                           sdk.MustNewDecFromStr("-10000.000100000001000000"),
						Margin:                          sdk.NewDec(2000),
						OpenNotional:                    sdk.NewDec(10000),
						LatestCumulativePremiumFraction: sdk.ZeroDec(),
						LastUpdatedBlockNumber:          2,
					},
				)),
				PositionChangedEventShouldBeEqual(&types.PositionChangedEvent{
					FinalPosition: types.Position{
						Pair:                            pairBtcUsdc,
						TraderAddress:                   alice.String(),
						Size_:                           sdk.MustNewDecFromStr("-10000.000100000001000000"),
						Margin:                          sdk.NewDec(2000),
						OpenNotional:                    sdk.NewDec(10000),
						LatestCumulativePremiumFraction: sdk.ZeroDec(),
						LastUpdatedBlockNumber:          2,
					},
					PositionNotional: sdk.NewDec(10_000),
					RealizedPnl:      sdk.ZeroDec(),
					BadDebt:          sdk.NewCoin(denoms.USDC, sdk.ZeroInt()),
					FundingPayment:   sdk.ZeroDec(),
					TransactionFee:   sdk.NewCoin(denoms.USDC, sdk.ZeroInt()),
					BlockHeight:      2,
					MarginToUser:     sdk.NewInt(-1000),
					ChangeReason:     types.ChangeReason_AddMargin,
				}),
				BalanceEqual(alice, denoms.USDC, sdk.ZeroInt()),
				ModuleBalanceEqual(types.PerpEFModuleAccount, denoms.USDC, sdk.NewInt(10)),
				ModuleBalanceEqual(types.FeePoolModuleAccount, denoms.USDC, sdk.NewInt(10)),
			),

		TC("Testing fails").
			Given(
				CreateCustomMarket(pairBtcUsdc),
				CreateCustomMarket(pairEthUsdc),

				SetBlockNumber(1),
				SetBlockTime(startBlockTime),
				FundAccount(alice, sdk.NewCoins(sdk.NewCoin(denoms.USDC, sdk.NewInt(1020)))),
				MarketOrder(alice, pairBtcUsdc, types.Direction_LONG, sdk.NewInt(1000), sdk.NewDec(10), sdk.ZeroDec()),
			).
			When(
				MoveToNextBlock(),
				AddMarginFail(alice, asset.MustNewPair("luna:usdt"), sdk.NewInt(1000), types.ErrPairNotFound),
				AddMarginFail(alice, pairEthUsdc, sdk.NewInt(1000), collections.ErrNotFound),
				AddMarginFail(alice, pairBtcUsdc, sdk.NewInt(1000), sdkerrors.ErrInsufficientFunds),

				RemoveMarginFail(alice, asset.MustNewPair("luna:usdt"), sdk.NewInt(1000), types.ErrPairNotFound),
				RemoveMarginFail(alice, pairEthUsdc, sdk.NewInt(1000), collections.ErrNotFound),
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
				CreateCustomMarket(pairBtcUsdc),
				SetBlockNumber(1),
				SetBlockTime(startBlockTime),
				FundAccount(alice, sdk.NewCoins(sdk.NewCoin(denoms.USDC, sdk.NewInt(1002)))),
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
					Size_:                           sdk.MustNewDecFromStr("999.999999000000001000"),
					Margin:                          sdk.NewDec(500),
					OpenNotional:                    sdk.NewDec(1000),
					LatestCumulativePremiumFraction: sdk.ZeroDec(),
					LastUpdatedBlockNumber:          2,
				})),
				PositionChangedEventShouldBeEqual(&types.PositionChangedEvent{
					FinalPosition: types.Position{
						Pair:                            pairBtcUsdc,
						TraderAddress:                   alice.String(),
						Size_:                           sdk.MustNewDecFromStr("999.999999000000001000"),
						Margin:                          sdk.NewDec(500),
						OpenNotional:                    sdk.NewDec(1000),
						LatestCumulativePremiumFraction: sdk.ZeroDec(),
						LastUpdatedBlockNumber:          2,
					},
					PositionNotional: sdk.NewDec(1000),
					RealizedPnl:      sdk.ZeroDec(),
					BadDebt:          sdk.NewCoin(denoms.USDC, sdk.ZeroInt()),
					FundingPayment:   sdk.ZeroDec(),
					TransactionFee:   sdk.NewCoin(denoms.USDC, sdk.ZeroInt()),
					BlockHeight:      2,
					MarginToUser:     sdk.NewInt(500),
					ChangeReason:     types.ChangeReason_RemoveMargin,
				}),
				BalanceEqual(alice, denoms.USDC, sdk.NewInt(500)),
				ModuleBalanceEqual(types.PerpEFModuleAccount, denoms.USDC, sdk.OneInt()),
				ModuleBalanceEqual(types.FeePoolModuleAccount, denoms.USDC, sdk.OneInt()),
			),

		TC("existing long position, remove almost all margin fails").
			Given(
				CreateCustomMarket(pairBtcUsdc),
				SetBlockNumber(1),
				SetBlockTime(startBlockTime),
				FundAccount(alice, sdk.NewCoins(sdk.NewCoin(denoms.USDC, sdk.NewInt(1002)))),
				MarketOrder(alice, pairBtcUsdc, types.Direction_LONG, sdk.NewInt(1000), sdk.OneDec(), sdk.ZeroDec()),
				MoveToNextBlock(),
			).
			When(
				RemoveMarginFail(alice, pairBtcUsdc, sdk.NewInt(999), types.ErrMarginRatioTooLow),
			).
			Then(
				PositionShouldBeEqual(alice, pairBtcUsdc, Position_PositionShouldBeEqualTo(types.Position{
					Pair:                            pairBtcUsdc,
					TraderAddress:                   alice.String(),
					Size_:                           sdk.MustNewDecFromStr("999.999999000000001000"),
					Margin:                          sdk.NewDec(1000),
					OpenNotional:                    sdk.NewDec(1000),
					LatestCumulativePremiumFraction: sdk.ZeroDec(),
					LastUpdatedBlockNumber:          1,
				})),
				BalanceEqual(alice, denoms.USDC, sdk.ZeroInt()),
				ModuleBalanceEqual(types.PerpEFModuleAccount, denoms.USDC, sdk.OneInt()),
				ModuleBalanceEqual(types.FeePoolModuleAccount, denoms.USDC, sdk.OneInt()),
			),

		TC("existing short position, remove margin").
			Given(
				CreateCustomMarket(pairBtcUsdc),
				SetBlockNumber(1),
				SetBlockTime(startBlockTime),
				FundAccount(alice, sdk.NewCoins(sdk.NewCoin(denoms.USDC, sdk.NewInt(1002)))),
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
					Size_:                           sdk.MustNewDecFromStr("-1000.000001000000001000"),
					Margin:                          sdk.NewDec(500),
					OpenNotional:                    sdk.NewDec(1000),
					LatestCumulativePremiumFraction: sdk.ZeroDec(),
					LastUpdatedBlockNumber:          2,
				})),
				PositionChangedEventShouldBeEqual(&types.PositionChangedEvent{
					FinalPosition: types.Position{
						Pair:                            pairBtcUsdc,
						TraderAddress:                   alice.String(),
						Size_:                           sdk.MustNewDecFromStr("-1000.000001000000001000"),
						Margin:                          sdk.NewDec(500),
						OpenNotional:                    sdk.NewDec(1000),
						LatestCumulativePremiumFraction: sdk.ZeroDec(),
						LastUpdatedBlockNumber:          2,
					},
					PositionNotional: sdk.NewDec(1000),
					RealizedPnl:      sdk.ZeroDec(),
					BadDebt:          sdk.NewCoin(denoms.USDC, sdk.ZeroInt()),
					FundingPayment:   sdk.ZeroDec(),
					TransactionFee:   sdk.NewCoin(denoms.USDC, sdk.ZeroInt()),
					BlockHeight:      2,
					MarginToUser:     sdk.NewInt(500),
					ChangeReason:     types.ChangeReason_RemoveMargin,
				}),
				BalanceEqual(alice, denoms.USDC, sdk.NewInt(500)),
				ModuleBalanceEqual(types.PerpEFModuleAccount, denoms.USDC, sdk.OneInt()),
				ModuleBalanceEqual(types.FeePoolModuleAccount, denoms.USDC, sdk.OneInt()),
			),

		TC("existing short position, remove almost all margin fails").
			Given(
				CreateCustomMarket(pairBtcUsdc),
				SetBlockNumber(1),
				SetBlockTime(startBlockTime),
				FundAccount(alice, sdk.NewCoins(sdk.NewCoin(denoms.USDC, sdk.NewInt(1002)))),
				MarketOrder(alice, pairBtcUsdc, types.Direction_SHORT, sdk.NewInt(1000), sdk.OneDec(), sdk.ZeroDec()),
				MoveToNextBlock(),
			).
			When(
				RemoveMarginFail(alice, pairBtcUsdc, sdk.NewInt(999), types.ErrMarginRatioTooLow),
			).
			Then(
				PositionShouldBeEqual(alice, pairBtcUsdc, Position_PositionShouldBeEqualTo(types.Position{
					Pair:                            pairBtcUsdc,
					TraderAddress:                   alice.String(),
					Size_:                           sdk.MustNewDecFromStr("-1000.000001000000001000"),
					Margin:                          sdk.NewDec(1000),
					OpenNotional:                    sdk.NewDec(1000),
					LatestCumulativePremiumFraction: sdk.ZeroDec(),
					LastUpdatedBlockNumber:          1,
				})),
				BalanceEqual(alice, denoms.USDC, sdk.ZeroInt()),
				ModuleBalanceEqual(types.PerpEFModuleAccount, denoms.USDC, sdk.OneInt()),
				ModuleBalanceEqual(types.FeePoolModuleAccount, denoms.USDC, sdk.OneInt()),
			),
	}

	NewTestSuite(t).WithTestCases(tc...).Run()
}
