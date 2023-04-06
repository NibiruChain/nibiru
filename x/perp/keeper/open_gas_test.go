package keeper_test

import (
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/common/denoms"
	testutilevents "github.com/NibiruChain/nibiru/x/common/testutil"
	. "github.com/NibiruChain/nibiru/x/common/testutil/action"
	"github.com/NibiruChain/nibiru/x/common/testutil/assertion"
	. "github.com/NibiruChain/nibiru/x/oracle/integration_test/action"
	. "github.com/NibiruChain/nibiru/x/perp/integration/action"
	perptypes "github.com/NibiruChain/nibiru/x/perp/types"
)

func TestOpenGasConsumed(t *testing.T) {
	ts := NewTestSuite(t)

	alice := testutilevents.AccAddress()
	pairBtcUsdc := asset.Registry.Pair(denoms.BTC, denoms.USDC)

	testCases := TestCases{
		TC("open position gas consumed").
			Given(
				createInitVPool(),
				SetBlockTime(time.Now()),
				SetBlockNumber(1),
				SetPairPrice(pairBtcUsdc, sdk.NewDec(10000)),
				FundAccount(alice, sdk.NewCoins(sdk.NewCoin(denoms.USDC, sdk.NewInt(1020)))),
			).
			When(
				OpenPosition(alice, pairBtcUsdc, perptypes.Side_BUY, sdk.NewInt(1000), sdk.NewDec(10), sdk.ZeroDec()),
			).Then(
			assertion.GasConsumedShouldBe(152645),
		),
	}

	ts.WithTestCases(testCases...).Run()
}
