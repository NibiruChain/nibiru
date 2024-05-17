package keeper_test

import (
	"context"
	"fmt"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	gethcommon "github.com/ethereum/go-ethereum/common"

	"github.com/NibiruChain/collections"

	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/app/codec"
	"github.com/NibiruChain/nibiru/eth"
	"github.com/NibiruChain/nibiru/eth/crypto/ethsecp256k1"
	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/common/testutil/testapp"
	"github.com/NibiruChain/nibiru/x/evm"
	"github.com/NibiruChain/nibiru/x/evm/evmtest"
	"github.com/NibiruChain/nibiru/x/evm/keeper"
	"github.com/NibiruChain/nibiru/x/evm/statedb"
)

type TestDeps struct {
	chain    *app.NibiruApp
	ctx      sdk.Context
	encCfg   codec.EncodingConfig
	k        keeper.Keeper
	genState *evm.GenesisState
	sender   Sender
}

func (deps TestDeps) GoCtx() context.Context {
	return sdk.WrapSDKContext(deps.ctx)
}

type Sender struct {
	EthAddr    gethcommon.Address
	PrivKey    *ethsecp256k1.PrivKey
	NibiruAddr sdk.AccAddress
}

func (s *KeeperSuite) SetupTest() TestDeps {
	testapp.EnsureNibiruPrefix()
	encCfg := app.MakeEncodingConfig()
	evm.RegisterInterfaces(encCfg.InterfaceRegistry)
	eth.RegisterInterfaces(encCfg.InterfaceRegistry)
	chain, ctx := testapp.NewNibiruTestAppAndContext()

	ethAddr, privKey, nibiruAddr := evmtest.NewEthAddrNibiruPair()
	return TestDeps{
		chain:    chain,
		ctx:      ctx,
		encCfg:   encCfg,
		k:        chain.EvmKeeper,
		genState: evm.DefaultGenesisState(),
		sender: Sender{
			EthAddr:    ethAddr,
			PrivKey:    privKey,
			NibiruAddr: nibiruAddr,
		},
	}
}

func (s *KeeperSuite) StateDB(deps *TestDeps) *statedb.StateDB {
	return statedb.New(deps.ctx, &deps.chain.EvmKeeper,
		statedb.NewEmptyTxConfig(
			gethcommon.BytesToHash(deps.ctx.HeaderHash().Bytes()),
		),
	)
}

func InvalidEthAddr() string { return "0x0000" }

type TestCase[In, Out any] struct {
	name string
	// setup: Optional setup function to create the scenario
	setup    func(deps *TestDeps)
	scenario func(deps *TestDeps) (
		req In,
		wantResp Out,
	)
	wantErr string
}

func (s *KeeperSuite) TestQueryNibiruAccount() {
	type In = *evm.QueryNibiruAccountRequest
	type Out = *evm.QueryNibiruAccountResponse
	testCases := []TestCase[In, Out]{
		{
			name: "sad: msg validation",
			scenario: func(deps *TestDeps) (req In, wantResp Out) {
				req = &evm.QueryNibiruAccountRequest{
					Address: InvalidEthAddr(),
				}
				wantResp = &evm.QueryNibiruAccountResponse{
					Address: sdk.AccAddress(gethcommon.Address{}.Bytes()).String(),
				}
				return req, wantResp
			},
			wantErr: "not a valid ethereum hex addr",
		},
		{
			name: "happy",
			scenario: func(deps *TestDeps) (req In, wantResp Out) {
				ethAddr, _, nibiruAddr := evmtest.NewEthAddrNibiruPair()
				req = &evm.QueryNibiruAccountRequest{
					Address: ethAddr.String(),
				}
				wantResp = &evm.QueryNibiruAccountResponse{
					Address:       nibiruAddr.String(),
					Sequence:      0,
					AccountNumber: 0,
				}
				return req, wantResp
			},
			wantErr: "",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			deps := s.SetupTest()
			req, wantResp := tc.scenario(&deps)
			goCtx := sdk.WrapSDKContext(deps.ctx)
			gotResp, err := deps.k.NibiruAccount(goCtx, req)
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
	type In = *evm.QueryEthAccountRequest
	type Out = *evm.QueryEthAccountResponse
	testCases := []TestCase[In, Out]{
		{
			name: "sad: msg validation",
			scenario: func(deps *TestDeps) (req In, wantResp Out) {
				req = &evm.QueryEthAccountRequest{
					Address: InvalidEthAddr(),
				}
				wantResp = &evm.QueryEthAccountResponse{
					Balance:  "0",
					CodeHash: gethcommon.BytesToHash(evm.EmptyCodeHash).Hex(),
					Nonce:    0,
				}
				return req, wantResp
			},
			wantErr: "not a valid ethereum hex addr",
		},
		{
			name: "happy: fund account + query",
			setup: func(deps *TestDeps) {
				chain := deps.chain
				ethAddr := deps.sender.EthAddr

				// fund account with 420 tokens
				coins := sdk.Coins{sdk.NewInt64Coin(evm.DefaultEVMDenom, 420)}
				err := chain.BankKeeper.MintCoins(deps.ctx, evm.ModuleName, coins)
				s.NoError(err)
				err = chain.BankKeeper.SendCoinsFromModuleToAccount(
					deps.ctx, evm.ModuleName, ethAddr.Bytes(), coins)
				s.Require().NoError(err)
			},
			scenario: func(deps *TestDeps) (req In, wantResp Out) {
				req = &evm.QueryEthAccountRequest{
					Address: deps.sender.EthAddr.Hex(),
				}
				wantResp = &evm.QueryEthAccountResponse{
					Balance:  "420",
					CodeHash: gethcommon.BytesToHash(evm.EmptyCodeHash).Hex(),
					Nonce:    0,
				}
				return req, wantResp
			},
			wantErr: "",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			deps := s.SetupTest()
			if tc.setup != nil {
				tc.setup(&deps)
			}
			req, wantResp := tc.scenario(&deps)
			goCtx := sdk.WrapSDKContext(deps.ctx)
			gotResp, err := deps.k.EthAccount(goCtx, req)
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
	type In = *evm.QueryValidatorAccountRequest
	type Out = *evm.QueryValidatorAccountResponse
	testCases := []TestCase[In, Out]{
		{
			name: "sad: msg validation",
			scenario: func(deps *TestDeps) (req In, wantResp Out) {
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
			name:  "happy: default values",
			setup: func(deps *TestDeps) {},
			scenario: func(deps *TestDeps) (req In, wantResp Out) {
				valopers := deps.chain.StakingKeeper.GetValidators(deps.ctx, 1)
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
			setup: func(deps *TestDeps) {},
			scenario: func(deps *TestDeps) (req In, wantResp Out) {
				valopers := deps.chain.StakingKeeper.GetValidators(deps.ctx, 1)
				valAddrBz := valopers[0].GetOperator().Bytes()
				consAddr := sdk.ConsAddress(valAddrBz)

				s.T().Log(
					"Send coins to validator to register in the account keeper.")
				coinsToSend := sdk.NewCoins(sdk.NewCoin(eth.EthBaseDenom, math.NewInt(69420)))
				valAddr := sdk.AccAddress(valAddrBz)
				s.NoError(testapp.FundAccount(
					deps.chain.BankKeeper,
					deps.ctx, valAddr,
					coinsToSend,
				))

				req = &evm.QueryValidatorAccountRequest{
					ConsAddress: consAddr.String(),
				}

				ak := deps.chain.AccountKeeper
				acc := ak.GetAccount(deps.ctx, valAddr)
				s.NoError(acc.SetAccountNumber(420), "acc: ", acc.String())
				s.NoError(acc.SetSequence(69), "acc: ", acc.String())
				ak.SetAccount(deps.ctx, acc)

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
			deps := s.SetupTest()
			if tc.setup != nil {
				tc.setup(&deps)
			}
			req, wantResp := tc.scenario(&deps)
			goCtx := sdk.WrapSDKContext(deps.ctx)
			gotResp, err := deps.k.ValidatorAccount(goCtx, req)
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
	type In = *evm.QueryStorageRequest
	type Out = *evm.QueryStorageResponse
	testCases := []TestCase[In, Out]{
		{
			name: "sad: msg validation",
			scenario: func(deps *TestDeps) (req In, wantResp Out) {
				req = &evm.QueryStorageRequest{
					Address: InvalidEthAddr(),
				}
				return req, wantResp
			},
			wantErr: "InvalidArgument",
		},
		{
			name: "happy",
			scenario: func(deps *TestDeps) (req In, wantResp Out) {
				addr := evmtest.NewEthAddr()
				storageKey := gethcommon.BytesToHash([]byte("storagekey"))
				req = &evm.QueryStorageRequest{
					Address: addr.Hex(),
					Key:     storageKey.String(),
				}

				stateDB := s.StateDB(deps)
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
			scenario: func(deps *TestDeps) (req In, wantResp Out) {
				addr := evmtest.NewEthAddr()
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
			deps := s.SetupTest()
			if tc.setup != nil {
				tc.setup(&deps)
			}
			req, wantResp := tc.scenario(&deps)
			goCtx := sdk.WrapSDKContext(deps.ctx)

			gotResp, err := deps.k.Storage(goCtx, req)
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
	type In = *evm.QueryCodeRequest
	type Out = *evm.QueryCodeResponse
	testCases := []TestCase[In, Out]{
		{
			name: "sad: msg validation",
			scenario: func(deps *TestDeps) (req In, wantResp Out) {
				req = &evm.QueryCodeRequest{
					Address: InvalidEthAddr(),
				}
				return req, wantResp
			},
			wantErr: "InvalidArgument",
		},
		{
			name: "happy",
			scenario: func(deps *TestDeps) (req In, wantResp Out) {
				addr := evmtest.NewEthAddr()
				req = &evm.QueryCodeRequest{
					Address: addr.Hex(),
				}

				stateDB := s.StateDB(deps)
				contractBytecode := []byte("bytecode")
				stateDB.SetCode(addr, contractBytecode)
				s.Require().NoError(stateDB.Commit())

				s.NotNil(stateDB.Keeper().GetAccount(deps.ctx, addr))
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
			deps := s.SetupTest()
			if tc.setup != nil {
				tc.setup(&deps)
			}
			req, wantResp := tc.scenario(&deps)
			goCtx := sdk.WrapSDKContext(deps.ctx)

			gotResp, err := deps.k.Code(goCtx, req)
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

// AssertModuleParamsEqual errors if the fields don't match. This function avoids
// failing the "EqualValues" check due to comparisons between nil and empty
// slices: `[]string(nil)` and `[]string{}`.
func AssertModuleParamsEqual(want, got evm.Params) error {
	errs := []error{}
	{
		want, got := want.EvmDenom, got.EvmDenom
		if want != got {
			errs = append(errs, ErrModuleParamsEquality(
				"evm_denom", want, got))
		}
	}
	{
		want, got := want.EnableCreate, got.EnableCreate
		if want != got {
			errs = append(errs, ErrModuleParamsEquality(
				"enable_create", want, got))
		}
	}
	{
		want, got := want.EnableCall, got.EnableCall
		if want != got {
			errs = append(errs, ErrModuleParamsEquality(
				"enable_call", want, got))
		}
	}
	{
		want, got := want.ChainConfig, got.ChainConfig
		if want != got {
			errs = append(errs, ErrModuleParamsEquality(
				"chain_config", want, got))
		}
	}
	return common.CombineErrors(errs...)
}

func ErrModuleParamsEquality(field string, want, got any) error {
	return fmt.Errorf(`failed AssetModuleParamsEqual on field %s: want "%v", got "%v"`, field, want, got)
}

func (s *KeeperSuite) TestQueryParams() {
	deps := s.SetupTest()
	want := evm.DefaultParams()
	deps.k.SetParams(deps.ctx, want)
	gotResp, err := deps.k.Params(deps.GoCtx(), nil)
	got := gotResp.Params
	s.Require().NoError(err)

	// Note that protobuf equals is more reliable than `s.Equal`
	s.Require().True(want.Equal(got), "want %s, got %s", want, got)

	// Empty params to test the setter
	want.ActivePrecompiles = []string{"new", "something"}
	deps.k.SetParams(deps.ctx, want)
	gotResp, err = deps.k.Params(deps.GoCtx(), nil)
	s.Require().NoError(err)
	got = gotResp.Params

	// Note that protobuf equals is more reliable than `s.Equal`
	s.Require().True(want.Equal(got), "want %s, got %s", want, got)
}
