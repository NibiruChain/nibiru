package ante_test

import (
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth/migrations/legacytx"

	"github.com/NibiruChain/nibiru/app/ante"
	evmtestutil "github.com/NibiruChain/nibiru/x/common/testutil/evm"
	"github.com/NibiruChain/nibiru/x/evm/evmtest"
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
				return evmtestutil.HappyCreateContractTx(deps)
			},
			wantErr: "",
		},
		{
			name: "happy: tx without gas, block gas limit 1000",
			ctxSetup: func(deps *evmtest.TestDeps) {
				cp := &tmproto.ConsensusParams{
					Block: &tmproto.BlockParams{MaxGas: 1000},
				}
				deps.Ctx = deps.Ctx.WithConsensusParams(cp)
			},
			txSetup: func(deps *evmtest.TestDeps) sdk.Tx {
				return legacytx.StdTx{
					Msgs: []sdk.Msg{
						evmtestutil.HappyCreateContractTx(deps),
					},
				}
			},
			wantErr: "",
		},
		{
			name: "happy: tx with gas wanted 500, block gas limit 1000",
			ctxSetup: func(deps *evmtest.TestDeps) {
				cp := &tmproto.ConsensusParams{
					Block: &tmproto.BlockParams{MaxGas: 1000},
				}
				deps.Ctx = deps.Ctx.WithConsensusParams(cp)
			},
			txSetup: func(deps *evmtest.TestDeps) sdk.Tx {
				return legacytx.StdTx{
					Msgs: []sdk.Msg{
						evmtestutil.HappyCreateContractTx(deps),
					},
					Fee: legacytx.StdFee{Gas: 500},
				}
			},
			wantErr: "",
		},
		{
			name: "sad: tx with gas wanted 1000, block gas limit 500",
			ctxSetup: func(deps *evmtest.TestDeps) {
				cp := &tmproto.ConsensusParams{
					Block: &tmproto.BlockParams{
						MaxGas: 500,
					},
				}
				deps.Ctx = deps.Ctx.WithConsensusParams(cp)
			},
			txSetup: func(deps *evmtest.TestDeps) sdk.Tx {
				return legacytx.StdTx{
					Msgs: []sdk.Msg{
						evmtestutil.HappyCreateContractTx(deps),
					},
					Fee: legacytx.StdFee{Gas: 1000},
				}
			},
			wantErr: "exceeds block gas limit",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			deps := evmtest.NewTestDeps()
			stateDB := deps.StateDB()
			anteDec := ante.AnteDecoratorGasWanted{}

			tx := tc.txSetup(&deps)
			s.Require().NoError(stateDB.Commit())

			deps.Ctx = deps.Ctx.WithIsCheckTx(true)
			if tc.ctxSetup != nil {
				tc.ctxSetup(&deps)
			}
			_, err := anteDec.AnteHandle(
				deps.Ctx, tx, false, evmtestutil.NextNoOpAnteHandler,
			)
			if tc.wantErr != "" {
				s.Require().ErrorContains(err, tc.wantErr)
				return
			}
			s.Require().NoError(err)
		})
	}
}
