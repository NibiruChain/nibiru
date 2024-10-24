package statedb_test

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/core/vm"

	"github.com/NibiruChain/nibiru/v2/x/common/testutil/testapp"
	"github.com/NibiruChain/nibiru/v2/x/evm"
	"github.com/NibiruChain/nibiru/v2/x/evm/embeds"
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
	assertionsBeforeRun := func(deps *evmtest.TestDeps) {
		precompiletest.AssertWasmCounterState(
			&s.Suite, *deps, wasmContract, 0,
		)
	}
	run := func(deps *evmtest.TestDeps) *vm.EVM {
		return test.IncrementWasmCounterWithExecuteMulti(
			&s.Suite, deps, wasmContract, 7,
		)
	}
	assertionsAfterRun := func(deps *evmtest.TestDeps) {
		precompiletest.AssertWasmCounterState(
			&s.Suite, *deps, wasmContract, 7,
		)
	}

	s.T().Log("Assert before transition")

	assertionsBeforeRun(&deps)

	s.T().Log("Populate dirty journal entries")

	deployArgs := []any{"name", "SYMBOL", uint8(18)}
	deployResp, err := evmtest.DeployContract(
		&deps,
		embeds.SmartContract_ERC20Minter,
		deployArgs...,
	)
	s.Require().NoError(err, deployResp)

	deps.EvmKeeper.ERC20().Mint()
	_, _, err := deps.EvmKeeper.ERC20().Mint(
		contract, deps.Sender.EthAddr, evm.EVM_MODULE_ADDRESS,
		big.NewInt(69_420), deps.Ctx,
	)

	evmtest.TransferWei()

	s.T().Log("Run state transition")

	evmObj := run(&deps)
	stateDB, ok := evmObj.StateDB.(*statedb.StateDB)
	s.Require().True(ok, "error retrieving StateDB from the EVM")
	s.Equal(0, stateDB.Journal.DirtiesLen())

	ctxBefore, _ := deps.Ctx.CacheContext()
	assertionsAfterRun(&deps)
	err := stateDB.Commit()
	s.NoError(err)
	assertionsAfterRun(&deps)

	s.Equal(0, stateDB.Journal.DirtiesLen())

	s.Require().EqualValues(ctxBefore, deps.Ctx,
		"StateDB should have been committed by the precompile",
	)
}
