package wasm_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	sdk "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/v2/x/wasm/types"
)

const firstCodeID = 1

type ibcReflectInitMsg struct {
	ReflectCodeID uint64 `json:"reflect_code_id"`
}

func (m ibcReflectInitMsg) getBytes(tb testing.TB) []byte {
	tb.Helper()

	initMsgBz, err := json.Marshal(m)
	require.NoError(tb, err)
	return initMsgBz
}

// ensure store code returns the expected response
func assertStoreCodeResponse(t *testing.T, data []byte, expected uint64) {
	t.Helper()
	var pStoreResp types.MsgStoreCodeResponse
	require.NoError(t, pStoreResp.Unmarshal(data))
	require.Equal(t, pStoreResp.CodeID, expected)
}

// ensure execution returns the expected data
func assertExecuteResponse(t *testing.T, data, expected []byte) {
	t.Helper()
	var pExecResp types.MsgExecuteContractResponse
	require.NoError(t, pExecResp.Unmarshal(data))
	require.Equal(t, pExecResp.Data, expected)
}

// ensures this returns a valid bech32 address and returns it
func parseInitResponse(t *testing.T, data []byte) string {
	t.Helper()
	var pInstResp types.MsgInstantiateContractResponse
	require.NoError(t, pInstResp.Unmarshal(data))
	require.NotEmpty(t, pInstResp.Address)
	addr := pInstResp.Address
	// ensure this is a valid sdk address
	_, err := sdk.AccAddressFromBech32(addr)
	require.NoError(t, err)
	return addr
}
