package wasmtesting

import (
	"github.com/NibiruChain/nibiru/v2/lib/wasmvm/wvm"

	sdk "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types"
)

type MockMsgDispatcher struct {
	DispatchSubmessagesFn func(ctx sdk.Context, contractAddr sdk.AccAddress, ibcPort string, msgs []wvm.SubMsg) ([]byte, error)
}

func (m MockMsgDispatcher) DispatchSubmessages(ctx sdk.Context, contractAddr sdk.AccAddress, ibcPort string, msgs []wvm.SubMsg) ([]byte, error) {
	if m.DispatchSubmessagesFn == nil {
		panic("not expected to be called")
	}
	return m.DispatchSubmessagesFn(ctx, contractAddr, ibcPort, msgs)
}
