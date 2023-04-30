package keeper_test

import (
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/common/denoms"
	"github.com/NibiruChain/nibiru/x/common/testutil"
	. "github.com/NibiruChain/nibiru/x/common/testutil/action"
	"github.com/NibiruChain/nibiru/x/common/testutil/assertion"
	. "github.com/NibiruChain/nibiru/x/oracle/integration_test/action"
	. "github.com/NibiruChain/nibiru/x/perp/integration/action/v2"
	v2types "github.com/NibiruChain/nibiru/x/perp/types/v2"
)

func TestOpenGasConsumed(t *testing.T) {
	ts := NewTestSuite(t)

	alice := testutil.AccAddress()
	PairBtcUsdc := asset.Registry.Pair(denoms.BTC, denoms.USDC)

	testCases := TestCases{
		TC("open position gas consumed").
			Given(
				createInitMarket(PairBtcUsdc),
				SetBlockTime(time.Now()),
				SetBlockNumber(1),
				SetPairPrice(PairBtcUsdc, sdk.NewDec(10000)),
				FundAccount(alice, sdk.NewCoins(sdk.NewCoin(denoms.USDC, sdk.NewInt(1020)))),
			).
			When(
				OpenPosition(
					alice, PairBtcUsdc, v2types.Direction_LONG,
					sdk.NewInt(1000), sdk.NewDec(10), sdk.ZeroDec(),
				),
			).Then(
			assertion.GasConsumedShouldBe(153539),
		),
	}

	ts.WithTestCases(testCases...).Run()
}
