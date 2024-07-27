package ante_test

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/app/ante"
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
				return evmtest.HappyTransferTx(deps, 0)
			},
			wantErr: "invalid type",
		},
		{
			name:    "happy: non evm message",
			txSetup: evmtest.NonEvmMsgTx,
			wantErr: "",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			deps := evmtest.NewTestDeps()
			anteDec := ante.AnteDecoratorPreventEtheruemTxMsgs{}
			tx := tc.txSetup(&deps)

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
