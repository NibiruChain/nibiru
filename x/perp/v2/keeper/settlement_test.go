package keeper_test

import (
	"github.com/stretchr/testify/require"
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/common/denoms"
	"github.com/NibiruChain/nibiru/x/common/testutil"
	. "github.com/NibiruChain/nibiru/x/common/testutil/action"
	. "github.com/NibiruChain/nibiru/x/common/testutil/assertion"
	. "github.com/NibiruChain/nibiru/x/perp/v2/integration/action"
	. "github.com/NibiruChain/nibiru/x/perp/v2/integration/assertion"
	"github.com/NibiruChain/nibiru/x/perp/v2/types"
)

func TestSettlePosition(t *testing.T) {
	pairBtcUsdc := asset.Registry.Pair(denoms.BTC, denoms.NUSD)
	startTime := time.Now()

	alice := testutil.AccAddress()
	bob := testutil.AccAddress()

	adminUser, err := sdk.AccAddressFromBech32(testutil.ADDR_SUDO_ROOT)
	require.NoError(t, err)

	tc := TestCases{
		TC("Happy path").When(
			CreateCustomMarket(
				pairBtcUsdc,
				WithPricePeg(sdk.OneDec()),
				WithSqrtDepth(sdk.NewDec(100_000)),
				WithEnabled(true),
			),
			SetBlockNumber(1),
			SetBlockTime(startTime),
			FundAccount(alice, sdk.NewCoins(sdk.NewCoin(types.TestingCollateralDenomNUSD, sdk.NewInt(10_200)))),
			MarketOrder(
				alice,
				pairBtcUsdc,
				types.Direction_LONG,
				sdk.NewInt(10_000),
				sdk.OneDec(),
				sdk.ZeroDec(),
			),
		).When(
			CloseMarket(pairBtcUsdc, adminUser),
			SettlePosition(pairBtcUsdc, 1, alice),
		).Then(
			PositionShouldNotExist(alice, pairBtcUsdc, 1),
		),

		TC("Happy path, but with bad debt").When(
			CreateCustomMarket(
				pairBtcUsdc,
				WithPricePeg(sdk.OneDec()),
				WithSqrtDepth(sdk.NewDec(100_000)),
				WithEnabled(true),
			),
			SetBlockNumber(1),
			SetBlockTime(startTime),
			FundAccount(alice, sdk.NewCoins(sdk.NewCoin(types.TestingCollateralDenomNUSD, sdk.NewInt(104)))), // need 4 because we need to pay for the close position fee
			FundAccount(bob, sdk.NewCoins(sdk.NewCoin(types.TestingCollateralDenomNUSD, sdk.NewInt(1_020)))),
			MarketOrder(
				alice,
				pairBtcUsdc,
				types.Direction_SHORT,
				sdk.NewInt(100),
				sdk.NewDec(10),
				sdk.ZeroDec(),
			),
			MarketOrder(
				bob,
				pairBtcUsdc,
				types.Direction_LONG,
				sdk.NewInt(1_000),
				sdk.NewDec(10),
				sdk.ZeroDec(),
			),
			QueryPosition(pairBtcUsdc, alice, QueryPosition_MarginRatioEquals(sdk.MustNewDecFromStr("-0.090096188479034643"))),
		).When(
			// Alice opened a short position (leverage x10) while bob a bigger long position
			// Price jumped by 10%, with a settlement price of 1.09
			// That creates a bad debt for alice
			// Her Realized Pnl is -101.01010101 and her margin is 100, so -1.01010101 is bad debt
			// Bob's Realized Pnl is 1010, so he has 1010 more than his margin

			CloseMarket(pairBtcUsdc, adminUser),
			SettlePosition(
				pairBtcUsdc,
				1,
				alice,
				SettlePositionChecker_PositionEquals(
					types.Position{
						TraderAddress:                   alice.String(),
						Pair:                            "ubtc:unusd",
						Size_:                           sdk.MustNewDecFromStr("0"),
						Margin:                          sdk.MustNewDecFromStr("0"),
						OpenNotional:                    sdk.MustNewDecFromStr("0"),
						LatestCumulativePremiumFraction: sdk.MustNewDecFromStr("0"),
						LastUpdatedBlockNumber:          1,
					},
				),
				SettlePositionChecker_MarginToVault(sdk.MustNewDecFromStr("-1.009493031710765502")),
				SettlePositionChecker_BadDebt(sdk.ZeroDec()),
			),
			SettlePosition(
				pairBtcUsdc,
				1,
				bob,
				SettlePositionChecker_MarginToVault(sdk.MustNewDecFromStr("-1076.990506968289234498")),
				SettlePositionChecker_BadDebt(sdk.ZeroDec()),
			),
		).Then(
			PositionShouldNotExist(alice, pairBtcUsdc, 1),
			PositionShouldNotExist(bob, pairBtcUsdc, 1),
			SetBlockNumber(2),
			BalanceEqual(alice, types.TestingCollateralDenomNUSD, sdk.NewInt(5)),
			BalanceEqual(bob, types.TestingCollateralDenomNUSD, sdk.NewInt(1095)),
		),

		TC("Error: can't settle on enabled market").When(
			CreateCustomMarket(
				pairBtcUsdc,
				WithPricePeg(sdk.OneDec()),
				WithSqrtDepth(sdk.NewDec(100_000)),
				WithEnabled(true),
			),
			SetBlockNumber(1),
			SetBlockTime(startTime),
			FundAccount(alice, sdk.NewCoins(sdk.NewCoin(types.TestingCollateralDenomNUSD, sdk.NewInt(10_200)))),
			MarketOrder(
				alice,
				pairBtcUsdc,
				types.Direction_LONG,
				sdk.NewInt(10_000),
				sdk.OneDec(),
				sdk.ZeroDec(),
			),
		).When(
			SettlePositionShouldFail(pairBtcUsdc, 1, alice),
		),

		TC("Error: can't settle on enabled market (with a live market in another version)").When(
			CreateCustomMarket(
				pairBtcUsdc,
				WithPricePeg(sdk.OneDec()),
				WithSqrtDepth(sdk.NewDec(100_000)),
				WithVersion(1),
				WithEnabled(true),
			),
			SetBlockNumber(1),
			SetBlockTime(startTime),
			FundAccount(alice, sdk.NewCoins(sdk.NewCoin(types.TestingCollateralDenomNUSD, sdk.NewInt(20_400)))),
			MarketOrder(
				alice,
				pairBtcUsdc,
				types.Direction_LONG,
				sdk.NewInt(10_000),
				sdk.OneDec(),
				sdk.ZeroDec(),
			),
		).When(
			CloseMarket(pairBtcUsdc, adminUser),
			CreateCustomMarket(
				pairBtcUsdc,
				WithPricePeg(sdk.OneDec()),
				WithSqrtDepth(sdk.NewDec(100_000)),
				WithVersion(2),
				WithEnabled(true),
			),
			SetBlockNumber(2),
			MarketOrder(
				alice,
				pairBtcUsdc,
				types.Direction_LONG,
				sdk.NewInt(10_000),
				sdk.OneDec(),
				sdk.ZeroDec(),
			),
		).Then(
			SettlePositionShouldFail(pairBtcUsdc, 3, alice), // can't settle on non existing market
			SettlePositionShouldFail(pairBtcUsdc, 2, alice), // can't settle on live market
			SettlePosition(pairBtcUsdc, 1, alice),

			PositionShouldNotExist(alice, pairBtcUsdc, 1),
			PositionShouldExist(alice, pairBtcUsdc, 2),
		),

		TC("Error: can't settle on non existing market").When(
			CreateCustomMarket(
				pairBtcUsdc,
				WithPricePeg(sdk.OneDec()),
				WithSqrtDepth(sdk.NewDec(100_000)),
				WithVersion(1),
				WithEnabled(true),
			),
			SetBlockNumber(1),
			SetBlockTime(startTime),
			FundAccount(alice, sdk.NewCoins(sdk.NewCoin(types.TestingCollateralDenomNUSD, sdk.NewInt(20_400)))),
			MarketOrder(
				alice,
				pairBtcUsdc,
				types.Direction_LONG,
				sdk.NewInt(10_000),
				sdk.OneDec(),
				sdk.ZeroDec(),
			),
		).When(
			CloseMarket(pairBtcUsdc, adminUser),
			CreateCustomMarket(
				pairBtcUsdc,
				WithPricePeg(sdk.OneDec()),
				WithSqrtDepth(sdk.NewDec(100_000)),
				WithVersion(2),
				WithEnabled(true),
			),
			SetBlockNumber(2),
			MarketOrder(
				alice,
				pairBtcUsdc,
				types.Direction_LONG,
				sdk.NewInt(10_000),
				sdk.OneDec(),
				sdk.ZeroDec(),
			),
		).Then(

			SettlePositionShouldFail(pairBtcUsdc, 2, alice),
			SettlePosition(pairBtcUsdc, 1, alice),

			PositionShouldNotExist(alice, pairBtcUsdc, 1),
			PositionShouldExist(alice, pairBtcUsdc, 2),
		),
	}

	NewTestSuite(t).WithTestCases(tc...).Run()
}
