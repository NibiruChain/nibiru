package evmante_test

import (
	"fmt"
	"math/big"
	"slices"

	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/holiman/uint256"

	"github.com/NibiruChain/nibiru/v2/evm"
	"github.com/NibiruChain/nibiru/v2/evm/evmante"
	"github.com/NibiruChain/nibiru/v2/evm/evmstate"
	"github.com/NibiruChain/nibiru/v2/evm/evmtest"
)

func (s *Suite) TestEthAnteIncrementNonce() {
	testCases := []struct {
		name    string
		txSetup func(deps *evmtest.TestDeps, sdb *evmstate.SDB) *evm.MsgEthereumTx
		wantErr string
		wantSeq uint64
	}{
		{
			name: "happy: single message",
			txSetup: func(deps *evmtest.TestDeps, sdb *evmstate.SDB) *evm.MsgEthereumTx {
				balance := big.NewInt(100)
				AddBalanceSigned(sdb, deps.Sender.EthAddr, balance)
				return evmtest.HappyTransferTx(deps, 0)
			},
			wantErr: "",
			wantSeq: 1,
		},
		{
			name: "sad: account does not exists",
			txSetup: func(deps *evmtest.TestDeps, sdb *evmstate.SDB) *evm.MsgEthereumTx {
				return evmtest.HappyTransferTx(deps, 0)
			},
			wantErr: "unknown address",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			deps := evmtest.NewTestDeps()
			sdb := deps.NewStateDB()

			msgEthTx := tc.txSetup(&deps, sdb)
			simulate := false
			unusedOpts := AnteOptionsForTests{}
			err := evmante.AnteStepIncrementNonce(
				sdb,
				sdb.Keeper(),
				msgEthTx,
				simulate,
				unusedOpts,
			)

			if tc.wantErr != "" {
				s.Require().ErrorContains(err, tc.wantErr)
				return
			}
			s.Require().NoError(err)

			s.True(deps.IsEqualSDB(sdb), "deps and sdb MUST still be connected.")
			if tc.wantSeq > 0 {
				acct := deps.EvmKeeper.GetAccount(sdb.Ctx(), deps.Sender.EthAddr)
				s.NotNil(acct)

				acct1 := deps.App.AccountKeeper.GetAccount(deps.Ctx(), deps.Sender.NibiruAddr)
				s.Require().NotNil(acct1, "deps.Ctx(), after commit - Cosmos SDK account should exist")
				acct2 := deps.App.AccountKeeper.GetAccount(sdb.Ctx(), deps.Sender.NibiruAddr)
				s.Require().NotNil(acct1, "sdb.Ctx(), after commit - Cosmos SDK account should exist")

				nonces := []uint64{
					acct.Nonce,
					acct1.GetSequence(),
					acct2.GetSequence(),
				}
				s.Require().Equal(slices.Repeat([]uint64{tc.wantSeq}, 3), nonces,
					"nonces must match no matter what query or context is used.")
			}
		})
	}
}

func (s *Suite) TestEthAnteIncrementNonceCheckTx() {
	deps := evmtest.NewTestDeps()
	deps.SetCtx(deps.Ctx().WithIsCheckTx(true))
	sdb := deps.NewStateDB()
	AddBalanceSigned(sdb, deps.Sender.EthAddr, big.NewInt(100))
	sdb.SetNonce(deps.Sender.EthAddr, 10)
	sdb.Commit()
	checkCtx := sdb.RootCtx()

	runAnteStep := func(nonce uint64) error {
		sdb = deps.EvmKeeper.NewSDB(
			checkCtx,
			deps.EvmKeeper.TxConfig(checkCtx, gethcommon.Hash{}),
		)
		return evmante.AnteStepIncrementNonce(
			sdb,
			sdb.Keeper(),
			evmtest.HappyTransferTx(&deps, nonce),
			false,
			AnteOptionsForTests{},
		)
	}
	requireState := func(wantNonce uint64) {
		acct := deps.EvmKeeper.GetAccount(checkCtx, deps.Sender.EthAddr)
		s.Require().NotNil(acct)
		s.Require().Equal(wantNonce, acct.Nonce)
	}

	s.Require().ErrorContains(runAnteStep(9), "invalid nonce; got 9, expected 10 or higher")
	requireState(10)

	for _, nonce := range []uint64{10, 11, 73, 74, 10} {
		s.Require().NoError(runAnteStep(nonce))
	}
	requireState(10)

	s.Require().ErrorContains(runAnteStep(75), "future nonce gap too large")
	requireState(10)
}

func (s *Suite) TestEthAnteIncrementNonceReCheckTx() {
	deps := evmtest.NewTestDeps()
	deps.SetCtx(deps.Ctx().WithIsReCheckTx(true))
	sdb := deps.NewStateDB()
	AddBalanceSigned(sdb, deps.Sender.EthAddr, big.NewInt(100))
	sdb.SetNonce(deps.Sender.EthAddr, 10)
	sdb.Commit()
	recheckCtx := sdb.RootCtx()

	runAnteStep := func(nonce uint64) error {
		sdb = deps.EvmKeeper.NewSDB(
			recheckCtx,
			deps.EvmKeeper.TxConfig(recheckCtx, gethcommon.Hash{}),
		)
		return evmante.AnteStepIncrementNonce(
			sdb,
			sdb.Keeper(),
			evmtest.HappyTransferTx(&deps, nonce),
			false,
			AnteOptionsForTests{},
		)
	}

	s.Require().True(evmstate.IsReCheckTxOnly(recheckCtx))
	s.Require().ErrorContains(runAnteStep(11), "invalid nonce; got 11, expected 10")

	s.Require().NoError(runAnteStep(10))

	acct := deps.EvmKeeper.GetAccount(recheckCtx, deps.Sender.EthAddr)
	s.Require().NotNil(acct)
	s.Require().Equal(uint64(10), acct.Nonce, "ReCheckTx must not persist SetNonce")
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
		if acc := sdb.Keeper().GetAccount(sdb.Ctx(), addr); acc == nil {
			acc = evmstate.NewEmptyAccount()
			acc.BalanceNwei = weiAbs
			fmt.Printf("About to call SetAccount with addr: %v, acc: %v\n", addr, acc)
			err := sdb.Keeper().SetAccount(sdb.Ctx(), addr, *acc)
			fmt.Printf("SetAccount result: %v\n", err)
		}
	} else {
		sdb.SubBalance(addr, weiAbs, reason)
	}
}
