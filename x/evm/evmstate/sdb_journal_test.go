package evmstate_test

import (
	"fmt"
	"math/big"

	"github.com/MakeNowJust/heredoc/v2"
	sdk "github.com/cosmos/cosmos-sdk/types"
	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/holiman/uint256"

	serverconfig "github.com/NibiruChain/nibiru/v2/app/server/config"
	"github.com/NibiruChain/nibiru/v2/x/evm"
	"github.com/NibiruChain/nibiru/v2/x/evm/embeds"
	"github.com/NibiruChain/nibiru/v2/x/evm/evmstate"
	"github.com/NibiruChain/nibiru/v2/x/evm/evmtest"
	"github.com/NibiruChain/nibiru/v2/x/evm/precompile/test"
	"github.com/NibiruChain/nibiru/v2/x/nutil"
	"github.com/NibiruChain/nibiru/v2/x/nutil/testutil/testapp"
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
	_, err = deps.EvmKeeper.CallContract(
		evmObj,
		deps.Sender.EthAddr, // caller
		&erc20,              // contract
		input,
		evm.Erc20GasLimitExecute,
		evm.COMMIT_READONLY, /*commit*/
		nil,
	)
	s.Require().NoError(err)

	// TODO: UD-DEBUG: test: assertion needed to replace dirty journals
	// s.Less(
	// 	0, evmObj.StateDB.(*evmstate.SDB).DebugDirtiesCount(),
	// 	"after a state modifying contract call (ERC20.mint), there should be dirty entries",
	// )

	evmObj.StateDB.(*evmstate.SDB).Commit()
	// TODO: UD-DEBUG: test: assertion needed to replace dirty journals
}

// AddBalanceSigned is only used in tests for convenience.
func AddBalanceSigned(sdb *evmstate.SDB, addr gethcommon.Address, wei *big.Int) {
	weiSign := wei.Sign()
	weiAbs, isOverflow := uint256.FromBig(new(big.Int).Abs(wei))
	if isOverflow {
		// TODO: Is there a better strategy than panicking here?
		panic(fmt.Errorf(
			"uint256 overflow occurred for big.Int value %s", wei))
	}

	reason := tracing.BalanceChangeTransfer
	if weiSign >= 0 {
		sdb.AddBalance(addr, weiAbs, reason)
	} else {
		sdb.SubBalance(addr, weiAbs, reason)
	}
}

func (s *Suite) TestCommitRemovesDirties_OnlyStateDB() {
	deps := evmtest.NewTestDeps()
	evmObj, _ := deps.NewEVM()
	sdb := evmObj.StateDB.(*evmstate.SDB)

	randomAcc := evmtest.NewEthPrivAcc().EthAddr
	balDelta := evm.NativeToWei(big.NewInt(4))
	// 2 dirties from [createObjectChange, balanceChange]
	AddBalanceSigned(sdb, randomAcc, balDelta)
	// 1 dirties from [balanceChange]
	AddBalanceSigned(sdb, randomAcc, balDelta)
	// 1 dirties from [balanceChange]
	AddBalanceSigned(sdb, randomAcc, balDelta)
	// TODO: UD-DEBUG: test: assertion needed to replace dirty journals
	// if sdb.DebugDirtiesCount() != 4 {
	// 	debugDirtiesCountMismatch(sdb, s.T())
	// 	s.FailNow("expected 4 dirty journal changes")
	// }

	s.T().Log("StateDB.Commit, then Dirties should be gone")
	sdb.Commit()
	// TODO: UD-DEBUG: test: assertion needed to replace dirty journals
	// if sdb.DebugDirtiesCount() != 0 {
	// 	debugDirtiesCountMismatch(sdb, s.T())
	// 	s.FailNow("expected 0 dirty journal changes")
	// }
}

func (s *Suite) TestContractCallsAnotherContract() {
	deps := evmtest.NewTestDeps()
	evmObj, _ := deps.NewEVM()
	// TODO: UD-DEBUG: refactor: sdb not needed here?
	// sdb := evmObj.StateDB.(*evmstate.SDB)

	s.Require().NoError(testapp.FundAccount(
		deps.App.BankKeeper,
		deps.Ctx(),
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
		_, err = deps.EvmKeeper.CallContract(
			evmObj,
			deps.Sender.EthAddr, // caller
			&erc20,              // contract
			contractInput,
			evm.Erc20GasLimitExecute,
			evm.COMMIT_ETH_TX, /*commit*/
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
		// TODO: UD-DEBUG: New assertion needed for test to replace dirty journal
		// checks
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
		deps.Ctx(),
		deps.Sender.NibiruAddr,
		sdk.NewCoins(sdk.NewCoin(evm.EVMBankDenom, sdk.NewInt(69_420))),
	))

	type SnapshotAssertion struct {
		SnapshotIdx         int
		WantPostRevertCount int64
		WantPreRevertCount  int64
		Text                string
	}

	s.T().Log("Set up helloworldcounter.wasm")
	wasmContracts := test.SetupWasmContracts(&deps, &s.Suite)
	helloWorldCounterWasm := wasmContracts[1]
	s.T().Logf("helloworldcounter.wasm - contract addr:\n%s\n", helloWorldCounterWasm)

	s.T().Log("commitEvmTx=true, expect 0 dirty journal entries")
	evmObj, sdb := deps.NewEVM()
	snapshots := []SnapshotAssertion{
		{
			SnapshotIdx:         sdb.SnapshotRevertIdx(),
			WantPreRevertCount:  7,
			WantPostRevertCount: 7,
			Text:                "init sdb with counter at 0 (committing at 7)",
		},
	}

	test.IncrementWasmCounterWithExecuteMulti(
		&s.Suite, &deps, evmObj, helloWorldCounterWasm, 7, true,
	)
	snapshots = append(snapshots, SnapshotAssertion{
		SnapshotIdx:         sdb.SnapshotRevertIdx(),
		WantPostRevertCount: 7,
		WantPreRevertCount:  7,
		Text:                "increment +7 with commit=true",
	})

	// if sdb.DebugDirtiesCount() != 0 {
	// 	debugDirtiesCountMismatch(sdb, s.T())
	// 	s.FailNowf("statedb dirty count mismatch", "expected 0 dirty journal changes, but instead got: %d", sdb.DebugDirtiesCount())
	// }

	s.T().Log("commitEvmTx=false, expect dirty journal entries")
	// evmObj, sdb = deps.NewEVM()
	test.IncrementWasmCounterWithExecuteMulti(
		&s.Suite, &deps, evmObj, helloWorldCounterWasm, 5, false,
	)
	snapshots = append(snapshots, SnapshotAssertion{
		SnapshotIdx:         sdb.SnapshotRevertIdx(),
		WantPostRevertCount: 7,
		WantPreRevertCount:  7 + 5,
		Text:                "increment +5 (commit=false)",
	})

	// s.T().Log("Expect exactly 1 dirty journal entry for the precompile snapshot")
	// if sdb.DebugDirtiesCount() != 1 {
	// 	debugDirtiesCountMismatch(sdb, s.T())
	// 	s.FailNowf("statedb dirty count mismatch", "expected 1 dirty journal change, but instead got: %d", sdb.DebugDirtiesCount())
	// }

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
	snapshots = append(snapshots, SnapshotAssertion{
		SnapshotIdx:         sdb.SnapshotRevertIdx(),
		WantPostRevertCount: 7 + 5,
		WantPreRevertCount:  7 + 5 + 50,
		Text:                "increment +50 (commit=false)",
	})
	s.T().Log(heredoc.Doc(`At this point, several precompile calls have succeeded.
The StateDB has not been committed. We expect to be able to revert to any 
snapshots and see prior states.`))
	test.AssertWasmCounterStateWithEvm(
		&s.Suite, deps, evmObj, helloWorldCounterWasm, 7+5+50,
	)

	errPanic := nutil.TryCatch(func() {
		// a revision that doesn't exist
		sdb.RevertToSnapshot(9000)
	})()
	s.Require().ErrorContains(errPanic, "revision id 9000 cannot be reverted")

	s.Equal(
		snapshots[len(snapshots)-1].SnapshotIdx+1, // plus 1 comes from the wasm counter query, which was an EVM precompile query
		sdb.SnapshotRevertIdx(),
		"snapshot index must be the same since we've only done read operations since the last snapshot",
	)

	for i := len(snapshots) - 1; i >= 0; i-- {
		snap := snapshots[i]
		s.T().Logf("assert snapshot: %+v", snap)

		test.AssertWasmCounterStateWithEvm(
			&s.Suite, deps, evmObj, helloWorldCounterWasm, snap.WantPreRevertCount,
		)
		sdb.RevertToSnapshot(snap.SnapshotIdx)
		test.AssertWasmCounterStateWithEvm(
			&s.Suite, deps, evmObj, helloWorldCounterWasm, snap.WantPostRevertCount,
		)
	}

	// sdb.RevertToSnapshot(2)
	// test.AssertWasmCounterStateWithEvm(
	// 	&s.Suite, deps, evmObj, helloWorldCounterWasm, 7+5,
	// )

	// sdb.RevertToSnapshot(0)
	// test.AssertWasmCounterStateWithEvm(
	// 	&s.Suite, deps, evmObj, helloWorldCounterWasm, 7,
	// )

	sdb.Commit()
	test.AssertWasmCounterState(
		&s.Suite, deps, helloWorldCounterWasm, 7,
	)
}
