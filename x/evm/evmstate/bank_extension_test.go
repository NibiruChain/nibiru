package evmstate_test

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	staking "github.com/cosmos/cosmos-sdk/x/staking/types"

	"github.com/NibiruChain/nibiru/v2/x/evm"
	"github.com/NibiruChain/nibiru/v2/x/evm/evmtest"
	"github.com/NibiruChain/nibiru/v2/x/nutil/testutil/testapp"
)

type FunctionalGasConsumedInvariantScenario struct {
	Setup   func(deps *evmtest.TestDeps)
	Measure func(deps *evmtest.TestDeps)
}

func (f FunctionalGasConsumedInvariantScenario) Run(s *Suite) {
	var (
		gasConsumedA uint64 // nil StateDB
		gasConsumedB uint64 // not nil StateDB
	)

	{
		deps := evmtest.NewTestDeps()

		f.Setup(&deps)

		gasConsumedBefore := deps.Ctx().GasMeter().GasConsumed()
		f.Measure(&deps)
		gasConsumedAfter := deps.Ctx().GasMeter().GasConsumed()
		gasConsumedA = gasConsumedAfter - gasConsumedBefore
	}

	{
		deps := evmtest.NewTestDeps()
		deps.NewStateDB()

		f.Setup(&deps)

		gasConsumedBefore := deps.Ctx().GasMeter().GasConsumed()
		f.Measure(&deps)
		gasConsumedAfter := deps.Ctx().GasMeter().GasConsumed()
		gasConsumedB = gasConsumedAfter - gasConsumedBefore
	}

	s.Equalf(
		fmt.Sprintf("%d", gasConsumedA),
		fmt.Sprintf("%d", gasConsumedB),
		"Gas consumed should be equal",
	)
}

// See [Suite.TestGasConsumedInvariantSend].
func (s *Suite) TestGasConsumedInvariantOther() {
	to := evmtest.NewEthPrivAcc() // arbitrary constant
	bankDenom := evm.EVMBankDenom
	coins := sdk.NewCoins(sdk.NewInt64Coin(bankDenom, 420)) // arbitrary constant
	// Use this value because the gas token is involved in both the BaseOp and
	// AfterOp of the "ForceGasInvariant" function.
	s.Run("MintCoins", func() {
		FunctionalGasConsumedInvariantScenario{
			Setup: func(deps *evmtest.TestDeps) {
				s.NoError(
					testapp.FundAccount(deps.App.BankKeeper, deps.Ctx(), deps.Sender.NibiruAddr, coins),
				)
			},
			Measure: func(deps *evmtest.TestDeps) {
				s.NoError(
					deps.App.BankKeeper.MintCoins(
						deps.Ctx(), evm.ModuleName, coins,
					),
				)
			},
		}.Run(s)
	})

	s.Run("BurnCoins", func() {
		FunctionalGasConsumedInvariantScenario{
			Setup: func(deps *evmtest.TestDeps) {
				s.NoError(
					testapp.FundModuleAccount(deps.App.BankKeeper, deps.Ctx(), evm.ModuleName, coins),
				)
			},
			Measure: func(deps *evmtest.TestDeps) {
				s.NoError(
					deps.App.BankKeeper.BurnCoins(
						deps.Ctx(), evm.ModuleName, coins,
					),
				)
			},
		}.Run(s)
	})

	s.Run("SendCoinsFromAccountToModule", func() {
		FunctionalGasConsumedInvariantScenario{
			Setup: func(deps *evmtest.TestDeps) {
				s.NoError(
					testapp.FundAccount(deps.App.BankKeeper, deps.Ctx(), deps.Sender.NibiruAddr, coins),
				)
			},
			Measure: func(deps *evmtest.TestDeps) {
				s.NoError(
					deps.App.BankKeeper.SendCoinsFromAccountToModule(
						deps.Ctx(), deps.Sender.NibiruAddr, evm.ModuleName, coins,
					),
				)
			},
		}.Run(s)
	})

	s.Run("SendCoinsFromModuleToAccount", func() {
		FunctionalGasConsumedInvariantScenario{
			Setup: func(deps *evmtest.TestDeps) {
				s.NoError(
					testapp.FundModuleAccount(deps.App.BankKeeper, deps.Ctx(), evm.ModuleName, coins),
				)
			},
			Measure: func(deps *evmtest.TestDeps) {
				s.NoError(
					deps.App.BankKeeper.SendCoinsFromModuleToAccount(
						deps.Ctx(), evm.ModuleName, to.NibiruAddr, coins,
					),
				)
			},
		}.Run(s)
	})

	s.Run("SendCoinsFromModuleToModule", func() {
		FunctionalGasConsumedInvariantScenario{
			Setup: func(deps *evmtest.TestDeps) {
				s.NoError(
					testapp.FundModuleAccount(deps.App.BankKeeper, deps.Ctx(), evm.ModuleName, coins),
				)
			},
			Measure: func(deps *evmtest.TestDeps) {
				s.NoError(
					deps.App.BankKeeper.SendCoinsFromModuleToModule(
						deps.Ctx(), evm.ModuleName, staking.NotBondedPoolName, coins,
					),
				)
			},
		}.Run(s)
	})
}
