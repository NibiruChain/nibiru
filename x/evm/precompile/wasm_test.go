package precompile_test

import (
	"encoding/json"
	"fmt"
	"math/big"

	wasm "github.com/CosmWasm/wasmd/x/wasm/types"

	"github.com/NibiruChain/nibiru/v2/x/common/testutil"
	"github.com/NibiruChain/nibiru/v2/x/evm/embeds"
	"github.com/NibiruChain/nibiru/v2/x/evm/evmtest"
	"github.com/NibiruChain/nibiru/v2/x/evm/precompile"
	"github.com/NibiruChain/nibiru/v2/x/evm/precompile/test"
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
		deps.Ctx,
		deps.Sender.EthAddr,
		&precompile.PrecompileAddr_Wasm,
		true,
		input,
		WasmGasLimitExecute,
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
		deps.Ctx,
		deps.Sender.EthAddr,
		&precompile.PrecompileAddr_Wasm,
		true,
		input,
		WasmGasLimitExecute,
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
		&s.Suite, &deps, wasmContract, 2)
	// count = 2
	test.AssertWasmCounterState(&s.Suite, deps, wasmContract, 2)
	s.assertWasmCounterStateRaw(deps, wasmContract, 2)
	// count += 67
	test.IncrementWasmCounterWithExecuteMulti(
		&s.Suite, &deps, wasmContract, 67)
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

	deps.ResetGasMeter()

	ethTxResp, _, err := deps.EvmKeeper.CallContractWithInput(
		deps.Ctx,
		deps.Sender.EthAddr,
		&precompile.PrecompileAddr_Wasm,
		true,
		input,
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

			callArgs := tc.callArgs
			input, err := abi.Pack(
				string(tc.methodName),
				callArgs...,
			)
			s.Require().NoError(err)

			ethTxResp, _, err := deps.EvmKeeper.CallContractWithInput(
				deps.Ctx,
				deps.Sender.EthAddr,
				&precompile.PrecompileAddr_Wasm,
				true,
				input,
				WasmGasLimitExecute,
			)
			s.Require().ErrorContains(err, tc.wantError, "ethTxResp %v", ethTxResp)
		})
	}
}
