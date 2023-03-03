package integration_test

import (
	"testing"

	"github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/common/testutil"
	. "github.com/NibiruChain/nibiru/x/perp/integration/action"
	. "github.com/NibiruChain/nibiru/x/testutil"
	. "github.com/NibiruChain/nibiru/x/testutil/action"
)

func TestHappyPath(t *testing.T) {
	ts := NewTestSuite(t)
	alice, bob := testutil.AccAddress(), testutil.AccAddress()

	tc := TestCases{

		TC("happy path").
			Given(
				CreateBaseVpool(),
				FundAccount(alice, types.NewCoins(types.NewCoin("uusdc", types.NewInt(1000)))),
				FundAccount(bob, types.NewCoins(types.NewCoin("uusdc", types.NewInt(1000)))),
			).
			Then(
				BalanceEqual(alice, types.NewCoins(types.NewCoin("uusdc", types.NewInt(1000)))),
			),

		TC("other test"),
	}

	ts.WithTestCases(tc...).Run()
}
