package keeper_test

import (
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/common/denoms"
	"github.com/NibiruChain/nibiru/x/common/testutil"
	. "github.com/NibiruChain/nibiru/x/common/testutil/action"
	. "github.com/NibiruChain/nibiru/x/perp/v2/integration/action"
	. "github.com/NibiruChain/nibiru/x/perp/v2/integration/assertion"
	"github.com/NibiruChain/nibiru/x/perp/v2/types"
)

func TestDisableMarket(t *testing.T) {
	pairBtcUsdc := asset.Registry.Pair(denoms.BTC, denoms.NUSD)
	startTime := time.Now()
	alice := testutil.AccAddress()

	tc := TestCases{
		TC("market can be disabled").
			Given(
				CreateCustomMarket(pairBtcUsdc),
				SetBlockTime(startTime),
				MarketShouldBeEqual(
					pairBtcUsdc,
					Market_EnableShouldBeEqualTo(true),
				),
			).
			When(
				CloseMarket(pairBtcUsdc),
			).
			Then(
				MarketShouldBeEqual(
					pairBtcUsdc,
					Market_EnableShouldBeEqualTo(false),
				),
			),
		TC("cannot open position on disabled market").
			Given(
				CreateCustomMarket(
					pairBtcUsdc,
					WithPricePeg(sdk.OneDec()),
					WithSqrtDepth(sdk.NewDec(100_000)),
				),
				SetBlockNumber(1),
				SetBlockTime(startTime),

				FundAccount(alice, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(1e6)))),
			).
			When(
				CloseMarket(pairBtcUsdc),
			).
			Then(
				MarketOrderFails(
					alice,
					pairBtcUsdc,
					types.Direction_LONG,
					sdk.NewInt(10_000),
					sdk.OneDec(),
					sdk.ZeroDec(),
					types.ErrMarketNotEnabled,
				),
			),
		TC("cannot close position on disabled market").When(
			CreateCustomMarket(
				pairBtcUsdc,
				WithPricePeg(sdk.OneDec()),
				WithSqrtDepth(sdk.NewDec(100_000)),
			),
			SetBlockNumber(1),
			SetBlockTime(startTime),
			FundAccount(alice, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(10_200)))),
			MarketOrder(
				alice,
				pairBtcUsdc,
				types.Direction_LONG,
				sdk.NewInt(10_000),
				sdk.OneDec(),
				sdk.ZeroDec(),
			),
		).When(
			CloseMarket(pairBtcUsdc),
		).Then(
			ClosePositionFails(alice, pairBtcUsdc, types.ErrMarketNotEnabled),
		),
		TC("cannot partial close position on disabled market").When(
			CreateCustomMarket(
				pairBtcUsdc,
				WithPricePeg(sdk.OneDec()),
				WithSqrtDepth(sdk.NewDec(100_000)),
			),
			SetBlockNumber(1),
			SetBlockTime(startTime),
			FundAccount(alice, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(10_200)))),
			MarketOrder(
				alice,
				pairBtcUsdc,
				types.Direction_LONG,
				sdk.NewInt(10_000),
				sdk.OneDec(),
				sdk.ZeroDec(),
			),
		).When(
			CloseMarket(pairBtcUsdc),
			AMMShouldBeEqual(pairBtcUsdc, AMM_SettlementPriceShoulBeEqual(sdk.MustNewDecFromStr("1.1"))),
		).Then(
			PartialCloseFails(alice, pairBtcUsdc, sdk.NewDec(5_000), types.ErrMarketNotEnabled),
		),
	}

	NewTestSuite(t).WithTestCases(tc...).Run()
}

func TestSettlePosition(t *testing.T) {
	pairBtcUsdc := asset.Registry.Pair(denoms.BTC, denoms.NUSD)
	startTime := time.Now()

	alice := testutil.AccAddress()

	tc := TestCases{
		TC("Happy path").When(
			CreateCustomMarket(
				pairBtcUsdc,
				WithPricePeg(sdk.OneDec()),
				WithSqrtDepth(sdk.NewDec(100_000)),
			),
			SetBlockNumber(1),
			SetBlockTime(startTime),
			FundAccount(alice, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(10_200)))),
			MarketOrder(
				alice,
				pairBtcUsdc,
				types.Direction_LONG,
				sdk.NewInt(10_000),
				sdk.OneDec(),
				sdk.ZeroDec(),
			),
		).When(
			CloseMarket(pairBtcUsdc),
			SettlePosition(pairBtcUsdc, 1, alice),
		).Then(
			PositionShouldNotExist(alice, pairBtcUsdc, 1),
		),

		TC("Error: can't settle on enabled market").When(
			CreateCustomMarket(
				pairBtcUsdc,
				WithPricePeg(sdk.OneDec()),
				WithSqrtDepth(sdk.NewDec(100_000)),
			),
			SetBlockNumber(1),
			SetBlockTime(startTime),
			FundAccount(alice, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(10_200)))),
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
			),
			SetBlockNumber(1),
			SetBlockTime(startTime),
			FundAccount(alice, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(20_400)))),
			MarketOrder(
				alice,
				pairBtcUsdc,
				types.Direction_LONG,
				sdk.NewInt(10_000),
				sdk.OneDec(),
				sdk.ZeroDec(),
			),
		).When(
			CloseMarket(pairBtcUsdc),
			CreateCustomMarket(
				pairBtcUsdc,
				WithPricePeg(sdk.OneDec()),
				WithSqrtDepth(sdk.NewDec(100_000)),
				WithVersion(2),
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
			),
			SetBlockNumber(1),
			SetBlockTime(startTime),
			FundAccount(alice, sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(20_400)))),
			MarketOrder(
				alice,
				pairBtcUsdc,
				types.Direction_LONG,
				sdk.NewInt(10_000),
				sdk.OneDec(),
				sdk.ZeroDec(),
			),
		).When(
			CloseMarket(pairBtcUsdc),
			CreateCustomMarket(
				pairBtcUsdc,
				WithPricePeg(sdk.OneDec()),
				WithSqrtDepth(sdk.NewDec(100_000)),
				WithVersion(2),
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
