package keeper_test

import (
	"bytes"
	"encoding/json"
	"strconv"
	"strings"
	"testing"

	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/stretchr/testify/require"

	sdktestdata "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/testutil/testdata"
	sdk "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/v2/x/wasm/keeper"

	"github.com/NibiruChain/nibiru/v2/app"
	"github.com/NibiruChain/nibiru/v2/x/nutil/testapp"
	"github.com/NibiruChain/nibiru/v2/x/nutil/testutil"
	wasmtestdata "github.com/NibiruChain/nibiru/v2/x/wasm/testdata"
	wasmtypes "github.com/NibiruChain/nibiru/v2/x/wasm/types"
)

func TestWasmBlockHooksTestApp(t *testing.T) {
	t.Run("no registry configured", func(t *testing.T) {
		wasmApp, ctx := testapp.NewNibiruTestAppAndContext()
		ctx = ctx.WithEventManager(sdk.NewEventManager())

		events := runWasmBlockHook(t, wasmApp, ctx, "begin")

		require.Empty(t, testutil.FindAbciEventsOfType(events, wasmtypes.WasmBlockHookPlanFailedEventType("begin_block")))
		require.Empty(t, testutil.FindAbciEventsOfType(events, wasmtypes.EventTypeWasmBlockHookSummary))
	})

	t.Run("invalid stored registry address is treated as unconfigured", func(t *testing.T) {
		wasmApp, ctx := testapp.NewNibiruTestAppAndContext()
		wasmApp.SudoKeeper.WasmBlockHooksContract.Set(ctx, "not-a-contract-address")
		ctx = ctx.WithEventManager(sdk.NewEventManager())

		events := runWasmBlockHook(t, wasmApp, ctx, "begin")

		require.Empty(t, testutil.FindAbciEventsOfType(events, wasmtypes.WasmBlockHookPlanFailedEventType("begin_block")))
		require.Empty(t, testutil.FindAbciEventsOfType(events, wasmtypes.EventTypeWasmBlockHookSummary))
	})

	t.Run("empty configured calls", func(t *testing.T) {
		wasmApp, ctx, sender, codeID := setupWasmBlockHooksTester(t)
		registryAddr := instantiateWasmBlockHooksTester(t, wasmApp, ctx, sender, codeID)
		executeWasmBlockHooksTesterConfig(t, wasmApp, ctx, sender, registryAddr, nil, ptr(false), []keeper.WasmSudoMsgCall{})
		wasmApp.SudoKeeper.WasmBlockHooksContract.Set(ctx, registryAddr.String())
		ctx = ctx.WithEventManager(sdk.NewEventManager())

		events := runWasmBlockHook(t, wasmApp, ctx, "begin")

		require.Empty(t, testutil.FindAbciEventsOfType(events, wasmtypes.WasmBlockHookPlanFailedEventType("begin_block")))
		assertWasmBlockHookSummary(t, events, "begin_block", 0, nil)
	})

	t.Run("single valid begin call", func(t *testing.T) {
		wasmApp, ctx, sender, codeID := setupWasmBlockHooksTester(t)
		targetAddr := instantiateWasmBlockHooksTester(t, wasmApp, ctx, sender, codeID)
		registryAddr := instantiateWasmBlockHooksTester(t, wasmApp, ctx, sender, codeID)
		executeWasmBlockHooksTesterConfig(t, wasmApp, ctx, sender, registryAddr, nil, ptr(false), []keeper.WasmSudoMsgCall{
			{ContractAddr: targetAddr.String(), Msg: json.RawMessage(`{"increment":{"by":7}}`)},
		})
		wasmApp.SudoKeeper.WasmBlockHooksContract.Set(ctx, registryAddr.String())
		ctx = ctx.WithEventManager(sdk.NewEventManager())

		events := runWasmBlockHook(t, wasmApp, ctx, "begin")

		require.Equal(t, uint64(7), queryWasmBlockHooksTesterState(t, wasmApp, ctx, targetAddr).Count)
		require.Empty(t, testutil.FindAbciEventsOfType(events, wasmtypes.WasmBlockHookPlanFailedEventType("begin_block")))
		assertWasmBlockHookSummary(t, events, "begin_block", 1, nil)
	})

	t.Run("single valid end call", func(t *testing.T) {
		wasmApp, ctx, sender, codeID := setupWasmBlockHooksTester(t)
		targetAddr := instantiateWasmBlockHooksTester(t, wasmApp, ctx, sender, codeID)
		registryAddr := instantiateWasmBlockHooksTester(t, wasmApp, ctx, sender, codeID)
		executeWasmBlockHooksTesterConfig(t, wasmApp, ctx, sender, registryAddr, nil, ptr(false), []keeper.WasmSudoMsgCall{
			{ContractAddr: targetAddr.String(), Msg: json.RawMessage(`{"increment":{"by":7}}`)},
		})
		wasmApp.SudoKeeper.WasmBlockHooksContract.Set(ctx, registryAddr.String())
		ctx = ctx.WithEventManager(sdk.NewEventManager())

		events := runWasmBlockHook(t, wasmApp, ctx, "end")

		require.Equal(t, uint64(7), queryWasmBlockHooksTesterState(t, wasmApp, ctx, targetAddr).Count)
		require.Empty(t, testutil.FindAbciEventsOfType(events, wasmtypes.WasmBlockHookPlanFailedEventType("end_block")))
		assertWasmBlockHookSummary(t, events, "end_block", 1, nil)
	})

	t.Run("registry query error skips calls", func(t *testing.T) {
		wasmApp, ctx, sender, codeID := setupWasmBlockHooksTester(t)
		registryAddr := instantiateWasmBlockHooksTester(t, wasmApp, ctx, sender, codeID)
		executeWasmBlockHooksTesterConfig(t, wasmApp, ctx, sender, registryAddr, nil, ptr(true), nil)
		wasmApp.SudoKeeper.WasmBlockHooksContract.Set(ctx, registryAddr.String())
		ctx = ctx.WithEventManager(sdk.NewEventManager())

		events := runWasmBlockHook(t, wasmApp, ctx, "begin")

		require.NotEmpty(t, testutil.FindAbciEventsOfType(events, wasmtypes.WasmBlockHookPlanFailedEventType("begin_block")))
		require.Empty(t, testutil.FindAbciEventsOfType(events, wasmtypes.EventTypeWasmBlockHookSummary))
	})

	t.Run("invalid calls do not block valid calls", func(t *testing.T) {
		wasmApp, ctx, sender, codeID := setupWasmBlockHooksTester(t)
		targetA := instantiateWasmBlockHooksTester(t, wasmApp, ctx, sender, codeID)
		targetB := instantiateWasmBlockHooksTester(t, wasmApp, ctx, sender, codeID)
		registryAddr := instantiateWasmBlockHooksTester(t, wasmApp, ctx, sender, codeID)
		executeWasmBlockHooksTesterConfig(t, wasmApp, ctx, sender, registryAddr, nil, ptr(false), []keeper.WasmSudoMsgCall{
			{ContractAddr: targetA.String(), Msg: json.RawMessage(`{"increment":{"by":7}}`)},
			{ContractAddr: "not-a-wasm-contract-address", Msg: json.RawMessage(`{"increment":{"by":99}}`)},
			{ContractAddr: targetB.String(), Msg: json.RawMessage(`"not-a-sudo-object"`)},
			{ContractAddr: targetB.String(), Msg: json.RawMessage(`{"increment":{"by":3}}`)},
		})
		wasmApp.SudoKeeper.WasmBlockHooksContract.Set(ctx, registryAddr.String())
		ctx = ctx.WithEventManager(sdk.NewEventManager())

		events := runWasmBlockHook(t, wasmApp, ctx, "end")

		require.Equal(t, uint64(7), queryWasmBlockHooksTesterState(t, wasmApp, ctx, targetA).Count)
		require.Equal(t, uint64(3), queryWasmBlockHooksTesterState(t, wasmApp, ctx, targetB).Count)
		assertWasmBlockHookSummary(t, events, "end_block", 4, []keeper.WasmBlockHookDispatchFailure{
			{Idx: 1, ContractAddr: "not-a-wasm-contract-address"},
			{Idx: 2, ContractAddr: targetB.String()},
		})
	})

	t.Run("target failure rolls back and later call continues", func(t *testing.T) {
		wasmApp, ctx, sender, codeID := setupWasmBlockHooksTester(t)
		targetA := instantiateWasmBlockHooksTester(t, wasmApp, ctx, sender, codeID)
		targetB := instantiateWasmBlockHooksTester(t, wasmApp, ctx, sender, codeID)
		targetC := instantiateWasmBlockHooksTester(t, wasmApp, ctx, sender, codeID)
		registryAddr := instantiateWasmBlockHooksTester(t, wasmApp, ctx, sender, codeID)
		executeWasmBlockHooksTesterConfig(t, wasmApp, ctx, sender, registryAddr, nil, ptr(false), []keeper.WasmSudoMsgCall{
			{ContractAddr: targetA.String(), Msg: json.RawMessage(`{"increment":{"by":7}}`)},
			{ContractAddr: targetB.String(), Msg: json.RawMessage(`{"fail_after_write":{"by":9}}`)},
			{ContractAddr: targetC.String(), Msg: json.RawMessage(`{"increment":{"by":3}}`)},
		})
		wasmApp.SudoKeeper.WasmBlockHooksContract.Set(ctx, registryAddr.String())
		ctx = ctx.WithEventManager(sdk.NewEventManager())

		events := runWasmBlockHook(t, wasmApp, ctx, "end")

		require.Equal(t, uint64(7), queryWasmBlockHooksTesterState(t, wasmApp, ctx, targetA).Count)
		require.Equal(t, uint64(0), queryWasmBlockHooksTesterState(t, wasmApp, ctx, targetB).Count)
		require.Nil(t, queryWasmBlockHooksTesterState(t, wasmApp, ctx, targetB).LastSudo)
		require.Equal(t, uint64(3), queryWasmBlockHooksTesterState(t, wasmApp, ctx, targetC).Count)
		require.False(t, hasWasmEventAttribute(events, targetB.String(), "method", "fail_after_write"))
		assertWasmBlockHookSummary(t, events, "end_block", 3, []keeper.WasmBlockHookDispatchFailure{
			{Idx: 1, ContractAddr: targetB.String()},
		})
	})
}

type wasmBlockHooksTesterState struct {
	Count    uint64  `json:"count"`
	LastSudo *string `json:"last_sudo"`
}

func setupWasmBlockHooksTester(t *testing.T) (*app.NibiruApp, sdk.Context, sdk.AccAddress, uint64) {
	t.Helper()

	wasmApp, ctx := testapp.NewNibiruTestAppAndContext()
	_, _, sender := sdktestdata.KeyTestPubAddr()
	codeID := storeWasmBlockHooksTester(t, wasmApp, ctx, sender)
	return wasmApp, ctx, sender, codeID
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
) sdk.AccAddress {
	t.Helper()

	msg := &wasmtypes.MsgInstantiateContract{
		Sender: sender.String(),
		CodeID: codeID,
		Label:  "wasm block hooks tester",
		Msg:    []byte(`{}`),
		Funds:  sdk.Coins{},
	}
	rsp, err := wasmApp.MsgServiceRouter().Handler(msg)(ctx, msg)
	require.NoError(t, err)

	var instantiateResp wasmtypes.MsgInstantiateContractResponse
	require.NoError(t, wasmApp.AppCodec().Unmarshal(rsp.Data, &instantiateResp))
	return sdk.MustAccAddressFromBech32(instantiateResp.Address)
}

func executeWasmBlockHooksTesterConfig(
	t *testing.T,
	wasmApp *app.NibiruApp,
	ctx sdk.Context,
	sender sdk.AccAddress,
	contractAddr sdk.AccAddress,
	count *uint64,
	queryError *bool,
	wasmSudoMsgCalls []keeper.WasmSudoMsgCall,
) {
	t.Helper()

	configMsg := map[string]any{
		"config": map[string]any{
			"count":               count,
			"query_error":         queryError,
			"wasm_sudo_msg_calls": wasmSudoMsgCalls,
		},
	}
	msgBz, err := json.Marshal(configMsg)
	require.NoError(t, err)
	msg := &wasmtypes.MsgExecuteContract{
		Sender:   sender.String(),
		Contract: contractAddr.String(),
		Msg:      msgBz,
		Funds:    sdk.Coins{},
	}
	_, err = wasmApp.MsgServiceRouter().Handler(msg)(ctx, msg)
	require.NoError(t, err)
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

func assertWasmBlockHookSummary(
	t *testing.T,
	events []abci.Event,
	hook string,
	total int,
	wantFailures []keeper.WasmBlockHookDispatchFailure,
) {
	t.Helper()

	summaryEvents := testutil.FindAbciEventsOfType(events, wasmtypes.EventTypeWasmBlockHookSummary)
	require.Len(t, summaryEvents, 1)
	attrs := eventAttributes(summaryEvents[0])
	require.Equal(t, hook, attrs[wasmtypes.AttributeKeyWasmBlockHook])
	require.Equal(t, strconv.Itoa(total), attrs[wasmtypes.AttributeKeyWasmBlockHookTotal])

	if len(wantFailures) == 0 {
		_, hasFailures := attrs[wasmtypes.AttributeKeyWasmBlockHookFailures]
		require.False(t, hasFailures)
		return
	}

	failuresJSON, ok := attrs[wasmtypes.AttributeKeyWasmBlockHookFailures]
	require.True(t, ok, "expected failures attribute on summary event")

	var gotFailures []keeper.WasmBlockHookDispatchFailure
	require.NoError(t, json.Unmarshal([]byte(failuresJSON), &gotFailures))
	require.Len(t, gotFailures, len(wantFailures))

	for idx, want := range wantFailures {
		got := gotFailures[idx]
		require.Equal(t, want.Idx, got.Idx)
		require.Equal(t, want.ContractAddr, got.ContractAddr)
		if want.Reason != "" {
			require.Equal(t, want.Reason, got.Reason)
		} else {
			require.NotEmpty(t, got.Reason)
		}
		if want.WasmSudoMsg != "" {
			require.Equal(t, want.WasmSudoMsg, got.WasmSudoMsg)
		} else {
			require.NotEmpty(t, got.WasmSudoMsg)
			require.True(t, json.Valid([]byte(got.WasmSudoMsg)))
		}
	}
}

func hasWasmEventAttribute(events []abci.Event, contractAddr string, key string, value string) bool {
	for _, event := range events {
		if event.Type != wasmtypes.WasmModuleEventType {
			continue
		}
		attrs := eventAttributes(event)
		if attrs[wasmtypes.AttributeKeyContractAddr] == contractAddr && attrs[key] == value {
			return true
		}
	}
	return false
}

func eventAttributes(event abci.Event) map[string]string {
	attrs := map[string]string{}
	for _, attr := range event.Attributes {
		attrs[attr.Key] = attr.Value
	}
	return attrs
}

func ptr[T any](v T) *T {
	return &v
}

func TestValidateWasmSudoMsgCalls(t *testing.T) {
	contractAddr := sdk.AccAddress(bytes.Repeat([]byte{1}, wasmtypes.ContractAddrLen)).String()
	sdkLenAddr := sdk.AccAddress(bytes.Repeat([]byte{2}, 20)).String()
	validMsg := json.RawMessage(`{"increment":{"by":7}}`)

	testCases := []struct {
		name        string
		calls       []keeper.WasmSudoMsgCall
		wantValid   int
		wantInvalid int
		wantErr     string
	}{
		{
			name: "empty calls",
		},
		{
			name: "single valid call",
			calls: []keeper.WasmSudoMsgCall{
				{ContractAddr: contractAddr, Msg: validMsg},
			},
			wantValid: 1,
		},
		{
			name: "too many calls",
			calls: func() []keeper.WasmSudoMsgCall {
				calls := make([]keeper.WasmSudoMsgCall, keeper.WasmBlockHookMaxDispatches+1)
				for idx := range calls {
					calls[idx] = keeper.WasmSudoMsgCall{ContractAddr: contractAddr, Msg: validMsg}
				}
				return calls
			}(),
			wantErr: "too many wasm sudo msg calls",
		},
		{
			name: "invalid bech32 target",
			calls: []keeper.WasmSudoMsgCall{
				{ContractAddr: "not-an-address", Msg: validMsg},
			},
			wantInvalid: 1,
		},
		{
			name: "valid and invalid siblings",
			calls: []keeper.WasmSudoMsgCall{
				{ContractAddr: contractAddr, Msg: validMsg},
				{ContractAddr: "not-an-address", Msg: validMsg},
			},
			wantValid:   1,
			wantInvalid: 1,
		},
		{
			name: "sdk length target",
			calls: []keeper.WasmSudoMsgCall{
				{ContractAddr: sdkLenAddr, Msg: validMsg},
			},
			wantInvalid: 1,
		},
		{
			name: "empty message",
			calls: []keeper.WasmSudoMsgCall{
				{ContractAddr: contractAddr},
			},
			wantInvalid: 1,
		},
		{
			name: "oversized message",
			calls: []keeper.WasmSudoMsgCall{
				{
					ContractAddr: contractAddr,
					Msg:          json.RawMessage(`{"payload":"` + strings.Repeat("x", keeper.WasmBlockHookMaxPayloadJSONSize) + `"}`),
				},
			},
			wantInvalid: 1,
		},
		{
			name: "json string payload",
			calls: []keeper.WasmSudoMsgCall{
				{ContractAddr: contractAddr, Msg: json.RawMessage(`"not-a-sudo-object"`)},
			},
			wantInvalid: 1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, invalid, err := keeper.ValidateWasmSudoMsgCalls(tc.calls)
			if tc.wantErr != "" {
				require.ErrorContains(t, err, tc.wantErr)
				return
			}
			require.NoError(t, err)
			require.Len(t, got, tc.wantValid)
			require.Len(t, invalid, tc.wantInvalid)
			for _, valid := range got {
				require.NotEmpty(t, valid.ContractAddr)
				require.True(t, json.Valid(valid.Msg))
			}
		})
	}
}
