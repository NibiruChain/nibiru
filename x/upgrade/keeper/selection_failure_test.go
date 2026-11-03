package keeper

import (
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/testutil"
	sdk "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types"
	"github.com/NibiruChain/nibiru/v2/lib/wasmvm/wvm"
)

func TestSelectionAttemptAtomicityAndErrorClassification(t *testing.T) {
	for _, tc := range []struct {
		name     string
		category string
		err      error
		panicVal any
	}{
		{name: "success"},
		{name: "contract rejection", category: SelectionContractError,
			err: fmt.Errorf("wrapped: %w", &wvm.ContractError{Message: "bad selection"})},
		{name: "VM gas exhaustion", category: SelectionOutOfGas, err: wvm.OutOfGasError{}},
		{name: "SDK gas exhaustion", category: SelectionOutOfGas,
			panicVal: sdk.ErrorOutOfGas{Descriptor: "contract execution"}},
		{name: "engine failure", err: errors.New("local engine unavailable")},
		{name: "unexpected panic", panicVal: "disk fault"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			key := sdk.NewKVStoreKey("upgrade")
			ctx := testutil.DefaultContext(key, sdk.NewTransientStoreKey("transient"))
			ctx = ctx.WithGasMeter(sdk.NewInfiniteGasMeter())
			ctx.KVStore(key).Set([]byte("original"), []byte("kept"))
			attempt := func(child sdk.Context) error {
				require.Equal(t, SelectionGasLimit, child.GasMeter().Limit())
				child.KVStore(key).Set([]byte("partial"), []byte("must be atomic"))
				child.EventManager().EmitEvent(sdk.NewEvent("selection_attempt"))
				if tc.panicVal != nil {
					panic(tc.panicVal)
				}
				return tc.err
			}
			if tc.panicVal != nil && tc.category == "" {
				require.PanicsWithValue(t, tc.panicVal, func() {
					_, _ = attemptHeightSelection(ctx, attempt)
				})
			} else {
				category, err := attemptHeightSelection(ctx, attempt)
				require.Equal(t, tc.category, category)
				if tc.err != nil || tc.panicVal != nil {
					require.Error(t, err)
				} else {
					require.NoError(t, err)
				}
				if tc.err != nil {
					require.ErrorIs(t, err, tc.err)
				}
			}
			if tc.err == nil && tc.panicVal == nil {
				require.NotNil(t, ctx.KVStore(key).Get([]byte("partial")))
				require.Len(t, ctx.EventManager().Events(), 1)
			} else {
				require.Nil(t, ctx.KVStore(key).Get([]byte("partial")))
				require.Empty(t, ctx.EventManager().Events())
			}
			require.Equal(t, []byte("kept"), ctx.KVStore(key).Get([]byte("original")))
		})
	}
}

func TestSelectionFailureGenesisRoundTrip(t *testing.T) {
	key := sdk.NewKVStoreKey("upgrade")
	ctx := testutil.DefaultContext(key, sdk.NewTransientStoreKey("transient"))
	k := Keeper{storeKey: key}
	failure := SelectionFailure{RequestID: 9, Name: "vNext", Height: 123, Category: SelectionContractError}
	k.setSelectionFailure(ctx, failure)
	bz := k.SelectionFailureGenesis(ctx)
	require.NoError(t, ValidateSelectionFailureGenesis(bz))
	k.ClearSelectionFailure(ctx)
	require.Nil(t, k.GetSelectionFailure(ctx))
	require.NoError(t, k.InitSelectionFailureGenesis(ctx, bz))
	require.Equal(t, &failure, k.GetSelectionFailure(ctx))
	require.JSONEq(t, string(bz), string(k.SelectionFailureGenesis(ctx)))

	require.NoError(t, k.InitSelectionFailureGenesis(ctx, json.RawMessage(`{}`)))
	require.Nil(t, k.GetSelectionFailure(ctx))
	for _, invalid := range []string{
		`{"selection_failure":{"request_id":"0","name":"vNext","height":"123","category":"contract_error"}}`,
		`{"selection_failure":{"request_id":"9","name":"vNext","height":"0","category":"contract_error"}}`,
		`{"selection_failure":{"request_id":"9","name":"vNext","height":"123","category":"host error text"}}`,
		`not json`,
	} {
		require.Error(t, ValidateSelectionFailureGenesis(json.RawMessage(invalid)))
		require.Error(t, k.InitSelectionFailureGenesis(ctx, json.RawMessage(invalid)))
	}
}

func TestSelectionFailurePersistsOutsideFailedAttempt(t *testing.T) {
	key := sdk.NewKVStoreKey("upgrade")
	ctx := testutil.DefaultContext(key, sdk.NewTransientStoreKey("transient"))
	k := Keeper{storeKey: key}
	category, err := attemptHeightSelection(ctx, func(child sdk.Context) error {
		child.KVStore(key).Set([]byte("partial_plan"), []byte("wrong"))
		child.GasMeter().ConsumeGas(SelectionGasLimit+1, "fixture infinite contract")
		return nil
	})
	require.Error(t, err)
	require.Equal(t, SelectionOutOfGas, category)
	k.setSelectionFailure(ctx, SelectionFailure{RequestID: 1, Name: "vNext", Height: 100, Category: category})
	require.Nil(t, ctx.KVStore(key).Get([]byte("partial_plan")))
	require.Equal(t, SelectionOutOfGas, k.GetSelectionFailure(ctx).Category)
}
