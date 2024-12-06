package keeper_test

import (
	"fmt"

	"github.com/NibiruChain/nibiru/v2/x/common/testutil/testapp"
	"github.com/NibiruChain/nibiru/v2/x/evm"
	"github.com/NibiruChain/nibiru/v2/x/evm/evmtest"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/rs/zerolog/log"
)

// TODO: UD-DEBUG:
// non-nil StaetDB bank send and record gas consumed
// nil StaetDB bank send and record gas consumed
// Both should be equal.

// TestGasConsumedInvariant: The "NibiruBankKeeper" is defined such that
// send operations are meant to have consistent gas consumption regardless of the
// whether the active "StateDB" instance is defined or undefined (nil). This is
// important to prevent consensus failures resulting from nodes reaching an
// inconsistent state after processing the same state transitions and gettng
// conflicting gas results.
func (s *Suite) TestGasConsumedInvariant() {
	to := evmtest.NewEthPrivAcc() // arbitrary constant

	testCases := []struct {
		GasConsumedInvariantScenario
		name string
	}{
		{
			name: "StateDB nil",
			GasConsumedInvariantScenario: GasConsumedInvariantScenario{
				BankDenom:  evm.EVMBankDenom,
				NilStateDB: true,
			},
		},

		{
			name: "StateDB defined",
			GasConsumedInvariantScenario: GasConsumedInvariantScenario{
				BankDenom:  evm.EVMBankDenom,
				NilStateDB: false,
			},
		},

		{
			name: "StateDB nil, uniba",
			GasConsumedInvariantScenario: GasConsumedInvariantScenario{
				BankDenom:  "uniba",
				NilStateDB: true,
			},
		},
	}

	s.T().Log("Check that gas consumption is equal in all scenarios")
	var first uint64
	for idx, tc := range testCases {
		s.Run(tc.name, func() {
			gasConsumed := tc.GasConsumedInvariantScenario.Run(s, to)
			if idx == 0 {
				first = gasConsumed
				return
			}
			// Each elem being equal to "first" implies that each elem is equal
			s.Equalf(
				fmt.Sprintf("%d", first),
				fmt.Sprintf("%d", gasConsumed),
				"Gas consumed should be equal",
			)
		})
	}

}

func (s *Suite) populateStateDB(deps *evmtest.TestDeps) {
	// evmtest.DeployAndExecuteERC20Transfer(deps, s.T())
	deps.NewStateDB()
	s.NotNil(deps.EvmKeeper.Bank.StateDB)
}

type GasConsumedInvariantScenario struct {
	BankDenom  string
	NilStateDB bool
}

func (scenario GasConsumedInvariantScenario) Run(
	s *Suite,
	to evmtest.EthPrivKeyAcc,
) (gasConsumed uint64) {
	bankDenom, nilStateDB := scenario.BankDenom, scenario.NilStateDB
	deps := evmtest.NewTestDeps()
	if nilStateDB {
		s.Require().Nil(deps.EvmKeeper.Bank.StateDB)
	} else {
		s.populateStateDB(&deps)
	}

	sendCoins := sdk.NewCoins(sdk.NewInt64Coin(bankDenom, 420))
	s.NoError(
		testapp.FundAccount(deps.App.BankKeeper, deps.Ctx, deps.Sender.NibiruAddr, sendCoins),
	)

	gasConsumedBefore := deps.Ctx.GasMeter().GasConsumed()
	s.NoError(
		deps.App.BankKeeper.SendCoins(
			deps.Ctx, deps.Sender.NibiruAddr, to.NibiruAddr, sendCoins,
		),
	)
	gasConsumedAfter := deps.Ctx.GasMeter().GasConsumed()
	log.Debug().Msgf("gasConsumedBefore: %d", gasConsumedBefore)
	log.Debug().Msgf("gasConsumedAfter: %d", gasConsumedAfter)
	log.Debug().Msgf("nilStateDB: %v", nilStateDB)

	s.GreaterOrEqualf(gasConsumedAfter, gasConsumedBefore,
		"gas meter consumed should not be negative: gas consumed after = %d, gas consumed before = %d ",
		gasConsumedAfter, gasConsumedBefore,
	)

	return gasConsumedAfter - gasConsumedBefore
}

func (s *Suite) TestGasConsumedInvariantNotNibi() {
	to := evmtest.NewEthPrivAcc() // arbitrary constant

	testCases := []struct {
		GasConsumedInvariantScenario
		name string
	}{
		{
			name: "StateDB nil A",
			GasConsumedInvariantScenario: GasConsumedInvariantScenario{
				BankDenom:  "dummy_token_A",
				NilStateDB: true,
			},
		},

		{
			name: "StateDB defined A",
			GasConsumedInvariantScenario: GasConsumedInvariantScenario{
				BankDenom:  "dummy_token_A",
				NilStateDB: false,
			},
		},

		{
			name: "StateDB nil B",
			GasConsumedInvariantScenario: GasConsumedInvariantScenario{
				BankDenom:  "dummy_token_B",
				NilStateDB: true,
			},
		},

		{
			name: "StateDB defined B",
			GasConsumedInvariantScenario: GasConsumedInvariantScenario{
				BankDenom:  "dummy_token_B",
				NilStateDB: false,
			},
		},
	}

	s.T().Log("Check that gas consumption is equal in all scenarios")
	var first uint64
	for idx, tc := range testCases {
		s.Run(tc.name, func() {
			gasConsumed := tc.GasConsumedInvariantScenario.Run(s, to)
			if idx == 0 {
				first = gasConsumed
				return
			}
			// Each elem being equal to "first" implies that each elem is equal
			s.Equalf(
				fmt.Sprintf("%d", first),
				fmt.Sprintf("%d", gasConsumed),
				"Gas consumed should be equal",
			)
		})
	}

}
