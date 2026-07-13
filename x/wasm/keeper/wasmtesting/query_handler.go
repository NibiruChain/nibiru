package wasmtesting

import (
	wasmvmtypes "github.com/NibiruChain/nibiru/v2/lib/wasmvm-ffi/wvm"

	sdk "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types"
)

type MockQueryHandler struct {
	HandleQueryFn func(ctx sdk.Context, request wasmvmtypes.QueryRequest, caller sdk.AccAddress) ([]byte, error)
}

func (m *MockQueryHandler) HandleQuery(ctx sdk.Context, caller sdk.AccAddress, request wasmvmtypes.QueryRequest) ([]byte, error) {
	if m.HandleQueryFn == nil {
		panic("not expected to be called")
	}
	return m.HandleQueryFn(ctx, request, caller)
}
