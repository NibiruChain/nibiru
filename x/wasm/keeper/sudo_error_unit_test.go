package keeper

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	wasmvm "github.com/NibiruChain/nibiru/v2/lib/wasmvm"
	"github.com/NibiruChain/nibiru/v2/lib/wasmvm/wvm"
	"github.com/NibiruChain/nibiru/v2/x/wasm/keeper/wasmtesting"
	"github.com/NibiruChain/nibiru/v2/x/wasm/types"
)

func TestSudoWithErrorCause(t *testing.T) {
	mock := wasmtesting.MockWasmEngine{}
	wasmtesting.MakeInstantiable(&mock)
	ctx, keepers := CreateTestInput(t, false, AvailableCapabilities, WithWasmEngine(&mock))
	example := SeedNewContractInstance(t, ctx, keepers, &mock)
	contractErr := &wvm.ContractError{Message: "selection rejected"}
	engineErr := errors.New("node-local engine failure")
	var returned error
	mock.SudoFn = func(wasmvm.Checksum, wvm.Env, []byte, wasmvm.KVStore,
		wasmvm.GoAPI, wasmvm.Querier, wasmvm.GasMeter, uint64,
		wvm.UFraction,
	) (*wvm.Response, uint64, error) {
		return nil, 0, returned
	}

	t.Run("contract rejection preserves its type", func(t *testing.T) {
		returned = contractErr
		_, err := keepers.WasmKeeper.SudoWithErrorCause(ctx, example.Contract, []byte(`{}`))
		require.ErrorIs(t, err, contractErr)
		var typed *wvm.ContractError
		require.ErrorAs(t, err, &typed)
	})
	t.Run("engine fault is not classified as a contract rejection", func(t *testing.T) {
		returned = engineErr
		_, err := keepers.WasmKeeper.SudoWithErrorCause(ctx, example.Contract, []byte(`{}`))
		require.ErrorIs(t, err, engineErr)
		var typed *wvm.ContractError
		require.NotErrorAs(t, err, &typed)
	})
	t.Run("existing sudo retains its SDK error", func(t *testing.T) {
		returned = contractErr
		_, err := keepers.WasmKeeper.Sudo(ctx, example.Contract, []byte(`{}`))
		require.ErrorIs(t, err, types.ErrExecuteFailed)
		require.Equal(t, "selection rejected: execute wasm contract failed", err.Error())
	})
}
