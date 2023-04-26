package binding

import (
	"encoding/json"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	wasmvmtypes "github.com/CosmWasm/wasmvm/types"
	perpkeeper "github.com/NibiruChain/nibiru/x/perp/keeper"
	"github.com/NibiruChain/nibiru/x/wasm/binding/cw_struct"
)

var _ wasmkeeper.Messenger = (*CustomWasmExecutor)(nil)

// CustomWasmExecutor is an extension of wasm/keeper.Messenger with its
// own custom `DispatchMsg` for CosmWasm execute calls on Nibiru.
type CustomWasmExecutor struct {
	Wasm wasmkeeper.Messenger
	Perp IExecutorPerp
}

// DispatchMsg encodes the wasmVM message and dispatches it.
func (messenger *CustomWasmExecutor) DispatchMsg(
	ctx sdk.Context,
	contractAddr sdk.AccAddress,
	contractIBCPortID string,
	wasmMsg wasmvmtypes.CosmosMsg,
) (events []sdk.Event, data [][]byte, err error) {

	// If the "Custom" field is set, we handle a BindingMsg.
	if wasmMsg.Custom != nil {

		var contractExecuteMsg cw_struct.BindingMsg
		if err := json.Unmarshal(wasmMsg.Custom, &contractExecuteMsg); err != nil {
			return events, data, sdkerrors.Wrapf(err, "wasmMsg: %s", wasmMsg.Custom)
		}

		switch {
		case contractExecuteMsg.OpenPosition != nil:
			cwMsg := contractExecuteMsg.OpenPosition
			_, err = messenger.Perp.OpenPosition(cwMsg, ctx)
			return events, data, err
		case contractExecuteMsg.ClosePosition != nil:
			cwMsg := contractExecuteMsg.ClosePosition
			_, err = messenger.Perp.ClosePosition(cwMsg, ctx)
			return events, data, err
		case contractExecuteMsg.AddMargin != nil:
			cwMsg := contractExecuteMsg.AddMargin
			_, err = messenger.Perp.AddMargin(cwMsg, ctx)
			return events, data, err
		case contractExecuteMsg.RemoveMargin != nil:
			cwMsg := contractExecuteMsg.RemoveMargin
			_, err = messenger.Perp.RemoveMargin(cwMsg, ctx)
			return events, data, err
		default:
			err = wasmvmtypes.InvalidRequest{
				Err:     "invalid bindings request",
				Request: wasmMsg.Custom}
			return events, data, err
		}
	}

	// The default execution path is to use the wasmkeeper.Messenger.
	return messenger.Wasm.DispatchMsg(ctx, contractAddr, contractIBCPortID, wasmMsg)
}

func CustomExecuteMsgHandler(
	perp perpkeeper.Keeper,
) func(wasmkeeper.Messenger) wasmkeeper.Messenger {
	return func(originalWasmMessenger wasmkeeper.Messenger) wasmkeeper.Messenger {
		return &CustomWasmExecutor{
			Wasm: originalWasmMessenger,
			Perp: &ExecutorPerp{Perp: perp},
		}
	}
}
