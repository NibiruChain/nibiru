package precompile_test

import (
	"encoding/json"
	"fmt"
	"math/big"

	wasm "github.com/CosmWasm/wasmd/x/wasm/types"

	"github.com/NibiruChain/nibiru/v2/x/common/testutil"
	"github.com/NibiruChain/nibiru/v2/x/common/testutil/testapp"
	"github.com/NibiruChain/nibiru/v2/x/evm/embeds"
	"github.com/NibiruChain/nibiru/v2/x/evm/evmtest"
	"github.com/NibiruChain/nibiru/v2/x/evm/precompile"
	"github.com/NibiruChain/nibiru/v2/x/evm/precompile/test"
	tokenfactory "github.com/NibiruChain/nibiru/v2/x/tokenfactory/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/suite"
)

type WasmSuite struct {
	suite.Suite
}

func (s *WasmSuite) TestExecuteHappy() {
	deps := evmtest.NewTestDeps()
	wasmContracts := test.SetupWasmContracts(&deps, &s.Suite)
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
		wasmContract.String(),
		msgArgsBz,
		funds,
	}
	input, err := embeds.SmartContract_Wasm.ABI.Pack(
		string(precompile.WasmMethod_execute),
		callArgs...,
	)
	s.Require().NoError(err)

	ethTxResp, _, err := deps.EvmKeeper.CallContractWithInput(
		deps.Ctx, deps.Sender.EthAddr, &precompile.PrecompileAddr_Wasm, true, input,
	)
	s.Require().NoError(err)
	s.Require().NotEmpty(ethTxResp.Ret)

	s.T().Log("Execute: mint tokens")
	coinDenom := tokenfactory.TFDenom{
		Creator:  wasmContract.String(),
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
		wasmContract.String(),
		msgArgsBz,
		funds,
	}
	input, err = embeds.SmartContract_Wasm.ABI.Pack(
		string(precompile.WasmMethod_execute),
		callArgs...,
	)
	s.Require().NoError(err)
	ethTxResp, _, err = deps.EvmKeeper.CallContractWithInput(
		deps.Ctx, deps.Sender.EthAddr, &precompile.PrecompileAddr_Wasm, true, input,
	)
	s.Require().NoError(err)
	s.Require().NotEmpty(ethTxResp.Ret)
	evmtest.AssertBankBalanceEqual(
		s.T(), deps, coinDenom, deps.Sender.EthAddr, big.NewInt(69420),
	)
}

func (s *WasmSuite) TestExecuteMultiHappy() {
	deps := evmtest.NewTestDeps()
	wasmContracts := test.SetupWasmContracts(&deps, &s.Suite)
	wasmContract := wasmContracts[1] // hello_world_counter.wasm

	// count = 0
	test.AssertWasmCounterState(&s.Suite, deps, wasmContract, 0)
	// count += 2
	test.IncrementWasmCounterWithExecuteMulti(
		&s.Suite, &deps, wasmContract, 2, true)
	// count = 2
	test.AssertWasmCounterState(&s.Suite, deps, wasmContract, 2)
	s.assertWasmCounterStateRaw(deps, wasmContract, 2)
	// count += 67
	test.IncrementWasmCounterWithExecuteMulti(
		&s.Suite, &deps, wasmContract, 67, true)
	// count = 69
	test.AssertWasmCounterState(&s.Suite, deps, wasmContract, 69)
	s.assertWasmCounterStateRaw(deps, wasmContract, 69)
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
	wasmContract sdk.AccAddress,
	wantCount int64,
) {
	keyBz := []byte(`state`)
	callArgs := []any{
		wasmContract.String(),
		keyBz,
	}
	input, err := embeds.SmartContract_Wasm.ABI.Pack(
		string(precompile.WasmMethod_queryRaw),
		callArgs...,
	)
	s.Require().NoError(err)

	ethTxResp, _, err := deps.EvmKeeper.CallContractWithInput(
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

	var typedResp test.QueryMsgCountResp
	s.NoError(json.Unmarshal(wasmMsg, &typedResp))
	s.EqualValues(wantCount, typedResp.Count)
	s.EqualValues(deps.Sender.NibiruAddr.String(), typedResp.Owner)
}

func (s *WasmSuite) TestSadArgsCount() {
	nonsenseArgs := []any{"nonsense", "args here", "to see if", "precompile is", "called"}
	testcases := []struct {
		name       string
		methodName precompile.PrecompileMethod
		callArgs   []any
		wantError  string
	}{
		{
			name:       "execute",
			methodName: precompile.WasmMethod_execute,
			callArgs:   nonsenseArgs,
			wantError:  "argument count mismatch: got 5 for 3",
		},
		{
			name:       "executeMulti",
			methodName: precompile.WasmMethod_executeMulti,
			callArgs:   nonsenseArgs,
			wantError:  "argument count mismatch: got 5 for 1",
		},
		{
			name:       "query",
			methodName: precompile.WasmMethod_query,
			callArgs:   nonsenseArgs,
			wantError:  "argument count mismatch: got 5 for 2",
		},
		{
			name:       "queryRaw",
			methodName: precompile.WasmMethod_queryRaw,
			callArgs:   nonsenseArgs,
			wantError:  "argument count mismatch: got 5 for 2",
		},
		{
			name:       "instantiate",
			methodName: precompile.WasmMethod_instantiate,
			callArgs:   nonsenseArgs[:4],
			wantError:  "argument count mismatch: got 4 for 5",
		},
		{
			name:       "invalid method name",
			methodName: "not_a_method",
			callArgs:   nonsenseArgs,
			wantError:  "method 'not_a_method' not found",
		},
	}

	abi := embeds.SmartContract_Wasm.ABI
	for _, tc := range testcases {
		s.Run(tc.name, func() {
			callArgs := tc.callArgs
			_, err := abi.Pack(
				string(tc.methodName),
				callArgs...,
			)
			s.Require().ErrorContains(err, tc.wantError)
		})
	}
}

func (s *WasmSuite) TestSadArgsExecute() {
	methodName := precompile.WasmMethod_execute
	contractAddr := testutil.AccAddress().String()
	wasmContractMsg := []byte(`
	{ "create_denom": {
		"subdenom": "ETH"
	   }
	}
	`)
	{
		wasmMsg := wasm.RawContractMessage(wasmContractMsg)
		s.Require().NoError(wasmMsg.ValidateBasic())
	}

	testcases := []struct {
		name       string
		methodName precompile.PrecompileMethod
		callArgs   []any
		wantError  string
	}{
		{
			name:       "valid arg types, should get VM error",
			methodName: methodName,
			callArgs: []any{
				// contractAddr
				contractAddr,
				// msgArgBz
				wasmContractMsg,
				// funds
				[]precompile.WasmBankCoin{},
			},
			wantError: "execute method called",
		},
		{
			name:       "contractAddr",
			methodName: methodName,
			callArgs: []any{
				// contractAddr
				contractAddr + "malformed", // mess up bech32
				// msgArgBz
				wasmContractMsg,
				// funds
				[]precompile.WasmBankCoin{},
			},
			wantError: "decoding bech32 failed",
		},
		{
			name:       "funds populated",
			methodName: methodName,
			callArgs: []any{
				// contractAddr
				contractAddr,
				// msgArgBz
				[]byte(`[]`),
				// funds
				[]precompile.WasmBankCoin{
					{
						Denom:  "x-123a!$",
						Amount: big.NewInt(123),
					},
					{
						Denom:  "xyz",
						Amount: big.NewInt(456),
					},
				},
			},
			wantError: "no such contract",
		},
	}

	abi := embeds.SmartContract_Wasm.ABI
	for _, tc := range testcases {
		s.Run(tc.name, func() {
			deps := evmtest.NewTestDeps()

			callArgs := tc.callArgs
			input, err := abi.Pack(
				string(tc.methodName),
				callArgs...,
			)
			s.Require().NoError(err)

			ethTxResp, _, err := deps.EvmKeeper.CallContractWithInput(
				deps.Ctx, deps.Sender.EthAddr, &precompile.PrecompileAddr_Wasm, true, input,
			)
			s.Require().ErrorContains(err, tc.wantError, "ethTxResp %v", ethTxResp)
		})
	}
}

type WasmExecuteMsg struct {
	ContractAddr string                    `json:"contractAddr"`
	MsgArgs      []byte                    `json:"msgArgs"`
	Funds        []precompile.WasmBankCoin `json:"funds"`
}

func (s *WasmSuite) TestExecuteMultiValidation() {
	deps := evmtest.NewTestDeps()

	s.Require().NoError(testapp.FundAccount(
		deps.App.BankKeeper,
		deps.Ctx,
		deps.Sender.NibiruAddr,
		sdk.NewCoins(sdk.NewCoin("unibi", sdk.NewInt(100))),
	))

	wasmContracts := SetupWasmContracts(&deps, &s.Suite)
	wasmContract := wasmContracts[1] // hello_world_counter.wasm

	invalidMsgArgsBz := []byte(`{"invalid": "json"}`) // Invalid message format
	validMsgArgsBz := []byte(`{"increment": {}}`)     // Valid increment message

	var emptyFunds []precompile.WasmBankCoin
	validFunds := []precompile.WasmBankCoin{{
		Denom:  "unibi",
		Amount: big.NewInt(100),
	}}
	invalidFunds := []precompile.WasmBankCoin{{
		Denom:  "invalid!denom",
		Amount: big.NewInt(100),
	}}

	testCases := []struct {
		name        string
		executeMsgs []WasmExecuteMsg
		wantError   string
	}{
		{
			name: "valid - single message",
			executeMsgs: []WasmExecuteMsg{
				{
					ContractAddr: wasmContract.String(),
					MsgArgs:      validMsgArgsBz,
					Funds:        emptyFunds,
				},
			},
			wantError: "",
		},
		{
			name: "valid - multiple messages",
			executeMsgs: []WasmExecuteMsg{
				{
					ContractAddr: wasmContract.String(),
					MsgArgs:      validMsgArgsBz,
					Funds:        validFunds,
				},
				{
					ContractAddr: wasmContract.String(),
					MsgArgs:      validMsgArgsBz,
					Funds:        emptyFunds,
				},
			},
			wantError: "",
		},
		{
			name: "invalid - malformed contract address",
			executeMsgs: []WasmExecuteMsg{
				{
					ContractAddr: "invalid-address",
					MsgArgs:      validMsgArgsBz,
					Funds:        emptyFunds,
				},
			},
			wantError: "decoding bech32 failed",
		},
		{
			name: "invalid - malformed message args",
			executeMsgs: []WasmExecuteMsg{
				{
					ContractAddr: wasmContract.String(),
					MsgArgs:      invalidMsgArgsBz,
					Funds:        emptyFunds,
				},
			},
			wantError: "unknown variant",
		},
		{
			name: "invalid - malformed funds",
			executeMsgs: []WasmExecuteMsg{
				{
					ContractAddr: wasmContract.String(),
					MsgArgs:      validMsgArgsBz,
					Funds:        invalidFunds,
				},
			},
			wantError: "invalid coins",
		},
		{
			name: "invalid - second message fails validation",
			executeMsgs: []WasmExecuteMsg{
				{
					ContractAddr: wasmContract.String(),
					MsgArgs:      validMsgArgsBz,
					Funds:        emptyFunds,
				},
				{
					ContractAddr: wasmContract.String(),
					MsgArgs:      invalidMsgArgsBz,
					Funds:        emptyFunds,
				},
			},
			wantError: "unknown variant",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			callArgs := []any{tc.executeMsgs}
			input, err := embeds.SmartContract_Wasm.ABI.Pack(
				string(precompile.WasmMethod_executeMulti),
				callArgs...,
			)
			s.Require().NoError(err)

			ethTxResp, err := deps.EvmKeeper.CallContractWithInput(
				deps.Ctx, deps.Sender.EthAddr, &precompile.PrecompileAddr_Wasm, true, input,
			)

			if tc.wantError != "" {
				s.Require().Error(err)
				s.Require().Contains(err.Error(), tc.wantError)
				s.Require().Nil(ethTxResp)
			} else {
				s.Require().NoError(err)
				s.Require().NotNil(ethTxResp)
				s.Require().NotEmpty(ethTxResp.Ret)
			}
		})
	}
}

// TestExecuteMultiPartialExecution ensures that no state changes occur if any message
// in the batch fails validation
func (s *WasmSuite) TestExecuteMultiPartialExecution() {
	deps := evmtest.NewTestDeps()
	wasmContracts := SetupWasmContracts(&deps, &s.Suite)
	wasmContract := wasmContracts[1] // hello_world_counter.wasm

	// First verify initial state is 0
	s.assertWasmCounterState(deps, wasmContract, 0)

	// Create a batch where the second message will fail validation
	executeMsgs := []WasmExecuteMsg{
		{
			ContractAddr: wasmContract.String(),
			MsgArgs:      []byte(`{"increment": {}}`),
			Funds:        []precompile.WasmBankCoin{},
		},
		{
			ContractAddr: wasmContract.String(),
			MsgArgs:      []byte(`{"invalid": "json"}`), // This will fail validation
			Funds:        []precompile.WasmBankCoin{},
		},
	}

	callArgs := []any{executeMsgs}
	input, err := embeds.SmartContract_Wasm.ABI.Pack(
		string(precompile.WasmMethod_executeMulti),
		callArgs...,
	)
	s.Require().NoError(err)

	ethTxResp, err := deps.EvmKeeper.CallContractWithInput(
		deps.Ctx, deps.Sender.EthAddr, &precompile.PrecompileAddr_Wasm, true, input,
	)

	// Verify that the call failed
	s.Require().Error(err)
	s.Require().Contains(err.Error(), "unknown variant")
	s.Require().Nil(ethTxResp)

	// Verify that no state changes occurred
	s.assertWasmCounterState(deps, wasmContract, 0)
}
