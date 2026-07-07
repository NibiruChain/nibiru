package keeper

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	wasmtypes "github.com/NibiruChain/nibiru/v2/x/wasm/types"
)

func TestValidateWasmBlockHookDispatches(t *testing.T) {
	contractAddr := sdk.AccAddress(bytes.Repeat([]byte{1}, wasmtypes.ContractAddrLen)).String()
	sdkLenAddr := RandomAccountAddress(t).String()
	validMsg := json.RawMessage(`{"increment":{"by":7}}`)

	testCases := []struct {
		name       string
		dispatches []wasmBlockHookDispatch
		wantErr    string
	}{
		{
			name: "empty plan",
		},
		{
			name: "single valid dispatch",
			dispatches: []wasmBlockHookDispatch{
				{ContractAddr: contractAddr, Msg: validMsg},
			},
		},
		{
			name: "too many dispatches",
			dispatches: func() []wasmBlockHookDispatch {
				dispatches := make([]wasmBlockHookDispatch, wasmBlockHookMaxDispatches+1)
				for idx := range dispatches {
					dispatches[idx] = wasmBlockHookDispatch{ContractAddr: contractAddr, Msg: validMsg}
				}
				return dispatches
			}(),
			wantErr: "too many wasm block hook dispatches",
		},
		{
			name: "invalid bech32 target",
			dispatches: []wasmBlockHookDispatch{
				{ContractAddr: "not-an-address", Msg: validMsg},
			},
			wantErr: "target address",
		},
		{
			name: "sdk length target",
			dispatches: []wasmBlockHookDispatch{
				{ContractAddr: sdkLenAddr, Msg: validMsg},
			},
			wantErr: "target address must be 32 bytes",
		},
		{
			name: "empty message",
			dispatches: []wasmBlockHookDispatch{
				{ContractAddr: contractAddr},
			},
			wantErr: "msg cannot be empty",
		},
		{
			name: "oversized message",
			dispatches: []wasmBlockHookDispatch{
				{
					ContractAddr: contractAddr,
					Msg:          json.RawMessage(`{"payload":"` + strings.Repeat("x", wasmBlockHookMaxPayloadJSONSize) + `"}`),
				},
			},
			wantErr: "msg too large",
		},
		{
			name: "json string payload",
			dispatches: []wasmBlockHookDispatch{
				{ContractAddr: contractAddr, Msg: json.RawMessage(`"not-a-sudo-object"`)},
			},
			wantErr: "msg must be a JSON object",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := validateWasmBlockHookDispatches(tc.dispatches)
			if tc.wantErr != "" {
				require.ErrorContains(t, err, tc.wantErr)
				return
			}
			require.NoError(t, err)
			require.Len(t, got, len(tc.dispatches))
			for idx := range got {
				require.Equal(t, tc.dispatches[idx].ContractAddr, got[idx].ContractAddr.String())
				require.JSONEq(t, string(tc.dispatches[idx].Msg), string(got[idx].Msg))
			}
		})
	}
}
