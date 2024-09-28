package precompile_test

import (
	"encoding/json"
	"fmt"
	"math/big"
	"os"

	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	wasm "github.com/CosmWasm/wasmd/x/wasm/types"

	"github.com/NibiruChain/nibiru/v2/app"
	"github.com/NibiruChain/nibiru/v2/x/evm/embeds"
	"github.com/NibiruChain/nibiru/v2/x/evm/evmtest"
	"github.com/NibiruChain/nibiru/v2/x/evm/precompile"
	tokenfactory "github.com/NibiruChain/nibiru/v2/x/tokenfactory/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/suite"
)

type WasmSuite struct {
	suite.Suite
}

type WasmContract struct {
	Addr   sdk.AccAddress
	CodeID uint64
}

// SetupWasmContracts stores all Wasm bytecode and has the "deps.Sender"
// instantiate each Wasm contract using the precompile.
func SetupWasmContracts(deps *evmtest.TestDeps, s *suite.Suite) (
	contracts []WasmContract,
) {
	wasmCodes := DeployWasmBytecode(s, deps.Ctx, deps.Sender.NibiruAddr, deps.App)

	otherArgs := []struct {
		InstMsg []byte
		Label   string
	}{
		{
			InstMsg: []byte("{}"),
			Label:   "https://github.com/NibiruChain/nibiru-wasm/blob/main/contracts/nibi-stargate/src/contract.rs",
		},
		{
			InstMsg: []byte(`{"count": 0}`),
			Label:   "https://github.com/NibiruChain/nibiru-wasm/tree/ec3ab9f09587a11fbdfbd4021c7617eca3912044/contracts/00-hello-world-counter",
		},
	}

	for wasmCodeIdx, wasmCode := range wasmCodes {
		s.T().Logf("Instantiate using Wasm precompile: %s", wasmCode.binPath)
		codeId := wasmCode.codeId

		m := wasm.MsgInstantiateContract{
			Admin:  "",
			CodeID: codeId,
			Label:  otherArgs[wasmCodeIdx].Label,
			Msg:    otherArgs[wasmCodeIdx].InstMsg,
			Funds:  []sdk.Coin{},
		}

		msgArgsBz, err := json.Marshal(m.Msg)
		s.NoError(err)

		var funds []precompile.WasmBankCoin
		fundsJson, err := m.Funds.MarshalJSON()
		s.NoErrorf(err, "fundsJson: %s", fundsJson)
		err = json.Unmarshal(fundsJson, &funds)
		s.Require().NoError(err)

		callArgs := []any{m.Admin, m.CodeID, msgArgsBz, m.Label, funds}
		input, err := embeds.SmartContract_Wasm.ABI.Pack(
			string(precompile.WasmMethod_instantiate),
			callArgs...,
		)
		s.Require().NoError(err)

		ethTxResp, err := deps.EvmKeeper.CallContractWithInput(
			deps.Ctx, deps.Sender.EthAddr, &precompile.PrecompileAddr_Wasm, true, input,
		)
		s.Require().NoError(err)
		s.Require().NotEmpty(ethTxResp.Ret)

		s.T().Log("Parse the response contract addr and response bytes")
		var contractAddrStr string
		var data []byte
		err = embeds.SmartContract_Wasm.ABI.UnpackIntoInterface(
			&[]any{&contractAddrStr, &data},
			string(precompile.WasmMethod_instantiate),
			ethTxResp.Ret,
		)
		s.Require().NoError(err)
		contractAddr, err := sdk.AccAddressFromBech32(contractAddrStr)
		s.NoError(err)
		contracts = append(contracts, WasmContract{
			Addr:   contractAddr,
			CodeID: codeId,
		})
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
	for _, pathToWasmBin := range []string{
		// nibi_stargate.wasm is a compiled version of:
		// https://github.com/NibiruChain/nibiru-wasm/blob/main/contracts/nibi-stargate/src/contract.rs
		"../../tokenfactory/fixture/nibi_stargate.wasm",

		// hello_world_counter.wasm is a compiled version of:
		// https://github.com/NibiruChain/nibiru-wasm/tree/ec3ab9f09587a11fbdfbd4021c7617eca3912044/contracts/00-hello-world-counter
		"./hello_world_counter.wasm",

		// Add other wasm bytecode here if needed...
	} {
		wasmBytecode, err := os.ReadFile(pathToWasmBin)
		s.Require().NoError(err)

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
		}{codeId, pathToWasmBin})
	}

	return codeIds
}

func (s *WasmSuite) TestExecuteHappy() {
	deps := evmtest.NewTestDeps()
	wasmContracts := SetupWasmContracts(&deps, &s.Suite)
	wasmContract := wasmContracts[0] // nibi_stargate.wasm

	s.T().Log("Execute: create denom")
	msgArgsBz := []byte(`
	{ "create_denom": { 
	    "subdenom": "ETH" 
	   }
	}
	`)

	var funds []precompile.WasmBankCoin
	fundsJson, err := json.Marshal(funds)
	s.NoErrorf(err, "fundsJson: %s", fundsJson)
	err = json.Unmarshal(fundsJson, &funds)
	s.Require().NoError(err, "fundsJson %s, funds %s", fundsJson, funds)

	callArgs := []any{
		wasmContract.Addr.String(),
		msgArgsBz,
		funds,
	}
	input, err := embeds.SmartContract_Wasm.ABI.Pack(
		string(precompile.WasmMethod_execute),
		callArgs...,
	)
	s.Require().NoError(err)

	ethTxResp, err := deps.EvmKeeper.CallContractWithInput(
		deps.Ctx, deps.Sender.EthAddr, &precompile.PrecompileAddr_Wasm, true, input,
	)
	s.Require().NoError(err)
	s.Require().NotEmpty(ethTxResp.Ret)

	s.T().Log("Execute: mint tokens")
	coinDenom := tokenfactory.TFDenom{
		Creator:  wasmContract.Addr.String(),
		Subdenom: "ETH",
	}.Denom().String()
	msgArgsBz = []byte(fmt.Sprintf(`
	{ 
		"mint": { 
			"coin": { "amount": "69420", "denom": "%s" }, 
			"mint_to": "%s" 
		} 
	}
	`, coinDenom, deps.Sender.NibiruAddr))
	callArgs = []any{
		wasmContract.Addr.String(),
		msgArgsBz,
		funds,
	}
	input, err = embeds.SmartContract_Wasm.ABI.Pack(
		string(precompile.WasmMethod_execute),
		callArgs...,
	)
	s.Require().NoError(err)
	ethTxResp, err = deps.EvmKeeper.CallContractWithInput(
		deps.Ctx, deps.Sender.EthAddr, &precompile.PrecompileAddr_Wasm, true, input,
	)
	s.Require().NoError(err)
	s.Require().NotEmpty(ethTxResp.Ret)
	evmtest.AssertBankBalanceEqual(
		s.T(), deps, coinDenom, deps.Sender.EthAddr, big.NewInt(69420),
	)
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

func (s *WasmSuite) TestExecuteMultiHappy() {
	deps := evmtest.NewTestDeps()
	wasmContracts := SetupWasmContracts(&deps, &s.Suite)
	wasmContract := wasmContracts[1] // hello_world_counter.wasm

	s.assertWasmCounterState(deps, wasmContract, 0)                // count = 0
	s.incrementWasmCounterWithExecuteMulti(&deps, wasmContract, 2) // count += 2
	s.assertWasmCounterState(deps, wasmContract, 2)                // count = 2
	s.assertWasmCounterStateRaw(deps, wasmContract, 2)
	s.incrementWasmCounterWithExecuteMulti(&deps, wasmContract, 67) // count += 67
	s.assertWasmCounterState(deps, wasmContract, 69)                // count = 69
	s.assertWasmCounterStateRaw(deps, wasmContract, 69)
}

// From IWasm.query of Wasm.sol:
//
//	```solidity
//	function query(
//	  string memory contractAddr,
//	  bytes memory req
//	) external view returns (bytes memory response);
//	```
func (s *WasmSuite) assertWasmCounterState(
	deps evmtest.TestDeps,
	wasmContract WasmContract,
	wantCount int64,
) {
	msgArgsBz := []byte(`
		{ 
		  "count": {}
		}
		`)

	callArgs := []any{
		// string memory contractAddr
		wasmContract.Addr.String(),
		// bytes memory req
		msgArgsBz,
	}
	input, err := embeds.SmartContract_Wasm.ABI.Pack(
		string(precompile.WasmMethod_query),
		callArgs...,
	)
	s.Require().NoError(err)

	ethTxResp, err := deps.EvmKeeper.CallContractWithInput(
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
		// &[]any{...}
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
func (s *WasmSuite) incrementWasmCounterWithExecuteMulti(
	deps *evmtest.TestDeps,
	wasmContract WasmContract,
	times uint,
) {
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
		{wasmContract.Addr.String(), msgArgsBz, funds},
	}
	if times == 0 {
		executeMsgs = executeMsgs[:0] // force empty
	} else {
		for i := uint(1); i < times; i++ {
			executeMsgs = append(executeMsgs, executeMsgs[0])
		}
	}
	s.Require().Len(executeMsgs, int(times)) // sanity check assertion

	callArgs := []any{
		executeMsgs,
	}
	input, err := embeds.SmartContract_Wasm.ABI.Pack(
		string(precompile.WasmMethod_executeMulti),
		callArgs...,
	)
	s.Require().NoError(err)

	ethTxResp, err := deps.EvmKeeper.CallContractWithInput(
		deps.Ctx, deps.Sender.EthAddr, &precompile.PrecompileAddr_Wasm, true, input,
	)
	s.Require().NoError(err)
	s.Require().NotEmpty(ethTxResp.Ret)
}

// From IWasm.query of Wasm.sol:
//
//	```solidity
//	function queryRaw(
//	  string memory contractAddr,
//	  bytes memory key
//	) external view returns (bytes memory response);
//	```
func (s *WasmSuite) assertWasmCounterStateRaw(
	deps evmtest.TestDeps,
	wasmContract WasmContract,
	wantCount int64,
) {
	keyBz := []byte(`state`)
	callArgs := []any{
		wasmContract.Addr.String(),
		keyBz,
	}
	input, err := embeds.SmartContract_Wasm.ABI.Pack(
		string(precompile.WasmMethod_queryRaw),
		callArgs...,
	)
	s.Require().NoError(err)

	ethTxResp, err := deps.EvmKeeper.CallContractWithInput(
		deps.Ctx, deps.Sender.EthAddr, &precompile.PrecompileAddr_Wasm, true, input,
	)
	s.Require().NoError(err)
	s.Require().NotEmpty(ethTxResp.Ret)

	s.T().Log("Parse the response contract addr and response bytes")
	s.T().Logf("ethTxResp.Ret: %s", ethTxResp.Ret)

	var queryResp []byte
	err = embeds.SmartContract_Wasm.ABI.UnpackIntoInterface(
		&queryResp,
		string(precompile.WasmMethod_queryRaw),
		ethTxResp.Ret,
	)
	s.Require().NoError(err)
	s.T().Logf("queryResp: %s", queryResp)

	var wasmMsg wasm.RawContractMessage
	s.NoError(wasmMsg.UnmarshalJSON(queryResp), queryResp)
	s.T().Logf("wasmMsg: %s", wasmMsg)
	s.NoError(wasmMsg.ValidateBasic())

	var typedResp QueryMsgCountResp
	s.NoError(json.Unmarshal(wasmMsg, &typedResp))
	s.EqualValues(wantCount, typedResp.Count)
	s.EqualValues(deps.Sender.NibiruAddr.String(), typedResp.Owner)
}
