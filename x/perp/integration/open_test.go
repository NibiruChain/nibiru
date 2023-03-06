package integration_test

import (
	"testing"

	"github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/common/denoms"
	"github.com/NibiruChain/nibiru/x/common/testutil"
	. "github.com/NibiruChain/nibiru/x/oracle/integration_test/action"
	. "github.com/NibiruChain/nibiru/x/perp/integration/action"
	types2 "github.com/NibiruChain/nibiru/x/perp/types"
	. "github.com/NibiruChain/nibiru/x/testutil"
	. "github.com/NibiruChain/nibiru/x/testutil/action"
)

func TestHappyPath(t *testing.T) {
	ts := NewTestSuite(t)
	alice := testutil.AccAddress()

	pairBtcUsdc := asset.Registry.Pair(denoms.BTC, denoms.USDC)

	tc := TestCases{

		TC("happy path").
			Given(
				CreateBaseVpool(),
				SetPairPrice(pairBtcUsdc, types.MustNewDecFromStr("2.1")),
				FundAccount(alice, types.NewCoins(types.NewCoin(denoms.USDC, types.NewInt(5000e6)))),
				OpenPosition(alice, pairBtcUsdc, types2.Side_BUY, types.NewInt(600e6), types.NewDec(10), types.NewDec(0)),
			).
			Then(
				BalanceShouldBeEqual(alice, types.NewCoins(types.NewCoin("uusdc", types.NewInt(1000)))),
			),

		TC("other test").
			Given(
				CreateBaseVpool(),
			).
			Then(),
	}

	ts.WithTestCases(tc...).Run()
}
