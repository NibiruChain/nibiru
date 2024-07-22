package app_test

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/eth"
	"github.com/NibiruChain/nibiru/x/evm/evmtest"
)

func (s *TestSuite) TestDynamicFeeChecker() {
	testCases := []struct {
		name         string
		txSetup      func(deps *evmtest.TestDeps) sdk.FeeTx
		ctxSetup     func(deps *evmtest.TestDeps)
		wantErr      string
		wantFee      int64
		wantPriority int64
	}{
		{
			name: "happy: genesis tx with sufficient fee",
			ctxSetup: func(deps *evmtest.TestDeps) {
				gasPrice := sdk.NewInt64Coin("unibi", 1)
				deps.Ctx = deps.Ctx.
					WithBlockHeight(0).
					WithMinGasPrices(
						sdk.NewDecCoins(sdk.NewDecCoinFromCoin(gasPrice)),
					).
					WithIsCheckTx(true)
			},
			txSetup: func(deps *evmtest.TestDeps) sdk.FeeTx {
				txMsg := happyCreateContractTx(deps)
				txBuilder := deps.EncCfg.TxConfig.NewTxBuilder()
				tx, err := txMsg.BuildTx(txBuilder, eth.EthBaseDenom)
				s.Require().NoError(err)
				return tx
			},
			wantErr:      "",
			wantFee:      gasLimitCreateContract().Int64(),
			wantPriority: 0,
		},
		{
			name: "sad: genesis tx insufficient fee",
			ctxSetup: func(deps *evmtest.TestDeps) {
				gasPrice := sdk.NewInt64Coin("unibi", 2)
				deps.Ctx = deps.Ctx.
					WithBlockHeight(0).
					WithMinGasPrices(
						sdk.NewDecCoins(sdk.NewDecCoinFromCoin(gasPrice)),
					).
					WithIsCheckTx(true)
			},
			txSetup: func(deps *evmtest.TestDeps) sdk.FeeTx {
				txMsg := happyCreateContractTx(deps)
				txBuilder := deps.EncCfg.TxConfig.NewTxBuilder()
				tx, err := txMsg.BuildTx(txBuilder, eth.EthBaseDenom)
				s.Require().NoError(err)
				return tx
			},
			wantErr: "insufficient fee",
		},
		{
			name: "happy: tx with sufficient fee",
			txSetup: func(deps *evmtest.TestDeps) sdk.FeeTx {
				txMsg := happyCreateContractTx(deps)
				txBuilder := deps.EncCfg.TxConfig.NewTxBuilder()
				tx, err := txMsg.BuildTx(txBuilder, eth.EthBaseDenom)
				s.Require().NoError(err)
				return tx
			},
			wantErr:      "",
			wantFee:      gasLimitCreateContract().Int64(),
			wantPriority: 0,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			deps := evmtest.NewTestDeps()
			checker := app.NewDynamicFeeChecker(deps.K)

			if tc.ctxSetup != nil {
				tc.ctxSetup(&deps)
			}

			fee, priority, err := checker(deps.Ctx, tc.txSetup(&deps))

			if tc.wantErr != "" {
				s.Require().ErrorContains(err, tc.wantErr)
				return
			}
			s.Require().NoError(err)
			s.Require().Equal(tc.wantFee, fee.AmountOf("unibi").Int64())
			s.Require().Equal(tc.wantPriority, priority)
		})
	}
}
