package ante_test

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/app/ante"
	evmtestutil "github.com/NibiruChain/nibiru/x/common/testutil/evm"
	"github.com/NibiruChain/nibiru/x/evm/evmtest"
)

func (s *AnteTestSuite) TestAnteDecoratorPreventEtheruemTxMsgs() {
	testCases := []struct {
		name    string
		txSetup func(deps *evmtest.TestDeps) sdk.Tx
		wantErr string
	}{
		{
			name: "sad: evm message",
			txSetup: func(deps *evmtest.TestDeps) sdk.Tx {
				return evmtestutil.HappyTransferTx(deps, 0)
			},
			wantErr: "invalid type",
		},
		{
			name:    "happy: non evm message",
			txSetup: evmtestutil.NonEvmMsgTx,
			wantErr: "",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			deps := evmtest.NewTestDeps()
			anteDec := ante.AnteDecoratorPreventEtheruemTxMsgs{}
			tx := tc.txSetup(&deps)

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
