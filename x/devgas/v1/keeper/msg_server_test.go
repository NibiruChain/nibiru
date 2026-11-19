package keeper_test

import (
	_ "embed"

	sdkmath "cosmossdk.io/math"

	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/testutil/testdata"
	sdk "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types"
	authtypes "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/auth/types"
	govtypes "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/gov/types"

	wasmtypes "github.com/NibiruChain/nibiru/v2/x/wasm/types"

	"github.com/NibiruChain/nibiru/v2/x/devgas/v1/types"
)

//go:embed testdata/reflect.wasm
var wasmContract []byte

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

// TestMsgsDeprecated checks that every x/devgas transaction message fails
// with [types.ErrDevgasDeprecated] and leaves the module state unchanged,
// including messages that were valid before the shutdown.
func (s *KeeperTestSuite) TestMsgsDeprecated() {
	s.SetupTest()
	_, _, sender := testdata.KeyTestPubAddr()
	s.Require().NoError(s.FundAccount(
		s.ctx, sender, sdk.NewCoins(sdk.NewCoin("stake", sdkmath.NewInt(1_000_000))),
	))
	_, _, withdrawer := testdata.KeyTestPubAddr()
	_, _, newWithdrawer := testdata.KeyTestPubAddr()

	registered := s.InstantiateContract(sender.String(), "")
	unregistered := s.InstantiateContract(sender.String(), "")
	feeShare := types.FeeShare{
		ContractAddress:   registered,
		DeployerAddress:   sender.String(),
		WithdrawerAddress: withdrawer.String(),
	}
	s.app.DevGasKeeper.SetFeeShare(s.ctx, feeShare)
	paramsBefore := s.app.DevGasKeeper.GetParams(s.ctx)
	s.Require().True(paramsBefore.EnableFeeShare)

	goCtx := sdk.WrapSDKContext(s.ctx)
	for _, tc := range []struct {
		desc string
		call func() error
	}{
		{
			desc: "register by the contract creator",
			call: func() error {
				_, err := s.devgasMsgServer.RegisterFeeShare(goCtx, &types.MsgRegisterFeeShare{
					ContractAddress:   unregistered,
					DeployerAddress:   sender.String(),
					WithdrawerAddress: withdrawer.String(),
				})
				return err
			},
		},
		{
			desc: "update a registered fee share",
			call: func() error {
				_, err := s.devgasMsgServer.UpdateFeeShare(goCtx, &types.MsgUpdateFeeShare{
					ContractAddress:   registered,
					DeployerAddress:   sender.String(),
					WithdrawerAddress: newWithdrawer.String(),
				})
				return err
			},
		},
		{
			desc: "cancel a registered fee share",
			call: func() error {
				_, err := s.devgasMsgServer.CancelFeeShare(goCtx, &types.MsgCancelFeeShare{
					ContractAddress: registered,
					DeployerAddress: sender.String(),
				})
				return err
			},
		},
		{
			desc: "update params from the gov module",
			call: func() error {
				params := types.DefaultParams()
				params.EnableFeeShare = false
				_, err := s.devgasMsgServer.UpdateParams(goCtx, &types.MsgUpdateParams{
					Authority: authtypes.NewModuleAddress(govtypes.ModuleName).String(),
					Params:    params,
				})
				return err
			},
		},
	} {
		s.Run(tc.desc, func() {
			s.Require().ErrorIs(tc.call(), types.ErrDevgasDeprecated)
		})
	}

	s.T().Log("state is unchanged")
	got, found := s.app.DevGasKeeper.GetFeeShare(
		s.ctx, sdk.MustAccAddressFromBech32(registered),
	)
	s.Require().True(found)
	s.Require().Equal(feeShare, got)
	s.Require().False(s.app.DevGasKeeper.IsFeeShareRegistered(
		s.ctx, sdk.MustAccAddressFromBech32(unregistered),
	))
	s.Require().Equal(paramsBefore, s.app.DevGasKeeper.GetParams(s.ctx))
}
