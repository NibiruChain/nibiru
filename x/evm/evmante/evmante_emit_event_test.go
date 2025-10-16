package evmante_test

import (
	"github.com/NibiruChain/nibiru/v2/x/evm"
	"github.com/NibiruChain/nibiru/v2/x/evm/evmante"
	"github.com/NibiruChain/nibiru/v2/x/evm/evmstate"
	"github.com/NibiruChain/nibiru/v2/x/evm/evmtest"
)

func (s *Suite) TestEthAnteEmitPendingEvent() {
	testCases := []AnteTC{
		{
			Name:        "happy: eth tx emitted event",
			EvmAnteStep: evmante.AnteStepEmitPendingEvent,
			TxSetup: func(deps *evmtest.TestDeps, sdb *evmstate.SDB) evm.Tx {
				tx := evmtest.HappyCreateContractTx(deps)
				return tx
			},
			WantErr: "",
			OnEnd: func(sdb *evmstate.SDB, tx evm.Tx) {
				// Verify that the pending ethereum tx event was emitted
				events := sdb.Ctx().EventManager().Events()
				s.Require().Greater(len(events), 0)
				event := events[len(events)-1]
				s.Require().Equal(evm.PendingEthereumTxEvent, event.Type)

				// TX hash attr must present
				attr, ok := event.GetAttribute(evm.PendingEthereumTxEventAttrEthHash)
				s.Require().True(ok, "tx hash attribute not found")
				s.Require().Equal(tx.Hash, attr.Value)

				// TX index attr must present
				attr, ok = event.GetAttribute(evm.PendingEthereumTxEventAttrIndex)
				s.Require().True(ok, "tx index attribute not found")
				s.Require().Equal("0", attr.Value)
			},
		},
	}

	RunAnteTCs(&s.Suite, testCases)
}
