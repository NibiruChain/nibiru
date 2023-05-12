package binding

import (
	"encoding/json"
	"fmt"
	oraclekeeper "github.com/NibiruChain/nibiru/x/oracle/keeper"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	wasmvmtypes "github.com/CosmWasm/wasmvm/types"

	"github.com/NibiruChain/nibiru/x/common/set"
	perpkeeper "github.com/NibiruChain/nibiru/x/perp/keeper/v1"
	"github.com/NibiruChain/nibiru/x/sudo"
	"github.com/NibiruChain/nibiru/x/wasm/binding/cw_struct"
)

var _ wasmkeeper.Messenger = (*CustomWasmExecutor)(nil)

// CustomWasmExecutor is an extension of wasm/keeper.Messenger with its
// own custom `DispatchMsg` for CosmWasm execute calls on Nibiru.
type CustomWasmExecutor struct {
	Wasm   wasmkeeper.Messenger
	Perp   ExecutorPerp
	Sudo   sudo.Keeper
	Oracle ExecutorOracle
}

// BindingExecuteMsgWrapper is a n override of CosmosMsg::Custom
// (json.RawMessage), which corresponds to `BindingExecuteMsgWrapper` in
// the bindings-perp.rs contract.
type BindingExecuteMsgWrapper struct {
	// Routes here refer to groups of modules on Nibiru. The idea behind setting
	// routes alongside the messae payload is to add information on which module
	// or group of modules a particular execute message belongs to.
	// For example, the perp bindings have route "perp".
	Route *string `json:"route,omitempty"`
	// ExecuteMsg is a json struct for ExecuteMsg::{
	//   OpenPosition, ClosePosition, AddMargin, RemoveMargin, ...} from the
	//   bindings smart contracts.
	ExecuteMsg *cw_struct.BindingMsg `json:"msg,omitempty"`
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
		var contractExecuteMsg BindingExecuteMsgWrapper
		if err := json.Unmarshal(wasmMsg.Custom, &contractExecuteMsg); err != nil {
			return events, data, sdkerrors.Wrapf(err, "wasmMsg: %s", wasmMsg.Custom)
		}

		switch {
		case contractExecuteMsg.ExecuteMsg.OpenPosition != nil:
			cwMsg := contractExecuteMsg.ExecuteMsg.OpenPosition
			_, err = messenger.Perp.OpenPosition(cwMsg, ctx)
			return events, data, err
		case contractExecuteMsg.ExecuteMsg.ClosePosition != nil:
			cwMsg := contractExecuteMsg.ExecuteMsg.ClosePosition
			_, err = messenger.Perp.ClosePosition(cwMsg, ctx)
			return events, data, err
		case contractExecuteMsg.ExecuteMsg.AddMargin != nil:
			cwMsg := contractExecuteMsg.ExecuteMsg.AddMargin
			_, err = messenger.Perp.AddMargin(cwMsg, ctx)
			return events, data, err
		case contractExecuteMsg.ExecuteMsg.RemoveMargin != nil:
			cwMsg := contractExecuteMsg.ExecuteMsg.RemoveMargin
			_, err = messenger.Perp.RemoveMargin(cwMsg, ctx)
			return events, data, err
		case contractExecuteMsg.ExecuteMsg.PegShift != nil:
			if err := messenger.CheckPermissions(contractAddr, ctx); err != nil {
				return events, data, err
			}
			cwMsg := contractExecuteMsg.ExecuteMsg.PegShift
			err = messenger.Perp.PegShift(cwMsg, contractAddr, ctx)
			return events, data, err
		case contractExecuteMsg.ExecuteMsg.DepthShift != nil:
			cwMsg := contractExecuteMsg.ExecuteMsg.DepthShift
			err = messenger.Perp.DepthShift(cwMsg, contractAddr, ctx)
			return events, data, err
		case contractExecuteMsg.ExecuteMsg.OracleParams != nil:
			cwMsg := contractExecuteMsg.ExecuteMsg.OracleParams
			messenger.Oracle.SetOracleParams(cwMsg, ctx)
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
	sudoKeeper sudo.Keeper,
	oracleKeeper oraclekeeper.Keeper,
) func(wasmkeeper.Messenger) wasmkeeper.Messenger {
	return func(originalWasmMessenger wasmkeeper.Messenger) wasmkeeper.Messenger {
		return &CustomWasmExecutor{
			Wasm:   originalWasmMessenger,
			Perp:   ExecutorPerp{Perp: perp},
			Sudo:   sudoKeeper,
			Oracle: ExecutorOracle{Oracle: oracleKeeper},
		}
	}
}

// CheckPermissions: Checks if a contract is contained within the set of sudo
// contracts defined in the x/sudo module. These smart contracts are able to
// execute certain permissioned functions.
// See https://www.notion.so/nibiru/Nibi-Perps-Admin-ADR-ad38991fffd34e7798618731be0fa922?pvs=4
func (messenger *CustomWasmExecutor) CheckPermissions(
	contract sdk.AccAddress, ctx sdk.Context,
) error {
	contracts, err := messenger.Sudo.GetSudoContracts(ctx)
	if err != nil {
		return err
	}
	hasPermission := set.New(contracts...).Has(contract.String())
	if !hasPermission {
		return fmt.Errorf(
			"insufficient permissions on smart contract: %s. The sudo contracts are: %s",
			contract, contracts,
		)
	}
	return nil
}
