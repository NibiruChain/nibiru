package evmante_test

import (
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/app/evmante"
	evmtestutil "github.com/NibiruChain/nibiru/x/common/testutil/evm"
	"github.com/NibiruChain/nibiru/x/evm/evmtest"
	"github.com/NibiruChain/nibiru/x/evm/statedb"
)

func (s *TestSuite) TestAnteDecEthIncrementSenderSequence() {
	testCases := []struct {
		name          string
		beforeTxSetup func(deps *evmtest.TestDeps, sdb *statedb.StateDB)
		txSetup       func(deps *evmtest.TestDeps) sdk.Tx
		wantErr       string
		wantSeq       uint64
	}{
		{
			name: "happy: single message",
			beforeTxSetup: func(deps *evmtest.TestDeps, sdb *statedb.StateDB) {
				balance := big.NewInt(100)
				sdb.AddBalance(deps.Sender.EthAddr, balance)
			},
			txSetup: func(deps *evmtest.TestDeps) sdk.Tx {
				return evmtestutil.HappyTransferTx(deps, 0)
			},
			wantErr: "",
			wantSeq: 1,
		},
		{
			name: "happy: two messages",
			beforeTxSetup: func(deps *evmtest.TestDeps, sdb *statedb.StateDB) {
				balance := big.NewInt(100)
				sdb.AddBalance(deps.Sender.EthAddr, balance)
			},
			txSetup: func(deps *evmtest.TestDeps) sdk.Tx {
				txMsgOne := evmtestutil.HappyTransferTx(deps, 0)
				txMsgTwo := evmtestutil.HappyTransferTx(deps, 1)

				txBuilder := deps.EncCfg.TxConfig.NewTxBuilder()
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
				return evmtestutil.HappyTransferTx(deps, 0)
			},
			wantErr: "unknown address",
		},
		{
			name:    "sad: tx with non evm message",
			txSetup: evmtestutil.NonEvmMsgTx,
			wantErr: "invalid message",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			deps := evmtest.NewTestDeps()
			stateDB := deps.StateDB()
			anteDec := evmante.NewAnteDecEthIncrementSenderSequence(&deps.Chain.EvmKeeper, deps.Chain.AccountKeeper)

			if tc.beforeTxSetup != nil {
				tc.beforeTxSetup(&deps, stateDB)
				s.Require().NoError(stateDB.Commit())
			}
			tx := tc.txSetup(&deps)

			_, err := anteDec.AnteHandle(
				deps.Ctx, tx, false, evmtestutil.NextNoOpAnteHandler,
			)
			if tc.wantErr != "" {
				s.Require().ErrorContains(err, tc.wantErr)
				return
			}
			s.Require().NoError(err)

			if tc.wantSeq > 0 {
				seq := deps.Chain.AccountKeeper.GetAccount(deps.Ctx, deps.Sender.NibiruAddr).GetSequence()
				s.Require().Equal(tc.wantSeq, seq)
			}
		})
	}
}
