package app_test

import (
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/eth"
	"github.com/NibiruChain/nibiru/x/evm/evmtest"
	"github.com/NibiruChain/nibiru/x/evm/statedb"
)

func (s *TestSuite) TestCanTransferDecorator() {
	testCases := []struct {
		name          string
		txSetup       func(deps *evmtest.TestDeps) sdk.FeeTx
		ctxSetup      func(deps *evmtest.TestDeps)
		beforeTxSetup func(deps *evmtest.TestDeps, sdb *statedb.StateDB)
		wantErr       string
	}{
		{
			name: "happy: signed tx, sufficient funds",
			beforeTxSetup: func(deps *evmtest.TestDeps, sdb *statedb.StateDB) {
				sdb.AddBalance(deps.Sender.EthAddr, big.NewInt(100))
			},
			txSetup: func(deps *evmtest.TestDeps) sdk.FeeTx {
				txMsg := happyTransfertTx(deps, 0)
				txBuilder := deps.EncCfg.TxConfig.NewTxBuilder()

				gethSigner := deps.Sender.GethSigner(deps.Chain.EvmKeeper.EthChainID(deps.Ctx))
				keyringSigner := deps.Sender.KeyringSigner
				err := txMsg.Sign(gethSigner, keyringSigner)
				s.Require().NoError(err)

				tx, err := txMsg.BuildTx(txBuilder, eth.EthBaseDenom)
				s.Require().NoError(err)

				return tx
			},
			wantErr: "",
		},
		{
			name: "sad: signed tx, insufficient funds",
			txSetup: func(deps *evmtest.TestDeps) sdk.FeeTx {
				txMsg := happyTransfertTx(deps, 0)
				txBuilder := deps.EncCfg.TxConfig.NewTxBuilder()

				gethSigner := deps.Sender.GethSigner(deps.Chain.EvmKeeper.EthChainID(deps.Ctx))
				keyringSigner := deps.Sender.KeyringSigner
				err := txMsg.Sign(gethSigner, keyringSigner)
				s.Require().NoError(err)

				tx, err := txMsg.BuildTx(txBuilder, eth.EthBaseDenom)
				s.Require().NoError(err)

				return tx
			},
			wantErr: "insufficient funds",
		},
		{
			name: "sad: unsigned tx",
			txSetup: func(deps *evmtest.TestDeps) sdk.FeeTx {
				txMsg := happyTransfertTx(deps, 0)
				txBuilder := deps.EncCfg.TxConfig.NewTxBuilder()

				tx, err := txMsg.BuildTx(txBuilder, eth.EthBaseDenom)
				s.Require().NoError(err)

				return tx
			},
			wantErr: "invalid transaction",
		},
		{
			name: "sad: tx with non evm message",
			txSetup: func(deps *evmtest.TestDeps) sdk.FeeTx {
				return nonEvmMsgTx(deps).(sdk.FeeTx)
			},
			wantErr: "invalid message",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			deps := evmtest.NewTestDeps()
			stateDB := deps.StateDB()
			anteDec := app.NewCanTransferDecorator(deps.Chain.AppKeepers)
			tx := tc.txSetup(&deps)

			if tc.ctxSetup != nil {
				tc.ctxSetup(&deps)
			}
			if tc.beforeTxSetup != nil {
				tc.beforeTxSetup(&deps, stateDB)
				err := stateDB.Commit()
				s.Require().NoError(err)
			}

			_, err := anteDec.AnteHandle(
				deps.Ctx, tx, false, NextNoOpAnteHandler,
			)
			if tc.wantErr != "" {
				s.Require().ErrorContains(err, tc.wantErr)
				return
			}
			s.Require().NoError(err)
		})
	}
}
