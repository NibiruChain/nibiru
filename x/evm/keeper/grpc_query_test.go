package keeper_test

import (
	"encoding/json"
	"fmt"
	"math/big"
	"strings"

	"cosmossdk.io/math"
	"github.com/NibiruChain/collections"
	sdk "github.com/cosmos/cosmos-sdk/types"
	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	gethparams "github.com/ethereum/go-ethereum/params"

	"github.com/NibiruChain/nibiru/v2/eth"
	"github.com/NibiruChain/nibiru/v2/x/common/testutil/testapp"
	"github.com/NibiruChain/nibiru/v2/x/evm"
	"github.com/NibiruChain/nibiru/v2/x/evm/embeds"
	"github.com/NibiruChain/nibiru/v2/x/evm/evmtest"
)

type TestCase[In, Out any] struct {
	name string
	// setup: Optional setup function to create the scenario
	setup    func(deps *evmtest.TestDeps)
	scenario func(deps *evmtest.TestDeps) (
		req In,
		wantResp Out,
	)
	onTestEnd func(deps *evmtest.TestDeps)
	wantErr   string
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

// TraceERC20Transfer returns a hardcoded JSON string representing the expected
// trace output of a successful ERC-20 token transfer (an EVM tx).
// Used to test the correctness of "TraceTx" and "TraceBlock".
func TraceERC20Transfer() string {
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
				coins := sdk.Coins{sdk.NewInt64Coin(evm.DefaultEVMDenom, 420)}
				err := testapp.FundAccount(deps.App.BankKeeper, deps.Ctx, ethAddr.Bytes(), coins)
				s.Require().NoError(err)
			},
			scenario: func(deps *evmtest.TestDeps) (req In, wantResp Out) {
				req = &evm.QueryEthAccountRequest{
					Address: deps.Sender.EthAddr.Hex(),
				}
				wantResp = &evm.QueryEthAccountResponse{
					Balance:       "420",
					BalanceWei:    "420" + strings.Repeat("0", 12),
					CodeHash:      gethcommon.BytesToHash(evm.EmptyCodeHash).Hex(),
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
				coins := sdk.Coins{sdk.NewInt64Coin(evm.DefaultEVMDenom, 420)}
				err := testapp.FundAccount(deps.App.BankKeeper, deps.Ctx, ethAddr.Bytes(), coins)
				s.Require().NoError(err)
			},
			scenario: func(deps *evmtest.TestDeps) (req In, wantResp Out) {
				req = &evm.QueryEthAccountRequest{
					Address: deps.Sender.NibiruAddr.String(),
				}
				wantResp = &evm.QueryEthAccountResponse{
					Balance:       "420",
					BalanceWei:    "420" + strings.Repeat("0", 12),
					CodeHash:      gethcommon.BytesToHash(evm.EmptyCodeHash).Hex(),
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
					Balance:    "0",
					BalanceWei: "0",
					CodeHash:   gethcommon.BytesToHash(evm.EmptyCodeHash).Hex(),
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
					Balance:       "0",
					BalanceWei:    "0",
					CodeHash:      gethcommon.BytesToHash(evm.EmptyCodeHash).Hex(),
					Nonce:         0,
					EthAddress:    ethAcc.EthAddr.String(),
					Bech32Address: ethAcc.NibiruAddr.String(),
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
					Balance:       "0",
					BalanceWei:    "0",
					CodeHash:      gethcommon.BytesToHash(evm.EmptyCodeHash).Hex(),
					Nonce:         0,
					EthAddress:    ethAcc.EthAddr.String(),
					Bech32Address: ethAcc.NibiruAddr.String(),
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
			goCtx := sdk.WrapSDKContext(deps.Ctx)
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
				valopers := deps.App.StakingKeeper.GetValidators(deps.Ctx, 1)
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
				valopers := deps.App.StakingKeeper.GetValidators(deps.Ctx, 1)
				valAddrBz := valopers[0].GetOperator().Bytes()
				consAddr := sdk.ConsAddress(valAddrBz)

				s.T().Log(
					"Send coins to validator to register in the account keeper.")
				coinsToSend := sdk.NewCoins(sdk.NewCoin(eth.EthBaseDenom, math.NewInt(69420)))
				valAddr := sdk.AccAddress(valAddrBz)
				s.NoError(testapp.FundAccount(
					deps.App.BankKeeper,
					deps.Ctx, valAddr,
					coinsToSend,
				))

				req = &evm.QueryValidatorAccountRequest{
					ConsAddress: consAddr.String(),
				}

				ak := deps.App.AccountKeeper
				acc := ak.GetAccount(deps.Ctx, valAddr)
				s.NoError(acc.SetAccountNumber(420), "acc: ", acc.String())
				s.NoError(acc.SetSequence(69), "acc: ", acc.String())
				ak.SetAccount(deps.Ctx, acc)

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
			goCtx := sdk.WrapSDKContext(deps.Ctx)
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

				stateDB := deps.StateDB()
				storageValue := gethcommon.BytesToHash([]byte("value"))

				stateDB.SetState(addr, storageKey, storageValue)
				s.NoError(stateDB.Commit())

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
			goCtx := sdk.WrapSDKContext(deps.Ctx)

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

				stateDB := deps.StateDB()
				contractBytecode := []byte("bytecode")
				stateDB.SetCode(addr, contractBytecode)
				s.Require().NoError(stateDB.Commit())

				s.NotNil(stateDB.Keeper().GetAccount(deps.Ctx, addr))
				s.NotNil(stateDB.GetCode(addr))

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
			goCtx := sdk.WrapSDKContext(deps.Ctx)

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
	deps.EvmKeeper.SetParams(deps.Ctx, want)
	gotResp, err := deps.EvmKeeper.Params(sdk.WrapSDKContext(deps.Ctx), nil)
	s.NoError(err)
	got := gotResp.Params
	s.Require().NoError(err)

	// Note that protobuf equals is more reliable than `s.Equal`
	s.Require().True(want.Equal(got), "want %s, got %s", want, got)

	// Empty params to test the setter
	want.EvmDenom = "wei"
	deps.EvmKeeper.SetParams(deps.Ctx, want)
	gotResp, err = deps.EvmKeeper.Params(sdk.WrapSDKContext(deps.Ctx), nil)
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
			name: "happy: eth call for erc20 token transfer",
			scenario: func(deps *evmtest.TestDeps) (req In, wantResp Out) {
				fungibleTokenContract := embeds.SmartContract_ERC20Minter

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
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			deps := evmtest.NewTestDeps()
			if tc.setup != nil {
				tc.setup(&deps)
			}
			req, wantResp := tc.scenario(&deps)
			gotResp, err := deps.App.EvmKeeper.EthCall(sdk.WrapSDKContext(deps.Ctx), req)
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
			wantErr: "InvalidArgument",
		},
		{
			name: "happy: zero balance",
			scenario: func(deps *evmtest.TestDeps) (req In, wantResp Out) {
				req = &evm.QueryBalanceRequest{
					Address: evmtest.NewEthPrivAcc().EthAddr.String(),
				}
				wantResp = &evm.QueryBalanceResponse{
					Balance:    "0",
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
				coins := sdk.Coins{sdk.NewInt64Coin(evm.DefaultEVMDenom, 420)}
				err := chain.BankKeeper.MintCoins(deps.Ctx, evm.ModuleName, coins)
				s.NoError(err)
				err = chain.BankKeeper.SendCoinsFromModuleToAccount(
					deps.Ctx, evm.ModuleName, ethAddr.Bytes(), coins)
				s.Require().NoError(err)
			},
			scenario: func(deps *evmtest.TestDeps) (req In, wantResp Out) {
				req = &evm.QueryBalanceRequest{
					Address: deps.Sender.EthAddr.Hex(),
				}
				wantResp = &evm.QueryBalanceResponse{
					Balance:    "420",
					BalanceWei: "420" + strings.Repeat("0", 12),
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
			goCtx := sdk.WrapSDKContext(deps.Ctx)
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
				zeroFee := math.NewInt(1)
				wantResp = &evm.QueryBaseFeeResponse{
					BaseFee: &zeroFee,
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
			goCtx := sdk.WrapSDKContext(deps.Ctx)
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
				coins := sdk.Coins{sdk.NewInt64Coin(evm.DefaultEVMDenom, 1000)}
				err := chain.BankKeeper.MintCoins(deps.Ctx, evm.ModuleName, coins)
				s.NoError(err)
				err = chain.BankKeeper.SendCoinsFromModuleToAccount(
					deps.Ctx, evm.ModuleName, ethAddr.Bytes(), coins)
				s.Require().NoError(err)

				// assert balance of 1000 * 10^12 wei
				resp, _ := deps.App.EvmKeeper.Balance(sdk.WrapSDKContext(deps.Ctx), &evm.QueryBalanceRequest{
					Address: deps.Sender.EthAddr.Hex(),
				})
				s.Equal("1000", resp.Balance)
				s.Require().Equal("1000"+strings.Repeat("0", 12), resp.BalanceWei)

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
			onTestEnd: func(deps *evmtest.TestDeps) {
			},
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
			goCtx := sdk.WrapSDKContext(deps.Ctx)
			gotResp, err := deps.EvmKeeper.EstimateGas(goCtx, req)
			if tc.wantErr != "" {
				s.Require().ErrorContains(err, tc.wantErr)
				return
			}
			s.Assert().NoError(err)
			s.EqualValues(wantResp, gotResp)

			if tc.onTestEnd != nil {
				tc.onTestEnd(&deps)
			}
		})
	}
}

func (s *Suite) TestTraceTx() {
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
				txMsg := evmtest.ExecuteNibiTransfer(deps, s.T())
				req = &evm.QueryTraceTxRequest{
					Msg: txMsg,
				}
				wantResp = TraceNibiTransfer()
				return req, wantResp
			},
			wantErr: "",
		},
		{
			name: "happy: trace erc-20 transfer tx",
			scenario: func(deps *evmtest.TestDeps) (req In, wantResp Out) {
				txMsg, predecessors := evmtest.DeployAndExecuteERC20Transfer(deps, s.T())

				req = &evm.QueryTraceTxRequest{
					Msg:          txMsg,
					Predecessors: predecessors,
				}
				wantResp = TraceERC20Transfer()
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
			req, _ := tc.scenario(&deps)
			goCtx := sdk.WrapSDKContext(deps.Ctx)
			gotResp, err := deps.EvmKeeper.TraceTx(goCtx, req)
			if tc.wantErr != "" {
				s.Require().ErrorContains(err, tc.wantErr)
				return
			}
			s.Assert().NoError(err)
			s.Assert().NotNil(gotResp)
			s.Assert().NotNil(gotResp.Data)

			// // Replace spaces in want resp
			// re := regexp.MustCompile(`[\s\n\r]+`)
			// wantResp = re.ReplaceAllString(wantResp, "")
			// actualResp := string(gotResp.Data)
			// if len(actualResp) > 1000 {
			// 	actualResp = actualResp[:len(wantResp)]
			// }
			// // FIXME: Why does this trace sometimes have gas 35050 and sometimes 35062?
			// // s.Equal(wantResp, actualResp)
			// replaceTimes := 1
			// hackedWantResp := strings.Replace(wantResp, "35062", "35050", replaceTimes)
			// s.True(
			// 	wantResp == actualResp || hackedWantResp == actualResp,
			// 	"got \"%s\", want \"%s\"", actualResp, wantResp,
			// )
		})
	}
}

func (s *Suite) TestTraceBlock() {
	type In = *evm.QueryTraceBlockRequest
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
			name:  "happy: simple nibi transfer tx",
			setup: nil,
			scenario: func(deps *evmtest.TestDeps) (req In, wantResp Out) {
				txMsg := evmtest.ExecuteNibiTransfer(deps, s.T())
				req = &evm.QueryTraceBlockRequest{
					Txs: []*evm.MsgEthereumTx{
						txMsg,
					},
				}
				wantResp = "[{\"result\":" + TraceNibiTransfer() + "}]"
				return req, wantResp
			},
			wantErr: "",
		},
		{
			name:  "happy: trace erc-20 transfer tx",
			setup: nil,
			scenario: func(deps *evmtest.TestDeps) (req In, wantResp Out) {
				txMsg, _ := evmtest.DeployAndExecuteERC20Transfer(deps, s.T())
				req = &evm.QueryTraceBlockRequest{
					Txs: []*evm.MsgEthereumTx{
						txMsg,
					},
				}
				wantResp = "[{\"result\":" + TraceERC20Transfer() // no end as it's trimmed
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
			req, _ := tc.scenario(&deps)
			goCtx := sdk.WrapSDKContext(deps.Ctx)
			gotResp, err := deps.EvmKeeper.TraceBlock(goCtx, req)
			if tc.wantErr != "" {
				s.Require().ErrorContains(err, tc.wantErr)
				return
			}
			s.Assert().NoError(err)
			s.Assert().NotNil(gotResp)
			s.Assert().NotNil(gotResp.Data)

			// Replace spaces in want resp
			// re := regexp.MustCompile(`[\s\n\r]+`)
			// wantResp = re.ReplaceAllString(wantResp, "")
			// actualResp := string(gotResp.Data)
			// if len(actualResp) > 1000 {
			// 	actualResp = actualResp[:len(wantResp)]
			// }
			// FIXME: Why does this trace sometimes have gas 35050 and sometimes 35062?
			// s.Equal(wantResp, actualResp)
			// replaceTimes := 1
			// hackedWantResp := strings.Replace(wantResp, "35062", "35050", replaceTimes)
			// s.True(
			// 	wantResp == actualResp || hackedWantResp == actualResp,
			// 	"got \"%s\", want \"%s\"", actualResp, wantResp,
			// )
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
				wantResp = TraceERC20Transfer()
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
			req, _ := tc.scenario(&deps)
			goCtx := sdk.WrapSDKContext(deps.Ctx)
			gotResp, err := deps.EvmKeeper.TraceCall(goCtx, req)
			if tc.wantErr != "" {
				s.Require().ErrorContains(err, tc.wantErr)
				return
			}
			s.Assert().NoError(err)
			s.Assert().NotNil(gotResp)
			s.Assert().NotNil(gotResp.Data)
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
					deps.Ctx,
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
					deps.Ctx,
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
			goCtx := sdk.WrapSDKContext(deps.Ctx)
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
