package keeper_test

import (
	"crypto/sha256"
	"fmt"

	"cosmossdk.io/math"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"

	_ "embed"

	"github.com/cosmos/cosmos-sdk/testutil/testdata"
	sdk "github.com/cosmos/cosmos-sdk/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	"github.com/NibiruChain/nibiru/v2/x/devgas/v1/types"
)

//go:embed testdata/reflect.wasm
var wasmContract []byte

func (s *KeeperTestSuite) StoreCode() {
	_, _, sender := testdata.KeyTestPubAddr()
	msg := wasmtypes.MsgStoreCodeFixture(func(m *wasmtypes.MsgStoreCode) {
		m.WASMByteCode = wasmContract
		m.Sender = sender.String()
	})
	rsp, err := s.app.MsgServiceRouter().Handler(msg)(s.ctx, msg)
	s.Require().NoError(err)
	var result wasmtypes.MsgStoreCodeResponse
	s.Require().NoError(s.app.AppCodec().Unmarshal(rsp.Data, &result))
	s.Require().Equal(uint64(1), result.CodeID)
	expHash := sha256.Sum256(wasmContract)
	s.Require().Equal(expHash[:], result.Checksum)
	// and
	info := s.app.WasmKeeper.GetCodeInfo(s.ctx, 1)
	s.Require().NotNil(info)
	s.Require().Equal(expHash[:], info.CodeHash)
	s.Require().Equal(sender.String(), info.Creator)
	s.Require().Equal(wasmtypes.DefaultParams().InstantiateDefaultPermission.With(sender), info.InstantiateConfig)
}

func (s *KeeperTestSuite) InstantiateContract(sender string, admin string) string {
	msgStoreCode := wasmtypes.MsgStoreCodeFixture(func(m *wasmtypes.MsgStoreCode) {
		m.WASMByteCode = wasmContract
		m.Sender = sender
	})
	_, err := s.app.MsgServiceRouter().Handler(msgStoreCode)(s.ctx, msgStoreCode)
	s.Require().NoError(err)

	msgInstantiate := wasmtypes.MsgInstantiateContractFixture(func(m *wasmtypes.MsgInstantiateContract) {
		m.Sender = sender
		m.Admin = admin
		m.Msg = []byte(`{}`)
	})
	resp, err := s.app.MsgServiceRouter().Handler(msgInstantiate)(s.ctx, msgInstantiate)
	s.Require().NoError(err)
	var result wasmtypes.MsgInstantiateContractResponse
	s.Require().NoError(s.app.AppCodec().Unmarshal(resp.Data, &result))
	contractInfo := s.app.WasmKeeper.GetContractInfo(s.ctx, sdk.MustAccAddressFromBech32(result.Address))
	s.Require().Equal(contractInfo.CodeID, uint64(1))
	s.Require().Equal(contractInfo.Admin, admin)
	s.Require().Equal(contractInfo.Creator, sender)

	return result.Address
}

func (s *KeeperTestSuite) TestGetContractAdminOrCreatorAddress() {
	_, _, sender := testdata.KeyTestPubAddr()
	_, _, admin := testdata.KeyTestPubAddr()
	_ = s.FundAccount(s.ctx, sender, sdk.NewCoins(sdk.NewCoin("stake", math.NewInt(1_000_000))))
	_ = s.FundAccount(s.ctx, admin, sdk.NewCoins(sdk.NewCoin("stake", math.NewInt(1_000_000))))

	noAdminContractAddress := s.InstantiateContract(sender.String(), "")
	withAdminContractAddress := s.InstantiateContract(sender.String(), admin.String())

	for _, tc := range []struct {
		desc            string
		contractAddress string
		deployerAddress string
		shouldErr       bool
	}{
		{
			desc:            "Success - Creator",
			contractAddress: noAdminContractAddress,
			deployerAddress: sender.String(),
			shouldErr:       false,
		},
		{
			desc:            "Success - Admin",
			contractAddress: withAdminContractAddress,
			deployerAddress: admin.String(),
			shouldErr:       false,
		},
		{
			desc:            "Error - Invalid deployer",
			contractAddress: noAdminContractAddress,
			deployerAddress: "Invalid",
			shouldErr:       true,
		},
	} {
		tc := tc
		s.Run(tc.desc, func() {
			if !tc.shouldErr {
				_, err := s.app.DevGasKeeper.GetContractAdminOrCreatorAddress(
					s.ctx, sdk.MustAccAddressFromBech32(tc.contractAddress), tc.deployerAddress,
				)
				s.NoError(err)
				return
			}
			_, err := s.app.DevGasKeeper.GetContractAdminOrCreatorAddress(
				s.ctx, sdk.MustAccAddressFromBech32(tc.contractAddress), tc.deployerAddress,
			)
			s.Error(err)
		})
	}
}

func (s *KeeperTestSuite) TestRegisterFeeShare() {
	_, _, sender := testdata.KeyTestPubAddr()
	_ = s.FundAccount(s.ctx, sender, sdk.NewCoins(sdk.NewCoin("stake", math.NewInt(1_000_000))))

	gov := s.app.AccountKeeper.GetModuleAddress(govtypes.ModuleName).String()
	govContract := s.InstantiateContract(sender.String(), gov)

	contractAddress := s.InstantiateContract(sender.String(), "")
	contractAddress2 := s.InstantiateContract(contractAddress, contractAddress)

	deployer := s.InstantiateContract(sender.String(), "")
	subContract := s.InstantiateContract(deployer, deployer)

	_, _, withdrawer := testdata.KeyTestPubAddr()

	for _, tc := range []struct {
		desc      string
		msg       *types.MsgRegisterFeeShare
		resp      *types.MsgRegisterFeeShareResponse
		shouldErr bool
	}{
		{
			desc: "Invalid contract address",
			msg: &types.MsgRegisterFeeShare{
				ContractAddress:   "Invalid",
				DeployerAddress:   sender.String(),
				WithdrawerAddress: withdrawer.String(),
			},
			resp:      &types.MsgRegisterFeeShareResponse{},
			shouldErr: true,
		},
		{
			desc: "Invalid deployer address",
			msg: &types.MsgRegisterFeeShare{
				ContractAddress:   contractAddress,
				DeployerAddress:   "Invalid",
				WithdrawerAddress: withdrawer.String(),
			},
			resp:      &types.MsgRegisterFeeShareResponse{},
			shouldErr: true,
		},
		{
			desc: "Invalid withdrawer address",
			msg: &types.MsgRegisterFeeShare{
				ContractAddress:   contractAddress,
				DeployerAddress:   sender.String(),
				WithdrawerAddress: "Invalid",
			},
			resp:      &types.MsgRegisterFeeShareResponse{},
			shouldErr: true,
		},
		{
			desc: "Success",
			msg: &types.MsgRegisterFeeShare{
				ContractAddress:   contractAddress,
				DeployerAddress:   sender.String(),
				WithdrawerAddress: withdrawer.String(),
			},
			resp:      &types.MsgRegisterFeeShareResponse{},
			shouldErr: false,
		},
		{
			desc: "Invalid withdraw address for factory contract",
			msg: &types.MsgRegisterFeeShare{
				ContractAddress:   contractAddress2,
				DeployerAddress:   sender.String(),
				WithdrawerAddress: sender.String(),
			},
			resp:      &types.MsgRegisterFeeShareResponse{},
			shouldErr: true,
		},
		{
			desc: "Success register factory contract to itself",
			msg: &types.MsgRegisterFeeShare{
				ContractAddress:   contractAddress2,
				DeployerAddress:   sender.String(),
				WithdrawerAddress: contractAddress2,
			},
			resp:      &types.MsgRegisterFeeShareResponse{},
			shouldErr: false,
		},
		{
			desc: "Invalid register gov contract withdraw address",
			msg: &types.MsgRegisterFeeShare{
				ContractAddress:   govContract,
				DeployerAddress:   sender.String(),
				WithdrawerAddress: sender.String(),
			},
			resp:      &types.MsgRegisterFeeShareResponse{},
			shouldErr: true,
		},
		{
			desc: "Success register gov contract withdraw address to self",
			msg: &types.MsgRegisterFeeShare{
				ContractAddress:   govContract,
				DeployerAddress:   sender.String(),
				WithdrawerAddress: govContract,
			},
			resp:      &types.MsgRegisterFeeShareResponse{},
			shouldErr: false,
		},
		{
			desc: "Success register contract from contract as admin",
			msg: &types.MsgRegisterFeeShare{
				ContractAddress:   subContract,
				DeployerAddress:   deployer,
				WithdrawerAddress: deployer,
			},
			resp:      &types.MsgRegisterFeeShareResponse{},
			shouldErr: false,
		},
	} {
		tc := tc
		s.Run(tc.desc, func() {
			goCtx := sdk.WrapSDKContext(s.ctx)
			if !tc.shouldErr {
				resp, err := s.devgasMsgServer.RegisterFeeShare(goCtx, tc.msg)
				s.Require().NoError(err)
				s.Require().Equal(resp, tc.resp)
			} else {
				resp, err := s.devgasMsgServer.RegisterFeeShare(goCtx, tc.msg)
				s.Require().Error(err)
				s.Require().Nil(resp)
			}
		})
	}
}

func (s *KeeperTestSuite) TestUpdateFeeShare() {
	_, _, sender := testdata.KeyTestPubAddr()
	_ = s.FundAccount(s.ctx, sender, sdk.NewCoins(sdk.NewCoin("stake", math.NewInt(1_000_000))))

	fmt.Printf("s.ctx: %v\n", s.ctx)
	fmt.Printf("s.ctx.BlockTime(): %v\n", s.ctx.BlockTime())
	fmt.Printf("s.ctx.BlockHeight(): %v\n", s.ctx.BlockHeight())
	contractAddress := s.InstantiateContract(sender.String(), "")
	_, _, withdrawer := testdata.KeyTestPubAddr()

	contractAddressNoRegisFeeShare := s.InstantiateContract(sender.String(), "")
	s.Require().NotEqual(contractAddress, contractAddressNoRegisFeeShare)

	// RegsisFeeShare
	goCtx := sdk.WrapSDKContext(s.ctx)
	msg := &types.MsgRegisterFeeShare{
		ContractAddress:   contractAddress,
		DeployerAddress:   sender.String(),
		WithdrawerAddress: withdrawer.String(),
	}
	_, err := s.devgasMsgServer.RegisterFeeShare(goCtx, msg)
	s.Require().NoError(err)
	_, _, newWithdrawer := testdata.KeyTestPubAddr()
	s.Require().NotEqual(withdrawer, newWithdrawer)

	for _, tc := range []struct {
		desc      string
		msg       *types.MsgUpdateFeeShare
		resp      *types.MsgCancelFeeShareResponse
		shouldErr bool
	}{
		{
			desc: "Invalid - contract not regis",
			msg: &types.MsgUpdateFeeShare{
				ContractAddress:   contractAddressNoRegisFeeShare,
				DeployerAddress:   sender.String(),
				WithdrawerAddress: newWithdrawer.String(),
			},
			resp:      nil,
			shouldErr: true,
		},
		{
			desc: "Invalid - Invalid DeployerAddress",
			msg: &types.MsgUpdateFeeShare{
				ContractAddress:   contractAddress,
				DeployerAddress:   "Invalid",
				WithdrawerAddress: newWithdrawer.String(),
			},
			resp:      nil,
			shouldErr: true,
		},
		{
			desc: "Invalid - Invalid WithdrawerAddress",
			msg: &types.MsgUpdateFeeShare{
				ContractAddress:   contractAddress,
				DeployerAddress:   sender.String(),
				WithdrawerAddress: "Invalid",
			},
			resp:      nil,
			shouldErr: true,
		},
		{
			desc: "Invalid - Invalid WithdrawerAddress not change",
			msg: &types.MsgUpdateFeeShare{
				ContractAddress:   contractAddress,
				DeployerAddress:   sender.String(),
				WithdrawerAddress: withdrawer.String(),
			},
			resp:      nil,
			shouldErr: true,
		},
		{
			desc: "Success",
			msg: &types.MsgUpdateFeeShare{
				ContractAddress:   contractAddress,
				DeployerAddress:   sender.String(),
				WithdrawerAddress: newWithdrawer.String(),
			},
			resp:      &types.MsgCancelFeeShareResponse{},
			shouldErr: false,
		},
	} {
		tc := tc
		s.Run(tc.desc, func() {
			goCtx := sdk.WrapSDKContext(s.ctx)
			if !tc.shouldErr {
				_, err := s.devgasMsgServer.UpdateFeeShare(goCtx, tc.msg)
				s.Require().NoError(err)
			} else {
				resp, err := s.devgasMsgServer.UpdateFeeShare(goCtx, tc.msg)
				s.Require().Error(err)
				s.Require().Nil(resp)
			}
		})
	}
}

func (s *KeeperTestSuite) TestCancelFeeShare() {
	_, _, sender := testdata.KeyTestPubAddr()
	_ = s.FundAccount(s.ctx, sender, sdk.NewCoins(sdk.NewCoin("stake", math.NewInt(1_000_000))))

	contractAddress := s.InstantiateContract(sender.String(), "")
	_, _, withdrawer := testdata.KeyTestPubAddr()

	// RegsisFeeShare
	goCtx := sdk.WrapSDKContext(s.ctx)
	msg := &types.MsgRegisterFeeShare{
		ContractAddress:   contractAddress,
		DeployerAddress:   sender.String(),
		WithdrawerAddress: withdrawer.String(),
	}
	_, err := s.devgasMsgServer.RegisterFeeShare(goCtx, msg)
	s.Require().NoError(err)

	for _, tc := range []struct {
		desc      string
		msg       *types.MsgCancelFeeShare
		resp      *types.MsgCancelFeeShareResponse
		shouldErr bool
	}{
		{
			desc: "Invalid - contract Address",
			msg: &types.MsgCancelFeeShare{
				ContractAddress: "Invalid",
				DeployerAddress: sender.String(),
			},
			resp:      nil,
			shouldErr: true,
		},
		{
			desc: "Invalid - deployer Address",
			msg: &types.MsgCancelFeeShare{
				ContractAddress: contractAddress,
				DeployerAddress: "Invalid",
			},
			resp:      nil,
			shouldErr: true,
		},
		{
			desc: "Success",
			msg: &types.MsgCancelFeeShare{
				ContractAddress: contractAddress,
				DeployerAddress: sender.String(),
			},
			resp:      &types.MsgCancelFeeShareResponse{},
			shouldErr: false,
		},
	} {
		tc := tc
		s.Run(tc.desc, func() {
			goCtx := sdk.WrapSDKContext(s.ctx)
			if !tc.shouldErr {
				resp, err := s.devgasMsgServer.CancelFeeShare(goCtx, tc.msg)
				s.Require().NoError(err)
				s.Require().Equal(resp, tc.resp)
			} else {
				resp, err := s.devgasMsgServer.CancelFeeShare(goCtx, tc.msg)
				s.Require().Error(err)
				s.Require().Equal(resp, tc.resp)
			}
		})
	}
}

func (s *KeeperTestSuite) TestUpdateParams() {
	govModuleAddr := authtypes.NewModuleAddress(govtypes.ModuleName).String()
	goCtx := sdk.WrapSDKContext(s.ctx)

	for _, tc := range []struct {
		desc      string
		msg       *types.MsgUpdateParams
		shouldErr bool
	}{
		{
			desc: "happy",
			msg: &types.MsgUpdateParams{
				Authority: govModuleAddr,
				Params:    types.DefaultParams().Sanitize(),
			},
			shouldErr: false,
		},
		{
			desc: "err - empty params",
			msg: &types.MsgUpdateParams{
				Authority: govModuleAddr,
				Params:    types.ModuleParams{},
			},
			shouldErr: true,
		},
		{
			desc: "err - nil params",
			msg: &types.MsgUpdateParams{
				Authority: govModuleAddr,
			},
			shouldErr: true,
		},
		{
			desc: "err - nil params",
			msg: &types.MsgUpdateParams{
				Authority: "not-an-authority",
				Params:    types.DefaultParams(),
			},
			shouldErr: true,
		},
	} {
		tc := tc
		s.Run(tc.desc, func() {
			if !tc.shouldErr {
				_, err := s.devgasMsgServer.UpdateParams(goCtx, tc.msg)
				s.NoError(err)
			} else {
				_, err := s.devgasMsgServer.UpdateParams(goCtx, tc.msg)
				s.Error(err)
			}
		})
	}
}
