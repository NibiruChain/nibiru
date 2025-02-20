package evmante_test

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth/migrations/legacytx"

	"github.com/NibiruChain/nibiru/v2/app/evmante"
	"github.com/NibiruChain/nibiru/v2/x/evm"

	"github.com/NibiruChain/nibiru/v2/x/evm/evmtest"
	tf "github.com/NibiruChain/nibiru/v2/x/tokenfactory/types"
)

func (s *TestSuite) TestEthEmitEventDecorator() {
	testCases := []struct {
		name    string
		txSetup func(deps *evmtest.TestDeps) sdk.Tx
		wantErr string
	}{
		{
			name: "sad: non ethereum tx",
			txSetup: func(deps *evmtest.TestDeps) sdk.Tx {
				return legacytx.StdTx{
					Msgs: []sdk.Msg{
						&tf.MsgMint{},
					},
				}
			},
			wantErr: "invalid message",
		},
		{
			name: "happy: eth tx emitted event",
			txSetup: func(deps *evmtest.TestDeps) sdk.Tx {
				tx := evmtest.HappyCreateContractTx(deps)
				return tx
			},
			wantErr: "",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			deps := evmtest.NewTestDeps()
			stateDB := deps.NewStateDB()
			anteDec := evmante.NewEthEmitEventDecorator(deps.App.AppKeepers.EvmKeeper)

			tx := tc.txSetup(&deps)
			s.Require().NoError(stateDB.Commit())

			_, err := anteDec.AnteHandle(
				deps.Ctx, tx, false, evmtest.NextNoOpAnteHandler,
			)
			if tc.wantErr != "" {
				s.Require().ErrorContains(err, tc.wantErr)
				return
			}
			s.Require().NoError(err)
			events := deps.Ctx.EventManager().Events()

			s.Require().Greater(len(events), 0)
			event := events[len(events)-1]
			s.Require().Equal(evm.PendingEthereumTxEvent, event.Type)

			// Convert tx to msg to get hash
			txMsg, ok := tx.GetMsgs()[0].(*evm.MsgEthereumTx)
			s.Require().True(ok)

			// TX hash attr must present
			attr, ok := event.GetAttribute(evm.PendingEthereumTxEventAttrEthHash)
			s.Require().True(ok, "tx hash attribute not found")
			s.Require().Equal(txMsg.Hash, attr.Value)

			// TX index attr must present
			attr, ok = event.GetAttribute(evm.PendingEthereumTxEventAttrIndex)
			s.Require().True(ok, "tx index attribute not found")
			s.Require().Equal("0", attr.Value)
		})
	}
}
