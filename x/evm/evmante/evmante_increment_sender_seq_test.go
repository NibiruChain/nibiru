package evmante_test

import (
	"fmt"
	"math/big"
	"slices"

	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/holiman/uint256"

	"github.com/NibiruChain/nibiru/v2/x/evm"
	"github.com/NibiruChain/nibiru/v2/x/evm/evmante"
	"github.com/NibiruChain/nibiru/v2/x/evm/evmstate"
	"github.com/NibiruChain/nibiru/v2/x/evm/evmtest"
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
