package keeper_test

import (
	"github.com/NibiruChain/collections"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"cosmossdk.io/math"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/x/common/testutil"
	oracletypes "github.com/NibiruChain/nibiru/x/oracle/types"
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
		s.Run(tc.name, func() {
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
			}.Denom()
			s.Equal(want.String(), resp.NewTokenDenom)

			if tc.postHook != nil {
				tc.postHook(s.ctx, s.app)
			}
		})
	}
}

func (s *TestSuite) TestChangeAdmin() {
	sbf := testutil.AccAddress().String()

	testCases := []struct {
		Name     string
		txMsg    *types.MsgChangeAdmin
		wantErr  string
		preHook  func(ctx sdk.Context, bapp *app.NibiruApp)
		postHook func(ctx sdk.Context, bapp *app.NibiruApp)
	}{
		{
			Name:    "sad: nil msg",
			txMsg:   nil,
			wantErr: "nil msg",
		},

		{
			Name: "sad: fail validate basic",
			txMsg: &types.MsgChangeAdmin{
				Sender: "sender", Denom: "tf/creator/nusd", NewAdmin: "new admin",
			},
			wantErr: "invalid sender",
		},

		{
			Name: "sad: non-admin tries to change admin",
			txMsg: &types.MsgChangeAdmin{
				Sender:   testutil.AccAddress().String(),
				Denom:    types.TFDenom{Creator: sbf, Subdenom: "ftt"}.Denom().String(),
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
			Name: "happy: SBF changes FTT admin",
			txMsg: &types.MsgChangeAdmin{
				Sender:   sbf,
				Denom:    types.TFDenom{Creator: sbf, Subdenom: "ftt"}.Denom().String(),
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
			Name: "sad: change admin for denom that doesn't exist ",
			txMsg: &types.MsgChangeAdmin{
				Sender:   sbf,
				Denom:    types.TFDenom{Creator: sbf, Subdenom: "ftt"}.Denom().String(),
				NewAdmin: testutil.AccAddress().String(),
			},
			wantErr: collections.ErrNotFound.Error(),
		},
	}

	for _, tc := range testCases {
		s.Run(tc.Name, func() {
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
		s.Run(tc.name, func() {
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

type TestCaseTx struct {
	// Name: identifier for the test case.
	Name string

	// SetupMsgs: a list of messages to broadcast in order that should execute
	// without error. These can be used to create complex scenarios.
	SetupMsgs []sdk.Msg

	// PreHook: an optional hook that runs before TestMsgs
	PreHook func(ctx sdk.Context, bapp *app.NibiruApp)

	TestMsgs []TestMsgElem

	// PostHook: an optional hook that runs after TestMsgs
	PostHook func(ctx sdk.Context, bapp *app.NibiruApp)
}

func (tc TestCaseTx) RunTest(s *TestSuite) {
	for _, txMsg := range tc.SetupMsgs {
		err := s.HandleMsg(txMsg)
		s.Require().NoError(err)
	}

	if tc.PreHook != nil {
		tc.PreHook(s.ctx, s.app)
	}

	for _, msgTc := range tc.TestMsgs {
		err := s.HandleMsg(msgTc.TestMsg)
		if msgTc.WantErr != "" {
			s.ErrorContains(err, msgTc.WantErr)
			continue
		}
		s.NoError(err)
	}

	if tc.PostHook != nil {
		tc.PostHook(s.ctx, s.app)
	}
}

type TestMsgElem struct {
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
		Denom:  tfdenom.Denom().String(),
		Amount: math.NewInt(69_420),
	}

	testCases := []TestCaseTx{
		{
			Name: "happy: mint and burn",
			SetupMsgs: []sdk.Msg{
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
						}.Denom().String(),
						Amount: math.NewInt(69_420),
					},
					MintTo: "",
				},
			},
			TestMsgs: []TestMsgElem{
				{
					TestMsg: &types.MsgBurn{
						Sender: addrs[0].String(),
						Coin: sdk.Coin{
							Denom: types.TFDenom{
								Creator:  addrs[0].String(),
								Subdenom: "nusd",
							}.Denom().String(),
							Amount: math.NewInt(1),
						},
						BurnFrom: "",
					},
					WantErr: "",
				},
			},
			PreHook: func(ctx sdk.Context, bapp *app.NibiruApp) {
				allDenoms := bapp.TokenFactoryKeeper.Store.Denoms.
					Iterate(ctx, collections.Range[string]{}).Values()
				s.Len(allDenoms, 1)
			},
			PostHook: func(ctx sdk.Context, bapp *app.NibiruApp) {
				allDenoms := bapp.TokenFactoryKeeper.Store.Denoms.
					Iterate(ctx, collections.Range[string]{}).Values()

				s.T().Log("Total supply should decrease by burned amount.")
				denom := allDenoms[0]
				s.Equal(
					math.NewInt(69_419), s.app.BankKeeper.GetSupply(s.ctx, denom.Denom().String()).Amount,
				)

				s.T().Log("Module account should be empty.")
				coin := s.app.BankKeeper.GetBalance(
					s.ctx, tfModuleAddr, denom.Denom().String())
				s.Equal(
					math.NewInt(0),
					coin.Amount,
				)
			},
		},

		{
			Name:      "sad: denom does not exist",
			SetupMsgs: []sdk.Msg{},
			TestMsgs: []TestMsgElem{
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
			Name: "sad: sender is not admin",
			SetupMsgs: []sdk.Msg{
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
					Denom:    tfdenom.Denom().String(),
					NewAdmin: addrs[1].String(),
				},
			},
			TestMsgs: []TestMsgElem{
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
			Name: "sad: blocked addrs",
			SetupMsgs: []sdk.Msg{
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
			TestMsgs: []TestMsgElem{
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
		s.Run(tc.Name, func() {
			s.SetupTest()
			tc.RunTest(s)
		})
	}
}

func (s *TestSuite) TestSetDenomMetadata() {
	_, addrs := testutil.PrivKeyAddressPairs(4)
	tfdenom := types.TFDenom{
		Creator:  addrs[0].String(),
		Subdenom: "nusd",
	}

	testCases := []TestCaseTx{
		{
			Name: "happy: set metadata",
			SetupMsgs: []sdk.Msg{
				&types.MsgCreateDenom{
					Sender:   addrs[0].String(),
					Subdenom: "nusd",
				},
			},
			TestMsgs: []TestMsgElem{
				{
					TestMsg: &types.MsgSetDenomMetadata{
						Sender:   addrs[0].String(),
						Metadata: tfdenom.DefaultBankMetadata(),
					},
					WantErr: "",
				},
				{
					TestMsg: &types.MsgSetDenomMetadata{
						Sender: addrs[0].String(),
						Metadata: banktypes.Metadata{
							Description: "US Dollar",
							DenomUnits: []*banktypes.DenomUnit{
								{
									Denom:    tfdenom.Denom().String(),
									Exponent: 0,
									Aliases:  []string{"unusd"},
								},
								{Denom: "USD", Exponent: 6},
							},
							Base:    tfdenom.Denom().String(),
							Display: "USD",
							Name:    "USD",
							Symbol:  "USD",
							URI:     "https://www.federalreserve.gov/aboutthefed/currency.htm",
						},
					},
					WantErr: "",
				},
			},
		}, // end case

		{
			Name: "sad: sender not admin",
			SetupMsgs: []sdk.Msg{
				&types.MsgCreateDenom{
					Sender:   addrs[0].String(),
					Subdenom: "nusd",
				},
			},
			TestMsgs: []TestMsgElem{
				{
					TestMsg: &types.MsgSetDenomMetadata{
						Sender:   addrs[1].String(),
						Metadata: tfdenom.DefaultBankMetadata(),
					},
					WantErr: types.ErrUnauthorized.Error(),
				},
			},
		}, // end case

		{
			Name: "sad: invalid sender",
			SetupMsgs: []sdk.Msg{
				&types.MsgCreateDenom{
					Sender:   addrs[0].String(),
					Subdenom: "nusd",
				},
			},
			TestMsgs: []TestMsgElem{
				{
					TestMsg: &types.MsgSetDenomMetadata{
						Sender:   "sender",
						Metadata: tfdenom.DefaultBankMetadata(),
					},
					WantErr: "invalid sender",
				},
			},
		}, // end case

		{
			Name: "sad: nil msg",
			TestMsgs: []TestMsgElem{
				{
					TestMsg: (*types.MsgSetDenomMetadata)(nil),
					WantErr: "nil msg",
				},
			},
		},

		{
			Name: "sad: metadata.base is not registered",
			SetupMsgs: []sdk.Msg{
				&types.MsgCreateDenom{
					Sender:   addrs[0].String(),
					Subdenom: "nusd",
				},
			},
			TestMsgs: []TestMsgElem{
				{
					TestMsg: &types.MsgSetDenomMetadata{
						Sender: addrs[0].String(),
						Metadata: banktypes.Metadata{
							DenomUnits: []*banktypes.DenomUnit{{
								Denom:    "ust",
								Exponent: 0,
							}},
							Base: "ust",
							// The following is necessary for x/bank denom validation
							Display: "ust",
							Name:    "ust",
							Symbol:  "ust",
						},
					},
					WantErr: collections.ErrNotFound.Error(),
				},
			},
		}, // end case

	}

	for _, tc := range testCases {
		s.Run(tc.Name, func() {
			s.SetupTest()
			tc.RunTest(s)
		})
	}
}

func (s *TestSuite) TestBurnNative() {
	_, addrs := testutil.PrivKeyAddressPairs(4)
	tfModuleAddr := authtypes.NewModuleAddress(types.ModuleName)

	testCases := []TestCaseTx{
		{
			Name:      "happy: burn",
			SetupMsgs: []sdk.Msg{},
			PreHook: func(ctx sdk.Context, bapp *app.NibiruApp) {
				coins := sdk.NewCoins(sdk.NewCoin("unibi", math.NewInt(123)))
				s.NoError(bapp.BankKeeper.MintCoins(ctx, types.ModuleName, coins))
				s.NoError(bapp.BankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, addrs[0], coins))
			},
			TestMsgs: []TestMsgElem{
				{
					TestMsg: &types.MsgBurnNative{
						Sender: addrs[0].String(),
						Coin:   sdk.NewCoin("unibi", math.NewInt(123)),
					},
					WantErr: "",
				},
			},
			PostHook: func(ctx sdk.Context, bapp *app.NibiruApp) {
				s.Equal(
					math.NewInt(0), s.app.BankKeeper.GetSupply(s.ctx, "unibi").Amount,
				)

				s.Equal(
					math.NewInt(0),
					s.app.BankKeeper.GetBalance(s.ctx, tfModuleAddr, "unibi").Amount,
				)

				s.Equal(
					math.NewInt(0),
					s.app.BankKeeper.GetBalance(s.ctx, addrs[0], "unibi").Amount,
				)
			},
		},

		{
			Name:      "sad: not enough funds",
			SetupMsgs: []sdk.Msg{},
			PreHook: func(ctx sdk.Context, bapp *app.NibiruApp) {
				coins := sdk.NewCoins(sdk.NewCoin("unibi", math.NewInt(123)))
				s.NoError(bapp.BankKeeper.MintCoins(ctx, types.ModuleName, coins))
				s.NoError(bapp.BankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, addrs[0], coins))
			},
			TestMsgs: []TestMsgElem{
				{
					TestMsg: &types.MsgBurnNative{
						Sender: addrs[0].String(),
						Coin:   sdk.NewCoin("unibi", math.NewInt(124)),
					},
					WantErr: "spendable balance 123unibi is smaller than 124unibi: insufficient funds",
				},
			},
			PostHook: func(ctx sdk.Context, bapp *app.NibiruApp) {
				s.Equal(
					math.NewInt(123), s.app.BankKeeper.GetSupply(s.ctx, "unibi").Amount,
				)

				s.Equal(
					math.NewInt(123),
					s.app.BankKeeper.GetBalance(s.ctx, addrs[0], "unibi").Amount,
				)
			},
		},

		{
			Name: "sad: nil msg",
			TestMsgs: []TestMsgElem{
				{
					TestMsg: (*types.MsgSetDenomMetadata)(nil),
					WantErr: "nil msg",
				},
			},
		},
	}

	for _, tc := range testCases {
		s.Run(tc.Name, func() {
			s.SetupTest()
			tc.RunTest(s)
		})
	}
}
