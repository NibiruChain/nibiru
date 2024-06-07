package app_test

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/eth"
	"github.com/NibiruChain/nibiru/x/evm/evmtest"
)

func (s *TestSuite) TestEthValidateBasicDecorator() {
	testCases := []struct {
		name     string
		ctxSetup func(deps *evmtest.TestDeps)
		txSetup  func(deps *evmtest.TestDeps) sdk.Tx
		wantErr  string
	}{
		{
			name: "sad: unsigned уер tx should fail chain id validation",
			txSetup: func(deps *evmtest.TestDeps) sdk.Tx {
				return happyCreateContractTx(deps)
			},
			wantErr: "invalid chain-id",
		},
		{
			name: "happy: ctx recheck should ignore validation",
			ctxSetup: func(deps *evmtest.TestDeps) {
				deps.Ctx = deps.Ctx.WithIsReCheckTx(true)
			},
			txSetup: func(deps *evmtest.TestDeps) sdk.Tx {
				return happyCreateContractTx(deps)
			},
			wantErr: "",
		},
		{
			name: "sad: tx not implementing protoTxProvider",
			txSetup: func(deps *evmtest.TestDeps) sdk.Tx {
				tx := happyCreateContractTx(deps)
				gethSigner := deps.Sender.GethSigner(InvalidChainID)
				keyringSigner := deps.Sender.KeyringSigner
				err := tx.Sign(gethSigner, keyringSigner)
				s.Require().NoError(err)
				return tx
			},
			wantErr: "didn't implement interface protoTxProvider",
		},
		{
			name: "happy: signed ethereum tx should pass validation",
			txSetup: func(deps *evmtest.TestDeps) sdk.Tx {
				txBuilder := deps.EncCfg.TxConfig.NewTxBuilder()
				tx, err := happyCreateContractTx(deps).BuildTx(txBuilder, eth.EthBaseDenom)
				s.Require().NoError(err)
				return tx
			},
			wantErr: "",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			deps := evmtest.NewTestDeps()
			stateDB := deps.StateDB()
			anteDec := app.NewEthValidateBasicDecorator(deps.Chain.AppKeepers)

			tx := tc.txSetup(&deps)
			s.Require().NoError(stateDB.Commit())

			if tc.ctxSetup != nil {
				tc.ctxSetup(&deps)
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
