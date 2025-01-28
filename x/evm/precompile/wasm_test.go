package precompile_test

import (
	"encoding/json"
	"fmt"
	"math/big"
	"testing"

	"cosmossdk.io/math"
	wasm "github.com/CosmWasm/wasmd/x/wasm/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/stretchr/testify/suite"

	"github.com/NibiruChain/nibiru/v2/eth"
	"github.com/NibiruChain/nibiru/v2/x/common/testutil"
	"github.com/NibiruChain/nibiru/v2/x/common/testutil/testapp"
	"github.com/NibiruChain/nibiru/v2/x/evm"
	"github.com/NibiruChain/nibiru/v2/x/evm/embeds"
	"github.com/NibiruChain/nibiru/v2/x/evm/evmtest"
	"github.com/NibiruChain/nibiru/v2/x/evm/precompile"
	"github.com/NibiruChain/nibiru/v2/x/evm/precompile/test"
	tokenfactory "github.com/NibiruChain/nibiru/v2/x/tokenfactory/types"
)

// rough gas limits for wasm execution - used in tests only
const (
	WasmGasLimitQuery   uint64 = 200_000
	WasmGasLimitExecute uint64 = 1_000_000
)

type WasmSuite struct {
	suite.Suite
}

func TestWasmSuite(t *testing.T) {
	suite.Run(t, new(WasmSuite))
}

func (s *WasmSuite) TestInstantiate() {
	deps := evmtest.NewTestDeps()
	evmObj, _ := deps.NewEVM()

	test.SetupWasmContracts(&deps, &s.Suite)

	contractInput, err := embeds.SmartContract_Wasm.ABI.Pack(
		string(precompile.WasmMethod_instantiate),
		"",                          // admin
		uint64(1),                   // codeId
		[]byte(`{}`),                // instantiateMsg
		"some non-empty label",      // label
		[]precompile.WasmBankCoin{}, // funds
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

	vals, err := embeds.SmartContract_Wasm.ABI.Unpack(
		string(precompile.WasmMethod_instantiate),
		ethTxResp.Ret,
	)
	s.Require().NoError(err)
	s.Require().NotEmpty(vals[0].(string))
}

func (s *WasmSuite) TestExecute() {
	deps := evmtest.NewTestDeps()
	evmObj, _ := deps.NewEVM()

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
	})

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

		evmObj, _ = deps.NewEVM()
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

func (s *WasmSuite) TestExecuteMulti() {
	deps := evmtest.NewTestDeps()
	evmObj, _ := deps.NewEVM()
	wasmContracts := test.SetupWasmContracts(&deps, &s.Suite)
	wasmContract := wasmContracts[1] // hello_world_counter.wasm

	// count = 0
	test.AssertWasmCounterState(&s.Suite, deps, wasmContract, 0)
	// count += 2
	test.IncrementWasmCounterWithExecuteMulti(
		&s.Suite, &deps, evmObj, wasmContract, 2, true)
	// count = 2
	test.AssertWasmCounterState(&s.Suite, deps, wasmContract, 2)
	// count += 67
	test.IncrementWasmCounterWithExecuteMulti(
		&s.Suite, &deps, evmObj, wasmContract, 67, true)
	// count = 69
	test.AssertWasmCounterState(&s.Suite, deps, wasmContract, 69)
}

func (s *WasmSuite) TestQueryRaw() {
	deps := evmtest.NewTestDeps()
	wasmContracts := test.SetupWasmContracts(&deps, &s.Suite)
	wasmContract := wasmContracts[1] // hello_world_counter.wasm

	contractInput, err := embeds.SmartContract_Wasm.ABI.Pack(
		string(precompile.WasmMethod_queryRaw),
		wasmContract.String(),
		[]byte(`state`),
	)
	s.Require().NoError(err)

	evmObj, _ := deps.NewEVM()
	queryResp, err := deps.EvmKeeper.CallContractWithInput(
		deps.Ctx,
		evmObj,
		deps.Sender.EthAddr,
		&precompile.PrecompileAddr_Wasm,
		false, // commit
		contractInput,
		WasmGasLimitQuery,
	)

	s.Require().NoError(err)
	s.Require().NotEmpty(queryResp.Ret)

	var respBz []byte
	err = embeds.SmartContract_Wasm.ABI.UnpackIntoInterface(
		&respBz,
		string(precompile.WasmMethod_queryRaw),
		queryResp.Ret,
	)
	s.Require().NoError(err, "ethTxResp: %s", queryResp)

	var typedResp test.QueryMsgCountResp
	s.NoError(json.Unmarshal(respBz, &typedResp))
	s.EqualValues(0, typedResp.Count)
	s.EqualValues(deps.Sender.NibiruAddr.String(), typedResp.Owner)
}

func (s *WasmSuite) TestQuerySmart() {
	deps := evmtest.NewTestDeps()
	wasmContracts := test.SetupWasmContracts(&deps, &s.Suite)
	wasmContract := wasmContracts[1] // hello_world_counter.wasm

	contractInput, err := embeds.SmartContract_Wasm.ABI.Pack(
		string(precompile.WasmMethod_query),
		wasmContract.String(),
		[]byte(`{"count": {}}`),
	)
	s.Require().NoError(err)

	evmObj, _ := deps.NewEVM()
	queryResp, err := deps.EvmKeeper.CallContractWithInput(
		deps.Ctx,
		evmObj,
		deps.Sender.EthAddr,
		&precompile.PrecompileAddr_Wasm,
		false, // commit
		contractInput,
		WasmGasLimitQuery,
	)

	s.Require().NoError(err)
	s.Require().NotEmpty(queryResp.Ret)

	var respBz []byte
	err = embeds.SmartContract_Wasm.ABI.UnpackIntoInterface(
		&respBz,
		string(precompile.WasmMethod_query),
		queryResp.Ret,
	)
	s.Require().NoError(err, "ethTxResp: %s", queryResp)

	var typedResp test.QueryMsgCountResp
	s.Require().NoError(json.Unmarshal(respBz, &typedResp))
	s.Require().EqualValues(0, typedResp.Count)
	s.Require().EqualValues(deps.Sender.NibiruAddr.String(), typedResp.Owner)
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
			evmObj, _ := deps.NewEVM()

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
		sdk.NewCoins(sdk.NewCoin(evm.EVMBankDenom, sdk.NewInt(100))),
	))

	wasmContracts := test.SetupWasmContracts(&deps, &s.Suite)
	wasmContract := wasmContracts[1] // hello_world_counter.wasm

	validMsgArgsBz := []byte(`{"increment": {}}`)     // Valid increment message
	invalidMsgArgsBz := []byte(`{"invalid": "json"}`) // Invalid message format

	var emptyFunds []precompile.WasmBankCoin
	validFunds := []precompile.WasmBankCoin{{
		Denom:  evm.EVMBankDenom,
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
			evmObj, _ := deps.NewEVM()
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
	evmObj, _ := deps.NewEVM()

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

// TestDirtyStateAttack4
//  1. Deploy a simple wasm contract that bank transfers NIBI to a recipient (Alice)
//  2. Calls the test contract
//     a. call the wasm precompile which calls the wasm contract with a bank transfer
//
// INITIAL STATE:
// - Test contract funds: 10 NIBI
// CONTRACT CALL:
// - Sends 1 NIBI to Alice via wasm precompile
// EXPECTED:
// - Test contract funds: 9 NIBI
// - Alice: 1 NIBI
func (s *WasmSuite) TestWasmPrecompileDirtyStateAttack4() {
	deps := evmtest.NewTestDeps()

	wasmContracts := test.SetupWasmContracts(&deps, &s.Suite)
	wasmContract := wasmContracts[2] // bank_transfer.wasm

	s.T().Log("Deploy Test Contract")
	deployResp, err := evmtest.DeployContract(
		&deps,
		embeds.SmartContract_TestDirtyStateAttack4,
	)
	s.Require().NoError(err)
	testContractAddr := deployResp.ContractAddr

	s.Run("Send 10 NIBI to test contract manually", func() {
		s.Require().NoError(testapp.FundAccount(
			deps.App.BankKeeper,
			deps.Ctx,
			eth.EthAddrToNibiruAddr(testContractAddr),
			sdk.NewCoins(sdk.NewCoin(evm.EVMBankDenom, sdk.NewInt(10e6))),
		))
	})

	alice := evmtest.NewEthPrivAcc()

	s.Run("call test contract", func() {
		msgArgsBz := []byte(fmt.Sprintf(`
		{
			"bank_transfer": {
				"recipient": "%s"
			}
		}
		`, alice.NibiruAddr))
		contractInput, err := embeds.SmartContract_TestDirtyStateAttack4.ABI.Pack(
			"attack",
			wasmContract.String(),
			msgArgsBz,
		)
		s.Require().NoError(err)

		evmObj, _ := deps.NewEVM()
		_, err = deps.EvmKeeper.CallContractWithInput(
			deps.Ctx,
			evmObj,
			deps.Sender.EthAddr,
			&testContractAddr,
			true,
			contractInput,
			evmtest.FunTokenGasLimitSendToEvm,
		)
		s.Require().NoError(err)

		balanceAlice := deps.App.BankKeeper.GetBalance(deps.Ctx, alice.NibiruAddr, evm.EVMBankDenom)
		s.Require().Equal(balanceAlice.Amount.BigInt(), big.NewInt(1e6))

		balanceTestContract := deps.App.BankKeeper.GetBalance(deps.Ctx, eth.EthAddrToNibiruAddr(testContractAddr), evm.EVMBankDenom)
		s.Require().Equal(balanceTestContract.Amount.BigInt(), big.NewInt(9e6))
	})
}

// TestDirtyStateAttack5
//  1. Deploy a simple wasm contract that stakes NIBI
//  2. Calls the test contract
//     a. call the wasm precompile which calls the wasm contract that stakes 5 NIBI
//
// INITIAL STATE:
// - Test contract funds: 10 NIBI
// CONTRACT CALL:
// - Sends 5 NIBI to the wasm contract
// - The wasm contract stakes 5 NIBI
// EXPECTED:
// - Test contract funds: 5 NIBI
// - Staked NIBI from wasm contract: 5 NIBI
// - Wasm contract: 0 NIBI
func (s *WasmSuite) TestWasmPrecompileDirtyStateAttack5() {
	deps := evmtest.NewTestDeps()

	wasmContracts := test.SetupWasmContracts(&deps, &s.Suite)
	wasmContract := wasmContracts[3] // staking.wasm

	s.T().Log("Deploy Test Contract")
	deployResp, err := evmtest.DeployContract(
		&deps,
		embeds.SmartContract_TestDirtyStateAttack5,
	)
	s.Require().NoError(err)
	testContractAddr := deployResp.ContractAddr

	s.Run("Mint 10 NIBI to test contract", func() {
		s.Require().NoError(testapp.FundAccount(
			deps.App.BankKeeper,
			deps.Ctx,
			eth.EthAddrToNibiruAddr(testContractAddr),
			sdk.NewCoins(sdk.NewCoin(evm.EVMBankDenom, sdk.NewInt(10e6))),
		))
	})

	validator := evmtest.NewEthPrivAcc()
	s.Run("create validator", func() {
		s.Require().NoError(testapp.FundAccount(
			deps.App.BankKeeper,
			deps.Ctx,
			validator.NibiruAddr,
			sdk.NewCoins(sdk.NewCoin(evm.EVMBankDenom, sdk.NewInt(10e6))),
		))

		createValMsg, err := stakingtypes.NewMsgCreateValidator(
			sdk.ValAddress(validator.NibiruAddr),
			validator.PrivKey.PubKey(),
			sdk.NewCoin(evm.EVMBankDenom, sdk.NewInt(10e6)),
			stakingtypes.NewDescription("validator0", "", "", "", ""),
			stakingtypes.NewCommissionRates(sdk.NewDec(1), sdk.NewDec(1), sdk.NewDec(1)),
			math.OneInt(),
		)
		s.Require().NoError(err)

		stakingMsgServer := stakingkeeper.NewMsgServerImpl(deps.App.StakingKeeper)
		resp, err := stakingMsgServer.CreateValidator(deps.Ctx, createValMsg)
		s.Require().NoError(err)
		s.Require().NotNil(resp)
	})

	s.Run("call test contract", func() {
		msgArgsBz := []byte(fmt.Sprintf(`{"run": {"validator": "%s"}}`, sdk.ValAddress(validator.NibiruAddr).String()))
		contractInput, err := embeds.SmartContract_TestDirtyStateAttack5.ABI.Pack(
			"attack",
			wasmContract.String(),
			msgArgsBz,
		)
		s.Require().NoError(err)

		evmObj, _ := deps.NewEVM()
		_, err = deps.EvmKeeper.CallContractWithInput(
			deps.Ctx,
			evmObj,
			deps.Sender.EthAddr,
			&testContractAddr,
			true,
			contractInput,
			evmtest.FunTokenGasLimitSendToEvm,
		)
		s.Require().NoError(err)

		testContractBalance := deps.App.BankKeeper.GetBalance(deps.Ctx, eth.EthAddrToNibiruAddr(testContractAddr), evm.EVMBankDenom)
		s.Require().Equal(testContractBalance.Amount.BigInt(), big.NewInt(5e6))

		wasmContractBalance := deps.App.BankKeeper.GetBalance(deps.Ctx, wasmContract, evm.EVMBankDenom)
		s.Require().Equal(wasmContractBalance.Amount.BigInt(), big.NewInt(0))

		delegations := deps.App.StakingKeeper.GetAllDelegatorDelegations(deps.Ctx, wasmContract)
		s.Require().Equal(len(delegations), 1)
		s.Require().Equal(sdk.ValAddress(validator.NibiruAddr).String(), delegations[0].ValidatorAddress)

		// after converting the wasm contract address to an eth address, check the balances
		wasmContractEthAddr := eth.NibiruAddrToEthAddr(wasmContract)
		balance := deps.App.BankKeeper.GetBalance(deps.Ctx, eth.EthAddrToNibiruAddr(wasmContractEthAddr), evm.EVMBankDenom)
		s.Require().Equal(big.NewInt(0), balance.Amount.BigInt())
	})
}
