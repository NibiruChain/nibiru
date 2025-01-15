package precompile_test

import (
	"encoding/json"
	"fmt"
	"math/big"

	wasm "github.com/CosmWasm/wasmd/x/wasm/types"
	gethcommon "github.com/ethereum/go-ethereum/common"
	gethcore "github.com/ethereum/go-ethereum/core/types"

	"github.com/NibiruChain/nibiru/v2/x/common/testutil"
	"github.com/NibiruChain/nibiru/v2/x/common/testutil/testapp"
	"github.com/NibiruChain/nibiru/v2/x/evm"
	"github.com/NibiruChain/nibiru/v2/x/evm/embeds"
	"github.com/NibiruChain/nibiru/v2/x/evm/evmtest"
	"github.com/NibiruChain/nibiru/v2/x/evm/precompile"
	"github.com/NibiruChain/nibiru/v2/x/evm/precompile/test"
	"github.com/NibiruChain/nibiru/v2/x/evm/statedb"
	tokenfactory "github.com/NibiruChain/nibiru/v2/x/tokenfactory/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/suite"
)

// rough gas limits for wasm execution - used in tests only
const (
	WasmGasLimitQuery   uint64 = 200_000
	WasmGasLimitExecute uint64 = 1_000_000
)

type WasmSuite struct {
	suite.Suite
}

func (s *WasmSuite) TestExecuteHappy() {
	deps := evmtest.NewTestDeps()
	stateDB := deps.EvmKeeper.NewStateDB(deps.Ctx, statedb.NewEmptyTxConfig(gethcommon.BytesToHash(deps.Ctx.HeaderHash())))
	evmObj := deps.EvmKeeper.NewEVM(deps.Ctx, evmtest.MOCK_GETH_MESSAGE, deps.EvmKeeper.GetEVMConfig(deps.Ctx), evm.NewNoOpTracer(), stateDB)
	wasmContracts := test.SetupWasmContracts(&deps, &s.Suite)
	wasmContract := wasmContracts[0] // nibi_stargate.wasm

	s.Run("create denom", func() {
		msgArgsBz := []byte(`
	{ 
		"create_denom": { 
	    "subdenom": "ETH" 
	  }
	}
	`)

		var funds []precompile.WasmBankCoin
		fundsJson, err := json.Marshal(funds)
		s.NoErrorf(err, "fundsJson: %s", fundsJson)
		err = json.Unmarshal(fundsJson, &funds)
		s.Require().NoError(err, "fundsJson %s, funds %s", fundsJson, funds)

		contractInput, err := embeds.SmartContract_Wasm.ABI.Pack(
			string(precompile.WasmMethod_execute),
			wasmContract.String(),
			msgArgsBz,
			funds,
		)
		s.Require().NoError(err)
		ethTxResp, err := deps.EvmKeeper.CallContractWithInput(
			deps.Ctx,
			evmObj,
			deps.Sender.EthAddr,
			&precompile.PrecompileAddr_Wasm,
			true,
			contractInput,
			WasmGasLimitExecute,
		)
		s.Require().NoError(err)
		s.Require().NotEmpty(ethTxResp.Ret)
	})

	s.T().Log("Execute: mint tokens")
	s.Run("mint tokens", func() {
		coinDenom := tokenfactory.TFDenom{
			Creator:  wasmContract.String(),
			Subdenom: "ETH",
		}.Denom().String()
		msgArgsBz := []byte(fmt.Sprintf(`
		{ 
			"mint": { 
				"coin": { "amount": "69420", "denom": "%s" }, 
				"mint_to": "%s" 
			} 
		}
		`, coinDenom, deps.Sender.NibiruAddr))
		contractInput, err := embeds.SmartContract_Wasm.ABI.Pack(
			string(precompile.WasmMethod_execute),
			wasmContract.String(),
			msgArgsBz,
			[]precompile.WasmBankCoin{},
		)
		s.Require().NoError(err)

		ethTxResp, err := deps.EvmKeeper.CallContractWithInput(
			deps.Ctx,
			evmObj,
			deps.Sender.EthAddr,
			&precompile.PrecompileAddr_Wasm,
			true,
			contractInput,
			WasmGasLimitExecute,
		)

		s.Require().NoError(err)
		s.Require().NotEmpty(ethTxResp.Ret)
		evmtest.AssertBankBalanceEqualWithDescription(
			s.T(), deps, coinDenom, deps.Sender.EthAddr, big.NewInt(69_420), "expect 69420 balance")
	})
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
	deps.ResetGasMeter()

	contractInput, err := embeds.SmartContract_Wasm.ABI.Pack(
		string(precompile.WasmMethod_queryRaw),
		wasmContract.String(),
		[]byte(`state`),
	)
	s.Require().NoError(err)
	txConfig := deps.EvmKeeper.TxConfig(deps.Ctx, gethcommon.BigToHash(big.NewInt(0)))
	stateDB := deps.EvmKeeper.NewStateDB(deps.Ctx, txConfig)
	evmCfg := deps.EvmKeeper.GetEVMConfig(deps.Ctx)
	evmMsg := gethcore.NewMessage(
		evm.EVM_MODULE_ADDRESS,
		&evm.EVM_MODULE_ADDRESS,
		deps.EvmKeeper.GetAccNonce(deps.Ctx, evm.EVM_MODULE_ADDRESS),
		big.NewInt(0),
		WasmGasLimitQuery,
		big.NewInt(0),
		big.NewInt(0),
		big.NewInt(0),
		contractInput,
		gethcore.AccessList{},
		false,
	)
	evmObj := deps.EvmKeeper.NewEVM(deps.Ctx, evmMsg, evmCfg, nil /*tracer*/, stateDB)

	ethTxResp, err := deps.EvmKeeper.CallContractWithInput(
		deps.Ctx,
		evmObj,
		deps.Sender.EthAddr,
		&precompile.PrecompileAddr_Wasm,
		true,
		contractInput,
		WasmGasLimitQuery,
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

			contractInput, err := abi.Pack(
				string(tc.methodName),
				tc.callArgs...,
			)
			s.Require().NoError(err)
			txConfig := deps.EvmKeeper.TxConfig(deps.Ctx, gethcommon.BigToHash(big.NewInt(0)))
			stateDB := deps.EvmKeeper.NewStateDB(deps.Ctx, txConfig)
			evmCfg := deps.EvmKeeper.GetEVMConfig(deps.Ctx)
			evmMsg := gethcore.NewMessage(
				evm.EVM_MODULE_ADDRESS,
				&evm.EVM_MODULE_ADDRESS,
				deps.EvmKeeper.GetAccNonce(deps.Ctx, evm.EVM_MODULE_ADDRESS),
				big.NewInt(0),
				WasmGasLimitExecute,
				big.NewInt(0),
				big.NewInt(0),
				big.NewInt(0),
				contractInput,
				gethcore.AccessList{},
				false,
			)
			evmObj := deps.EvmKeeper.NewEVM(deps.Ctx, evmMsg, evmCfg, nil /*tracer*/, stateDB)

			ethTxResp, err := deps.EvmKeeper.CallContractWithInput(
				deps.Ctx,
				evmObj,
				deps.Sender.EthAddr,
				&precompile.PrecompileAddr_Wasm,
				true,
				contractInput,
				WasmGasLimitExecute,
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

	wasmContracts := test.SetupWasmContracts(&deps, &s.Suite)
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
			contractInput, err := embeds.SmartContract_Wasm.ABI.Pack(
				string(precompile.WasmMethod_executeMulti),
				tc.executeMsgs,
			)
			s.Require().NoError(err)
			txConfig := deps.EvmKeeper.TxConfig(deps.Ctx, gethcommon.BigToHash(big.NewInt(0)))
			stateDB := deps.EvmKeeper.NewStateDB(deps.Ctx, txConfig)
			evmCfg := deps.EvmKeeper.GetEVMConfig(deps.Ctx)
			evmMsg := gethcore.NewMessage(
				evm.EVM_MODULE_ADDRESS,
				&evm.EVM_MODULE_ADDRESS,
				deps.EvmKeeper.GetAccNonce(deps.Ctx, evm.EVM_MODULE_ADDRESS),
				big.NewInt(0),
				WasmGasLimitExecute,
				big.NewInt(0),
				big.NewInt(0),
				big.NewInt(0),
				contractInput,
				gethcore.AccessList{},
				false,
			)
			evmObj := deps.EvmKeeper.NewEVM(deps.Ctx, evmMsg, evmCfg, nil /*tracer*/, stateDB)
			ethTxResp, err := deps.EvmKeeper.CallContractWithInput(
				deps.Ctx,
				evmObj,
				deps.Sender.EthAddr,
				&precompile.PrecompileAddr_Wasm,
				true,
				contractInput,
				WasmGasLimitExecute,
			)

			if tc.wantError != "" {
				s.Require().ErrorContains(err, tc.wantError)
				return
			}
			s.Require().NoError(err)
			s.NotNil(ethTxResp)
			s.NotEmpty(ethTxResp.Ret)
		})
	}
}

// TestExecuteMultiPartialExecution ensures that no state changes occur if any message
// in the batch fails validation
func (s *WasmSuite) TestExecuteMultiPartialExecution() {
	deps := evmtest.NewTestDeps()
	wasmContracts := test.SetupWasmContracts(&deps, &s.Suite)
	wasmContract := wasmContracts[1] // hello_world_counter.wasm

	// First verify initial state is 0
	test.AssertWasmCounterState(&s.Suite, deps, wasmContract, 0)

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

	contractInput, err := embeds.SmartContract_Wasm.ABI.Pack(
		string(precompile.WasmMethod_executeMulti),
		executeMsgs,
	)
	s.Require().NoError(err)
	txConfig := deps.EvmKeeper.TxConfig(deps.Ctx, gethcommon.BigToHash(big.NewInt(0)))
	stateDB := deps.EvmKeeper.NewStateDB(deps.Ctx, txConfig)
	evmCfg := deps.EvmKeeper.GetEVMConfig(deps.Ctx)
	evmMsg := gethcore.NewMessage(
		evm.EVM_MODULE_ADDRESS,
		&evm.EVM_MODULE_ADDRESS,
		deps.EvmKeeper.GetAccNonce(deps.Ctx, evm.EVM_MODULE_ADDRESS),
		big.NewInt(0),
		WasmGasLimitExecute,
		big.NewInt(0),
		big.NewInt(0),
		big.NewInt(0),
		contractInput,
		gethcore.AccessList{},
		false,
	)
	evmObj := deps.EvmKeeper.NewEVM(deps.Ctx, evmMsg, evmCfg, nil /*tracer*/, stateDB)
	ethTxResp, err := deps.EvmKeeper.CallContractWithInput(
		deps.Ctx,
		evmObj,
		deps.Sender.EthAddr,
		&precompile.PrecompileAddr_Wasm,
		true,
		contractInput,
		WasmGasLimitExecute,
	)

	// Verify that the call failed
	s.Require().Error(err, "ethTxResp: ", ethTxResp)
	s.Require().Contains(err.Error(), "unknown variant")

	// Verify that no state changes occurred
	test.AssertWasmCounterState(&s.Suite, deps, wasmContract, 0)
}
