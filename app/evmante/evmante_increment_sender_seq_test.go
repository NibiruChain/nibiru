package evmante_test

import (
	"fmt"
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"
	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/holiman/uint256"

	"github.com/NibiruChain/nibiru/v2/app/evmante"
	state "github.com/NibiruChain/nibiru/v2/x/evm/evmstate"
	"github.com/NibiruChain/nibiru/v2/x/evm/evmtest"
)

func (s *TestSuite) TestAnteDecEthIncrementSenderSequence() {
	testCases := []struct {
		name          string
		beforeTxSetup func(deps *evmtest.TestDeps, sdb *state.SDB)
		txSetup       func(deps *evmtest.TestDeps) sdk.Tx
		wantErr       string
		wantSeq       uint64
	}{
		{
			name: "happy: single message",
			beforeTxSetup: func(deps *evmtest.TestDeps, sdb *state.SDB) {
				balance := big.NewInt(100)
				AddBalanceSigned(sdb, deps.Sender.EthAddr, balance)
			},
			txSetup: func(deps *evmtest.TestDeps) sdk.Tx {
				return evmtest.HappyTransferTx(deps, 0)
			},
			wantErr: "",
			wantSeq: 1,
		},
		{
			name: "happy: two messages",
			beforeTxSetup: func(deps *evmtest.TestDeps, sdb *state.SDB) {
				balance := big.NewInt(100)
				AddBalanceSigned(sdb, deps.Sender.EthAddr, balance)
			},
			txSetup: func(deps *evmtest.TestDeps) sdk.Tx {
				txMsgOne := evmtest.HappyTransferTx(deps, 0)
				txMsgTwo := evmtest.HappyTransferTx(deps, 1)

				txBuilder := deps.App.GetTxConfig().NewTxBuilder()
				s.Require().NoError(txBuilder.SetMsgs(txMsgOne, txMsgTwo))

				tx := txBuilder.GetTx()
				return tx
			},
			wantErr: "",
			wantSeq: 2,
		},
		{
			name: "sad: account does not exists",
			txSetup: func(deps *evmtest.TestDeps) sdk.Tx {
				return evmtest.HappyTransferTx(deps, 0)
			},
			wantErr: "unknown address",
		},
		{
			name:    "sad: tx with non evm message",
			txSetup: evmtest.NonEvmMsgTx,
			wantErr: "invalid message",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			deps := evmtest.NewTestDeps()
			sdb := deps.NewStateDB()
			anteDec := evmante.NewAnteDecEthIncrementSenderSequence(deps.App.EvmKeeper, deps.App.AccountKeeper)

			if tc.beforeTxSetup != nil {
				tc.beforeTxSetup(&deps, sdb)
				sdb.Commit()
			}
			tx := tc.txSetup(&deps)

			_, err := anteDec.AnteHandle(
				deps.Ctx(), tx, false, evmtest.NextNoOpAnteHandler,
			)
			if tc.wantErr != "" {
				s.Require().ErrorContains(err, tc.wantErr)
				return
			}
			s.Require().NoError(err)

			if tc.wantSeq > 0 {
				seq := deps.App.AccountKeeper.GetAccount(deps.Ctx(), deps.Sender.NibiruAddr).GetSequence()
				s.Require().Equal(tc.wantSeq, seq)
			}
		})
	}
}

// AddBalanceSigned is only used in tests for convenience.
func AddBalanceSigned(sdb *state.SDB, addr gethcommon.Address, wei *big.Int) {
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
