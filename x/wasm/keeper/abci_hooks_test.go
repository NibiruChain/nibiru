package keeper_test

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/NibiruChain/nibiru/v2/x/wasm/keeper"
	abci "github.com/cometbft/cometbft/abci/types"
	sdktestdata "github.com/cosmos/cosmos-sdk/testutil/testdata"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/v2/app"
	"github.com/NibiruChain/nibiru/v2/x/nutil/testutil/testapp"
	wasmtestdata "github.com/NibiruChain/nibiru/v2/x/wasm/testdata"
	wasmtypes "github.com/NibiruChain/nibiru/v2/x/wasm/types"
)

func TestWasmBlockHooksTestApp(t *testing.T) {
	testCases := []struct {
		name        string
		hook        string
		mode        string
		configure   bool
		wantCount   uint64
		wantFailure bool
	}{
		{
			name:      "no registry configured",
			hook:      "begin",
			mode:      "single_valid",
			wantCount: 0,
		},
		{
			name:      "begin block empty plan",
			hook:      "begin",
			mode:      "empty",
			configure: true,
			wantCount: 0,
		},
		{
			name:      "begin block single valid dispatch",
			hook:      "begin",
			mode:      "single_valid",
			configure: true,
			wantCount: 7,
		},
		{
			name:      "end block single valid dispatch",
			hook:      "end",
			mode:      "single_valid",
			configure: true,
			wantCount: 7,
		},
		{
			name:        "registry query error skips dispatch",
			hook:        "begin",
			mode:        "query_error",
			configure:   true,
			wantCount:   0,
			wantFailure: true,
		},
		{
			name:        "malformed registry plan skips all dispatch",
			hook:        "begin",
			mode:        "mixed_valid_and_invalid",
			configure:   true,
			wantCount:   0,
			wantFailure: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			wasmApp, ctx := testapp.NewNibiruTestAppAndContext()
			_, _, sender := sdktestdata.KeyTestPubAddr()
			codeID := storeWasmBlockHooksTester(t, wasmApp, ctx, sender)
			targetAddr := instantiateWasmBlockHooksTester(t, wasmApp, ctx, sender, codeID, "empty", nil)
			registryAddr := instantiateWasmBlockHooksTester(t, wasmApp, ctx, sender, codeID, tc.mode, &targetAddr)
			if tc.configure {
				wasmApp.SudoKeeper.WasmBlockHooksContract.Set(ctx, registryAddr.String())
			}

			ctx = ctx.WithEventManager(sdk.NewEventManager())
			events := runWasmBlockHook(t, wasmApp, ctx, tc.hook)

			state := queryWasmBlockHooksTesterState(t, wasmApp, ctx, targetAddr)
			require.Equal(t, tc.wantCount, state.Count)
			require.Equal(t, tc.wantFailure, hasWasmBlockHookFailureEvent(events))
		})
	}
}

type wasmBlockHooksTesterState struct {
	Count    uint64  `json:"count"`
	LastSudo *string `json:"last_sudo"`
}

func storeWasmBlockHooksTester(t *testing.T, wasmApp *app.NibiruApp, ctx sdk.Context, sender sdk.AccAddress) uint64 {
	t.Helper()

	msg := wasmtypes.MsgStoreCodeFixture(func(m *wasmtypes.MsgStoreCode) {
		m.WASMByteCode = wasmtestdata.WasmBlockHooksTesterContractWasm
		m.Sender = sender.String()
	})
	rsp, err := wasmApp.MsgServiceRouter().Handler(msg)(ctx, msg)
	require.NoError(t, err)

	var storeResp wasmtypes.MsgStoreCodeResponse
	require.NoError(t, wasmApp.AppCodec().Unmarshal(rsp.Data, &storeResp))
	return storeResp.CodeID
}

func instantiateWasmBlockHooksTester(
	t *testing.T,
	wasmApp *app.NibiruApp,
	ctx sdk.Context,
	sender sdk.AccAddress,
	codeID uint64,
	registryMode string,
	targetAddr *sdk.AccAddress,
) sdk.AccAddress {
	t.Helper()

	var targetAddrStr *string
	if targetAddr != nil {
		addr := targetAddr.String()
		targetAddrStr = &addr
	}
	initMsg, err := json.Marshal(map[string]any{
		"count":                   uint64(0),
		"registry_mode":           map[string]any{registryMode: map[string]any{}},
		"target_addr":             targetAddrStr,
		"valid_payload_increment": uint64(7),
	})
	require.NoError(t, err)

	msg := &wasmtypes.MsgInstantiateContract{
		Sender: sender.String(),
		CodeID: codeID,
		Label:  "wasm block hooks tester",
		Msg:    initMsg,
		Funds:  sdk.Coins{},
	}
	rsp, err := wasmApp.MsgServiceRouter().Handler(msg)(ctx, msg)
	require.NoError(t, err)

	var instantiateResp wasmtypes.MsgInstantiateContractResponse
	require.NoError(t, wasmApp.AppCodec().Unmarshal(rsp.Data, &instantiateResp))
	return sdk.MustAccAddressFromBech32(instantiateResp.Address)
}

func runWasmBlockHook(t *testing.T, wasmApp *app.NibiruApp, ctx sdk.Context, hook string) []abci.Event {
	t.Helper()

	switch hook {
	case "begin":
		return wasmApp.BeginBlocker(ctx, abci.RequestBeginBlock{Header: ctx.BlockHeader()}).Events
	case "end":
		return wasmApp.EndBlocker(ctx, abci.RequestEndBlock{Height: ctx.BlockHeight()}).Events
	default:
		t.Fatalf("unexpected wasm block hook kind: %s", hook)
	}
	return nil
}

func queryWasmBlockHooksTesterState(
	t *testing.T,
	wasmApp *app.NibiruApp,
	ctx sdk.Context,
	contractAddr sdk.AccAddress,
) wasmBlockHooksTesterState {
	t.Helper()

	queryResp, err := wasmApp.WasmKeeper.QuerySmart(ctx, contractAddr, []byte(`{"state":{}}`))
	require.NoError(t, err)

	var state wasmBlockHooksTesterState
	require.NoError(t, json.Unmarshal(queryResp, &state))
	return state
}

func hasWasmBlockHookFailureEvent(events []abci.Event) bool {
	for _, event := range events {
		if event.Type == wasmtypes.EventTypeWasmBlockHookFailure {
			return true
		}
	}
	return false
}

func TestValidateWasmBlockHookDispatches(t *testing.T) {
	contractAddr := sdk.AccAddress(bytes.Repeat([]byte{1}, wasmtypes.ContractAddrLen)).String()
	sdkLenAddr := sdk.AccAddress(bytes.Repeat([]byte{2}, 20)).String()
	validMsg := json.RawMessage(`{"increment":{"by":7}}`)

	testCases := []struct {
		name       string
		dispatches []keeper.WasmBlockHookDispatch
		wantErr    string
	}{
		{
			name: "empty plan",
		},
		{
			name: "single valid dispatch",
			dispatches: []keeper.WasmBlockHookDispatch{
				{ContractAddr: contractAddr, Msg: validMsg},
			},
		},
		{
			name: "too many dispatches",
			dispatches: func() []keeper.WasmBlockHookDispatch {
				dispatches := make([]keeper.WasmBlockHookDispatch, keeper.WasmBlockHookMaxDispatches+1)
				for idx := range dispatches {
					dispatches[idx] = keeper.WasmBlockHookDispatch{ContractAddr: contractAddr, Msg: validMsg}
				}
				return dispatches
			}(),
			wantErr: "too many wasm block hook dispatches",
		},
		{
			name: "invalid bech32 target",
			dispatches: []keeper.WasmBlockHookDispatch{
				{ContractAddr: "not-an-address", Msg: validMsg},
			},
			wantErr: "target address",
		},
		{
			name: "sdk length target",
			dispatches: []keeper.WasmBlockHookDispatch{
				{ContractAddr: sdkLenAddr, Msg: validMsg},
			},
			wantErr: "target address must be 32 bytes",
		},
		{
			name: "empty message",
			dispatches: []keeper.WasmBlockHookDispatch{
				{ContractAddr: contractAddr},
			},
			wantErr: "msg cannot be empty",
		},
		{
			name: "oversized message",
			dispatches: []keeper.WasmBlockHookDispatch{
				{
					ContractAddr: contractAddr,
					Msg:          json.RawMessage(`{"payload":"` + strings.Repeat("x", keeper.WasmBlockHookMaxPayloadJSONSize) + `"}`),
				},
			},
			wantErr: "msg too large",
		},
		{
			name: "json string payload",
			dispatches: []keeper.WasmBlockHookDispatch{
				{ContractAddr: contractAddr, Msg: json.RawMessage(`"not-a-sudo-object"`)},
			},
			wantErr: "msg must be a JSON object",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := keeper.ValidateWasmBlockHookDispatches(tc.dispatches)
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
