package keeper_test

import (
	"testing"

	"github.com/NibiruChain/collections"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/x/common/testutil"
	oracletypes "github.com/NibiruChain/nibiru/x/oracle/types"
	"github.com/NibiruChain/nibiru/x/tokenfactory/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
)

func (s *TestSuite) TestCreateDenom() {
	_, addrs := testutil.PrivKeyAddressPairs(4)

	testCases := []struct {
		name     string
		txMsg    *types.MsgCreateDenom
		wantErr  string
		preHook  func(ctx sdk.Context, bapp *app.NibiruApp)
		postHook func(ctx sdk.Context, bapp *app.NibiruApp)
	}{
		{
			name:    "happy path",
			txMsg:   &types.MsgCreateDenom{Sender: addrs[0].String(), Subdenom: "nusd"},
			wantErr: "",
			preHook: func(ctx sdk.Context, bapp *app.NibiruApp) {
				allDenoms := bapp.TokenFactoryKeeper.Store.Denoms.
					Iterate(ctx, collections.Range[string]{}).Values()
				s.Len(allDenoms, 0)
			},
			postHook: func(ctx sdk.Context, bapp *app.NibiruApp) {
				allDenoms := bapp.TokenFactoryKeeper.Store.Denoms.
					Iterate(ctx, collections.Range[string]{}).Values()
				s.Len(allDenoms, 1)
				s.Equal(allDenoms[0].Subdenom, "nusd")
			},
		},
		{
			name:    "creating the same denom a second time should fail",
			txMsg:   &types.MsgCreateDenom{Sender: addrs[0].String(), Subdenom: "nusd"},
			wantErr: "attempting to create denom that is already registered",
			preHook: func(ctx sdk.Context, bapp *app.NibiruApp) {
				allDenoms := bapp.TokenFactoryKeeper.Store.Denoms.
					Iterate(ctx, collections.Range[string]{}).Values()
				s.Len(allDenoms, 0)
				_, err := bapp.TokenFactoryKeeper.CreateDenom(
					sdk.WrapSDKContext(s.ctx), &types.MsgCreateDenom{
						Sender:   addrs[0].String(),
						Subdenom: "nusd",
					},
				)
				s.NoError(err)
			},
			postHook: func(ctx sdk.Context, bapp *app.NibiruApp) {},
		},

		{
			name:    "sad: nil msg",
			txMsg:   nil,
			wantErr: "nil msg",
		},

		{
			name:    "sad: sender",
			txMsg:   &types.MsgCreateDenom{Sender: "sender", Subdenom: "nusd"},
			wantErr: "invalid creator",
		},

		{
			name:    "sad: denom",
			txMsg:   &types.MsgCreateDenom{Sender: addrs[0].String(), Subdenom: ""},
			wantErr: "denom format error",
		},
	}

	for _, tc := range testCases {
		s.T().Run(tc.name, func(t *testing.T) {
			s.SetupTest()
			if tc.preHook != nil {
				tc.preHook(s.ctx, s.app)
			}

			resp, err := s.app.TokenFactoryKeeper.CreateDenom(
				sdk.WrapSDKContext(s.ctx), tc.txMsg,
			)

			if tc.wantErr != "" {
				s.Error(err)
				s.ErrorContains(err, tc.wantErr)
				return
			}

			s.NoError(err)
			want := types.TFDenom{
				Creator:  tc.txMsg.Sender,
				Subdenom: tc.txMsg.Subdenom,
			}.String()
			s.Equal(want, resp.NewTokenDenom)

			if tc.postHook != nil {
				tc.postHook(s.ctx, s.app)
			}
		})
	}
}

func (s *TestSuite) TestChangeAdmin() {
	sbf := testutil.AccAddress().String()

	testCases := []struct {
		name     string
		txMsg    *types.MsgChangeAdmin
		wantErr  string
		preHook  func(ctx sdk.Context, bapp *app.NibiruApp)
		postHook func(ctx sdk.Context, bapp *app.NibiruApp)
	}{
		{
			name:    "sad: nil msg",
			txMsg:   nil,
			wantErr: "nil msg",
		},

		{
			name: "sad: fail validate basic",
			txMsg: &types.MsgChangeAdmin{
				Sender: "sender", Denom: "tf/creator/nusd", NewAdmin: "new admin"},
			wantErr: "invalid sender",
		},

		{
			name: "sad: non-admin tries to change admin",
			txMsg: &types.MsgChangeAdmin{
				Sender:   testutil.AccAddress().String(),
				Denom:    types.TFDenom{Creator: sbf, Subdenom: "ftt"}.String(),
				NewAdmin: testutil.AccAddress().String(),
			},
			wantErr: "only the current admin can set a new admin",
			preHook: func(ctx sdk.Context, bapp *app.NibiruApp) {
				_, err := bapp.TokenFactoryKeeper.CreateDenom(
					sdk.WrapSDKContext(ctx), &types.MsgCreateDenom{
						Sender:   sbf,
						Subdenom: "ftt",
					},
				)
				s.NoError(err)
			},
		},

		{
			name: "happy: SBF changes FTT admin",
			txMsg: &types.MsgChangeAdmin{
				Sender:   sbf,
				Denom:    types.TFDenom{Creator: sbf, Subdenom: "ftt"}.String(),
				NewAdmin: testutil.AccAddress().String(),
			},
			wantErr: "",
			preHook: func(ctx sdk.Context, bapp *app.NibiruApp) {
				_, err := bapp.TokenFactoryKeeper.CreateDenom(
					sdk.WrapSDKContext(ctx), &types.MsgCreateDenom{
						Sender:   sbf,
						Subdenom: "ftt",
					},
				)
				s.NoError(err)
			},
		},

		{
			name: "sad: change admin for denom that doesn't exist ",
			txMsg: &types.MsgChangeAdmin{
				Sender:   sbf,
				Denom:    types.TFDenom{Creator: sbf, Subdenom: "ftt"}.String(),
				NewAdmin: testutil.AccAddress().String()},
			wantErr: collections.ErrNotFound.Error(),
		},
	}

	for _, tc := range testCases {
		s.T().Run(tc.name, func(t *testing.T) {
			s.SetupTest()
			if tc.preHook != nil {
				tc.preHook(s.ctx, s.app)
			}

			_, err := s.app.TokenFactoryKeeper.ChangeAdmin(
				sdk.WrapSDKContext(s.ctx), tc.txMsg,
			)

			if tc.wantErr != "" {
				s.Error(err)
				s.ErrorContains(err, tc.wantErr)
				return
			}

			s.T().Log("expect new admin to be set in state.")
			s.NoError(err)
			authData, err := s.app.TokenFactoryKeeper.Store.GetDenomAuthorityMetadata(
				s.ctx, tc.txMsg.Denom)
			s.NoError(err)
			s.Equal(authData.Admin, tc.txMsg.NewAdmin)

			if tc.postHook != nil {
				tc.postHook(s.ctx, s.app)
			}
		})
	}
}

func (s *TestSuite) TestUpdateModuleParams() {
	testCases := []struct {
		name    string
		txMsg   *types.MsgUpdateModuleParams
		wantErr string
	}{
		{
			name:    "sad: nil msg",
			txMsg:   nil,
			wantErr: "nil msg",
		},

		{
			name: "sad: fail validate basic",
			txMsg: &types.MsgUpdateModuleParams{
				Authority: "authority",
				Params:    types.DefaultModuleParams(),
			},
			wantErr: "invalid authority",
		},

		{
			name: "sad: must be gov proposal form x/gov module account",
			txMsg: &types.MsgUpdateModuleParams{
				Authority: testutil.AccAddress().String(),
				Params:    types.DefaultModuleParams(),
			},
			wantErr: "expected gov account as only signer for proposal message",
		},

		{
			name: "happy: new params",
			txMsg: &types.MsgUpdateModuleParams{
				Authority: testutil.GovModuleAddr().String(),
				Params: types.ModuleParams{
					DenomCreationGasConsume: 69_420,
				},
			},
			wantErr: "",
		},
	}

	for _, tc := range testCases {
		s.T().Run(tc.name, func(t *testing.T) {
			s.SetupTest()
			_, err := s.app.TokenFactoryKeeper.UpdateModuleParams(
				sdk.WrapSDKContext(s.ctx), tc.txMsg,
			)

			if tc.wantErr != "" {
				s.Error(err)
				s.ErrorContains(err, tc.wantErr)
				return
			}
			s.NoError(err)

			params, err := s.app.TokenFactoryKeeper.Store.ModuleParams.Get(s.ctx)
			s.Require().NoError(err)
			s.Equal(params, tc.txMsg.Params)
		})
	}
}

type SdkMsgTestCase struct {
	TestMsg sdk.Msg
	WantErr string
}

func (s *TestSuite) TestMintBurn() {
	_, addrs := testutil.PrivKeyAddressPairs(4)
	tfModuleAddr := authtypes.NewModuleAddress(types.ModuleName)
	tfdenom := types.TFDenom{
		Creator:  addrs[0].String(),
		Subdenom: "nusd",
	}
	nusd69420 := sdk.Coin{
		Denom:  tfdenom.String(),
		Amount: sdk.NewInt(69_420),
	}

	testCases := []struct {
		name      string
		setupMsgs []sdk.Msg
		testMsgs  []SdkMsgTestCase
		preHook   func(ctx sdk.Context, bapp *app.NibiruApp)
		postHook  func(ctx sdk.Context, bapp *app.NibiruApp)
	}{
		{
			name: "happy: mint and burn",
			setupMsgs: []sdk.Msg{
				&types.MsgCreateDenom{
					Sender:   addrs[0].String(),
					Subdenom: "nusd",
				},

				&types.MsgMint{
					Sender: addrs[0].String(),
					Coin: sdk.Coin{
						Denom: types.TFDenom{
							Creator:  addrs[0].String(),
							Subdenom: "nusd",
						}.String(),
						Amount: sdk.NewInt(69_420),
					},
					MintTo: "",
				},
			},
			testMsgs: []SdkMsgTestCase{
				{
					TestMsg: &types.MsgBurn{
						Sender: addrs[0].String(),
						Coin: sdk.Coin{
							Denom: types.TFDenom{
								Creator:  addrs[0].String(),
								Subdenom: "nusd",
							}.String(),
							Amount: sdk.NewInt(1),
						},
						BurnFrom: "",
					},
					WantErr: "",
				},
			},
			preHook: func(ctx sdk.Context, bapp *app.NibiruApp) {
				allDenoms := bapp.TokenFactoryKeeper.Store.Denoms.
					Iterate(ctx, collections.Range[string]{}).Values()
				s.Len(allDenoms, 1)
			},
			postHook: func(ctx sdk.Context, bapp *app.NibiruApp) {
				allDenoms := bapp.TokenFactoryKeeper.Store.Denoms.
					Iterate(ctx, collections.Range[string]{}).Values()

				s.T().Log("Minting changes total supply, but burning does not.")
				denom := allDenoms[0]
				s.Equal(
					sdk.NewInt(69_420), s.app.BankKeeper.GetSupply(s.ctx, denom.String()).Amount,
				)

				s.T().Log("We burned 1 token, so it should be in the module account.")
				coin := s.app.BankKeeper.GetBalance(
					s.ctx, tfModuleAddr, denom.String())
				s.Equal(
					sdk.NewInt(1),
					coin.Amount,
				)

			},
		},

		{
			name:      "sad: denom does not exist",
			setupMsgs: []sdk.Msg{},
			testMsgs: []SdkMsgTestCase{
				{
					TestMsg: &types.MsgMint{
						Sender: addrs[0].String(),
						Coin:   nusd69420,
						MintTo: "",
					},
					WantErr: collections.ErrNotFound.Error(),
				},
				{
					TestMsg: &types.MsgBurn{
						Sender:   addrs[0].String(),
						Coin:     nusd69420,
						BurnFrom: "",
					},
					WantErr: collections.ErrNotFound.Error(),
				},
			},
		},

		{
			name: "sad: sender is not admin",
			setupMsgs: []sdk.Msg{

				&types.MsgCreateDenom{
					Sender:   addrs[0].String(),
					Subdenom: "nusd",
				},

				&types.MsgMint{
					Sender: addrs[0].String(),
					Coin:   nusd69420,
					MintTo: "",
				},

				&types.MsgChangeAdmin{
					Sender:   addrs[0].String(),
					Denom:    tfdenom.String(),
					NewAdmin: addrs[1].String(),
				},
			},
			testMsgs: []SdkMsgTestCase{
				{
					TestMsg: &types.MsgMint{
						Sender: addrs[0].String(),
						Coin:   nusd69420,
						MintTo: "",
					},
					WantErr: types.ErrUnauthorized.Error(),
				},
				{
					TestMsg: &types.MsgBurn{
						Sender:   addrs[0].String(),
						Coin:     nusd69420,
						BurnFrom: "",
					},
					WantErr: types.ErrUnauthorized.Error(),
				},
			},
		},

		{
			name: "sad: blocked addrs",
			setupMsgs: []sdk.Msg{

				&types.MsgCreateDenom{
					Sender:   addrs[0].String(),
					Subdenom: "nusd",
				},

				&types.MsgMint{
					Sender: addrs[0].String(),
					Coin:   nusd69420,
					MintTo: "",
				},
			},
			testMsgs: []SdkMsgTestCase{
				{
					TestMsg: &types.MsgMint{
						Sender: addrs[0].String(),
						Coin:   nusd69420,
						MintTo: authtypes.NewModuleAddress(oracletypes.ModuleName).String(),
					},
					WantErr: types.ErrBlockedAddress.Error(),
				},
				{
					TestMsg: &types.MsgBurn{
						Sender:   addrs[0].String(),
						Coin:     nusd69420,
						BurnFrom: authtypes.NewModuleAddress(oracletypes.ModuleName).String(),
					},
					WantErr: types.ErrBlockedAddress.Error(),
				},
			},
		},
	}

	for _, tc := range testCases {
		s.T().Run(tc.name, func(t *testing.T) {
			s.SetupTest()

			for _, txMsg := range tc.setupMsgs {
				err := s.HandleMsg(txMsg)
				s.Require().NoError(err)
			}

			if tc.preHook != nil {
				tc.preHook(s.ctx, s.app)
			}

			for _, msgTc := range tc.testMsgs {
				err := s.HandleMsg(msgTc.TestMsg)
				if msgTc.WantErr != "" {
					s.ErrorContains(err, msgTc.WantErr)
					continue
				}
				s.NoError(err)
			}

			if tc.postHook != nil {
				tc.postHook(s.ctx, s.app)
			}
		})
	}
}
