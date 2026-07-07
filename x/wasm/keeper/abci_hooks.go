package keeper

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"

	sdk "github.com/cosmos/cosmos-sdk/types"

	wasmtypes "github.com/NibiruChain/nibiru/v2/x/wasm/types"
)

const (
	// WasmBlockHookMaxDispatches bounds the number of target sudo calls a
	// registry can request for one block lifecycle hook.
	WasmBlockHookMaxDispatches = 32
	// WasmBlockHookMaxPayloadJSONSize bounds each registry-provided sudo payload.
	WasmBlockHookMaxPayloadJSONSize = 16 * 1024
)

// wasmBlockHookKind identifies which block lifecycle hook is being dispatched.
type wasmBlockHookKind string

const (
	wasmBlockHookBeginBlock wasmBlockHookKind = "begin_block"
	wasmBlockHookEndBlock   wasmBlockHookKind = "end_block"
)

var (
	// wasmBlockHookBeginBlockQuery is the registry smart query for begin-block
	// wasm_sudo_msg_calls.
	wasmBlockHookBeginBlockQuery = []byte(`{"begin_block_plan":{}}`)
	// wasmBlockHookEndBlockQuery is the registry smart query for end-block
	// wasm_sudo_msg_calls.
	wasmBlockHookEndBlockQuery = []byte(`{"end_block_plan":{}}`)
)

// WasmSudoMsgCall is one registry-selected target sudo call.
type WasmSudoMsgCall struct {
	ContractAddr string          `json:"contract_addr"`
	Msg          json.RawMessage `json:"msg"`
}

// ValidatedWasmSudoMsgCall is a sudo call after host-side address and JSON
// payload validation.
type ValidatedWasmSudoMsgCall struct {
	ContractAddr sdk.AccAddress
	Msg          json.RawMessage
}

// WasmBlockHookDispatchFailure records one failed dispatch attempt from the
// registry plan for debugging and indexer replay.
type WasmBlockHookDispatchFailure struct {
	Idx          int    `json:"idx"`
	ContractAddr string `json:"contract_addr"`
	Reason       string `json:"reason"`
	WasmSudoMsg  string `json:"wasm_sudo_msg"`
}

// BeginBlockWasmHooks queries the configured registry contract for a begin-block
// wasm_sudo_msg_calls list and executes each target sudo call in isolation.
func (k Keeper) BeginBlockWasmHooks(ctx sdk.Context) {
	k.runWasmBlockHook(ctx, wasmBlockHookBeginBlock)
}

// EndBlockWasmHooks queries the configured registry contract for an end-block
// wasm_sudo_msg_calls list and executes each target sudo call in isolation.
func (k Keeper) EndBlockWasmHooks(ctx sdk.Context) {
	k.runWasmBlockHook(ctx, wasmBlockHookEndBlock)
}

// runWasmBlockHook loads wasm_sudo_msg_calls for one hook and executes
// each valid target call behind an isolated cache context. Registry query errors
// skip the whole hook, while item validation and target errors are collected in
// the summary failures payload and do not prevent later calls from running.
func (k Keeper) runWasmBlockHook(ctx sdk.Context, hookKind wasmBlockHookKind) {
	wasmSudoMsgCalls, configured, err := k.queryWasmSudoMsgCalls(ctx, hookKind)
	if err != nil {
		// Record a registry query or plan decode failure without aborting block
		// processing.
		ctx.EventManager().EmitEvent(sdk.NewEvent(
			wasmtypes.WasmBlockHookPlanFailedEventType(string(hookKind)),
			sdk.NewAttribute(sdk.AttributeKeyModule, wasmtypes.ModuleName),
			sdk.NewAttribute(wasmtypes.AttributeKeyWasmBlockHookError, err.Error()),
		))
		return
	}
	if !configured {
		return
	}

	failures := make([]WasmBlockHookDispatchFailure, 0)
	for idx, wasmSudoMsg := range wasmSudoMsgCalls {
		validated, err := wasmSudoMsg.Validate(idx)
		if err != nil {
			failures = appendWasmBlockHookDispatchFailure(failures, idx, wasmSudoMsg, err)
			continue
		}

		cacheCtx, writeCacheCtx := ctx.CacheContext()
		cacheCtx = cacheCtx.WithEventManager(sdk.NewEventManager())

		_, err = k.Sudo(cacheCtx, validated.ContractAddr, validated.Msg)
		if err != nil {
			failures = appendWasmBlockHookDispatchFailure(failures, idx, wasmSudoMsg, err)
			continue
		}

		writeCacheCtx()
		ctx.EventManager().EmitEvents(cacheCtx.EventManager().Events())
	}

	attrs := []sdk.Attribute{
		sdk.NewAttribute(sdk.AttributeKeyModule, wasmtypes.ModuleName),
		sdk.NewAttribute(wasmtypes.AttributeKeyWasmBlockHook, string(hookKind)),
		sdk.NewAttribute(wasmtypes.AttributeKeyWasmBlockHookTotal, strconv.Itoa(len(wasmSudoMsgCalls))),
	}
	if len(failures) > 0 {
		failuresJSON, _ := json.Marshal(failures) // json serde is type safe here
		attrs = append(attrs, sdk.NewAttribute(wasmtypes.AttributeKeyWasmBlockHookFailures, string(failuresJSON)))
	}
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(wasmtypes.EventTypeWasmBlockHookSummary, attrs...),
	)
}

// queryWasmSudoMsgCalls asks the configured registry contract for the current
// hook's wasm_sudo_msg_calls. An unset registry is treated as feature-off and
// returns no calls.
func (k Keeper) queryWasmSudoMsgCalls(
	ctx sdk.Context,
	hookKind wasmBlockHookKind,
) (wasmSudoMsgCalls []WasmSudoMsgCall, enabled bool, err error) {
	if k.wasmBlockHooksContractSource == nil {
		return nil, false, nil
	}

	registryAddr, configured := k.wasmBlockHooksContractSource.GetWasmBlockHooksContract(ctx)
	if !configured || registryAddr == nil {
		return nil, false, nil
	}

	queryMsg, err := wasmBlockHookRegistryQueryMsg(hookKind)
	if err != nil {
		return nil, true, err
	}

	queryResp, err := k.QuerySmart(ctx, registryAddr, queryMsg)
	if err != nil {
		return nil, true, fmt.Errorf("query wasm block hook registry: %w", err)
	}

	if err := json.Unmarshal(queryResp, &wasmSudoMsgCalls); err != nil {
		return nil, true, fmt.Errorf("decode wasm block hook registry response: %w", err)
	}
	if len(wasmSudoMsgCalls) > WasmBlockHookMaxDispatches {
		return nil, true, fmt.Errorf(
			"too many wasm sudo msg calls: got %d, max %d",
			len(wasmSudoMsgCalls), WasmBlockHookMaxDispatches,
		)
	}
	return wasmSudoMsgCalls, true, nil
}

// wasmBlockHookRegistryQueryMsg returns the registry smart query for the
// requested block lifecycle hook.
func wasmBlockHookRegistryQueryMsg(hookKind wasmBlockHookKind) ([]byte, error) {
	switch hookKind {
	case wasmBlockHookBeginBlock:
		return wasmBlockHookBeginBlockQuery, nil
	case wasmBlockHookEndBlock:
		return wasmBlockHookEndBlockQuery, nil
	default:
		return nil, fmt.Errorf("unknown wasm block hook kind: %s", hookKind)
	}
}

// ValidateWasmSudoMsgCall enforces fixed v1 host guardrails for one registry
// selected sudo call.
func (wasmSudoMsg WasmSudoMsgCall) Validate(
	idx int,
) (ValidatedWasmSudoMsgCall, error) {
	contractAddr, err := sdk.AccAddressFromBech32(wasmSudoMsg.ContractAddr)
	if err != nil {
		return ValidatedWasmSudoMsgCall{}, fmt.Errorf("wasm sudo msg %d target address: %w", idx, err)
	}
	if len(contractAddr) != wasmtypes.ContractAddrLen {
		return ValidatedWasmSudoMsgCall{}, fmt.Errorf(
			"wasm sudo msg %d target address must be %d bytes, got %d",
			idx, wasmtypes.ContractAddrLen, len(contractAddr),
		)
	}

	msg := bytes.TrimSpace(wasmSudoMsg.Msg)
	if len(msg) == 0 {
		return ValidatedWasmSudoMsgCall{}, fmt.Errorf("wasm sudo msg %d msg cannot be empty", idx)
	}
	if len(msg) > WasmBlockHookMaxPayloadJSONSize {
		return ValidatedWasmSudoMsgCall{}, fmt.Errorf(
			"wasm sudo msg %d msg too large: got %d bytes, max %d",
			idx, len(msg), WasmBlockHookMaxPayloadJSONSize,
		)
	}
	if !json.Valid(msg) {
		return ValidatedWasmSudoMsgCall{}, fmt.Errorf("wasm sudo msg %d msg must be valid JSON", idx)
	}
	if msg[0] != '{' {
		return ValidatedWasmSudoMsgCall{}, fmt.Errorf("wasm sudo msg %d msg must be a JSON object", idx)
	}

	return ValidatedWasmSudoMsgCall{
		ContractAddr: contractAddr,
		Msg:          append(json.RawMessage(nil), msg...),
	}, nil
}

// ValidateWasmSudoMsgCalls enforces list-level bounds and returns valid calls
// plus item-level validation errors. Invalid items do not reject valid siblings.
func ValidateWasmSudoMsgCalls(
	wasmSudoMsgCalls []WasmSudoMsgCall,
) ([]ValidatedWasmSudoMsgCall, []error, error) {
	if len(wasmSudoMsgCalls) > WasmBlockHookMaxDispatches {
		return nil, nil, fmt.Errorf(
			"too many wasm sudo msg calls: got %d, max %d",
			len(wasmSudoMsgCalls), WasmBlockHookMaxDispatches,
		)
	}

	validated := make([]ValidatedWasmSudoMsgCall, 0, len(wasmSudoMsgCalls))
	invalid := make([]error, 0)
	for idx, wasmSudoMsg := range wasmSudoMsgCalls {
		validatedWasmSudoMsg, err := wasmSudoMsg.Validate(idx)
		if err != nil {
			invalid = append(invalid, err)
			continue
		}
		validated = append(validated, validatedWasmSudoMsg)
	}

	return validated, invalid, nil
}

// appendWasmBlockHookDispatchFailure records one failed dispatch with the
// original registry item for replay debugging.
func appendWasmBlockHookDispatchFailure(
	failures []WasmBlockHookDispatchFailure,
	idx int,
	wasmSudoMsg WasmSudoMsgCall,
	reason error,
) []WasmBlockHookDispatchFailure {
	wasmSudoMsgJSON, err := json.Marshal(wasmSudoMsg)
	if err != nil {
		wasmSudoMsgJSON = []byte(fmt.Sprintf(
			`{"contract_addr":%q,"msg":%s}`,
			wasmSudoMsg.ContractAddr,
			wasmSudoMsg.Msg,
		))
	}
	return append(failures, WasmBlockHookDispatchFailure{
		Idx:          idx,
		ContractAddr: wasmSudoMsg.ContractAddr,
		Reason:       reason.Error(),
		WasmSudoMsg:  string(wasmSudoMsgJSON),
	})
}
