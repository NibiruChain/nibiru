package statedb_test

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/core/vm"

	"github.com/NibiruChain/nibiru/v2/x/common/testutil/testapp"
	"github.com/NibiruChain/nibiru/v2/x/evm"
	"github.com/NibiruChain/nibiru/v2/x/evm/evmtest"
	"github.com/NibiruChain/nibiru/v2/x/evm/precompile/test"
	precompiletest "github.com/NibiruChain/nibiru/v2/x/evm/precompile/test"
	"github.com/NibiruChain/nibiru/v2/x/evm/statedb"
)

func (s *Suite) TestPrecompileSnapshots() {
	deps := evmtest.NewTestDeps()
	bankDenom := evm.EVMBankDenom
	s.Require().NoError(testapp.FundAccount(
		deps.App.BankKeeper,
		deps.Ctx,
		deps.Sender.NibiruAddr,
		sdk.NewCoins(sdk.NewCoin(bankDenom, sdk.NewInt(69_420))),
	))

	s.T().Log("Set up helloworldcounter.wasm")
	wasmContract := precompiletest.SetupWasmContracts(&deps, &s.Suite)[1]
	type Transition struct {
		Run                 func(deps *evmtest.TestDeps) *vm.EVM
		AssertionsBeforeRun func(deps *evmtest.TestDeps)
	}
	fmt.Printf("wasmContract: %v\n", wasmContract)
	stateTransitions := []Transition{
		{
			AssertionsBeforeRun: func(deps *evmtest.TestDeps) {
				precompiletest.AssertWasmCounterState(
					&s.Suite, *deps, wasmContract, 0,
				)
			},
			Run: func(deps *evmtest.TestDeps) *vm.EVM {
				return test.IncrementWasmCounterWithExecuteMulti(
					&s.Suite, deps, wasmContract, 7,
				)
			},
		},
		{
			AssertionsBeforeRun: func(deps *evmtest.TestDeps) {
				precompiletest.AssertWasmCounterState(
					&s.Suite, *deps, wasmContract, 7,
				)
			},
			Run: func(deps *evmtest.TestDeps) *vm.EVM {
				return test.IncrementWasmCounterWithExecuteMulti(
					&s.Suite, deps, wasmContract, 5,
				)
			},
		},
		{
			AssertionsBeforeRun: func(deps *evmtest.TestDeps) {
				precompiletest.AssertWasmCounterState(
					&s.Suite, *deps, wasmContract, 12,
				)
			},
		},
	}

	s.T().Log("Assert before transition")

	transitionIdx := 0
	st := stateTransitions[transitionIdx]
	st.AssertionsBeforeRun(&deps)

	s.T().Log("Run state transition")

	evmObj := st.Run(&deps)
	stateDB, ok := evmObj.StateDB.(*statedb.StateDB)
	s.Require().True(ok, "error retrieving StateDB from the EVM")

	entries := stateDB.Journal.EntriesCopy()
	s.Require().Len(entries, 13, "expect 13 journal entries")
	s.Equal(0, stateDB.Journal.DirtiesLen())

	assertionsAfter := stateTransitions[transitionIdx+1].AssertionsBeforeRun
	assertionsAfter(&deps)

	// st.AssertionsBeforeRun(&deps)
}
