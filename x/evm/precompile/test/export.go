package test

import (
	"encoding/json"
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
	"github.com/NibiruChain/nibiru/v2/x/evm"
	"github.com/NibiruChain/nibiru/v2/x/evm/embeds"
	"github.com/NibiruChain/nibiru/v2/x/evm/evmtest"
	"github.com/NibiruChain/nibiru/v2/x/evm/precompile"
	"github.com/NibiruChain/nibiru/v2/x/evm/statedb"
)

// rough gas limits for wasm execution - used in tests only
const (
	WasmGasLimitInstantiate uint64 = 1_000_000
	WasmGasLimitExecute     uint64 = 10_000_000
	WasmGasLimitQuery       uint64 = 200_000
)

// SetupWasmContracts stores all Wasm bytecode and has the "deps.Sender"
// instantiate each Wasm contract using the precompile.
func SetupWasmContracts(deps *evmtest.TestDeps, s *suite.Suite) (
	contracts []sdk.AccAddress,
) {
	wasmCodes := deployWasmBytecode(s, deps.Ctx, deps.Sender.NibiruAddr, deps.App)

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
		{
			InstantiateMsg: []byte(`{}`),
			Label:          "https://github.com/k-yang/nibiru-wasm-plus/tree/main/contracts/bank-transfer/",
		},
		{
			InstantiateMsg: []byte(`{}`),
			Label:          "https://github.com/k-yang/nibiru-wasm-plus/tree/main/contracts/staking/",
		},
	}

	for i, wasmCode := range wasmCodes {
		s.T().Logf("Instantiate %s", wasmCode.binPath)

		wasmPermissionedKeeper := wasmkeeper.NewDefaultPermissionKeeper(deps.App.WasmKeeper)
		contractAddr, _, err := wasmPermissionedKeeper.Instantiate(
			deps.Ctx,
			wasmCode.codeId,
			deps.Sender.NibiruAddr,
			deps.Sender.NibiruAddr,
			instantiateArgs[i].InstantiateMsg,
			instantiateArgs[i].Label,
			sdk.Coins{},
		)
		s.NoError(err)
		contracts = append(contracts, contractAddr)
	}

	return contracts
}

// deployWasmBytecode is a setup function that stores all Wasm bytecode used in
// the test suite.
func deployWasmBytecode(
	s *suite.Suite,
	ctx sdk.Context,
	sender sdk.AccAddress,
	app *app.NibiruApp,
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
		"x/evm/precompile/test/hello_world_counter.wasm",

		// bank_transfer.wasm is a compiled version of:
		// https://github.com/k-yang/nibiru-wasm-plus/tree/main/contracts/bank-transfer/
		"x/evm/precompile/test/bank_transfer.wasm",

		// staking.wasm is a compiled version of:
		// https://github.com/k-yang/nibiru-wasm-plus/tree/main/contracts/staking/
		"x/evm/precompile/test/staking.wasm",

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
		wasmPermissionedKeeper := wasmkeeper.NewDefaultPermissionKeeper(app.WasmKeeper)
		codeId, _, err := wasmPermissionedKeeper.Create(
			ctx, sender, wasmBytecode, &wasm.AccessConfig{
				Permission: wasm.AccessTypeEverybody,
			},
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
) {
	msgArgsBz := []byte(`
		{ 
		  "count": {}
		}
`)

	resp, err := deps.App.WasmKeeper.QuerySmart(deps.Ctx, wasmContract, msgArgsBz)
	s.Require().NoError(err)
	s.Require().NotEmpty(resp)

	var typedResp QueryMsgCountResp
	s.NoError(json.Unmarshal(resp, &typedResp))

	s.EqualValues(wantCount, typedResp.Count)
	s.EqualValues(deps.Sender.NibiruAddr.String(), typedResp.Owner)
}

func AssertWasmCounterStateWithEvm(
	s *suite.Suite,
	deps evmtest.TestDeps,
	evmObj *vm.EVM,
	wasmContract sdk.AccAddress,
	wantCount int64,
) {
	msgArgsBz := []byte(`
		{ 
		  "count": {}
		}
	`)

	contractInput, err := embeds.SmartContract_Wasm.ABI.Pack(
		string(precompile.WasmMethod_query),
		wasmContract.String(),
		msgArgsBz,
	)
	s.Require().NoError(err)

	evmResp, err := deps.EvmKeeper.CallContract(
		deps.Ctx,
		evmObj,
		deps.Sender.EthAddr,
		&precompile.PrecompileAddr_Wasm,
		contractInput,
		WasmGasLimitQuery,
		evm.COMMIT_READONLY, /*commit*/
		nil,
	)
	s.Require().NoError(err)
	s.Require().NotEmpty(evmResp.Ret)

	var queryResp []byte
	s.NoError(embeds.SmartContract_Wasm.ABI.UnpackIntoInterface(
		&queryResp,
		string(precompile.WasmMethod_query),
		evmResp.Ret,
	))

	var typedResp QueryMsgCountResp
	s.NoError(json.Unmarshal(queryResp, &typedResp))

	s.EqualValues(wantCount, typedResp.Count)
	s.EqualValues(deps.Sender.NibiruAddr.String(), typedResp.Owner)
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
	evmObj *vm.EVM,
	wasmContract sdk.AccAddress,
	times uint,
	commit bool,
) {
	msgArgsBz := []byte(`
	{ 
	  "increment": {}
	}
	`)

	// The "times" arg determines the number of messages in the executeMsgs slice
	executeMsgs := []struct {
		ContractAddr string                    `json:"contractAddr"`
		MsgArgs      []byte                    `json:"msgArgs"`
		Funds        []precompile.WasmBankCoin `json:"funds"`
	}{}

	for i := uint(0); i < times; i++ {
		executeMsgs = append(executeMsgs, struct {
			ContractAddr string                    `json:"contractAddr"`
			MsgArgs      []byte                    `json:"msgArgs"`
			Funds        []precompile.WasmBankCoin `json:"funds"`
		}{wasmContract.String(), msgArgsBz, []precompile.WasmBankCoin{}})
	}

	contractInput, err := embeds.SmartContract_Wasm.ABI.Pack(
		string(precompile.WasmMethod_executeMulti),
		executeMsgs,
	)
	s.Require().NoError(err)

	ethTxResp, err := deps.EvmKeeper.CallContract(
		deps.Ctx,
		evmObj,
		deps.Sender.EthAddr,
		&precompile.PrecompileAddr_Wasm,
		contractInput,
		WasmGasLimitExecute,
		commit, /*commit*/
		nil,
	)
	s.Require().NoError(err)
	s.Require().NotEmpty(ethTxResp.Ret)
	if commit {
		s.Require().NoError(evmObj.StateDB.(*statedb.StateDB).Commit())
	}
}
