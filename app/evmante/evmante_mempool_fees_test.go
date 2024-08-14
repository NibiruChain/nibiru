package evmante_test

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	"github.com/NibiruChain/nibiru/v2/app/evmante"
	"github.com/NibiruChain/nibiru/v2/x/evm/evmtest"
)

func (s *TestSuite) TestMempoolGasFeeDecorator() {
	testCases := []struct {
		name     string
		txSetup  func(deps *evmtest.TestDeps) sdk.Tx
		ctxSetup func(deps *evmtest.TestDeps)
		wantErr  string
	}{
		{
			name: "happy: min gas price is 0",
			txSetup: func(deps *evmtest.TestDeps) sdk.Tx {
				tx := evmtest.HappyCreateContractTx(deps)
				return tx
			},
			wantErr: "",
		},
		{
			name: "happy: min gas price is not zero, sufficient fee",
			ctxSetup: func(deps *evmtest.TestDeps) {
				gasPrice := sdk.NewInt64Coin("unibi", 1)
				deps.Ctx = deps.Ctx.
					WithMinGasPrices(
						sdk.NewDecCoins(sdk.NewDecCoinFromCoin(gasPrice)),
					).
					WithIsCheckTx(true)
			},
			txSetup: func(deps *evmtest.TestDeps) sdk.Tx {
				tx := evmtest.HappyCreateContractTx(deps)
				return tx
			},
			wantErr: "",
		},
		{
			name: "sad: insufficient fee",
			ctxSetup: func(deps *evmtest.TestDeps) {
				gasPrice := sdk.NewInt64Coin("unibi", 2)
				deps.Ctx = deps.Ctx.
					WithMinGasPrices(
						sdk.NewDecCoins(sdk.NewDecCoinFromCoin(gasPrice)),
					).
					WithIsCheckTx(true)
			},
			txSetup: func(deps *evmtest.TestDeps) sdk.Tx {
				tx := evmtest.HappyCreateContractTx(deps)
				return tx
			},
			wantErr: "insufficient fee",
		},
		{
			name: "sad: tx with non evm message",
			ctxSetup: func(deps *evmtest.TestDeps) {
				gasPrice := sdk.NewInt64Coin("unibi", 1)
				deps.Ctx = deps.Ctx.
					WithMinGasPrices(
						sdk.NewDecCoins(sdk.NewDecCoinFromCoin(gasPrice)),
					).
					WithIsCheckTx(true)
			},
			txSetup: func(deps *evmtest.TestDeps) sdk.Tx {
				gasLimit := uint64(10)
				fees := sdk.NewCoins(sdk.NewInt64Coin("unibi", int64(gasLimit)))
				msg := &banktypes.MsgSend{
					FromAddress: deps.Sender.NibiruAddr.String(),
					ToAddress:   evmtest.NewEthPrivAcc().NibiruAddr.String(),
					Amount:      sdk.NewCoins(sdk.NewInt64Coin("unibi", 1)),
				}
				return buildTx(deps, true, msg, gasLimit, fees)
			},
			wantErr: "invalid message",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			deps := evmtest.NewTestDeps()
			anteDec := evmante.NewMempoolGasPriceDecorator(&deps.App.AppKeepers.EvmKeeper)

			tx := tc.txSetup(&deps)

			if tc.ctxSetup != nil {
				tc.ctxSetup(&deps)
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
