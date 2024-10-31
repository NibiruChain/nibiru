package statedb_test

import (
	"fmt"
	"math/big"
	"strings"
	"testing"

	"github.com/MakeNowJust/heredoc/v2"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/core/vm"

	"github.com/NibiruChain/nibiru/v2/x/evm/keeper"

	serverconfig "github.com/NibiruChain/nibiru/v2/app/server/config"
	"github.com/NibiruChain/nibiru/v2/x/common"
	"github.com/NibiruChain/nibiru/v2/x/common/testutil/testapp"
	"github.com/NibiruChain/nibiru/v2/x/evm"
	"github.com/NibiruChain/nibiru/v2/x/evm/embeds"
	"github.com/NibiruChain/nibiru/v2/x/evm/evmtest"
	"github.com/NibiruChain/nibiru/v2/x/evm/precompile/test"
	"github.com/NibiruChain/nibiru/v2/x/evm/statedb"
)

func (s *Suite) TestComplexJournalChanges() {
	deps := evmtest.NewTestDeps()
	bankDenom := evm.EVMBankDenom
	s.Require().NoError(testapp.FundAccount(
		deps.App.BankKeeper,
		deps.Ctx,
		deps.Sender.NibiruAddr,
		sdk.NewCoins(sdk.NewCoin(bankDenom, sdk.NewInt(69_420))),
	))

	s.T().Log("Set up helloworldcounter.wasm")
	helloWorldCounterWasm := test.SetupWasmContracts(&deps, &s.Suite)[1]
	fmt.Printf("wasmContract: %s\n", helloWorldCounterWasm)

	s.T().Log("Assert before transition")
	test.AssertWasmCounterState(
		&s.Suite, deps, helloWorldCounterWasm, 0,
	)

	deployArgs := []any{"name", "SYMBOL", uint8(18)}
	deployResp, err := evmtest.DeployContract(
		&deps,
		embeds.SmartContract_ERC20Minter,
		deployArgs...,
	)
	s.Require().NoError(err, deployResp)

	contract := deployResp.ContractAddr
	to, amount := deps.Sender.EthAddr, big.NewInt(69_420)
	input, err := deps.EvmKeeper.ERC20().ABI.Pack("mint", to, amount)
	s.Require().NoError(err)
	_, evmObj, err := deps.EvmKeeper.CallContractWithInput(
		deps.Ctx,
		deps.Sender.EthAddr,
		&contract,
		true,
		input,
		keeper.Erc20GasLimitExecute,
	)
	s.Require().NoError(err)

	s.Run("Populate dirty journal entries. Remove with Commit", func() {
		stateDB := evmObj.StateDB.(*statedb.StateDB)
		s.Equal(0, stateDB.DebugDirtiesCount())

		randomAcc := evmtest.NewEthPrivAcc().EthAddr
		balDelta := evm.NativeToWei(big.NewInt(4))
		// 2 dirties from [createObjectChange, balanceChange]
		stateDB.AddBalance(randomAcc, balDelta)
		// 1 dirties from [balanceChange]
		stateDB.AddBalance(randomAcc, balDelta)
		// 1 dirties from [balanceChange]
		stateDB.SubBalance(randomAcc, balDelta)
		if stateDB.DebugDirtiesCount() != 4 {
			debugDirtiesCountMismatch(stateDB, s.T())
			s.FailNow("expected 4 dirty journal changes")
		}

		s.T().Log("StateDB.Commit, then Dirties should be gone")
		err = stateDB.Commit()
		s.NoError(err)
		if stateDB.DebugDirtiesCount() != 0 {
			debugDirtiesCountMismatch(stateDB, s.T())
			s.FailNow("expected 0 dirty journal changes")
		}
	})

	s.Run("Emulate a contract that calls another contract", func() {
		randomAcc := evmtest.NewEthPrivAcc().EthAddr
		to, amount := randomAcc, big.NewInt(69_000)
		input, err := embeds.SmartContract_ERC20Minter.ABI.Pack("transfer", to, amount)
		s.Require().NoError(err)

		leftoverGas := serverconfig.DefaultEthCallGasLimit
		_, _, err = evmObj.Call(
			vm.AccountRef(deps.Sender.EthAddr),
			contract,
			input,
			leftoverGas,
			big.NewInt(0),
		)
		s.Require().NoError(err)
		stateDB := evmObj.StateDB.(*statedb.StateDB)
		if stateDB.DebugDirtiesCount() != 2 {
			debugDirtiesCountMismatch(stateDB, s.T())
			s.FailNow("expected 2 dirty journal changes")
		}

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
			vm.AccountRef(contract),
			contract,
			input,
			leftoverGas,
			big.NewInt(0),
		)
		s.Require().ErrorContains(err, vm.ErrExecutionReverted.Error())
	})

	s.Run("Precompile calls populate snapshots", func() {
		s.T().Log("commitEvmTx=true, expect 0 dirty journal entries")
		commitEvmTx := true
		evmObj = test.IncrementWasmCounterWithExecuteMulti(
			&s.Suite, &deps, helloWorldCounterWasm, 7, commitEvmTx,
		)
		// assertions after run
		test.AssertWasmCounterState(
			&s.Suite, deps, helloWorldCounterWasm, 7,
		)
		stateDB, ok := evmObj.StateDB.(*statedb.StateDB)
		s.Require().True(ok, "error retrieving StateDB from the EVM")
		if stateDB.DebugDirtiesCount() != 0 {
			debugDirtiesCountMismatch(stateDB, s.T())
			s.FailNow("expected 0 dirty journal changes")
		}

		s.T().Log("commitEvmTx=false, expect dirty journal entries")
		commitEvmTx = false
		evmObj = test.IncrementWasmCounterWithExecuteMulti(
			&s.Suite, &deps, helloWorldCounterWasm, 5, commitEvmTx,
		)
		stateDB, ok = evmObj.StateDB.(*statedb.StateDB)
		s.Require().True(ok, "error retrieving StateDB from the EVM")

		s.T().Log("Expect exactly 0 dirty journal entry for the precompile snapshot")
		if stateDB.DebugDirtiesCount() != 0 {
			debugDirtiesCountMismatch(stateDB, s.T())
			s.FailNow("expected 0 dirty journal changes")
		}

		s.T().Log("Expect no change since the StateDB has not been committed")
		test.AssertWasmCounterState(
			&s.Suite, deps, helloWorldCounterWasm, 7, // 7 = 7 + 0
		)

		s.T().Log("Expect change to persist on the StateDB cacheCtx")
		cacheCtx := stateDB.GetCacheContext()
		s.NotNil(cacheCtx)
		deps.Ctx = *cacheCtx
		test.AssertWasmCounterState(
			&s.Suite, deps, helloWorldCounterWasm, 12, // 12 = 7 + 5
		)
		// NOTE: that the [StateDB.Commit] fn has not been called yet. We're still
		// mid-transaction.

		s.T().Log("EVM revert operation should bring about the old state")
		err = test.IncrementWasmCounterWithExecuteMultiViaVMCall(
			&s.Suite, &deps, helloWorldCounterWasm, 50, commitEvmTx, evmObj,
		)
		stateDBPtr := evmObj.StateDB.(*statedb.StateDB)
		s.Require().Equal(stateDB, stateDBPtr)
		s.Require().NoError(err)
		s.T().Log(heredoc.Doc(`At this point, 2 precompile calls have succeeded.
One that increments the counter to 7 + 5, and another for +50. 
The StateDB has not been committed. We expect to be able to revert to both
snapshots and see the prior states.`))
		cacheCtx = stateDB.GetCacheContext()
		deps.Ctx = *cacheCtx
		test.AssertWasmCounterState(
			&s.Suite, deps, helloWorldCounterWasm, 7+5+50,
		)

		errFn := common.TryCatch(func() {
			// There were only two EVM calls.
			// Thus, there are only 2 snapshots: 0 and 1.
			// We should not be able to revert to a third one.
			stateDB.RevertToSnapshot(2)
		})
		s.Require().ErrorContains(errFn(), "revision id 2 cannot be reverted")

		stateDB.RevertToSnapshot(1)
		cacheCtx = stateDB.GetCacheContext()
		s.NotNil(cacheCtx)
		deps.Ctx = *cacheCtx
		test.AssertWasmCounterState(
			&s.Suite, deps, helloWorldCounterWasm, 7+5,
		)

		stateDB.RevertToSnapshot(0)
		cacheCtx = stateDB.GetCacheContext()
		s.NotNil(cacheCtx)
		deps.Ctx = *cacheCtx
		test.AssertWasmCounterState(
			&s.Suite, deps, helloWorldCounterWasm, 7, // state before precompile called
		)

		err = stateDB.Commit()
		deps.Ctx = stateDB.GetEvmTxContext()
		test.AssertWasmCounterState(
			&s.Suite, deps, helloWorldCounterWasm, 7, // state before precompile called
		)
	})
}

func debugDirtiesCountMismatch(db *statedb.StateDB, t *testing.T) string {
	lines := []string{}
	dirties := db.DebugDirties()
	stateObjects := db.DebugStateObjects()
	for addr, dirtyCountForAddr := range dirties {
		lines = append(lines, fmt.Sprintf("Dirty addr: %s, dirtyCountForAddr=%d", addr, dirtyCountForAddr))

		// Inspect the actual state object
		maybeObj := stateObjects[addr]
		if maybeObj == nil {
			lines = append(lines, "  no state object found!")
			continue
		}
		obj := *maybeObj

		lines = append(lines, fmt.Sprintf("  balance: %s", obj.Balance()))
		lines = append(lines, fmt.Sprintf("  suicided: %v", obj.Suicided))
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
	return ""
}
