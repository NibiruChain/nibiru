package statedb_test

import (
	"fmt"
	"math/big"
	"strings"
	"testing"

	"github.com/MakeNowJust/heredoc/v2"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/holiman/uint256"

	serverconfig "github.com/NibiruChain/nibiru/v2/app/server/config"
	"github.com/NibiruChain/nibiru/v2/x/common"
	"github.com/NibiruChain/nibiru/v2/x/common/testutil/testapp"
	"github.com/NibiruChain/nibiru/v2/x/evm"
	"github.com/NibiruChain/nibiru/v2/x/evm/embeds"
	"github.com/NibiruChain/nibiru/v2/x/evm/evmtest"
	"github.com/NibiruChain/nibiru/v2/x/evm/keeper"
	"github.com/NibiruChain/nibiru/v2/x/evm/precompile/test"
	"github.com/NibiruChain/nibiru/v2/x/evm/statedb"
)

func (s *Suite) TestCommitRemovesDirties() {
	deps := evmtest.NewTestDeps()
	evmObj, _ := deps.NewEVM()

	deployResp, err := evmtest.DeployContract(
		&deps,
		embeds.SmartContract_ERC20MinterWithMetadataUpdates,
		"name",
		"SYMBOL",
		uint8(18),
	)
	s.Require().NoError(err, deployResp)
	erc20 := deployResp.ContractAddr

	input, err := deps.EvmKeeper.ERC20().ABI.Pack("mint", deps.Sender.EthAddr, big.NewInt(69_420))
	s.Require().NoError(err)
	_, err = deps.EvmKeeper.CallContractWithInput(
		deps.Ctx,
		evmObj,
		deps.Sender.EthAddr, // caller
		&erc20,              // contract
		true,                // commit
		input,
		keeper.Erc20GasLimitExecute,
		nil,
	)
	s.Require().NoError(err)
	s.Require().EqualValues(0, evmObj.StateDB.(*statedb.StateDB).DebugDirtiesCount())
}

func (s *Suite) TestCommitRemovesDirties_OnlyStateDB() {
	deps := evmtest.NewTestDeps()
	evmObj, _ := deps.NewEVM()
	stateDB := evmObj.StateDB.(*statedb.StateDB)

	randomAcc := evmtest.NewEthPrivAcc().EthAddr
	balDelta := evm.NativeToWei(big.NewInt(4))
	// 2 dirties from [createObjectChange, balanceChange]
	stateDB.AddBalanceSigned(randomAcc, balDelta)
	// 1 dirties from [balanceChange]
	stateDB.AddBalanceSigned(randomAcc, balDelta)
	// 1 dirties from [balanceChange]
	stateDB.AddBalanceSigned(randomAcc, balDelta)
	if stateDB.DebugDirtiesCount() != 4 {
		debugDirtiesCountMismatch(stateDB, s.T())
		s.FailNow("expected 4 dirty journal changes")
	}

	s.T().Log("StateDB.Commit, then Dirties should be gone")
	err := stateDB.Commit()
	s.NoError(err)
	if stateDB.DebugDirtiesCount() != 0 {
		debugDirtiesCountMismatch(stateDB, s.T())
		s.FailNow("expected 0 dirty journal changes")
	}
}

func (s *Suite) TestContractCallsAnotherContract() {
	deps := evmtest.NewTestDeps()
	evmObj, _ := deps.NewEVM()
	stateDB := evmObj.StateDB.(*statedb.StateDB)

	s.Require().NoError(testapp.FundAccount(
		deps.App.BankKeeper,
		deps.Ctx,
		deps.Sender.NibiruAddr,
		sdk.NewCoins(sdk.NewCoin(evm.EVMBankDenom, sdk.NewInt(69_420))),
	))

	deployResp, err := evmtest.DeployContract(
		&deps,
		embeds.SmartContract_ERC20MinterWithMetadataUpdates,
		"name",
		"SYMBOL",
		uint8(18),
	)
	s.Require().NoError(err, deployResp)
	erc20 := deployResp.ContractAddr

	s.Run("Mint 69_420 tokens", func() {
		contractInput, err := deps.EvmKeeper.ERC20().ABI.Pack("mint", deps.Sender.EthAddr, big.NewInt(69_420))
		s.Require().NoError(err)
		_, err = deps.EvmKeeper.CallContractWithInput(
			deps.Ctx,
			evmObj,
			deps.Sender.EthAddr, // caller
			&erc20,              // contract
			true,                // commit
			contractInput,
			keeper.Erc20GasLimitExecute,
			nil,
		)
		s.Require().NoError(err)
	})

	randomAcc := evmtest.NewEthPrivAcc().EthAddr
	contractInput, err := embeds.SmartContract_ERC20MinterWithMetadataUpdates.ABI.Pack("transfer", randomAcc, big.NewInt(69_000))
	s.Require().NoError(err)

	s.Run("Transfer 69_000 tokens", func() {
		s.T().Log("Transfer 69_000 tokens")

		_, _, err = evmObj.Call(
			vm.AccountRef(deps.Sender.EthAddr),
			erc20,
			contractInput,
			serverconfig.DefaultEthCallGasLimit,
			uint256.NewInt(0),
		)
		s.Require().NoError(err)
		if stateDB.DebugDirtiesCount() != 2 {
			debugDirtiesCountMismatch(stateDB, s.T())
			s.FailNowf("expected 2 dirty journal changes", "%#v", stateDB.Journal)
		}
	})

	s.Run("Transfer 69_000 tokens", func() {
		// The contract calling itself is invalid in this context.
		// Note the comment in vm.Contract:
		//
		// type Contract struct {
		// 	// CallerAddress is the result of the caller which initialized this
		// 	// contract. However when the "call method" is delegated this value
		// 	// needs to be initialized to that of the caller's caller.
		// 	CallerAddress common.Address
		// 	// ...
		// 	}
		// 	//

		_, _, err = evmObj.Call(
			vm.AccountRef(erc20),
			erc20,
			contractInput,
			serverconfig.DefaultEthCallGasLimit,
			uint256.NewInt(0),
		)
		s.Require().ErrorContains(err, vm.ErrExecutionReverted.Error())
	})
}

func (s *Suite) TestJournalReversion() {
	deps := evmtest.NewTestDeps()
	s.Require().NoError(testapp.FundAccount(
		deps.App.BankKeeper,
		deps.Ctx,
		deps.Sender.NibiruAddr,
		sdk.NewCoins(sdk.NewCoin(evm.EVMBankDenom, sdk.NewInt(69_420))),
	))

	s.T().Log("Set up helloworldcounter.wasm")
	wasmContracts := test.SetupWasmContracts(&deps, &s.Suite)
	helloWorldCounterWasm := wasmContracts[1]
	fmt.Printf("wasmContract: %s\n", helloWorldCounterWasm)

	s.T().Log("commitEvmTx=true, expect 0 dirty journal entries")
	evmObj, stateDB := deps.NewEVM()
	test.IncrementWasmCounterWithExecuteMulti(
		&s.Suite, &deps, evmObj, helloWorldCounterWasm, 7, true,
	)
	if stateDB.DebugDirtiesCount() != 0 {
		debugDirtiesCountMismatch(stateDB, s.T())
		s.FailNowf("statedb dirty count mismatch", "expected 0 dirty journal changes, but instead got: %d", stateDB.DebugDirtiesCount())
	}

	s.T().Log("commitEvmTx=false, expect dirty journal entries")
	evmObj, stateDB = deps.NewEVM()
	test.IncrementWasmCounterWithExecuteMulti(
		&s.Suite, &deps, evmObj, helloWorldCounterWasm, 5, false,
	)
	s.T().Log("Expect exactly 1 dirty journal entry for the precompile snapshot")
	if stateDB.DebugDirtiesCount() != 1 {
		debugDirtiesCountMismatch(stateDB, s.T())
		s.FailNowf("statedb dirty count mismatch", "expected 1 dirty journal change, but instead got: %d", stateDB.DebugDirtiesCount())
	}

	s.T().Log("Expect to see the pending changes included in the EVM context")
	test.AssertWasmCounterStateWithEvm(
		&s.Suite, deps, evmObj, helloWorldCounterWasm, 7+5,
	)
	s.T().Log("Expect to see the pending changes not included in cosmos ctx")
	test.AssertWasmCounterState(
		&s.Suite, deps, helloWorldCounterWasm, 7,
	)

	// NOTE: that the [StateDB.Commit] fn has not been called yet. We're still
	// mid-transaction.

	s.T().Log("EVM revert operation should bring about the old state")
	test.IncrementWasmCounterWithExecuteMulti(
		&s.Suite, &deps, evmObj, helloWorldCounterWasm, 50, false,
	)
	s.T().Log(heredoc.Doc(`At this point, 2 precompile calls have succeeded.
One that increments the counter to 7 + 5, and another for +50. 
The StateDB has not been committed. We expect to be able to revert to both
snapshots and see the prior states.`))
	test.AssertWasmCounterStateWithEvm(
		&s.Suite, deps, evmObj, helloWorldCounterWasm, 7+5+50,
	)

	errFn := common.TryCatch(func() {
		// a revision that doesn't exist
		stateDB.RevertToSnapshot(9000)
	})
	s.Require().ErrorContains(errFn(), "revision id 9000 cannot be reverted")

	stateDB.RevertToSnapshot(2)
	test.AssertWasmCounterStateWithEvm(
		&s.Suite, deps, evmObj, helloWorldCounterWasm, 7+5,
	)

	stateDB.RevertToSnapshot(0)
	test.AssertWasmCounterStateWithEvm(
		&s.Suite, deps, evmObj, helloWorldCounterWasm, 7,
	)

	s.Require().NoError(stateDB.Commit())
	s.Require().EqualValues(0, stateDB.DebugDirtiesCount())
	test.AssertWasmCounterState(
		&s.Suite, deps, helloWorldCounterWasm, 7,
	)
}

func debugDirtiesCountMismatch(db *statedb.StateDB, t *testing.T) {
	lines := []string{}
	dirties := db.DebugDirties()
	stateObjects := db.DebugStateObjects()
	for addr, dirtyCount := range dirties {
		lines = append(lines, fmt.Sprintf("Dirty addr: %s, dirtyCount=%d", addr, dirtyCount))

		// Inspect the actual state object
		maybeObj := stateObjects[addr]
		if maybeObj == nil {
			lines = append(lines, "  no state object found!")
			continue
		}
		obj := *maybeObj

		lines = append(lines, fmt.Sprintf("  balance: %s", obj.Balance()))
		lines = append(lines, fmt.Sprintf("  suicided: %v", obj.SelfDestructed))
		lines = append(lines, fmt.Sprintf("  dirtyCode: %v", obj.DirtyCode))

		// Print storage state
		lines = append(lines, fmt.Sprintf("  len(obj.DirtyStorage) entries: %d", len(obj.DirtyStorage)))
		for k, v := range obj.DirtyStorage {
			lines = append(lines, fmt.Sprintf("    key: %s, value: %s", k.Hex(), v.Hex()))
			origVal := obj.OriginStorage[k]
			lines = append(lines, fmt.Sprintf("    origin value: %s", origVal.Hex()))
		}
	}

	t.Log("debugDirtiesCountMismatch:\n", strings.Join(lines, "\n"))
}
