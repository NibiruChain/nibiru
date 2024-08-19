package evmante_test

import (
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/v2/app/evmante"
	"github.com/NibiruChain/nibiru/v2/x/evm/evmtest"
	"github.com/NibiruChain/nibiru/v2/x/evm/statedb"
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
				return evmtest.HappyTransferTx(deps, 0)
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
				txMsgOne := evmtest.HappyTransferTx(deps, 0)
				txMsgTwo := evmtest.HappyTransferTx(deps, 1)

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
			stateDB := deps.StateDB()
			anteDec := evmante.NewAnteDecEthIncrementSenderSequence(&deps.App.EvmKeeper, deps.App.AccountKeeper)

			if tc.beforeTxSetup != nil {
				tc.beforeTxSetup(&deps, stateDB)
				s.Require().NoError(stateDB.Commit())
			}
			tx := tc.txSetup(&deps)

			_, err := anteDec.AnteHandle(
				deps.Ctx, tx, false, evmtest.NextNoOpAnteHandler,
			)
			if tc.wantErr != "" {
				s.Require().ErrorContains(err, tc.wantErr)
				return
			}
			s.Require().NoError(err)

			if tc.wantSeq > 0 {
				seq := deps.App.AccountKeeper.GetAccount(deps.Ctx, deps.Sender.NibiruAddr).GetSequence()
				s.Require().Equal(tc.wantSeq, seq)
			}
		})
	}
}
