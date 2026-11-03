package keeper

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"

	sdk "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types"
	"github.com/NibiruChain/nibiru/v2/lib/wasmvm/wvm"
)

// SelectionGasLimit is consensus-defined, not a node's query gas setting.
// Selection only reads the pending target/configuration and writes one plan.
const SelectionGasLimit uint64 = 1_000_000

var selectionFailureKey = []byte{0x05}

const (
	SelectionContractError = "contract_error"
	SelectionOutOfGas      = "out_of_gas"
)

// SelectionFailure is a host receipt for an atomic contract call that could
// not persist its own failed status. Contract state remains canonical for
// ordinary schedules; this receipt only suppresses the failed request ID.
// Never persist arbitrary engine error strings, which can differ by node.
type SelectionFailure struct {
	RequestID uint64 `json:"request_id,string"`
	Name      string `json:"name"`
	Height    int64  `json:"height,string"`
	Category  string `json:"category"`
}

type selectionFailureGenesis struct {
	Failure *SelectionFailure `json:"selection_failure,omitempty"`
}

func (f SelectionFailure) validate() error {
	if f.RequestID == 0 || f.Name == "" || f.Height <= 0 {
		return fmt.Errorf("invalid upgrade selection failure identity or height")
	}
	if f.Category != SelectionContractError && f.Category != SelectionOutOfGas {
		return fmt.Errorf("invalid upgrade selection failure category %q", f.Category)
	}
	return nil
}

func (k Keeper) GetSelectionFailure(ctx sdk.Context) *SelectionFailure {
	bz := ctx.KVStore(k.storeKey).Get(selectionFailureKey)
	if bz == nil {
		return nil
	}
	var failure SelectionFailure
	if err := json.Unmarshal(bz, &failure); err != nil {
		panic(fmt.Errorf("decode upgrade selection failure: %w", err))
	}
	if err := failure.validate(); err != nil {
		panic(err)
	}
	return &failure
}

func (k Keeper) ClearSelectionFailure(ctx sdk.Context) {
	ctx.KVStore(k.storeKey).Delete(selectionFailureKey)
}

// SelectionFailureGenesis preserves the exceptional native receipt alongside
// Wasm's export of the four canonical contract state families.
func (k Keeper) SelectionFailureGenesis(ctx sdk.Context) json.RawMessage {
	bz, err := json.Marshal(selectionFailureGenesis{Failure: k.GetSelectionFailure(ctx)})
	if err != nil {
		panic(err)
	}
	return bz
}

func ValidateSelectionFailureGenesis(data json.RawMessage) error {
	_, err := decodeSelectionFailureGenesis(data)
	return err
}

func decodeSelectionFailureGenesis(data json.RawMessage) (*SelectionFailure, error) {
	if len(data) == 0 {
		return nil, nil
	}
	var state selectionFailureGenesis
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("decode upgrade genesis: %w", err)
	}
	if state.Failure != nil {
		if err := state.Failure.validate(); err != nil {
			return nil, err
		}
	}
	return state.Failure, nil
}

func (k Keeper) InitSelectionFailureGenesis(ctx sdk.Context, data json.RawMessage) error {
	failure, err := decodeSelectionFailureGenesis(data)
	if err != nil {
		return err
	}
	k.ClearSelectionFailure(ctx)
	if failure != nil {
		k.setSelectionFailure(ctx, *failure)
	}
	return nil
}

func (k Keeper) setSelectionFailure(ctx sdk.Context, failure SelectionFailure) {
	if err := failure.validate(); err != nil {
		panic(err)
	}
	bz, err := json.Marshal(failure)
	if err != nil {
		panic(err)
	}
	ctx.KVStore(k.storeKey).Set(selectionFailureKey, bz)
}

// TrySelectExecutionHeight leaves an unexpected deterministic contract failure
// visible but terminal for that request. Unknown host errors propagate to the
// caller, which must stop before committing rather than invent a failed plan.
func (k Keeper) TrySelectExecutionHeight(ctx sdk.Context) error {
	if !k.ContractActive(ctx) {
		return nil
	}
	target, err := k.TimeTarget(ctx)
	if err != nil {
		return err
	}
	if target == nil {
		return nil
	}
	// A time target has not reached its execution height yet. Preserve the
	// SDK early-binary guard even while CurrentPlan intentionally returns nil.
	if k.HasHandler(target.Name) {
		return fmt.Errorf("BINARY UPDATED BEFORE TRIGGER! UPGRADE %q - in binary but not executed on chain. Downgrade your binary", target.Name)
	}
	if failure := k.GetSelectionFailure(ctx); failure != nil && failure.RequestID == target.ID {
		return nil
	}
	// Compare seconds/nanoseconds without UnixNano overflow or wall-clock time.
	seconds := ctx.BlockTime().Unix()
	if seconds < 0 || uint64(seconds) < target.TargetTime/1_000_000_000 ||
		(uint64(seconds) == target.TargetTime/1_000_000_000 &&
			uint64(ctx.BlockTime().Nanosecond()) < target.TargetTime%1_000_000_000) {
		return nil
	}
	category, err := attemptHeightSelection(ctx, k.SelectExecutionHeight)
	if category == "" {
		return err
	}
	k.setSelectionFailure(ctx, SelectionFailure{
		RequestID: target.ID, Name: target.Name, Height: ctx.BlockHeight(), Category: category,
	})
	ctx.Logger().Error("upgrade execution height selection failed", "upgrade", target.Name, "err", err)
	ctx.EventManager().EmitEvent(sdk.NewEvent("upgrade_selection_failed",
		sdk.NewAttribute("request_id", strconv.FormatUint(target.ID, 10)),
		sdk.NewAttribute("upgrade", target.Name),
		sdk.NewAttribute("height", strconv.FormatInt(ctx.BlockHeight(), 10)),
		sdk.NewAttribute("reason_code", category),
	))
	return nil
}

// attemptHeightSelection only catches consensus-deterministic failures. The
// failed cache and its events never escape; the parent persists the receipt.
func attemptHeightSelection(ctx sdk.Context, run func(sdk.Context) error) (category string, err error) {
	cacheCtx, write := ctx.CacheContext()
	cacheCtx = cacheCtx.WithGasMeter(sdk.NewGasMeter(SelectionGasLimit)).
		WithBlockGasMeter(sdk.NewInfiniteGasMeter())
	defer func() {
		if recovered := recover(); recovered != nil {
			if _, ok := recovered.(sdk.ErrorOutOfGas); ok {
				category, err = SelectionOutOfGas, fmt.Errorf("upgrade selection exhausted its consensus gas limit")
				return
			}
			panic(recovered)
		}
	}()
	err = run(cacheCtx)
	if err != nil {
		var contractErr *wvm.ContractError
		var gasErr wvm.OutOfGasError
		if errors.As(err, &contractErr) {
			return SelectionContractError, err
		}
		if errors.As(err, &gasErr) {
			return SelectionOutOfGas, err
		}
		return "", err
	}
	write()
	return "", nil
}
