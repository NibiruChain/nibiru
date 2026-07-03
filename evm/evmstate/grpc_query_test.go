package evmstate_test

import (
	"encoding/json"
	"fmt"
	"math/big"
	"strings"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	gethcore "github.com/ethereum/go-ethereum/core/types"
	gethparams "github.com/ethereum/go-ethereum/params"

	"github.com/NibiruChain/nibiru/v2/x/collections"

	"github.com/NibiruChain/nibiru/v2/app/appconst"
	"github.com/NibiruChain/nibiru/v2/eth"
	"github.com/NibiruChain/nibiru/v2/evm"
	"github.com/NibiruChain/nibiru/v2/evm/embeds"
	"github.com/NibiruChain/nibiru/v2/evm/evmtest"
	"github.com/NibiruChain/nibiru/v2/x/nutil/testutil/testapp"
	"github.com/NibiruChain/nibiru/v2/x/sudo"
)

type TestCase[In, Out any] struct {
	name string
	// setup: Optional setup function to create the scenario
	setup    func(deps *evmtest.TestDeps)
	scenario func(deps *evmtest.TestDeps) (
		req In,
		wantResp Out,
	)
	onTestEnd func(deps *evmtest.TestDeps, gotResp Out)
	wantErr   string
}

const zeroGasSentinelGasLimit uint64 = 780_749

type zeroGasSentinelFixture struct {
	to   gethcommon.Address
	data []byte
}

func setupZeroGasSentinelFixture(deps *evmtest.TestDeps) zeroGasSentinelFixture {
	deployResp, err := evmtest.DeployContract(deps, embeds.SmartContract_TestERC20)
	if err != nil {
		panic(err)
	}
	data, err := deployResp.ContractData.ABI.Pack(
		"transfer", gethcommon.HexToAddress("0x0000000000000000000000000000000000010f2C"), big.NewInt(0),
	)
	if err != nil {
		panic(err)
	}
	to := deployResp.ContractAddr
	deps.App.SudoKeeper.ZeroGasActors.Set(deps.Ctx(), sudo.ZeroGasActors{
		AlwaysZeroGasContracts: []string{to.Hex()},
	})
	return zeroGasSentinelFixture{to: to, data: data}
}

func setupZeroGasSentinelArgs(deps *evmtest.TestDeps) evm.JsonTxArgs {
	fixture := setupZeroGasSentinelFixture(deps)
	gas := hexutil.Uint64(zeroGasSentinelGasLimit)
	return evm.JsonTxArgs{
		From:                 &deps.Sender.EthAddr,
		To:                   &fixture.to,
		Value:                (*hexutil.Big)(big.NewInt(0)),
		Gas:                  &gas,
		GasPrice:             (*hexutil.Big)(big.NewInt(1_000_000_000_000)),
		MaxFeePerGas:         (*hexutil.Big)(big.NewInt(1_200_000_000_000)),
		MaxPriorityFeePerGas: (*hexutil.Big)(big.NewInt(0)),
		Data:                 (*hexutil.Bytes)(&fixture.data),
	}
}

func setupZeroGasSentinelTx(deps *evmtest.TestDeps) *evm.MsgEthereumTx {
	args := setupZeroGasSentinelArgs(deps)
	tx := evm.NewTx(&evm.EvmTxArgs{
		ChainID:   deps.EvmKeeper.EthChainID(deps.Ctx()),
		Nonce:     deps.EvmKeeper.GetAccNonce(deps.Ctx(), deps.Sender.EthAddr),
		GasLimit:  zeroGasSentinelGasLimit,
		Input:     args.GetData(),
		GasFeeCap: args.MaxFeePerGas.ToInt(),
		GasTipCap: args.MaxPriorityFeePerGas.ToInt(),
		Amount:    big.NewInt(0),
		To:        args.To,
	})
	tx.From = deps.Sender.EthAddr.Hex()
	if err := tx.Sign(deps.GethSigner(), deps.Sender.KeyringSigner); err != nil {
		panic(err)
	}
	return tx
}

func TraceResCallTracer_ZeroGasSentinel(
	fromAddr, toAddr gethcommon.Address,
) (traceResFields map[string]string) {
	f := make(map[string]string)
	gas := hexutil.Uint64(zeroGasSentinelGasLimit)
	f["from"] = fromAddr.Hex()
	f["gas"] = gas.String()
	f["to"] = toAddr.Hex()
	f["value"] = hexutil.Uint64(0).String()
	f["type"] = "CALL"
	return f
}

func InvalidEthAddr() string { return "0x0000" }

// TraceNibiTransfer returns a hardcoded JSON string representing the expected
// trace output of a successful "ether" (unibi) token transfer.
// Used to test the correctness of "TraceTx" and "TraceBlock".
//   - Note that the struct logs are empty. That is because a simple token
//     transfer does not involve contract operations.
func TraceNibiTransfer() string {
	return fmt.Sprintf(`{
	  "gas": %d,
	  "failed": false,
	  "returnValue": "",
	  "structLogs": []
	}`, gethparams.TxGas)
}

func TraceResCallTracer_ERC20Transfer(
	fromAddr, toAddr gethcommon.Address,
) (traceResFields map[string]string) {
	gas := hexutil.Uint64(51_046)     // "0xc766"
	gasUsed := hexutil.Uint64(33_946) // "0x849a"

	f := make(map[string]string)
	f["from"] = fromAddr.Hex()
	f["gas"] = gas.String()
	f["gasUsed"] = gasUsed.String()
	f["to"] = toAddr.Hex()
	f["value"] = hexutil.Uint64(0).String()
	f["type"] = "CALL"
	return f
}

func TraceResCallTracer_NibiTransfer(
	fromAddr, toAddr gethcommon.Address,
) (traceResFields map[string]string) {
	f := make(map[string]string)

	gas := hexutil.Uint64(gethparams.TxGas) // <- 21000 == 0x5208
	gasUsed := hexutil.Uint64(gethparams.TxGas)

	f["from"] = fromAddr.Hex()
	f["gas"] = gas.String()
	f["gasUsed"] = gasUsed.String()
	f["to"] = toAddr.Hex()
	f["input"] = "0x"
	f["value"] = hexutil.Uint64(0).String()
	f["type"] = "CALL"
	return f
}

// TraceOutputERC20Transfer returns a hardcoded JSON string representing the
// expected trace output of a successful ERC-20 token transfer (an EVM tx). Used
// to test the correctness of "TraceTx" and "TraceBlock".
func TraceOutputERC20Transfer() string {
	return `{
	   "gas": 35062,
	   "failed": false,
	   "returnValue": "0000000000000000000000000000000000000000000000000000000000000001",
	   "structLogs": [
		  {
			 "pc": 0,
			 "op": "PUSH1",
			 "gas": 30578,
			 "gasCost": 3,
			 "depth": 1,
			 "stack": []
		  }`
}

func (s *Suite) TestQueryEvmAccount() {
	type In = *evm.QueryEthAccountRequest
	type Out = *evm.QueryEthAccountResponse
	testCases := []TestCase[In, Out]{
		{
			name: "happy: fund account + query eth addr",
			setup: func(deps *evmtest.TestDeps) {
				// fund account with 420 tokens
				ethAddr := deps.Sender.EthAddr
				coins := sdk.Coins{sdk.NewInt64Coin(evm.EVMBankDenom, 420)}
				err := testapp.FundAccount(deps.App.BankKeeper, deps.Ctx(), ethAddr.Bytes(), coins)
				s.Require().NoError(err)
			},
			scenario: func(deps *evmtest.TestDeps) (req In, wantResp Out) {
				req = &evm.QueryEthAccountRequest{
					Address: deps.Sender.EthAddr.Hex(),
				}
				wantResp = &evm.QueryEthAccountResponse{
					BalanceWei:    "420" + strings.Repeat("0", 12),
					CodeHash:      evm.EmptyCodeHash.Hex(),
					Nonce:         0,
					EthAddress:    deps.Sender.EthAddr.String(),
					Bech32Address: deps.Sender.NibiruAddr.String(),
				}
				return req, wantResp
			},
			wantErr: "",
		},
		{
			name: "happy: fund account + query nibiru bech32 addr",
			setup: func(deps *evmtest.TestDeps) {
				// fund account with 420 tokens
				ethAddr := deps.Sender.EthAddr
				coins := sdk.Coins{sdk.NewInt64Coin(evm.EVMBankDenom, 420)}
				err := testapp.FundAccount(deps.App.BankKeeper, deps.Ctx(), ethAddr.Bytes(), coins)
				s.Require().NoError(err)
			},
			scenario: func(deps *evmtest.TestDeps) (req In, wantResp Out) {
				req = &evm.QueryEthAccountRequest{
					Address: deps.Sender.NibiruAddr.String(),
				}
				wantResp = &evm.QueryEthAccountResponse{
					BalanceWei:    "420" + strings.Repeat("0", 12),
					CodeHash:      evm.EmptyCodeHash.Hex(),
					Nonce:         0,
					EthAddress:    deps.Sender.EthAddr.String(),
					Bech32Address: deps.Sender.NibiruAddr.String(),
				}
				return req, wantResp
			},
			wantErr: "",
		},
		{
			name: "sad: msg validation",
			scenario: func(deps *evmtest.TestDeps) (req In, wantResp Out) {
				req = &evm.QueryEthAccountRequest{
					Address: InvalidEthAddr(),
				}
				wantResp = &evm.QueryEthAccountResponse{
					BalanceWei: "0",
					CodeHash:   evm.EmptyCodeHash.Hex(),
					Nonce:      0,
				}
				return req, wantResp
			},
			wantErr: "not a valid ethereum hex addr",
		},
		{
			name: "happy: nonexistent account (hex addr input)",
			scenario: func(deps *evmtest.TestDeps) (req In, wantResp Out) {
				ethAcc := evmtest.NewEthPrivAcc()
				req = &evm.QueryEthAccountRequest{
					Address: ethAcc.EthAddr.String(),
				}
				wantResp = &evm.QueryEthAccountResponse{
					BalanceWei:    "0",
					CodeHash:      evm.EmptyCodeHash.Hex(),
					EthAddress:    ethAcc.EthAddr.Hex(),
					Bech32Address: ethAcc.NibiruAddr.String(),
					Nonce:         0,
				}
				return req, wantResp
			},
			wantErr: "",
		},
		{
			name: "happy: nonexistent account (bech32 input)",
			scenario: func(deps *evmtest.TestDeps) (req In, wantResp Out) {
				ethAcc := evmtest.NewEthPrivAcc()
				req = &evm.QueryEthAccountRequest{
					Address: ethAcc.NibiruAddr.String(),
				}
				wantResp = &evm.QueryEthAccountResponse{
					BalanceWei:    "0",
					CodeHash:      evm.EmptyCodeHash.Hex(),
					EthAddress:    ethAcc.EthAddr.Hex(),
					Bech32Address: ethAcc.NibiruAddr.String(),
					Nonce:         0,
				}
				return req, wantResp
			},
			wantErr: "",
		},
		{
			name: "happy: wasm contract has blank Eth address",
			scenario: func(deps *evmtest.TestDeps) (req In, wantResp Out) {
				wasmContractAddr, err := sdk.AccAddressFromBech32(
					// Address for the Eris deplyoment on mainnet
					"nibi1udqqx30cw8nwjxtl4l28ym9hhrp933zlq8dqxfjzcdhvl8y24zcqpzmh8m",
				)
				s.Require().NoError(err)
				s.Equal(appconst.ADDR_LEN_WASM_CONTRACT, len(wasmContractAddr))
				s.NotEqual(appconst.ADDR_LEN_EOA, len(wasmContractAddr))
				req = &evm.QueryEthAccountRequest{
					Address: wasmContractAddr.String(),
				}
				wantResp = &evm.QueryEthAccountResponse{
					BalanceWei:    "0",
					CodeHash:      evm.EmptyCodeHash.Hex(),
					EthAddress:    "", // <- MUST be blank
					Bech32Address: wasmContractAddr.String(),
					Nonce:         0,
				}
				return req, wantResp
			},
			wantErr: "",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			deps := evmtest.NewTestDeps()
			if tc.setup != nil {
				tc.setup(&deps)
			}
			req, wantResp := tc.scenario(&deps)
			goCtx := sdk.WrapSDKContext(deps.Ctx())
			gotResp, err := deps.EvmKeeper.EthAccount(goCtx, req)
			if tc.wantErr != "" {
				s.Require().ErrorContains(err, tc.wantErr)
				return
			}
			s.Assert().NoError(err)
			s.EqualValues(wantResp, gotResp)
		})
	}
}

func (s *Suite) TestQueryValidatorAccount() {
	type In = *evm.QueryValidatorAccountRequest
	type Out = *evm.QueryValidatorAccountResponse
	testCases := []TestCase[In, Out]{
		{
			name: "sad: msg validation",
			scenario: func(deps *evmtest.TestDeps) (req In, wantResp Out) {
				req = &evm.QueryValidatorAccountRequest{
					ConsAddress: "nibi1invalidaddr",
				}
				wantResp = &evm.QueryValidatorAccountResponse{
					AccountAddress: sdk.AccAddress(gethcommon.Address{}.Bytes()).String(),
				}
				return req, wantResp
			},
			wantErr: "decoding bech32 failed",
		},
		{
			name: "sad: validator account not found",
			scenario: func(deps *evmtest.TestDeps) (req In, wantResp Out) {
				req = &evm.QueryValidatorAccountRequest{
					ConsAddress: "nibivalcons1ea4ef7wsatlnaj9ry3zylymxv53f9ntrjecc40",
				}
				wantResp = &evm.QueryValidatorAccountResponse{
					AccountAddress: sdk.AccAddress(gethcommon.Address{}.Bytes()).String(),
				}
				return req, wantResp
			},
			wantErr: "validator not found",
		},
		{
			name:  "happy: default values",
			setup: func(deps *evmtest.TestDeps) {},
			scenario: func(deps *evmtest.TestDeps) (req In, wantResp Out) {
				valopers := deps.App.StakingKeeper.GetValidators(deps.Ctx(), 1)
				valAddrBz := valopers[0].GetOperator().Bytes()
				_, err := sdk.ConsAddressFromBech32(valopers[0].OperatorAddress)
				s.ErrorContains(err, "expected nibivalcons, got nibivaloper")
				consAddr := sdk.ConsAddress(valAddrBz)

				req = &evm.QueryValidatorAccountRequest{
					ConsAddress: consAddr.String(),
				}
				wantResp = &evm.QueryValidatorAccountResponse{
					AccountAddress: sdk.AccAddress(valAddrBz).String(),
					Sequence:       0,
					AccountNumber:  0,
				}
				return req, wantResp
			},
			wantErr: "",
		},
		{
			name:  "happy: with nonce",
			setup: func(deps *evmtest.TestDeps) {},
			scenario: func(deps *evmtest.TestDeps) (req In, wantResp Out) {
				valopers := deps.App.StakingKeeper.GetValidators(deps.Ctx(), 1)
				valAddrBz := valopers[0].GetOperator().Bytes()
				consAddr := sdk.ConsAddress(valAddrBz)

				s.T().Log(
					"Send coins to validator to register in the account keeper.")
				coinsToSend := sdk.NewCoins(sdk.NewCoin(eth.EthBaseDenom, sdkmath.NewInt(69420)))
				valAddr := sdk.AccAddress(valAddrBz)
				s.NoError(testapp.FundAccount(
					deps.App.BankKeeper,
					deps.Ctx(), valAddr,
					coinsToSend,
				))

				req = &evm.QueryValidatorAccountRequest{
					ConsAddress: consAddr.String(),
				}

				ak := deps.App.AccountKeeper
				acc := ak.GetAccount(deps.Ctx(), valAddr)
				s.NoError(acc.SetAccountNumber(420), "acc: ", acc.String())
				s.NoError(acc.SetSequence(69), "acc: ", acc.String())
				ak.SetAccount(deps.Ctx(), acc)

				wantResp = &evm.QueryValidatorAccountResponse{
					AccountAddress: sdk.AccAddress(valAddrBz).String(),
					Sequence:       69,
					AccountNumber:  420,
				}
				return req, wantResp
			},
			wantErr: "",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			deps := evmtest.NewTestDeps()
			if tc.setup != nil {
				tc.setup(&deps)
			}
			req, wantResp := tc.scenario(&deps)
			goCtx := sdk.WrapSDKContext(deps.Ctx())
			gotResp, err := deps.EvmKeeper.ValidatorAccount(goCtx, req)
			if tc.wantErr != "" {
				s.Require().ErrorContains(err, tc.wantErr)
				return
			}
			s.Assert().NoError(err)
			s.EqualValues(wantResp, gotResp)
		})
	}
}

func (s *Suite) TestQueryStorage() {
	type In = *evm.QueryStorageRequest
	type Out = *evm.QueryStorageResponse
	testCases := []TestCase[In, Out]{
		{
			name: "sad: msg validation",
			scenario: func(deps *evmtest.TestDeps) (req In, wantResp Out) {
				req = &evm.QueryStorageRequest{
					Address: InvalidEthAddr(),
				}
				return req, wantResp
			},
			wantErr: "InvalidArgument",
		},
		{
			name: "happy",
			scenario: func(deps *evmtest.TestDeps) (req In, wantResp Out) {
				addr := evmtest.NewEthPrivAcc().EthAddr
				storageKey := gethcommon.BytesToHash([]byte("storagekey"))
				req = &evm.QueryStorageRequest{
					Address: addr.Hex(),
					Key:     storageKey.String(),
				}

				sdb := deps.NewStateDB()
				storageValue := gethcommon.BytesToHash([]byte("value"))

				sdb.SetState(addr, storageKey, storageValue)
				sdb.Commit()

				wantResp = &evm.QueryStorageResponse{
					Value: storageValue.String(),
				}
				return req, wantResp
			},
			wantErr: "",
		},
		{
			name: "happy: no committed state",
			scenario: func(deps *evmtest.TestDeps) (req In, wantResp Out) {
				addr := evmtest.NewEthPrivAcc().EthAddr
				storageKey := gethcommon.BytesToHash([]byte("storagekey"))
				req = &evm.QueryStorageRequest{
					Address: addr.Hex(),
					Key:     storageKey.String(),
				}

				wantResp = &evm.QueryStorageResponse{
					Value: gethcommon.BytesToHash([]byte{}).String(),
				}
				return req, wantResp
			},
			wantErr: "",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			deps := evmtest.NewTestDeps()
			if tc.setup != nil {
				tc.setup(&deps)
			}
			req, wantResp := tc.scenario(&deps)
			goCtx := sdk.WrapSDKContext(deps.Ctx())

			gotResp, err := deps.EvmKeeper.Storage(goCtx, req)
			if tc.wantErr != "" {
				s.Require().ErrorContains(err, tc.wantErr)
				return
			}
			s.Assert().NoError(err)
			s.EqualValues(wantResp, gotResp)
		})
	}
}

func (s *Suite) TestQueryCode() {
	type In = *evm.QueryCodeRequest
	type Out = *evm.QueryCodeResponse
	testCases := []TestCase[In, Out]{
		{
			name: "sad: msg validation",
			scenario: func(deps *evmtest.TestDeps) (req In, wantResp Out) {
				req = &evm.QueryCodeRequest{
					Address: InvalidEthAddr(),
				}
				return req, wantResp
			},
			wantErr: "InvalidArgument",
		},
		{
			name: "happy",
			scenario: func(deps *evmtest.TestDeps) (req In, wantResp Out) {
				addr := evmtest.NewEthPrivAcc().EthAddr
				req = &evm.QueryCodeRequest{
					Address: addr.Hex(),
				}

				sdb := deps.NewStateDB()
				contractBytecode := []byte("bytecode")
				sdb.SetCode(addr, contractBytecode)
				sdb.Commit()

				s.NotNil(sdb.Keeper().GetAccount(deps.Ctx(), addr))
				s.NotNil(sdb.GetCode(addr))

				wantResp = &evm.QueryCodeResponse{
					Code: contractBytecode,
				}
				return req, wantResp
			},
			wantErr: "",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			deps := evmtest.NewTestDeps()
			if tc.setup != nil {
				tc.setup(&deps)
			}
			req, wantResp := tc.scenario(&deps)
			goCtx := sdk.WrapSDKContext(deps.Ctx())

			gotResp, err := deps.EvmKeeper.Code(goCtx, req)
			if tc.wantErr != "" {
				s.Require().ErrorContains(err, tc.wantErr)
				return
			}
			s.Assert().NoError(err)
			s.EqualValues(wantResp, gotResp, "want hex (%s), got hex (%s)",
				collections.HumanizeBytes(wantResp.Code),
				collections.HumanizeBytes(gotResp.Code),
			)
		})
	}
}

func (s *Suite) TestQueryParams() {
	deps := evmtest.NewTestDeps()
	want := evm.DefaultParams()
	want.WasmPlugins = []evm.WasmPlugin{
		{
			Name: evm.WasmPluginNameXOracle,
			Addr: deps.Sender.NibiruAddr.String(),
		},
	}
	err := deps.EvmKeeper.SetParams(deps.Ctx(), want)
	s.NoError(err)

	gotResp, err := deps.EvmKeeper.Params(sdk.WrapSDKContext(deps.Ctx()), nil)
	s.NoError(err)
	got := gotResp.Params
	s.Require().NoError(err)

	// Note that protobuf equals is more reliable than `s.Equal`
	s.Require().True(want.Equal(got), "want %s, got %s", want, got)

	// Empty params to test the setter
	want.EVMChannels = []string{"channel-420"}
	err = deps.EvmKeeper.SetParams(deps.Ctx(), want)
	s.NoError(err)
	gotResp, err = deps.EvmKeeper.Params(sdk.WrapSDKContext(deps.Ctx()), nil)
	s.Require().NoError(err)
	got = gotResp.Params

	// Note that protobuf equals is more reliable than `s.Equal`
	s.Require().True(want.Equal(got), "want %s, got %s", want, got)
}

func (s *Suite) TestQueryEthCall() {
	type In = *evm.EthCallRequest
	type Out = *evm.MsgEthereumTxResponse
	testCases := []TestCase[In, Out]{
		{
			name: "sad: msg invalid msg",
			scenario: func(deps *evmtest.TestDeps) (req In, wantResp Out) {
				return nil, nil
			},
			wantErr: "InvalidArgument",
		},
		{
			name: "sad: invalid args",
			scenario: func(deps *evmtest.TestDeps) (req In, wantResp Out) {
				return &evm.EthCallRequest{Args: []byte("invalid")}, nil
			},
			wantErr: "InvalidArgument",
		},
		{
			name: "happy: eth call to deploy erc20 contract",
			scenario: func(deps *evmtest.TestDeps) (req In, wantResp Out) {
				fungibleTokenContract := embeds.SmartContract_ERC20MinterWithMetadataUpdates
				jsonTxArgs, err := json.Marshal(&evm.JsonTxArgs{
					From: &deps.Sender.EthAddr,
					Data: (*hexutil.Bytes)(&fungibleTokenContract.Bytecode),
				})
				s.Require().NoError(err)
				return &evm.EthCallRequest{Args: jsonTxArgs}, &evm.MsgEthereumTxResponse{
					Hash: "0x0000000000000000000000000000000000000000000000000000000000000000",
				}
			},
			wantErr: "",
		},
		{
			name: "happy: zero-gas call with wallet-managed fee fields",
			scenario: func(deps *evmtest.TestDeps) (req In, wantResp Out) {
				jsonTxArgs, err := json.Marshal(setupZeroGasSentinelArgs(deps))
				s.Require().NoError(err)
				return &evm.EthCallRequest{Args: jsonTxArgs}, &evm.MsgEthereumTxResponse{
					Hash: "0x0000000000000000000000000000000000000000000000000000000000000000",
				}
			},
			wantErr: "",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			deps := evmtest.NewTestDeps()
			if tc.setup != nil {
				tc.setup(&deps)
			}
			req, wantResp := tc.scenario(&deps)
			gotResp, err := deps.App.EvmKeeper.EthCall(sdk.WrapSDKContext(deps.Ctx()), req)
			if tc.wantErr != "" {
				s.Require().ErrorContains(err, tc.wantErr)
				return
			}
			s.Assert().NoError(err)
			s.Assert().Empty(wantResp.VmError)
			s.Assert().Equal(wantResp.Hash, gotResp.Hash)
		})
	}
}

func (s *Suite) TestQueryBalance() {
	type In = *evm.QueryBalanceRequest
	type Out = *evm.QueryBalanceResponse
	setBankMetadata := func(deps *evmtest.TestDeps, denom string, decimals uint32, symbol string) {
		deps.App.BankKeeper.SetDenomMetaData(deps.Ctx(), banktypes.Metadata{
			DenomUnits: []*banktypes.DenomUnit{
				{Denom: denom, Exponent: 0},
				{Denom: symbol, Exponent: decimals},
			},
			Base:    denom,
			Display: symbol,
			Name:    symbol,
			Symbol:  symbol,
		})
	}
	fundBank := func(deps *evmtest.TestDeps, addr sdk.AccAddress, denom string, amount int64) {
		s.Require().NoError(testapp.FundAccount(
			deps.App.BankKeeper,
			deps.Ctx(),
			addr,
			sdk.NewCoins(sdk.NewInt64Coin(denom, amount)),
		))
	}

	testCases := []TestCase[In, Out]{
		{
			name: "sad: msg validation",
			scenario: func(deps *evmtest.TestDeps) (req In, wantResp Out) {
				req = &evm.QueryBalanceRequest{
					Address: InvalidEthAddr(),
				}
				wantResp = &evm.QueryBalanceResponse{
					BalanceWei: "0",
				}
				return req, wantResp
			},
			wantErr: "not a valid ethereum hex address",
		},
		{
			name: "happy: zero balance",
			scenario: func(deps *evmtest.TestDeps) (req In, wantResp Out) {
				req = &evm.QueryBalanceRequest{
					Address: evmtest.NewEthPrivAcc().EthAddr.String(),
				}
				wantResp = &evm.QueryBalanceResponse{
					BalanceWei: "0",
				}
				return req, wantResp
			},
			wantErr: "",
		},
		{
			name: "happy: non zero balance",
			setup: func(deps *evmtest.TestDeps) {
				chain := deps.App
				ethAddr := deps.Sender.EthAddr

				// fund account with 420 tokens
				coins := sdk.Coins{sdk.NewInt64Coin(evm.EVMBankDenom, 420)}
				err := chain.BankKeeper.MintCoins(deps.Ctx(), evm.ModuleName, coins)
				s.NoError(err)
				err = chain.BankKeeper.SendCoinsFromModuleToAccount(
					deps.Ctx(), evm.ModuleName, ethAddr.Bytes(), coins)
				s.Require().NoError(err)
			},
			scenario: func(deps *evmtest.TestDeps) (req In, wantResp Out) {
				req = &evm.QueryBalanceRequest{
					Address: deps.Sender.EthAddr.Hex(),
				}
				wantResp = &evm.QueryBalanceResponse{
					BalanceWei: "420" + strings.Repeat("0", 12),
				}
				return req, wantResp
			},
			wantErr: "",
		},
		{
			name: "happy: native NIBI token by denom",
			setup: func(deps *evmtest.TestDeps) {
				setBankMetadata(deps, appconst.DENOM_UNIBI, 6, "NIBI")
				fundBank(deps, deps.Sender.NibiruAddr, appconst.DENOM_UNIBI, 420)
				deps.DeployWNIBI(&s.Suite)
			},
			scenario: func(deps *evmtest.TestDeps) (req In, wantResp Out) {
				req = &evm.QueryBalanceRequest{
					Address: deps.Sender.EthAddr.Hex(),
					Token:   appconst.DENOM_UNIBI,
				}
				wantResp = &evm.QueryBalanceResponse{
					BalanceWei: "420" + strings.Repeat("0", 12),
					Bank: &evm.BalanceBank{
						Symbol:       "NIBI",
						BalanceHuman: "0.00042",
						Decimals:     6,
						CoinDenom:    appconst.DENOM_UNIBI,
						BalanceBase:  "420",
					},
					Erc20: &evm.BalanceERC20{
						Address:      deps.EvmKeeper.GetParams(deps.Ctx()).CanonicalWnibi.Hex(),
						Symbol:       "WNIBI",
						BalanceHuman: "0",
						Decimals:     18,
						Name:         "Wrapped Nibiru",
						BalanceBase:  "0",
					},
				}
				return req, wantResp
			},
			wantErr: "",
		},
		{
			name: "happy: native NIBI token by WNIBI alias",
			setup: func(deps *evmtest.TestDeps) {
				setBankMetadata(deps, appconst.DENOM_UNIBI, 6, "NIBI")
				deps.DeployWNIBI(&s.Suite)
			},
			scenario: func(deps *evmtest.TestDeps) (req In, wantResp Out) {
				req = &evm.QueryBalanceRequest{
					Address: deps.Sender.EthAddr.Hex(),
					Token:   "WNIBI",
				}
				wantResp = &evm.QueryBalanceResponse{
					BalanceWei: "0",
					Bank: &evm.BalanceBank{
						Symbol:       "NIBI",
						BalanceHuman: "0",
						Decimals:     6,
						CoinDenom:    appconst.DENOM_UNIBI,
						BalanceBase:  "0",
					},
					Erc20: &evm.BalanceERC20{
						Address:      deps.EvmKeeper.GetParams(deps.Ctx()).CanonicalWnibi.Hex(),
						Symbol:       "WNIBI",
						BalanceHuman: "0",
						Decimals:     18,
						Name:         "Wrapped Nibiru",
						BalanceBase:  "0",
					},
				}
				return req, wantResp
			},
			wantErr: "",
		},
		{
			name: "happy: FunToken mapping populates bank and erc20",
			setup: func(deps *evmtest.TestDeps) {
				bankDenom := "ufuntoken"
				funtoken := evmtest.CreateFunTokenForBankCoin(*deps, bankDenom, &s.Suite)
				fundBank(deps, deps.Sender.NibiruAddr, bankDenom, 123)
				deps.GenState.FuntokenMappings = []evm.FunToken{funtoken}
			},
			scenario: func(deps *evmtest.TestDeps) (req In, wantResp Out) {
				funtoken := deps.GenState.FuntokenMappings[0]
				req = &evm.QueryBalanceRequest{
					Address: deps.Sender.EthAddr.Hex(),
					Token:   funtoken.BankDenom,
				}
				wantResp = &evm.QueryBalanceResponse{
					BalanceWei: "0",
					Bank: &evm.BalanceBank{
						Symbol:       "TEST",
						BalanceHuman: "0.000000000000000123",
						Decimals:     18,
						CoinDenom:    funtoken.BankDenom,
						BalanceBase:  "123",
					},
					Erc20: &evm.BalanceERC20{
						Address:      funtoken.Erc20Addr.Hex(),
						Symbol:       "TEST",
						BalanceHuman: "0",
						Decimals:     18,
						Name:         "Name for ufuntoken",
						BalanceBase:  "0",
					},
				}
				return req, wantResp
			},
			wantErr: "",
		},
		{
			name: "happy: ERC20 address without FunToken populates erc20 only",
			setup: func(deps *evmtest.TestDeps) {
				deployResp, err := evmtest.DeployContract(
					deps,
					embeds.SmartContract_ERC20MinterWithMetadataUpdates,
					"Lonely Token", "LONE", uint8(6),
				)
				s.Require().NoError(err)
				deps.GenState.FuntokenMappings = []evm.FunToken{
					evm.NewFunToken(deployResp.ContractAddr, "unused", false),
				}
			},
			scenario: func(deps *evmtest.TestDeps) (req In, wantResp Out) {
				erc20 := deps.GenState.FuntokenMappings[0].Erc20Addr
				req = &evm.QueryBalanceRequest{
					Address: deps.Sender.EthAddr.Hex(),
					Token:   erc20.Hex(),
				}
				wantResp = &evm.QueryBalanceResponse{
					BalanceWei: "0",
					Erc20: &evm.BalanceERC20{
						Address:      erc20.Hex(),
						Symbol:       "LONE",
						BalanceHuman: "0",
						Decimals:     6,
						Name:         "Lonely Token",
						BalanceBase:  "0",
					},
				}
				return req, wantResp
			},
			wantErr: "",
		},
		{
			name: "happy: bank denom without FunToken populates bank only",
			setup: func(deps *evmtest.TestDeps) {
				setBankMetadata(deps, "ubankonly", 6, "BANK")
				fundBank(deps, deps.Sender.NibiruAddr, "ubankonly", 1_234_567)
			},
			scenario: func(deps *evmtest.TestDeps) (req In, wantResp Out) {
				req = &evm.QueryBalanceRequest{
					Address: deps.Sender.EthAddr.Hex(),
					Token:   "ubankonly",
				}
				wantResp = &evm.QueryBalanceResponse{
					BalanceWei: "0",
					Bank: &evm.BalanceBank{
						Symbol:       "BANK",
						BalanceHuman: "1.234567",
						Decimals:     6,
						CoinDenom:    "ubankonly",
						BalanceBase:  "1234567",
					},
				}
				return req, wantResp
			},
			wantErr: "",
		},
		{
			name: "happy: bank denom that looks like short hex avoids ERC20 index collision",
			setup: func(deps *evmtest.TestDeps) {
				setBankMetadata(deps, "deadbeef", 6, "BANK")
				setBankMetadata(deps, "ucollision", 6, "COLLIDE")
				fundBank(deps, deps.Sender.NibiruAddr, "deadbeef", 1_234_567)
				s.Require().NoError(deps.EvmKeeper.FunTokens.SafeInsert(
					deps.Ctx(),
					gethcommon.HexToAddress("deadbeef"),
					"ucollision",
					false,
				))
			},
			scenario: func(deps *evmtest.TestDeps) (req In, wantResp Out) {
				req = &evm.QueryBalanceRequest{
					Address: deps.Sender.EthAddr.Hex(),
					Token:   "deadbeef",
				}
				wantResp = &evm.QueryBalanceResponse{
					BalanceWei: "0",
					Bank: &evm.BalanceBank{
						Symbol:       "BANK",
						BalanceHuman: "1.234567",
						Decimals:     6,
						CoinDenom:    "deadbeef",
						BalanceBase:  "1234567",
					},
				}
				return req, wantResp
			},
			wantErr: "",
		},
		{
			name: "happy: USDC alias resolves to known mainnet ERC20 before mapping lookup",
			setup: func(deps *evmtest.TestDeps) {
				setBankMetadata(deps, "uusdc", 6, "USDC")
				fundBank(deps, deps.Sender.NibiruAddr, "uusdc", 7_000_000)
				s.Require().NoError(deps.EvmKeeper.FunTokens.SafeInsert(
					deps.Ctx(),
					gethcommon.HexToAddress("0x0829F361A05D993d5CEb035cA6DF3446b060970b"),
					"uusdc",
					false,
				))
			},
			scenario: func(deps *evmtest.TestDeps) (req In, wantResp Out) {
				req = &evm.QueryBalanceRequest{
					Address: deps.Sender.EthAddr.Hex(),
					Token:   "USDC",
				}
				wantResp = &evm.QueryBalanceResponse{
					BalanceWei: "0",
					Bank: &evm.BalanceBank{
						Symbol:       "USDC",
						BalanceHuman: "7",
						Decimals:     6,
						CoinDenom:    "uusdc",
						BalanceBase:  "7000000",
					},
				}
				return req, wantResp
			},
			wantErr: "",
		},
		{
			name: "happy: wasm address can query bank side without erc20 side",
			setup: func(deps *evmtest.TestDeps) {
				setBankMetadata(deps, "uwasmcoin", 6, "WASM")
				wasmContractAddr, err := sdk.AccAddressFromBech32(
					"nibi1udqqx30cw8nwjxtl4l28ym9hhrp933zlq8dqxfjzcdhvl8y24zcqpzmh8m",
				)
				s.Require().NoError(err)
				fundBank(deps, wasmContractAddr, "uwasmcoin", 2_500_000)
			},
			scenario: func(deps *evmtest.TestDeps) (req In, wantResp Out) {
				wasmContractAddr, err := sdk.AccAddressFromBech32(
					"nibi1udqqx30cw8nwjxtl4l28ym9hhrp933zlq8dqxfjzcdhvl8y24zcqpzmh8m",
				)
				s.Require().NoError(err)
				s.Equal(appconst.ADDR_LEN_WASM_CONTRACT, len(wasmContractAddr))
				req = &evm.QueryBalanceRequest{
					Address: wasmContractAddr.String(),
					Token:   "uwasmcoin",
				}
				wantResp = &evm.QueryBalanceResponse{
					BalanceWei: "0",
					Bank: &evm.BalanceBank{
						Symbol:       "WASM",
						BalanceHuman: "2.5",
						Decimals:     6,
						CoinDenom:    "uwasmcoin",
						BalanceBase:  "2500000",
					},
				}
				return req, wantResp
			},
			wantErr: "",
		},
		{
			name: "happy: unrecognized token preserves legacy response",
			scenario: func(deps *evmtest.TestDeps) (req In, wantResp Out) {
				req = &evm.QueryBalanceRequest{
					Address: deps.Sender.EthAddr.Hex(),
					Token:   "not a denom",
				}
				wantResp = &evm.QueryBalanceResponse{
					BalanceWei: "0",
				}
				return req, wantResp
			},
			wantErr: "",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			deps := evmtest.NewTestDeps()
			if tc.setup != nil {
				tc.setup(&deps)
			}
			req, wantResp := tc.scenario(&deps)
			goCtx := sdk.WrapSDKContext(deps.Ctx())
			gotResp, err := deps.EvmKeeper.Balance(goCtx, req)
			if tc.wantErr != "" {
				s.Require().ErrorContains(err, tc.wantErr)
				return
			}
			s.Assert().NoError(err)
			s.EqualValues(wantResp, gotResp)
		})
	}
}

func (s *Suite) TestQueryBaseFee() {
	type In = *evm.QueryBaseFeeRequest
	type Out = *evm.QueryBaseFeeResponse
	testCases := []TestCase[In, Out]{
		{
			name: "happy: base fee value",
			scenario: func(deps *evmtest.TestDeps) (req In, wantResp Out) {
				req = &evm.QueryBaseFeeRequest{}
				defaultFeeWei := sdkmath.NewIntFromBigInt(evm.BASE_FEE_WEI)
				defaultFeeUnibi := sdkmath.NewIntFromBigInt(evm.BASE_FEE_MICRONIBI)
				wantResp = &evm.QueryBaseFeeResponse{
					BaseFee:      &defaultFeeWei,
					BaseFeeUnibi: &defaultFeeUnibi,
				}
				return req, wantResp
			},
			wantErr: "",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			deps := evmtest.NewTestDeps()
			if tc.setup != nil {
				tc.setup(&deps)
			}
			req, wantResp := tc.scenario(&deps)
			goCtx := sdk.WrapSDKContext(deps.Ctx())
			gotResp, err := deps.EvmKeeper.BaseFee(goCtx, req)
			if tc.wantErr != "" {
				s.Require().ErrorContains(err, tc.wantErr)
				return
			}
			s.Assert().NoError(err)
			s.EqualValues(wantResp, gotResp)
		})
	}
}

func (s *Suite) TestEstimateGasForEvmCallType() {
	type In = *evm.EthCallRequest
	type Out = *evm.EstimateGasResponse
	testCases := []TestCase[In, Out]{
		{
			name: "sad: nil query",
			scenario: func(deps *evmtest.TestDeps) (req In, wantResp Out) {
				req = nil
				wantResp = nil
				return req, wantResp
			},
			wantErr: "InvalidArgument",
		},
		{
			name: "sad: insufficient gas cap",
			scenario: func(deps *evmtest.TestDeps) (req In, wantResp Out) {
				req = &evm.EthCallRequest{
					GasCap: gethparams.TxGas - 1,
				}
				return req, nil
			},
			wantErr: "InvalidArgument",
		},
		{
			name: "sad: invalid args",
			scenario: func(deps *evmtest.TestDeps) (req In, wantResp Out) {
				req = &evm.EthCallRequest{
					Args:   []byte{0, 0, 0},
					GasCap: gethparams.TxGas,
				}
				return req, nil
			},
			wantErr: "InvalidArgument",
		},
		{
			name: "happy: estimate gas for transfer",
			scenario: func(deps *evmtest.TestDeps) (req In, wantResp Out) {
				// fund the account
				chain := deps.App
				ethAddr := deps.Sender.EthAddr
				coins := sdk.Coins{sdk.NewInt64Coin(evm.EVMBankDenom, 1000)}
				err := chain.BankKeeper.MintCoins(deps.Ctx(), evm.ModuleName, coins)
				s.NoError(err)
				err = chain.BankKeeper.SendCoinsFromModuleToAccount(
					deps.Ctx(), evm.ModuleName, ethAddr.Bytes(), coins)
				s.Require().NoError(err)

				// assert balance of 1000 * 10^12 wei
				resp, _ := deps.App.EvmKeeper.Balance(sdk.WrapSDKContext(deps.Ctx()), &evm.QueryBalanceRequest{
					Address: deps.Sender.EthAddr.Hex(),
				})
				wantBal := "1000" + strings.Repeat("0", 12)
				s.Require().Equal(wantBal, resp.BalanceWei)

				// Send Eth call to transfer from the account - 5 * 10^12 wei
				recipient := evmtest.NewEthPrivAcc().EthAddr
				amountToSend := hexutil.Big(*evm.NativeToWei(big.NewInt(5)))
				gasLimitArg := hexutil.Uint64(100000)

				jsonTxArgs, err := json.Marshal(&evm.JsonTxArgs{
					From:  &deps.Sender.EthAddr,
					To:    &recipient,
					Value: &amountToSend,
					Gas:   &gasLimitArg,
				})
				s.Require().NoError(err)
				req = &evm.EthCallRequest{
					Args:   jsonTxArgs,
					GasCap: gethparams.TxGas,
				}
				wantResp = &evm.EstimateGasResponse{
					Gas: gethparams.TxGas,
				}
				return req, wantResp
			},
			wantErr: "",
		},
		{
			name: "happy: zero-gas estimate with wallet-managed fee fields",
			scenario: func(deps *evmtest.TestDeps) (req In, wantResp Out) {
				jsonTxArgs, err := json.Marshal(setupZeroGasSentinelArgs(deps))
				s.Require().NoError(err)
				req = &evm.EthCallRequest{
					Args:   jsonTxArgs,
					GasCap: zeroGasSentinelGasLimit,
				}
				return req, nil
			},
			onTestEnd: func(deps *evmtest.TestDeps, gotResp Out) {
				s.Require().NotNil(gotResp)
				s.Require().Greater(gotResp.Gas, uint64(gethparams.TxGas))
				s.Require().LessOrEqual(gotResp.Gas, zeroGasSentinelGasLimit)
			},
			wantErr: "",
		},
		{
			name: "sad: insufficient balance for transfer",
			scenario: func(deps *evmtest.TestDeps) (req In, wantResp Out) {
				recipient := evmtest.NewEthPrivAcc().EthAddr
				amountToSend := hexutil.Big(*evm.NativeToWei(big.NewInt(10)))

				jsonTxArgs, err := json.Marshal(&evm.JsonTxArgs{
					From:  &deps.Sender.EthAddr,
					To:    &recipient,
					Value: &amountToSend,
				})
				s.Require().NoError(err)
				req = &evm.EthCallRequest{
					Args:   jsonTxArgs,
					GasCap: gethparams.TxGas,
				}
				wantResp = nil
				return req, wantResp
			},
			wantErr: "insufficient balance for transfer",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			deps := evmtest.NewTestDeps()
			if tc.setup != nil {
				tc.setup(&deps)
			}
			req, wantResp := tc.scenario(&deps)
			goCtx := sdk.WrapSDKContext(deps.Ctx())
			gotResp, err := deps.EvmKeeper.EstimateGas(goCtx, req)
			if tc.wantErr != "" {
				s.Require().ErrorContains(err, tc.wantErr)
				return
			}
			s.Assert().NoError(err)
			if tc.onTestEnd != nil {
				tc.onTestEnd(&deps, gotResp)
			} else {
				s.EqualValues(wantResp, gotResp)
			}
		})
	}
}

func (s *Suite) TestTraceTx() {
	type In = *evm.QueryTraceTxRequest
	type Out = map[string]string

	testCases := []TestCase[In, Out]{
		{
			name: "sad: nil query",
			scenario: func(deps *evmtest.TestDeps) (req In, wantResp Out) {
				return nil, nil
			},
			wantErr: "InvalidArgument",
		},
		{
			name: "happy: simple nibi transfer tx",
			scenario: func(deps *evmtest.TestDeps) (req In, wantResp Out) {
				txMsg, _ := evmtest.ExecuteNibiTransfer(deps, s.T())
				req = &evm.QueryTraceTxRequest{
					Msg: txMsg,
				}
				toAddr := txMsg.AsTransaction().To()
				s.Require().NotNil(toAddr)
				wantResp = TraceResCallTracer_NibiTransfer(
					deps.Sender.EthAddr, // fromAddr
					*toAddr,             // toAddr
				)
				return req, wantResp
			},
			wantErr: "",
		},
		{
			name: "happy: trace erc-20 transfer tx",
			scenario: func(deps *evmtest.TestDeps) (req In, wantResp Out) {
				txMsg, predecessors, erc20Addr := evmtest.DeployAndExecuteERC20Transfer(deps, s.T())
				req = &evm.QueryTraceTxRequest{
					Msg:          txMsg,
					Predecessors: predecessors,
				}
				wantResp = TraceResCallTracer_ERC20Transfer(
					deps.Sender.EthAddr, // fromAddr
					erc20Addr,           // toAddr
				)
				return req, wantResp
			},
			wantErr: "",
		},
		{
			name: "happy: zero-gas trace tx with wallet-managed fee fields",
			scenario: func(deps *evmtest.TestDeps) (req In, wantResp Out) {
				txMsg := setupZeroGasSentinelTx(deps)
				s.Require().Equal(uint8(gethcore.DynamicFeeTxType), txMsg.AsTransaction().Type())
				toAddr := txMsg.AsTransaction().To()
				s.Require().NotNil(toAddr)
				req = &evm.QueryTraceTxRequest{
					Msg: txMsg,
				}
				wantResp = TraceResCallTracer_ZeroGasSentinel(
					deps.Sender.EthAddr,
					*toAddr,
				)
				return req, wantResp
			},
			wantErr: "",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			deps := evmtest.NewTestDeps()
			if tc.setup != nil {
				tc.setup(&deps)
			}
			req, wantResp := tc.scenario(&deps)
			goCtx := sdk.WrapSDKContext(deps.Ctx())
			gotResp, err := deps.EvmKeeper.TraceTx(goCtx, req)
			if tc.wantErr != "" {
				s.Require().ErrorContains(err, tc.wantErr)
				return
			}
			s.Assert().NoError(err)
			s.Assert().NotNil(gotResp)
			s.Assert().NotNil(gotResp.Data)
			s.Require().True(json.Valid(gotResp.Data), "expected json.RawMessage")
			s.T().Logf("gotResp.Data: %s", gotResp.Data)

			// TraceTx returns a single TxTraceResult object.
			gotTraceRes := make(map[string]string)
			s.Require().NoError(
				json.Unmarshal(gotResp.Data, &gotTraceRes),
				"expect map[string]string for TxTraceResult.Result field",
			)

			// Field-by-field comparison with address normalization (EIP-55 vs lowercase)
			for k, v := range wantResp {
				gotV, ok := gotTraceRes[k]
				s.Require().Truef(ok,
					"expect result.%s field to be present: { got: %s, want: %s }",
					k, gotTraceRes, wantResp,
				)

				switch k {
				case "from", "to":
					// Compare as addresses (case-insensitive, byte-equal)
					s.Equalf(
						gethcommon.HexToAddress(v),
						gethcommon.HexToAddress(gotV),
						`mismatch in trace result { key: %s, testCase: "%s" }`, k, tc.name,
					)
				default:
					// If it's hex, ignore case; else exact
					if strings.HasPrefix(gotV, "0x") {
						v = strings.ToLower(v)
						gotV = strings.ToLower(gotV)
					}
					s.Equalf(v, gotV, `mismatch in trace result { key: %s, testCase: "%s" }`, k, tc.name)
				}
			}
		})
	}
}

func (s *Suite) TestTraceBlock() {
	type In = *evm.QueryTraceBlockRequest
	type Out = map[string]string
	testCases := []TestCase[In, Out]{
		{
			name: "sad: nil query",
			scenario: func(deps *evmtest.TestDeps) (req In, wantResp Out) {
				return nil, nil
			},
			wantErr: "InvalidArgument",
		},
		{
			name:  "happy: simple nibi transfer tx",
			setup: nil,
			scenario: func(deps *evmtest.TestDeps) (req In, wantResp Out) {
				txMsg, _ := evmtest.ExecuteNibiTransfer(deps, s.T())
				req = &evm.QueryTraceBlockRequest{
					Txs: []*evm.MsgEthereumTx{
						txMsg,
					},
				}

				toAddr := txMsg.AsTransaction().To()
				s.Require().NotNil(toAddr)
				wantResp = TraceResCallTracer_NibiTransfer(
					deps.Sender.EthAddr, // fromAddr
					*toAddr,             // toAddr
				)
				return req, wantResp
			},
			wantErr: "",
		},
		{
			name:  "happy: trace erc-20 transfer tx",
			setup: nil,
			scenario: func(deps *evmtest.TestDeps) (req In, wantResp Out) {
				txMsg, _, erc20Addr := evmtest.DeployAndExecuteERC20Transfer(deps, s.T())
				req = &evm.QueryTraceBlockRequest{
					Txs: []*evm.MsgEthereumTx{
						txMsg,
					},
				}
				wantResp = TraceResCallTracer_ERC20Transfer(
					deps.Sender.EthAddr, // fromAddr
					erc20Addr,           // toAddr
				)
				return req, wantResp
			},
			wantErr: "",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			deps := evmtest.NewTestDeps()
			if tc.setup != nil {
				tc.setup(&deps)
			}
			req, wantResp := tc.scenario(&deps)
			goCtx := sdk.WrapSDKContext(deps.Ctx())
			gotResp, err := deps.EvmKeeper.TraceBlock(goCtx, req)
			if tc.wantErr != "" {
				s.Require().ErrorContains(err, tc.wantErr)
				return
			}
			s.Assert().NoError(err)
			s.Assert().NotNil(gotResp)
			s.Assert().NotNil(gotResp.Data)

			var (
				// Trace results encoded in the TraceBlock response.
				gotTraceResults []evm.TxTraceResult
				gotTraceResBz   []byte // raw bytes corresponding to "gotTraceRes"
				// JSON object for the "result" [evm.TxTraceResult]
				gotTraceRes map[string]string
			)
			s.Require().Truef(json.Valid(gotResp.Data),
				"expect valid JSON data. { got: %s }", gotResp.Data)
			err = json.Unmarshal(gotResp.Data, &gotTraceResults)
			s.Require().NoErrorf(err,
				"expect %T in the data field of the response", gotTraceResults)
			s.Require().Greater(len(gotTraceResults), 0)

			gotTraceResBz, err = json.Marshal(gotTraceResults[0].Result)
			s.Require().NoError(err, "result field should be JSON-able")

			err = json.Unmarshal(gotTraceResBz, &gotTraceRes)
			s.Require().NoErrorf(err,
				"expect map[string]string for TxTraceResult.Result field, got %s",
				gotTraceResults[0].Result)

			for k, v := range wantResp {
				gotV, ok := gotTraceRes[k]
				s.Require().Truef(ok,
					"expect result.%s field to be present: { got: %s, want: %s }",
					k, gotTraceRes, wantResp,
				)

				switch k {
				case "from", "to":
					// Compare as addresses (case-insensitive, byte-equal)
					s.Equalf(
						gethcommon.HexToAddress(v),
						gethcommon.HexToAddress(gotV),
						`mismatch in trace result { key: %s, testCase: "%s" }`, k, tc.name,
					)
				default:
					// If it's hex, ignore case; else exact
					if strings.HasPrefix(gotV, "0x") {
						v = strings.ToLower(v)
						gotV = strings.ToLower(gotV)
					}
					s.Equalf(v, gotV, `mismatch in trace result { key: %s, testCase: "%s" }`, k, tc.name)
				}
			}
		})
	}
}

func (s *Suite) TestTraceCall() {
	type In = *evm.QueryTraceTxRequest
	type Out = string

	testCases := []TestCase[In, Out]{
		{
			name: "sad: nil query",
			scenario: func(deps *evmtest.TestDeps) (req In, wantResp Out) {
				return nil, ""
			},
			wantErr: "InvalidArgument",
		},
		{
			name: "happy: simple nibi transfer tx",
			scenario: func(deps *evmtest.TestDeps) (req In, wantResp Out) {
				recipient := evmtest.NewEthPrivAcc().EthAddr
				gas := uint64(21000)
				txArgs := evm.JsonTxArgs{
					From:  &deps.Sender.EthAddr,
					To:    &recipient,
					Value: (*hexutil.Big)(big.NewInt(1e12)),
					Gas:   (*hexutil.Uint64)(&gas),
				}
				req = &evm.QueryTraceTxRequest{
					Msg: txArgs.ToMsgEthTx(),
				}
				wantResp = TraceNibiTransfer()
				return req, wantResp
			},
			wantErr: "",
		},
		{
			name: "happy: trace erc-20 transfer tx",
			scenario: func(deps *evmtest.TestDeps) (req In, wantResp Out) {
				s.T().Log("Deploy ERC20")

				deployResp, err := evmtest.DeployContract(
					deps, embeds.SmartContract_TestERC20,
				)
				s.Require().NoError(err)
				data, err := deployResp.ContractData.ABI.Pack(
					"transfer", evmtest.NewEthPrivAcc().EthAddr, new(big.Int).SetUint64(1000),
				)
				s.Require().NoError(err)
				gas := uint64(1e6)

				txArgs := evm.JsonTxArgs{
					From: &deps.Sender.EthAddr,
					To:   &deployResp.ContractAddr,
					Data: (*hexutil.Bytes)(&data),
					Gas:  (*hexutil.Uint64)(&gas),
				}
				req = &evm.QueryTraceTxRequest{
					Msg: txArgs.ToMsgEthTx(),
				}
				wantResp = TraceOutputERC20Transfer()
				return req, wantResp
			},
			wantErr: "",
		},
		{
			name: "happy: zero-gas trace call with wallet-managed fee fields",
			scenario: func(deps *evmtest.TestDeps) (req In, wantResp Out) {
				txArgs := setupZeroGasSentinelArgs(deps)
				req = &evm.QueryTraceTxRequest{
					Msg: txArgs.ToMsgEthTx(),
				}
				return req, ""
			},
			wantErr: "",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			deps := evmtest.NewTestDeps()
			if tc.setup != nil {
				tc.setup(&deps)
			}
			req, _ := tc.scenario(&deps)
			goCtx := sdk.WrapSDKContext(deps.Ctx())
			gotResp, err := deps.EvmKeeper.TraceCall(goCtx, req)
			if tc.wantErr != "" {
				s.Require().ErrorContains(err, tc.wantErr)
				return
			}
			s.NoError(err)
			s.NotNil(gotResp)
			s.NotNil(gotResp.Data)
			s.Require().Truef(json.Valid(gotResp.Data),
				"expect valid JSON data. { got: %s }", gotResp.Data)
		})
	}
}

func (s *Suite) TestQueryFunTokenMapping() {
	type In = *evm.QueryFunTokenMappingRequest
	type Out = *evm.QueryFunTokenMappingResponse
	testCases := []TestCase[In, Out]{
		{
			name: "sad: no token mapping",
			scenario: func(deps *evmtest.TestDeps) (req In, wantResp Out) {
				req = &evm.QueryFunTokenMappingRequest{
					Token: "unibi",
				}
				wantResp = &evm.QueryFunTokenMappingResponse{
					FunToken: nil,
				}
				return req, wantResp
			},
			wantErr: "token mapping not found for unibi",
		},
		{
			name: "happy: token mapping exists from cosmos coin -> ERC20 token",
			setup: func(deps *evmtest.TestDeps) {
				_ = deps.EvmKeeper.FunTokens.SafeInsert(
					deps.Ctx(),
					gethcommon.HexToAddress("0xAEf9437FF23D48D73271a41a8A094DEc9ac71477"),
					"unibi",
					true,
				)
			},
			scenario: func(deps *evmtest.TestDeps) (req In, wantResp Out) {
				req = &evm.QueryFunTokenMappingRequest{
					Token: "unibi",
				}
				wantResp = &evm.QueryFunTokenMappingResponse{
					FunToken: &evm.FunToken{
						Erc20Addr: eth.EIP55Addr{
							Address: gethcommon.HexToAddress("0xAEf9437FF23D48D73271a41a8A094DEc9ac71477"),
						},
						BankDenom:      "unibi",
						IsMadeFromCoin: true,
					},
				}
				return req, wantResp
			},
			wantErr: "",
		},
		{
			name: "happy: token mapping exists from ERC20 token -> cosmos coin",
			setup: func(deps *evmtest.TestDeps) {
				_ = deps.EvmKeeper.FunTokens.SafeInsert(
					deps.Ctx(),
					gethcommon.HexToAddress("0xAEf9437FF23D48D73271a41a8A094DEc9ac71477"),
					"unibi",
					true,
				)
			},
			scenario: func(deps *evmtest.TestDeps) (req In, wantResp Out) {
				req = &evm.QueryFunTokenMappingRequest{
					Token: "0xAEf9437FF23D48D73271a41a8A094DEc9ac71477",
				}
				wantResp = &evm.QueryFunTokenMappingResponse{
					FunToken: &evm.FunToken{
						Erc20Addr: eth.EIP55Addr{
							Address: gethcommon.HexToAddress("0xAEf9437FF23D48D73271a41a8A094DEc9ac71477"),
						},
						BankDenom:      "unibi",
						IsMadeFromCoin: true,
					},
				}
				return req, wantResp
			},
			wantErr: "",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			deps := evmtest.NewTestDeps()
			if tc.setup != nil {
				tc.setup(&deps)
			}
			req, wantResp := tc.scenario(&deps)
			goCtx := sdk.WrapSDKContext(deps.Ctx())
			gotResp, err := deps.EvmKeeper.FunTokenMapping(goCtx, req)
			if tc.wantErr != "" {
				s.Require().ErrorContains(err, tc.wantErr)
				return
			}
			s.Assert().NoError(err)
			s.EqualValues(wantResp, gotResp)
		})
	}
}
