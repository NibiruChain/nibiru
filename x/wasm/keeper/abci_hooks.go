package keeper

import (
	"bytes"
	"encoding/json"
	"fmt"

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
	// dispatch plans.
	wasmBlockHookBeginBlockQuery = []byte(`{"begin_block_plan":{}}`)
	// wasmBlockHookEndBlockQuery is the registry smart query for end-block
	// dispatch plans.
	wasmBlockHookEndBlockQuery = []byte(`{"end_block_plan":{}}`)
)

// WasmBlockHookDispatch is one registry-selected target sudo call.
type WasmBlockHookDispatch struct {
	ContractAddr string          `json:"contract_addr"`
	Msg          json.RawMessage `json:"msg"`
}

// ValidatedWasmBlockHookDispatch is a dispatch item after host-side address and
// JSON payload validation.
type ValidatedWasmBlockHookDispatch struct {
	ContractAddr sdk.AccAddress
	Msg          json.RawMessage
}

// BeginBlockWasmHooks queries the configured registry contract for a begin-block
// dispatch plan and executes each target sudo call in an isolated cache context.
func (k Keeper) BeginBlockWasmHooks(ctx sdk.Context) {
	k.runWasmBlockHookPlanner(ctx, wasmBlockHookBeginBlock)
}

// EndBlockWasmHooks queries the configured registry contract for an end-block
// dispatch plan and executes each target sudo call in an isolated cache context.
func (k Keeper) EndBlockWasmHooks(ctx sdk.Context) {
	k.runWasmBlockHookPlanner(ctx, wasmBlockHookEndBlock)
}

// runWasmBlockHookPlanner loads a registry plan for one hook and executes each
// validated target call behind an isolated cache context. Registry or plan
// errors skip the whole hook, while target errors discard only that target's
// writes/events and allow later dispatches to run.
func (k Keeper) runWasmBlockHookPlanner(ctx sdk.Context, hookKind wasmBlockHookKind) {
	dispatches, err := k.queryWasmBlockHookPlan(ctx, hookKind)
	if err != nil {
		emitWasmBlockHookFailureEvent(ctx, hookKind, err)
		return
	}

	for _, dispatch := range dispatches {
		cacheCtx, writeCache := ctx.CacheContext()
		cacheCtx = cacheCtx.WithEventManager(sdk.NewEventManager())

		_, err := k.Sudo(cacheCtx, dispatch.ContractAddr, dispatch.Msg)
		if err != nil {
			emitWasmBlockHookFailureEvent(ctx, hookKind, err, dispatch.ContractAddr)
			continue
		}

		writeCache()
		ctx.EventManager().EmitEvents(cacheCtx.EventManager().Events())
	}
}

// queryWasmBlockHookPlan asks the configured registry contract for the current
// hook's dispatch plan. An unset registry is treated as feature-off and returns
// no dispatches.
func (k Keeper) queryWasmBlockHookPlan(
	ctx sdk.Context,
	hookKind wasmBlockHookKind,
) ([]ValidatedWasmBlockHookDispatch, error) {
	if k.wasmBlockHooksContractSource == nil {
		return nil, nil
	}

	registryAddr, configured := k.wasmBlockHooksContractSource.GetWasmBlockHooksContract(ctx)
	if !configured {
		return nil, nil
	}

	queryMsg, err := wasmBlockHookRegistryQueryMsg(hookKind)
	if err != nil {
		return nil, err
	}

	queryResp, err := k.QuerySmart(ctx, registryAddr, queryMsg)
	if err != nil {
		return nil, fmt.Errorf("query wasm block hook registry: %w", err)
	}

	var dispatches []WasmBlockHookDispatch
	if err := json.Unmarshal(queryResp, &dispatches); err != nil {
		return nil, fmt.Errorf("decode wasm block hook registry response: %w", err)
	}

	return ValidateWasmBlockHookDispatches(dispatches)
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

// ValidateWasmBlockHookDispatches enforces the fixed v1 host guardrails before
// any target sudo execution begins. Any invalid item rejects the whole plan for
// the current block hook.
func ValidateWasmBlockHookDispatches(
	dispatches []WasmBlockHookDispatch,
) ([]ValidatedWasmBlockHookDispatch, error) {
	if len(dispatches) > WasmBlockHookMaxDispatches {
		return nil, fmt.Errorf(
			"too many wasm block hook dispatches: got %d, max %d",
			len(dispatches), WasmBlockHookMaxDispatches,
		)
	}

	validated := make([]ValidatedWasmBlockHookDispatch, 0, len(dispatches))
	for idx, dispatch := range dispatches {
		contractAddr, err := sdk.AccAddressFromBech32(dispatch.ContractAddr)
		if err != nil {
			return nil, fmt.Errorf("dispatch %d target address: %w", idx, err)
		}
		if len(contractAddr) != wasmtypes.ContractAddrLen {
			return nil, fmt.Errorf(
				"dispatch %d target address must be %d bytes, got %d",
				idx, wasmtypes.ContractAddrLen, len(contractAddr),
			)
		}

		msg := bytes.TrimSpace(dispatch.Msg)
		if len(msg) == 0 {
			return nil, fmt.Errorf("dispatch %d msg cannot be empty", idx)
		}
		if len(msg) > WasmBlockHookMaxPayloadJSONSize {
			return nil, fmt.Errorf(
				"dispatch %d msg too large: got %d bytes, max %d",
				idx, len(msg), WasmBlockHookMaxPayloadJSONSize,
			)
		}
		if !json.Valid(msg) {
			return nil, fmt.Errorf("dispatch %d msg must be valid JSON", idx)
		}
		if msg[0] != '{' {
			return nil, fmt.Errorf("dispatch %d msg must be a JSON object", idx)
		}

		validated = append(validated, ValidatedWasmBlockHookDispatch{
			ContractAddr: contractAddr,
			Msg:          append(json.RawMessage(nil), msg...),
		})
	}

	return validated, nil
}

// emitWasmBlockHookFailureEvent records registry, plan, or target execution
// failure without aborting block processing. Target failures include the target
// contract address when one is available.
func emitWasmBlockHookFailureEvent(
	ctx sdk.Context,
	hookKind wasmBlockHookKind,
	err error,
	contractAddr ...sdk.AccAddress,
) {
	attrs := []sdk.Attribute{
		sdk.NewAttribute(sdk.AttributeKeyModule, wasmtypes.ModuleName),
		sdk.NewAttribute(wasmtypes.AttributeKeyWasmBlockHook, string(hookKind)),
		sdk.NewAttribute(wasmtypes.AttributeKeyWasmBlockHookError, err.Error()),
	}
	if len(contractAddr) > 0 {
		attrs = append(attrs, sdk.NewAttribute(wasmtypes.AttributeKeyContractAddr, contractAddr[0].String()))
	}

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		wasmtypes.EventTypeWasmBlockHookFailure,
		attrs...,
	))
}
