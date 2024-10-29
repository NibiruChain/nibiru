package test

import (
	"encoding/json"
	"math/big"
	"os"
	"os/exec"
	"path"
	"strings"

	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	wasm "github.com/CosmWasm/wasmd/x/wasm/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/stretchr/testify/suite"

	"github.com/NibiruChain/nibiru/v2/app"
	serverconfig "github.com/NibiruChain/nibiru/v2/app/server/config"
	"github.com/NibiruChain/nibiru/v2/x/evm/embeds"
	"github.com/NibiruChain/nibiru/v2/x/evm/evmtest"
	"github.com/NibiruChain/nibiru/v2/x/evm/precompile"
)

// SetupWasmContracts stores all Wasm bytecode and has the "deps.Sender"
// instantiate each Wasm contract using the precompile.
func SetupWasmContracts(deps *evmtest.TestDeps, s *suite.Suite) (
	contracts []sdk.AccAddress,
) {
	wasmCodes := DeployWasmBytecode(s, deps.Ctx, deps.Sender.NibiruAddr, deps.App)

	instantiateArgs := []struct {
		InstantiateMsg []byte
		Label          string
	}{
		{
			InstantiateMsg: []byte("{}"),
			Label:          "https://github.com/NibiruChain/nibiru-wasm/blob/main/contracts/nibi-stargate/src/contract.rs",
		},
		{
			InstantiateMsg: []byte(`{"count": 0}`),
			Label:          "https://github.com/NibiruChain/nibiru-wasm/tree/ec3ab9f09587a11fbdfbd4021c7617eca3912044/contracts/00-hello-world-counter",
		},
	}

	for i, wasmCode := range wasmCodes {
		s.T().Logf("Instantiate using Wasm precompile: %s", wasmCode.binPath)
		codeId := wasmCode.codeId

		m := wasm.MsgInstantiateContract{
			Admin:  "",
			CodeID: codeId,
			Label:  instantiateArgs[i].Label,
			Msg:    instantiateArgs[i].InstantiateMsg,
		}

		msgArgsBz, err := json.Marshal(m.Msg)
		s.NoError(err)

		callArgs := []interface{}{m.Admin, m.CodeID, msgArgsBz, m.Label, []precompile.WasmBankCoin{}}
		input, err := embeds.SmartContract_Wasm.ABI.Pack(
			string(precompile.WasmMethod_instantiate),
			callArgs...,
		)
		s.Require().NoError(err)

		ethTxResp, _, err := deps.EvmKeeper.CallContractWithInput(
			deps.Ctx, deps.Sender.EthAddr, &precompile.PrecompileAddr_Wasm, true, input,
		)
		s.Require().NoError(err)
		s.Require().NotEmpty(ethTxResp.Ret)

		s.T().Log("Parse the response contract addr and response bytes")
		vals, err := embeds.SmartContract_Wasm.ABI.Unpack(string(precompile.WasmMethod_instantiate), ethTxResp.Ret)
		s.Require().NoError(err)

		contractAddr, err := sdk.AccAddressFromBech32(vals[0].(string))
		s.NoError(err)
		contracts = append(contracts, contractAddr)
	}

	return contracts
}

// DeployWasmBytecode is a setup function that stores all Wasm bytecode used in
// the test suite.
func DeployWasmBytecode(
	s *suite.Suite,
	ctx sdk.Context,
	sender sdk.AccAddress,
	nibiru *app.NibiruApp,
) (codeIds []struct {
	codeId  uint64
	binPath string
},
) {
	// rootPath, _ := exec.Command("go list -m -f {{.Dir}}").Output()
	// Run: go list -m -f {{.Dir}}
	// This returns the path to the root of the project.
	rootPathBz, err := exec.Command("go", "list", "-m", "-f", "{{.Dir}}").Output()
	s.Require().NoError(err)
	rootPath := strings.Trim(string(rootPathBz), "\n")
	for _, wasmFile := range []string{
		// nibi_stargate.wasm is a compiled version of:
		// https://github.com/NibiruChain/nibiru-wasm/blob/main/contracts/nibi-stargate/src/contract.rs
		"x/tokenfactory/fixture/nibi_stargate.wasm",

		// hello_world_counter.wasm is a compiled version of:
		// https://github.com/NibiruChain/nibiru-wasm/tree/ec3ab9f09587a11fbdfbd4021c7617eca3912044/contracts/00-hello-world-counter
		"x/evm/precompile/hello_world_counter.wasm",

		// Add other wasm bytecode here if needed...
	} {
		binPath := path.Join(rootPath, wasmFile)
		wasmBytecode, err := os.ReadFile(binPath)
		s.Require().NoErrorf(
			err,
			"path %s, pathToWasmBin %s", binPath,
		)

		// The "Create" fn is private on the nibiru.WasmKeeper. By placing it as the
		// decorated keeper in PermissionedKeeper type, we can access "Create" as a
		// public fn.
		wasmPermissionedKeeper := wasmkeeper.NewDefaultPermissionKeeper(nibiru.WasmKeeper)
		instantiateAccess := &wasm.AccessConfig{
			Permission: wasm.AccessTypeEverybody,
		}
		codeId, _, err := wasmPermissionedKeeper.Create(
			ctx, sender, wasmBytecode, instantiateAccess,
		)
		s.Require().NoError(err)
		codeIds = append(codeIds, struct {
			codeId  uint64
			binPath string
		}{codeId, binPath})
	}

	return codeIds
}

// From IWasm.query of Wasm.sol:
//
//	```solidity
//	function query(
//	  string memory contractAddr,
//	  bytes memory req
//	) external view returns (bytes memory response);
//	```
func AssertWasmCounterState(
	s *suite.Suite,
	deps evmtest.TestDeps,
	wasmContract sdk.AccAddress,
	wantCount int64,
) (evmObj *vm.EVM) {
	msgArgsBz := []byte(`
		{ 
		  "count": {}
		}
		`)

	callArgs := []interface{}{
		// string memory contractAddr
		wasmContract.String(),
		// bytes memory req
		msgArgsBz,
	}
	input, err := embeds.SmartContract_Wasm.ABI.Pack(
		string(precompile.WasmMethod_query),
		callArgs...,
	)
	s.Require().NoError(err)

	ethTxResp, evmObj, err := deps.EvmKeeper.CallContractWithInput(
		deps.Ctx, deps.Sender.EthAddr, &precompile.PrecompileAddr_Wasm, true, input,
	)
	s.Require().NoError(err)
	s.Require().NotEmpty(ethTxResp.Ret)

	s.T().Log("Parse the response contract addr and response bytes")
	s.T().Logf("ethTxResp.Ret: %s", ethTxResp.Ret)
	var queryResp []byte
	err = embeds.SmartContract_Wasm.ABI.UnpackIntoInterface(
		// Since there's only one return value, don't unpack as a slice.
		// If there were two or more return values, we'd use
		// &[]interface{}{...}
		&queryResp,
		string(precompile.WasmMethod_query),
		ethTxResp.Ret,
	)
	s.Require().NoError(err)
	s.T().Logf("queryResp: %s", queryResp)

	s.T().Log("Response is a JSON-encoded struct from the Wasm contract")
	var wasmMsg wasm.RawContractMessage
	err = json.Unmarshal(queryResp, &wasmMsg)
	s.NoError(err)
	s.NoError(wasmMsg.ValidateBasic())
	var typedResp QueryMsgCountResp
	err = json.Unmarshal(wasmMsg, &typedResp)
	s.NoError(err)

	s.EqualValues(wantCount, typedResp.Count)
	s.EqualValues(deps.Sender.NibiruAddr.String(), typedResp.Owner)
	return evmObj
}

// Result of QueryMsg::Count from the [hello_world_counter] Wasm contract:
//
//	```rust
//	#[cw_serde]
//	pub struct State {
//	    pub count: i64,
//	    pub owner: Addr,
//	}
//	```
//
// [hello_world_counter]: https://github.com/NibiruChain/nibiru-wasm/tree/ec3ab9f09587a11fbdfbd4021c7617eca3912044/contracts/00-hello-world-counter
type QueryMsgCountResp struct {
	Count int64  `json:"count"`
	Owner string `json:"owner"`
}

// From evm/embeds/contracts/Wasm.sol:
//
//	```solidity
//	struct WasmExecuteMsg {
//	  string contractAddr;
//	  bytes msgArgs;
//	  BankCoin[] funds;
//	}
//
//	/// @notice Identical to "execute", except for multiple contract calls.
//	function executeMulti(
//	  WasmExecuteMsg[] memory executeMsgs
//	) payable external returns (bytes[] memory responses);
//	```
//
// The increment call corresponds to the ExecuteMsg from
// the [hello_world_counter] Wasm contract:
//
//	```rust
//	#[cw_serde]
//	pub enum ExecuteMsg {
//	    Increment {},         // Increase count by 1
//	    Reset { count: i64 }, // Reset to any i64 value
//	}
//	```
//
// [hello_world_counter]: https://github.com/NibiruChain/nibiru-wasm/tree/ec3ab9f09587a11fbdfbd4021c7617eca3912044/contracts/00-hello-world-counter
func IncrementWasmCounterWithExecuteMulti(
	s *suite.Suite,
	deps *evmtest.TestDeps,
	wasmContract sdk.AccAddress,
	times uint,
	finalizeTx bool,
) (evmObj *vm.EVM) {
	msgArgsBz := []byte(`
	{ 
	  "increment": {}
	}
	`)

	// Parse funds argument.
	var funds []precompile.WasmBankCoin // blank funds
	fundsJson, err := json.Marshal(funds)
	s.NoErrorf(err, "fundsJson: %s", fundsJson)
	err = json.Unmarshal(fundsJson, &funds)
	s.Require().NoError(err, "fundsJson %s, funds %s", fundsJson, funds)

	// The "times" arg determines the number of messages in the executeMsgs slice
	executeMsgs := []struct {
		ContractAddr string                    `json:"contractAddr"`
		MsgArgs      []byte                    `json:"msgArgs"`
		Funds        []precompile.WasmBankCoin `json:"funds"`
	}{
		{wasmContract.String(), msgArgsBz, funds},
	}
	if times == 0 {
		executeMsgs = executeMsgs[:0] // force empty
	} else {
		for i := uint(1); i < times; i++ {
			executeMsgs = append(executeMsgs, executeMsgs[0])
		}
	}
	s.Require().Len(executeMsgs, int(times)) // sanity check assertion

	callArgs := []interface{}{
		executeMsgs,
	}
	input, err := embeds.SmartContract_Wasm.ABI.Pack(
		string(precompile.WasmMethod_executeMulti),
		callArgs...,
	)
	s.Require().NoError(err)

	ethTxResp, evmObj, err := deps.EvmKeeper.CallContractWithInput(
		deps.Ctx, deps.Sender.EthAddr, &precompile.PrecompileAddr_Wasm, finalizeTx, input,
	)
	s.Require().NoError(err)
	s.Require().NotEmpty(ethTxResp.Ret)
	return evmObj
}

func IncrementWasmCounterWithExecuteMultiViaVMCall(
	s *suite.Suite,
	deps *evmtest.TestDeps,
	wasmContract sdk.AccAddress,
	times uint,
	finalizeTx bool,
	evmObj *vm.EVM,
) error {
	msgArgsBz := []byte(`
	{ 
	  "increment": {}
	}
	`)

	// Parse funds argument.
	var funds []precompile.WasmBankCoin // blank funds
	fundsJson, err := json.Marshal(funds)
	s.NoErrorf(err, "fundsJson: %s", fundsJson)
	err = json.Unmarshal(fundsJson, &funds)
	s.Require().NoError(err, "fundsJson %s, funds %s", fundsJson, funds)

	// The "times" arg determines the number of messages in the executeMsgs slice
	executeMsgs := []struct {
		ContractAddr string                    `json:"contractAddr"`
		MsgArgs      []byte                    `json:"msgArgs"`
		Funds        []precompile.WasmBankCoin `json:"funds"`
	}{
		{wasmContract.String(), msgArgsBz, funds},
	}
	if times == 0 {
		executeMsgs = executeMsgs[:0] // force empty
	} else {
		for i := uint(1); i < times; i++ {
			executeMsgs = append(executeMsgs, executeMsgs[0])
		}
	}
	s.Require().Len(executeMsgs, int(times)) // sanity check assertion

	callArgs := []interface{}{
		executeMsgs,
	}
	input, err := embeds.SmartContract_Wasm.ABI.Pack(
		string(precompile.WasmMethod_executeMulti),
		callArgs...,
	)
	s.Require().NoError(err)

	contract := precompile.PrecompileAddr_Wasm
	leftoverGas := serverconfig.DefaultEthCallGasLimit
	_, _, err = evmObj.Call(
		vm.AccountRef(deps.Sender.EthAddr),
		contract,
		input,
		leftoverGas,
		big.NewInt(0),
	)
	return err
}
