package keeper_test

import (
	"testing"

	"github.com/NibiruChain/collections"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/x/common/testutil"
	"github.com/NibiruChain/nibiru/x/tokenfactory/types"
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
			name:    "sad: nil tx msg",
			txMsg:   nil,
			wantErr: "nil tx msg",
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
			name:    "sad: nil tx msg",
			txMsg:   nil,
			wantErr: "nil tx msg",
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
			name:    "sad: nil tx msg",
			txMsg:   nil,
			wantErr: "nil tx msg",
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
