package keeper

import (
	"bytes"
	"encoding/json"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	wasmtypes "github.com/NibiruChain/nibiru/v2/x/wasm/types"
)

const (
	wasmBlockHookMaxDispatches      = 32
	wasmBlockHookMaxPayloadJSONSize = 16 * 1024
)

type wasmBlockHookKind string

const (
	wasmBlockHookBeginBlock wasmBlockHookKind = "begin_block"
	wasmBlockHookEndBlock   wasmBlockHookKind = "end_block"
)

var (
	wasmBlockHookBeginBlockQuery = []byte(`{"begin_block_plan":{}}`)
	wasmBlockHookEndBlockQuery   = []byte(`{"end_block_plan":{}}`)
)

type wasmBlockHookDispatch struct {
	ContractAddr string          `json:"contract_addr"`
	Msg          json.RawMessage `json:"msg"`
}

type validatedWasmBlockHookDispatch struct {
	ContractAddr sdk.AccAddress
	Msg          json.RawMessage
}

// BeginBlockWasmHooks queries the configured registry contract for a begin-block
// dispatch plan. Target sudo execution is intentionally left to a later slice.
func (k Keeper) BeginBlockWasmHooks(ctx sdk.Context) {
	k.runWasmBlockHookPlanner(ctx, wasmBlockHookBeginBlock)
}

// EndBlockWasmHooks queries the configured registry contract for an end-block
// dispatch plan. Target sudo execution is intentionally left to a later slice.
func (k Keeper) EndBlockWasmHooks(ctx sdk.Context) {
	k.runWasmBlockHookPlanner(ctx, wasmBlockHookEndBlock)
}

func (k Keeper) runWasmBlockHookPlanner(ctx sdk.Context, hookKind wasmBlockHookKind) {
	_, err := k.queryWasmBlockHookPlan(ctx, hookKind)
	if err != nil {
		emitWasmBlockHookFailureEvent(ctx, hookKind, err)
	}

	// TODO: Instantiate the fixture contract in nibi-chain tests once its Wasm
	// artifact is built, then add target sudo dispatch with per-target cache
	// rollback in the next implementation slice.
}

func (k Keeper) queryWasmBlockHookPlan(
	ctx sdk.Context,
	hookKind wasmBlockHookKind,
) ([]validatedWasmBlockHookDispatch, error) {
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

	var dispatches []wasmBlockHookDispatch
	if err := json.Unmarshal(queryResp, &dispatches); err != nil {
		return nil, fmt.Errorf("decode wasm block hook registry response: %w", err)
	}

	return validateWasmBlockHookDispatches(dispatches)
}

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

func validateWasmBlockHookDispatches(
	dispatches []wasmBlockHookDispatch,
) ([]validatedWasmBlockHookDispatch, error) {
	if len(dispatches) > wasmBlockHookMaxDispatches {
		return nil, fmt.Errorf(
			"too many wasm block hook dispatches: got %d, max %d",
			len(dispatches), wasmBlockHookMaxDispatches,
		)
	}

	validated := make([]validatedWasmBlockHookDispatch, 0, len(dispatches))
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
		if len(msg) > wasmBlockHookMaxPayloadJSONSize {
			return nil, fmt.Errorf(
				"dispatch %d msg too large: got %d bytes, max %d",
				idx, len(msg), wasmBlockHookMaxPayloadJSONSize,
			)
		}
		if !json.Valid(msg) {
			return nil, fmt.Errorf("dispatch %d msg must be valid JSON", idx)
		}
		if msg[0] != '{' {
			return nil, fmt.Errorf("dispatch %d msg must be a JSON object", idx)
		}

		validated = append(validated, validatedWasmBlockHookDispatch{
			ContractAddr: contractAddr,
			Msg:          append(json.RawMessage(nil), msg...),
		})
	}

	return validated, nil
}

func emitWasmBlockHookFailureEvent(ctx sdk.Context, hookKind wasmBlockHookKind, err error) {
	ctx.EventManager().EmitEvent(sdk.NewEvent(
		wasmtypes.EventTypeWasmBlockHookFailure,
		sdk.NewAttribute(sdk.AttributeKeyModule, wasmtypes.ModuleName),
		sdk.NewAttribute(wasmtypes.AttributeKeyWasmBlockHook, string(hookKind)),
		sdk.NewAttribute(wasmtypes.AttributeKeyWasmBlockHookError, err.Error()),
	))
}
