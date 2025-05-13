package ante_test

import (
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/v2/app/ante"
	"github.com/NibiruChain/nibiru/v2/x/evm/evmtest"
)

func (s *AnteTestSuite) TestGasWantedDecorator() {
	testCases := []struct {
		name     string
		ctxSetup func(deps *evmtest.TestDeps)
		txSetup  func(deps *evmtest.TestDeps) sdk.Tx
		wantErr  string
	}{
		{
			name: "happy: non fee tx type",
			txSetup: func(deps *evmtest.TestDeps) sdk.Tx {
				return evmtest.HappyCreateContractTx(deps)
			},
			wantErr: "",
		},
		{
			name: "happy: tx without gas, block gas limit 1000",
			ctxSetup: func(deps *evmtest.TestDeps) {
				cp := tmproto.ConsensusParams{
					Block: &tmproto.BlockParams{MaxGas: 1000},
				}
				deps.Ctx = deps.Ctx.WithConsensusParams(cp)
			},
			txSetup: func(deps *evmtest.TestDeps) sdk.Tx {
				return evmtest.HappyCreateContractTx(deps)
			},
			wantErr: "",
		},
		{
			name: "happy: tx with gas wanted 500, block gas limit 1000",
			ctxSetup: func(deps *evmtest.TestDeps) {
				cp := tmproto.ConsensusParams{
					Block: &tmproto.BlockParams{MaxGas: 1000},
				}
				deps.Ctx = deps.Ctx.WithConsensusParams(cp)
			},
			txSetup: func(deps *evmtest.TestDeps) sdk.Tx {
				builder := deps.App.GetTxConfig().NewTxBuilder()
				_ = builder.SetMsgs(
					evmtest.HappyCreateContractTx(deps),
				)
				builder.SetGasLimit(500)
				return builder.GetTx()
			},
			wantErr: "",
		},
		{
			name: "sad: tx with gas wanted 1000, block gas limit 500",
			ctxSetup: func(deps *evmtest.TestDeps) {
				cp := tmproto.ConsensusParams{
					Block: &tmproto.BlockParams{
						MaxGas: 500,
					},
				}
				deps.Ctx = deps.Ctx.WithConsensusParams(cp)
			},
			txSetup: func(deps *evmtest.TestDeps) sdk.Tx {
				builder := deps.App.GetTxConfig().NewTxBuilder()
				_ = builder.SetMsgs(
					evmtest.HappyCreateContractTx(deps),
				)
				builder.SetGasLimit(1000)
				return builder.GetTx()
			},
			wantErr: "exceeds block gas limit",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			deps := evmtest.NewTestDeps(s.T().TempDir())
			stateDB := deps.NewStateDB()
			anteDec := ante.AnteDecoratorGasWanted{}

			tx := tc.txSetup(&deps)
			s.Require().NoError(stateDB.Commit())

			deps.Ctx = deps.Ctx.WithIsCheckTx(true)
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
