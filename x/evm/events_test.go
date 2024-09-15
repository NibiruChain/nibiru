package evm_test

import (
	abci "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/v2/x/evm"
)

func (s *Suite) TestFromABCIEvent() {
	for _, tc := range []struct {
		name string
		test func()
	}{
		{
			name: "EventTxLog - happy",
			test: func() {
				typed := &evm.EventTxLog{
					TxLogs: []string{},
				}

				e, err := sdk.TypedEventToEvent(typed)
				s.NoError(err)

				gotTyped, err := new(evm.EventTxLog).FromABCIEvent(abci.Event(e))
				s.NoError(err)

				s.EqualValues(typed, gotTyped)
			},
		},
	} {
		s.Run(tc.name, tc.test)
	}
}
