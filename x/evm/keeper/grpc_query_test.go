package keeper_test

import (
	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/app/codec"
	"github.com/NibiruChain/nibiru/eth"
	"github.com/NibiruChain/nibiru/eth/crypto/ethsecp256k1"
	"github.com/NibiruChain/nibiru/x/common/testutil/testapp"
	"github.com/NibiruChain/nibiru/x/evm"
	"github.com/NibiruChain/nibiru/x/evm/evmtest"
	"github.com/NibiruChain/nibiru/x/evm/keeper"
	sdk "github.com/cosmos/cosmos-sdk/types"
	gethcommon "github.com/ethereum/go-ethereum/common"
)

type TestDeps struct {
	chain    *app.NibiruApp
	ctx      sdk.Context
	encCfg   codec.EncodingConfig
	k        keeper.Keeper
	genState *evm.GenesisState
	sender Sender
}

type Sender struct {
	EthAddr gethcommon.Address
	PrivKey *ethsecp256k1.PrivKey
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

func InvalidEthAddr() string { return "0x0000" }

func (s *KeeperSuite) TestQueryNibiruAccount() {
	type In = *evm.QueryNibiruAccountRequest
	type Out = *evm.QueryNibiruAccountResponse
	testCases := []struct {
		name     string
		scenario func() (
			req In,
			wantResp Out,
		)
		wantErr string
	}{
		{
			name: "sad: msg validation",
			scenario: func() (req In, wantResp Out) {
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
			scenario: func() (req In, wantResp Out) {
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
			req, wantResp := tc.scenario()
			deps := s.SetupTest()
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
	testCases := []struct {
		name     string
		scenario func(deps *TestDeps) (
			req In,
			wantResp Out,
		)
		// Optional setup function to create the scenario
		setup func(deps *TestDeps)
		wantErr string
	}{
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
			wantErr: "",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			deps := s.SetupTest()
			req, wantResp := tc.scenario(&deps)
			if tc.setup != nil {
				tc.setup(&deps)
			}
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
