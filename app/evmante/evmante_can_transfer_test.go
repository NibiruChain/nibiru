package evmante_test

import (
	"github.com/NibiruChain/nibiru/v2/app/evmante"
	"github.com/NibiruChain/nibiru/v2/eth"
	"github.com/NibiruChain/nibiru/v2/x/common/testutil/testapp"
	"github.com/NibiruChain/nibiru/v2/x/evm/evmtest"
	"github.com/NibiruChain/nibiru/v2/x/evm/statedb"
	sdk "github.com/cosmos/cosmos-sdk/types"
	gethcore "github.com/ethereum/go-ethereum/core/types"
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
				s.NoError(
					testapp.FundAccount(
						deps.App.BankKeeper,
						deps.Ctx,
						deps.Sender.NibiruAddr,
						sdk.NewCoins(sdk.NewInt64Coin(eth.EthBaseDenom, 100)),
					),
				)
			},
			txSetup: func(deps *evmtest.TestDeps) sdk.FeeTx {
				txMsg := evmtest.HappyTransferTx(deps, 0)
				txBuilder := deps.EncCfg.TxConfig.NewTxBuilder()

				gethSigner := gethcore.LatestSignerForChainID(deps.App.EvmKeeper.EthChainID(deps.Ctx))
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
				txMsg := evmtest.HappyTransferTx(deps, 0)
				txBuilder := deps.EncCfg.TxConfig.NewTxBuilder()

				gethSigner := gethcore.LatestSignerForChainID(deps.App.EvmKeeper.EthChainID(deps.Ctx))
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
				txMsg := evmtest.HappyTransferTx(deps, 0)
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
				return evmtest.NonEvmMsgTx(deps).(sdk.FeeTx)
			},
			wantErr: "invalid message",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			deps := evmtest.NewTestDeps()
			stateDB := deps.StateDB()
			anteDec := evmante.NewCanTransferDecorator(&deps.App.AppKeepers.EvmKeeper)
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
				deps.Ctx, tx, false, evmtest.NextNoOpAnteHandler,
			)
			if tc.wantErr != "" {
				s.Require().ErrorContains(err, tc.wantErr)
				return
			}
			s.Require().NoError(err)
		})
	}
}
