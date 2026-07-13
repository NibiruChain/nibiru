package wasmtesting

import (
	"errors"

	"github.com/NibiruChain/nibiru/v2/lib/wasmvm/wvm"

	sdk "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types"
)

type MockMessageHandler struct {
	DispatchMsgFn func(ctx sdk.Context, contractAddr sdk.AccAddress, contractIBCPortID string, msg wvm.CosmosMsg) (events []sdk.Event, data [][]byte, err error)
}

func (m *MockMessageHandler) DispatchMsg(ctx sdk.Context, contractAddr sdk.AccAddress, contractIBCPortID string, msg wvm.CosmosMsg) (events []sdk.Event, data [][]byte, err error) {
	if m.DispatchMsgFn == nil {
		panic("not expected to be called")
	}
	return m.DispatchMsgFn(ctx, contractAddr, contractIBCPortID, msg)
}

func NewCapturingMessageHandler() (*MockMessageHandler, *[]wvm.CosmosMsg) {
	var messages []wvm.CosmosMsg
	return &MockMessageHandler{
		DispatchMsgFn: func(ctx sdk.Context, contractAddr sdk.AccAddress, contractIBCPortID string, msg wvm.CosmosMsg) (events []sdk.Event, data [][]byte, err error) {
			messages = append(messages, msg)
			// return one data item so that this doesn't cause an error in submessage processing (it takes the first element from data)
			return nil, [][]byte{{1}}, nil
		},
	}, &messages
}

func NewErroringMessageHandler() *MockMessageHandler {
	return &MockMessageHandler{
		DispatchMsgFn: func(ctx sdk.Context, contractAddr sdk.AccAddress, contractIBCPortID string, msg wvm.CosmosMsg) (events []sdk.Event, data [][]byte, err error) {
			return nil, nil, errors.New("test, ignore")
		},
	}
}
