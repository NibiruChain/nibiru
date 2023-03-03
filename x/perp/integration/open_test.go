package integration_test

import (
	testutil2 "github.com/NibiruChain/nibiru/x/common/testutil"
	. "github.com/NibiruChain/nibiru/x/perp/integration/action"
	. "github.com/NibiruChain/nibiru/x/testutil"
	. "github.com/NibiruChain/nibiru/x/testutil/action"
	"github.com/cosmos/cosmos-sdk/types"
	"testing"
)

func TestHappyPath(t *testing.T) {
	ts := NewTestSuite(t)
	alice, bob := testutil2.AccAddress(), testutil2.AccAddress()

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
