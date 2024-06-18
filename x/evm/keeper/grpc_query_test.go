package keeper_test

import (
	"encoding/json"
	"fmt"
	"math/big"
	"regexp"

	"cosmossdk.io/math"
	"github.com/NibiruChain/collections"
	sdk "github.com/cosmos/cosmos-sdk/types"
	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	gethparams "github.com/ethereum/go-ethereum/params"

	"github.com/NibiruChain/nibiru/eth"
	"github.com/NibiruChain/nibiru/x/common/testutil/testapp"
	"github.com/NibiruChain/nibiru/x/evm/evmtest"
	"github.com/NibiruChain/nibiru/x/evm/types"
)

type TestCase[In, Out any] struct {
	name string
	// setup: Optional setup function to create the scenario
	setup    func(deps *evmtest.TestDeps)
	scenario func(deps *evmtest.TestDeps) (
		req In,
		wantResp Out,
	)
	wantErr string
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

func (s *KeeperSuite) TestQueryNibiruAccount() {
	type In = *types.QueryNibiruAccountRequest
	type Out = *types.QueryNibiruAccountResponse
	testCases := []TestCase[In, Out]{
		{
			name: "sad: msg validation",
			scenario: func(deps *evmtest.TestDeps) (req In, wantResp Out) {
				req = &types.QueryNibiruAccountRequest{
					Address: InvalidEthAddr(),
				}
				wantResp = &types.QueryNibiruAccountResponse{
					Address: sdk.AccAddress(gethcommon.Address{}.Bytes()).String(),
				}
				return req, wantResp
			},
			wantErr: "not a valid ethereum hex addr",
		},
		{
			name: "happy: not existing account",
			scenario: func(deps *evmtest.TestDeps) (req In, wantResp Out) {
				ethAcc := evmtest.NewEthAccInfo()
				req = &types.QueryNibiruAccountRequest{
					Address: ethAcc.EthAddr.String(),
				}
				wantResp = &types.QueryNibiruAccountResponse{
					Address:       ethAcc.NibiruAddr.String(),
					Sequence:      0,
					AccountNumber: 0,
				}
				return req, wantResp
			},
			wantErr: "",
		},
		{
			name: "happy: existing account",
			scenario: func(deps *evmtest.TestDeps) (req In, wantResp Out) {
				ethAcc := evmtest.NewEthAccInfo()
				accountKeeper := deps.Chain.AccountKeeper
				account := accountKeeper.NewAccountWithAddress(deps.Ctx, ethAcc.NibiruAddr)
				accountKeeper.SetAccount(deps.Ctx, account)

				req = &types.QueryNibiruAccountRequest{
					Address: ethAcc.EthAddr.String(),
				}
				wantResp = &types.QueryNibiruAccountResponse{
					Address:       ethAcc.NibiruAddr.String(),
					Sequence:      account.GetSequence(),
					AccountNumber: account.GetAccountNumber(),
				}
				return req, wantResp
			},
			wantErr: "",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			deps := evmtest.NewTestDeps()
			req, wantResp := tc.scenario(&deps)
			goCtx := sdk.WrapSDKContext(deps.Ctx)
			gotResp, err := deps.K.NibiruAccount(goCtx, req)
			if tc.wantErr != "" {
				s.Require().ErrorContains(err, tc.wantErr)
				return
			}
			s.Assert().NoError(err)
			s.EqualValues(wantResp, gotResp)
		})
	}
}

func (s *KeeperSuite) TestQueryEthAccount() {
	type In = *types.QueryEthAccountRequest
	type Out = *types.QueryEthAccountResponse
	testCases := []TestCase[In, Out]{
		{
			name: "sad: msg validation",
			scenario: func(deps *evmtest.TestDeps) (req In, wantResp Out) {
				req = &types.QueryEthAccountRequest{
					Address: InvalidEthAddr(),
				}
				wantResp = &types.QueryEthAccountResponse{
					Balance:  "0",
					CodeHash: gethcommon.BytesToHash(types.EmptyCodeHash).Hex(),
					Nonce:    0,
				}
				return req, wantResp
			},
			wantErr: "not a valid ethereum hex addr",
		},
		{
			name: "happy: fund account + query",
			setup: func(deps *evmtest.TestDeps) {
				chain := deps.Chain
				ethAddr := deps.Sender.EthAddr

				// fund account with 420 tokens
				coins := sdk.Coins{sdk.NewInt64Coin(types.DefaultEVMDenom, 420)}
				err := chain.BankKeeper.MintCoins(deps.Ctx, types.ModuleName, coins)
				s.NoError(err)
				err = chain.BankKeeper.SendCoinsFromModuleToAccount(
					deps.Ctx, types.ModuleName, ethAddr.Bytes(), coins)
				s.Require().NoError(err)
			},
			scenario: func(deps *evmtest.TestDeps) (req In, wantResp Out) {
				req = &types.QueryEthAccountRequest{
					Address: deps.Sender.EthAddr.Hex(),
				}
				wantResp = &types.QueryEthAccountResponse{
					Balance:  "420",
					CodeHash: gethcommon.BytesToHash(types.EmptyCodeHash).Hex(),
					Nonce:    0,
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
			gotResp, err := deps.K.EthAccount(goCtx, req)
			if tc.wantErr != "" {
				s.Require().ErrorContains(err, tc.wantErr)
				return
			}
			s.Assert().NoError(err)
			s.EqualValues(wantResp, gotResp)
		})
	}
}

func (s *KeeperSuite) TestQueryValidatorAccount() {
	type In = *types.QueryValidatorAccountRequest
	type Out = *types.QueryValidatorAccountResponse
	testCases := []TestCase[In, Out]{
		{
			name: "sad: msg validation",
			scenario: func(deps *evmtest.TestDeps) (req In, wantResp Out) {
				req = &types.QueryValidatorAccountRequest{
					ConsAddress: "nibi1invalidaddr",
				}
				wantResp = &types.QueryValidatorAccountResponse{
					AccountAddress: sdk.AccAddress(gethcommon.Address{}.Bytes()).String(),
				}
				return req, wantResp
			},
			wantErr: "decoding bech32 failed",
		},
		{
			name: "sad: validator account not found",
			scenario: func(deps *evmtest.TestDeps) (req In, wantResp Out) {
				req = &types.QueryValidatorAccountRequest{
					ConsAddress: "nibivalcons1ea4ef7wsatlnaj9ry3zylymxv53f9ntrjecc40",
				}
				wantResp = &types.QueryValidatorAccountResponse{
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
				valopers := deps.Chain.StakingKeeper.GetValidators(deps.Ctx, 1)
				valAddrBz := valopers[0].GetOperator().Bytes()
				_, err := sdk.ConsAddressFromBech32(valopers[0].OperatorAddress)
				s.ErrorContains(err, "expected nibivalcons, got nibivaloper")
				consAddr := sdk.ConsAddress(valAddrBz)

				req = &types.QueryValidatorAccountRequest{
					ConsAddress: consAddr.String(),
				}
				wantResp = &types.QueryValidatorAccountResponse{
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
				valopers := deps.Chain.StakingKeeper.GetValidators(deps.Ctx, 1)
				valAddrBz := valopers[0].GetOperator().Bytes()
				consAddr := sdk.ConsAddress(valAddrBz)

				s.T().Log(
					"Send coins to validator to register in the account keeper.")
				coinsToSend := sdk.NewCoins(sdk.NewCoin(eth.EthBaseDenom, math.NewInt(69420)))
				valAddr := sdk.AccAddress(valAddrBz)
				s.NoError(testapp.FundAccount(
					deps.Chain.BankKeeper,
					deps.Ctx, valAddr,
					coinsToSend,
				))

				req = &types.QueryValidatorAccountRequest{
					ConsAddress: consAddr.String(),
				}

				ak := deps.Chain.AccountKeeper
				acc := ak.GetAccount(deps.Ctx, valAddr)
				s.NoError(acc.SetAccountNumber(420), "acc: ", acc.String())
				s.NoError(acc.SetSequence(69), "acc: ", acc.String())
				ak.SetAccount(deps.Ctx, acc)

				wantResp = &types.QueryValidatorAccountResponse{
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
			gotResp, err := deps.K.ValidatorAccount(goCtx, req)
			if tc.wantErr != "" {
				s.Require().ErrorContains(err, tc.wantErr)
				return
			}
			s.Assert().NoError(err)
			s.EqualValues(wantResp, gotResp)
		})
	}
}

func (s *KeeperSuite) TestQueryStorage() {
	type In = *types.QueryStorageRequest
	type Out = *types.QueryStorageResponse
	testCases := []TestCase[In, Out]{
		{
			name: "sad: msg validation",
			scenario: func(deps *evmtest.TestDeps) (req In, wantResp Out) {
				req = &types.QueryStorageRequest{
					Address: InvalidEthAddr(),
				}
				return req, wantResp
			},
			wantErr: "InvalidArgument",
		},
		{
			name: "happy",
			scenario: func(deps *evmtest.TestDeps) (req In, wantResp Out) {
				addr := evmtest.NewEthAccInfo().EthAddr
				storageKey := gethcommon.BytesToHash([]byte("storagekey"))
				req = &types.QueryStorageRequest{
					Address: addr.Hex(),
					Key:     storageKey.String(),
				}

				stateDB := deps.StateDB()
				storageValue := gethcommon.BytesToHash([]byte("value"))

				stateDB.SetState(addr, storageKey, storageValue)
				s.NoError(stateDB.Commit())

				wantResp = &types.QueryStorageResponse{
					Value: storageValue.String(),
				}
				return req, wantResp
			},
			wantErr: "",
		},
		{
			name: "happy: no committed state",
			scenario: func(deps *evmtest.TestDeps) (req In, wantResp Out) {
				addr := evmtest.NewEthAccInfo().EthAddr
				storageKey := gethcommon.BytesToHash([]byte("storagekey"))
				req = &types.QueryStorageRequest{
					Address: addr.Hex(),
					Key:     storageKey.String(),
				}

				wantResp = &types.QueryStorageResponse{
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

			gotResp, err := deps.K.Storage(goCtx, req)
			if tc.wantErr != "" {
				s.Require().ErrorContains(err, tc.wantErr)
				return
			}
			s.Assert().NoError(err)
			s.EqualValues(wantResp, gotResp)
		})
	}
}

func (s *KeeperSuite) TestQueryCode() {
	type In = *types.QueryCodeRequest
	type Out = *types.QueryCodeResponse
	testCases := []TestCase[In, Out]{
		{
			name: "sad: msg validation",
			scenario: func(deps *evmtest.TestDeps) (req In, wantResp Out) {
				req = &types.QueryCodeRequest{
					Address: InvalidEthAddr(),
				}
				return req, wantResp
			},
			wantErr: "InvalidArgument",
		},
		{
			name: "happy",
			scenario: func(deps *evmtest.TestDeps) (req In, wantResp Out) {
				addr := evmtest.NewEthAccInfo().EthAddr
				req = &types.QueryCodeRequest{
					Address: addr.Hex(),
				}

				stateDB := deps.StateDB()
				contractBytecode := []byte("bytecode")
				stateDB.SetCode(addr, contractBytecode)
				s.Require().NoError(stateDB.Commit())

				s.NotNil(stateDB.Keeper().GetAccount(deps.Ctx, addr))
				s.NotNil(stateDB.GetCode(addr))

				wantResp = &types.QueryCodeResponse{
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

			gotResp, err := deps.K.Code(goCtx, req)
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

func (s *KeeperSuite) TestQueryParams() {
	deps := evmtest.NewTestDeps()
	want := types.DefaultParams()
	deps.K.SetParams(deps.Ctx, want)
	gotResp, err := deps.K.Params(deps.GoCtx(), nil)
	s.NoError(err)
	got := gotResp.Params
	s.Require().NoError(err)

	// Note that protobuf equals is more reliable than `s.Equal`
	s.Require().True(want.Equal(got), "want %s, got %s", want, got)

	// Empty params to test the setter
	want.ActivePrecompiles = []string{"new", "something"}
	deps.K.SetParams(deps.Ctx, want)
	gotResp, err = deps.K.Params(deps.GoCtx(), nil)
	s.Require().NoError(err)
	got = gotResp.Params

	// Note that protobuf equals is more reliable than `s.Equal`
	s.Require().True(want.Equal(got), "want %s, got %s", want, got)
}

func (s *KeeperSuite) TestQueryEthCall() {
	type In = *types.EthCallRequest
	type Out = *types.MsgEthereumTxResponse
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
				return &types.EthCallRequest{Args: []byte("invalid")}, nil
			},
			wantErr: "InvalidArgument",
		},
		{
			name: "happy: eth call for erc20 token transfer",
			scenario: func(deps *evmtest.TestDeps) (req In, wantResp Out) {
				fungibleTokenContract := evmtest.SmartContract_FunToken.Load(s.T())

				jsonTxArgs, err := json.Marshal(&types.JsonTxArgs{
					From: &deps.Sender.EthAddr,
					Data: (*hexutil.Bytes)(&fungibleTokenContract.Bytecode),
				})
				s.Require().NoError(err)
				return &types.EthCallRequest{Args: jsonTxArgs}, &types.MsgEthereumTxResponse{
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
			gotResp, err := deps.Chain.EvmKeeper.EthCall(deps.GoCtx(), req)
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

func (s *KeeperSuite) TestQueryBalance() {
	type In = *types.QueryBalanceRequest
	type Out = *types.QueryBalanceResponse
	testCases := []TestCase[In, Out]{
		{
			name: "sad: msg validation",
			scenario: func(deps *evmtest.TestDeps) (req In, wantResp Out) {
				req = &types.QueryBalanceRequest{
					Address: InvalidEthAddr(),
				}
				wantResp = &types.QueryBalanceResponse{
					Balance: "0",
				}
				return req, wantResp
			},
			wantErr: "InvalidArgument",
		},
		{
			name: "happy: zero balance",
			scenario: func(deps *evmtest.TestDeps) (req In, wantResp Out) {
				req = &types.QueryBalanceRequest{
					Address: evmtest.NewEthAccInfo().EthAddr.String(),
				}
				wantResp = &types.QueryBalanceResponse{
					Balance: "0",
				}
				return req, wantResp
			},
			wantErr: "",
		},
		{
			name: "happy: non zero balance",
			setup: func(deps *evmtest.TestDeps) {
				chain := deps.Chain
				ethAddr := deps.Sender.EthAddr

				// fund account with 420 tokens
				coins := sdk.Coins{sdk.NewInt64Coin(types.DefaultEVMDenom, 420)}
				err := chain.BankKeeper.MintCoins(deps.Ctx, types.ModuleName, coins)
				s.NoError(err)
				err = chain.BankKeeper.SendCoinsFromModuleToAccount(
					deps.Ctx, types.ModuleName, ethAddr.Bytes(), coins)
				s.Require().NoError(err)
			},
			scenario: func(deps *evmtest.TestDeps) (req In, wantResp Out) {
				req = &types.QueryBalanceRequest{
					Address: deps.Sender.EthAddr.Hex(),
				}
				wantResp = &types.QueryBalanceResponse{
					Balance: "420",
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
			gotResp, err := deps.K.Balance(goCtx, req)
			if tc.wantErr != "" {
				s.Require().ErrorContains(err, tc.wantErr)
				return
			}
			s.Assert().NoError(err)
			s.EqualValues(wantResp, gotResp)
		})
	}
}

func (s *KeeperSuite) TestQueryBaseFee() {
	type In = *types.QueryBaseFeeRequest
	type Out = *types.QueryBaseFeeResponse
	testCases := []TestCase[In, Out]{
		{
			name: "happy: base fee value",
			scenario: func(deps *evmtest.TestDeps) (req In, wantResp Out) {
				req = &types.QueryBaseFeeRequest{}
				zeroFee := math.NewInt(0)
				wantResp = &types.QueryBaseFeeResponse{
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
			gotResp, err := deps.K.BaseFee(goCtx, req)
			if tc.wantErr != "" {
				s.Require().ErrorContains(err, tc.wantErr)
				return
			}
			s.Assert().NoError(err)
			s.EqualValues(wantResp, gotResp)
		})
	}
}

func (s *KeeperSuite) TestEstimateGasForEvmCallType() {
	type In = *types.EthCallRequest
	type Out = *types.EstimateGasResponse
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
				req = &types.EthCallRequest{
					GasCap: gethparams.TxGas - 1,
				}
				return req, nil
			},
			wantErr: "InvalidArgument",
		},
		{
			name: "sad: invalid args",
			scenario: func(deps *evmtest.TestDeps) (req In, wantResp Out) {
				req = &types.EthCallRequest{
					Args:   []byte{0, 0, 0},
					GasCap: gethparams.TxGas,
				}
				return req, nil
			},
			wantErr: "InvalidArgument",
		},
		{
			name: "happy: estimate gas for transfer",
			setup: func(deps *evmtest.TestDeps) {
				chain := deps.Chain
				ethAddr := deps.Sender.EthAddr
				coins := sdk.Coins{sdk.NewInt64Coin(types.DefaultEVMDenom, 1000)}
				err := chain.BankKeeper.MintCoins(deps.Ctx, types.ModuleName, coins)
				s.NoError(err)
				err = chain.BankKeeper.SendCoinsFromModuleToAccount(
					deps.Ctx, types.ModuleName, ethAddr.Bytes(), coins)
				s.Require().NoError(err)
			},
			scenario: func(deps *evmtest.TestDeps) (req In, wantResp Out) {
				recipient := evmtest.NewEthAccInfo().EthAddr
				amountToSend := hexutil.Big(*big.NewInt(10))
				gasLimitArg := hexutil.Uint64(100000)

				jsonTxArgs, err := json.Marshal(&types.JsonTxArgs{
					From:  &deps.Sender.EthAddr,
					To:    &recipient,
					Value: &amountToSend,
					Gas:   &gasLimitArg,
				})
				s.Require().NoError(err)
				req = &types.EthCallRequest{
					Args:   jsonTxArgs,
					GasCap: gethparams.TxGas,
				}
				wantResp = &types.EstimateGasResponse{
					Gas: gethparams.TxGas,
				}
				return req, wantResp
			},
			wantErr: "",
		},
		{
			name: "sad: insufficient balance for transfer",
			scenario: func(deps *evmtest.TestDeps) (req In, wantResp Out) {
				recipient := evmtest.NewEthAccInfo().EthAddr
				amountToSend := hexutil.Big(*big.NewInt(10))

				jsonTxArgs, err := json.Marshal(&types.JsonTxArgs{
					From:  &deps.Sender.EthAddr,
					To:    &recipient,
					Value: &amountToSend,
				})
				s.Require().NoError(err)
				req = &types.EthCallRequest{
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
			gotResp, err := deps.K.EstimateGas(goCtx, req)
			if tc.wantErr != "" {
				s.Require().ErrorContains(err, tc.wantErr)
				return
			}
			s.Assert().NoError(err)
			s.EqualValues(wantResp, gotResp)
		})
	}
}

func (s *KeeperSuite) TestTestTraceTx() {
	type In = *types.QueryTraceTxRequest
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
				req = &types.QueryTraceTxRequest{
					Msg: txMsg,
				}
				wantResp = TraceNibiTransfer()
				return req, wantResp
			},
			wantErr: "",
		},
		{
			"happy: trace erc-20 transfer tx",
			nil,
			func(deps *evmtest.TestDeps) (req In, wantResp Out) {
				txMsg, predecessors := evmtest.ExecuteERC20Transfer(deps, s.T())

				req = &types.QueryTraceTxRequest{
					Msg:          txMsg,
					Predecessors: predecessors,
				}
				wantResp = TraceERC20Transfer()
				return req, wantResp
			},
			"",
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
			gotResp, err := deps.K.TraceTx(goCtx, req)
			if tc.wantErr != "" {
				s.Require().ErrorContains(err, tc.wantErr)
				return
			}
			s.Assert().NoError(err)
			s.Assert().NotNil(gotResp)
			s.Assert().NotNil(gotResp.Data)

			// Replace spaces in want resp
			re := regexp.MustCompile(`[\s\n\r]+`)
			wantResp = re.ReplaceAllString(wantResp, "")
			actualResp := string(gotResp.Data)
			if len(actualResp) > 1000 {
				actualResp = actualResp[:len(wantResp)]
			}
			s.Assert().Equal(wantResp, actualResp)
		})
	}
}

func (s *KeeperSuite) TestTestTraceBlock() {
	type In = *types.QueryTraceBlockRequest
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
				req = &types.QueryTraceBlockRequest{
					Txs: []*types.MsgEthereumTx{
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
				txMsg, _ := evmtest.ExecuteERC20Transfer(deps, s.T())
				req = &types.QueryTraceBlockRequest{
					Txs: []*types.MsgEthereumTx{
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
			req, wantResp := tc.scenario(&deps)
			goCtx := sdk.WrapSDKContext(deps.Ctx)
			gotResp, err := deps.K.TraceBlock(goCtx, req)
			if tc.wantErr != "" {
				s.Require().ErrorContains(err, tc.wantErr)
				return
			}
			s.Assert().NoError(err)
			s.Assert().NotNil(gotResp)
			s.Assert().NotNil(gotResp.Data)

			// Replace spaces in want resp
			re := regexp.MustCompile(`[\s\n\r]+`)
			wantResp = re.ReplaceAllString(wantResp, "")
			actualResp := string(gotResp.Data)
			if len(actualResp) > 1000 {
				actualResp = actualResp[:len(wantResp)]
			}
			s.Assert().Equal(wantResp, actualResp)
		})
	}
}
