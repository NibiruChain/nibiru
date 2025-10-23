package keeper_test

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/v2/app"
	"github.com/NibiruChain/nibiru/v2/x/nutil/set"
	"github.com/NibiruChain/nibiru/v2/x/nutil/testutil"
	"github.com/NibiruChain/nibiru/v2/x/nutil/testutil/testapp"
	"github.com/NibiruChain/nibiru/v2/x/sudo"
	"github.com/NibiruChain/nibiru/v2/x/sudo/keeper"
	"github.com/NibiruChain/nibiru/v2/x/sudo/sudomodule"
)

func setup() (*app.NibiruApp, keeper.Keeper, sdk.Context) {
	nibiru, ctx := testapp.NewNibiruTestAppAndContext()
	return nibiru, nibiru.SudoKeeper, ctx
}

func (s *Suite) TestGenesis() {
	for _, testCase := range []struct {
		name     string
		genState *sudo.GenesisState
		panic    bool
		empty    bool
	}{
		{
			name:     "default genesis (empty)",
			genState: sudo.DefaultGenesis(),
			panic:    true,
		},
		{
			name: "happy genesis with contracts",
			genState: &sudo.GenesisState{
				Sudoers: sudo.Sudoers{
					Root: testutil.AccAddress().String(),
					Contracts: []string{
						testutil.AccAddress().String(),
						testutil.AccAddress().String(),
						testutil.AccAddress().String(),
					},
				},
			},
			empty: false,
		},
		{
			name:     "nil genesis (panic)",
			genState: nil,
			panic:    true,
		},
		{
			name: "invalid genesis (panic)",
			genState: &sudo.GenesisState{
				Sudoers: sudo.Sudoers{
					Root:      "root",
					Contracts: []string{"contract"},
				},
			},
			panic: true,
		},
	} {
		s.Run(testCase.name, func() {
			// Setup
			nibiru, k, ctx := setup()

			// InitGenesis
			if testCase.panic {
				s.Require().Panics(func() {
					k.InitGenesis(ctx, *testCase.genState)
				})
				return
			}
			s.Require().NotPanics(func() {
				k.InitGenesis(ctx, *testCase.genState)
			})

			// ExportGenesis
			got := k.ExportGenesis(ctx)
			s.Require().NotNil(got)

			// Validate
			if testCase.empty {
				// We only run this when we expect empty or null values.
				// Otherwise, it resets the fields of the struct.
				testutil.Fill(got)
			}

			// Handle ZeroGasActors comparison - if original is nil, exported should be default
			if testCase.genState.ZeroGasActors == nil {
				s.Require().NotNil(got.ZeroGasActors)
				s.Require().Equal(sudo.DefaultZeroGasActors(), *got.ZeroGasActors)
				// Set to nil for comparison of other fields
				got.ZeroGasActors = nil
				testCase.genState.ZeroGasActors = nil
			}
			s.Require().EqualValues(*testCase.genState, *got)

			// Validate with AppModule
			cdc := sudo.ModuleCdc
			s.Require().Panics(func() {
				// failing case
				appModule := sudomodule.AppModule{}
				_ = appModule.ExportGenesis(ctx, cdc)
			})
			appModule := sudomodule.NewAppModule(cdc, nibiru.SudoKeeper)
			jsonBz := appModule.ExportGenesis(ctx, cdc)
			err := appModule.ValidateGenesis(cdc, nil, jsonBz)
			s.Require().NoErrorf(err, "exportedGenesis: %s", jsonBz)
		})
	}
}

func (s *Suite) TestSudo_AddContracts() {
	exampleAddrs := []string{
		"nibi1zaavvzxez0elundtn32qnk9lkm8kmcsz44g7xl",
		"nibi1ah8gqrtjllhc5ld4rxgl4uglvwl93ag0sh6e6v",
		"nibi1x5zknk8va44th5vjpg0fagf0lxx0rvurpmp8gs",
	}

	for _, tc := range []struct {
		name        string
		start       []string
		delta       []string
		end         []string
		shouldError bool
	}{
		{
			name:  "happy - add 1",
			start: []string{exampleAddrs[0]},
			delta: []string{exampleAddrs[1]},
			end:   []string{exampleAddrs[0], exampleAddrs[1]},
		},
		{
			name:  "happy - add multiple",
			start: []string{exampleAddrs[0]},
			delta: []string{exampleAddrs[1], exampleAddrs[2]},
			end:   []string{exampleAddrs[0], exampleAddrs[1], exampleAddrs[2]},
		},
		{
			name:        "sad - invalid addr",
			start:       []string{exampleAddrs[0]},
			delta:       []string{"not-an-address"},
			shouldError: true,
		},
		{
			name:  "empty start",
			start: []string{},
			delta: []string{exampleAddrs[1], exampleAddrs[2]},
			end:   []string{exampleAddrs[1], exampleAddrs[2]},
		},
	} {
		s.Run(tc.name, func() {
			_, _, _ = setup()
			root := testutil.AccAddress().String()
			sudoers := keeper.Sudoers{
				Root:      root,
				Contracts: set.New(tc.start...),
			}

			newContractsState, err := sudoers.AddContracts(tc.delta)
			if tc.shouldError {
				s.Require().Error(err)
				return
			}
			s.Require().NoErrorf(err, "newState: %s", newContractsState.ToSlice())
		})
	}
}

func (s *Suite) TestMsgServer_ChangeRoot() {
	app, _, ctx := setup()

	_, err := app.SudoKeeper.Sudoers.Get(ctx)
	s.Require().NoError(err)

	actualRoot := testutil.AccAddress().String()
	newRoot := testutil.AccAddress().String()
	fakeRoot := testutil.AccAddress().String()

	app.SudoKeeper.Sudoers.Set(ctx, sudo.Sudoers{
		Root: actualRoot,
	})

	// try to change root with non-root account
	msgServer := app.SudoKeeper
	_, err = msgServer.ChangeRoot(
		sdk.WrapSDKContext(ctx),
		&sudo.MsgChangeRoot{Sender: fakeRoot, NewRoot: newRoot},
	)
	s.Require().EqualError(err, "unauthorized: missing sudo permissions")

	// try to change root with root account
	_, err = msgServer.ChangeRoot(
		sdk.WrapSDKContext(ctx),
		&sudo.MsgChangeRoot{Sender: actualRoot, NewRoot: newRoot},
	)
	s.Require().NoError(err)

	// check that root has changed
	sudoers, err := app.SudoKeeper.Sudoers.Get(ctx)
	s.Require().NoError(err)

	s.Require().Equal(newRoot, sudoers.Root)
}

func (s *Suite) TestSudo_FromPbSudoers() {
	for _, tc := range []struct {
		name string
		in   sudo.Sudoers
		out  keeper.Sudoers
	}{
		{
			name: "empty",
			in:   sudo.Sudoers{},
			out: keeper.Sudoers{
				Root:      "",
				Contracts: set.Set[string]{},
			},
		},
		{
			name: "happy",
			in:   sudo.Sudoers{Root: "root", Contracts: []string{"contractA", "contractB"}},
			out: keeper.Sudoers{
				Root:      "root",
				Contracts: set.New[string]("contractA", "contractB"),
			},
		},
	} {
		s.Run(tc.name, func() {
			out := keeper.SudoersFromPb(tc.in)
			s.EqualValuesf(tc.out.Contracts, out.Contracts, "out: %s", out.String())
			s.EqualValuesf(tc.out.Root, out.Root, "out: %s", out.String())

			pbSudoers := out.ToPb()
			for _, contract := range tc.in.Contracts {
				s.True(set.New(pbSudoers.Contracts...).Has(contract))
			}
		})
	}
}

func (s *Suite) TestKeeper_AddContracts() {
	root := "nibi1ggpg3vluy09qmfkgwsgkumhmmv2z44rdafn6qa"
	exampleAddrs := []string{
		"nibi1zaavvzxez0elundtn32qnk9lkm8kmcsz44g7xl",
		"nibi1ah8gqrtjllhc5ld4rxgl4uglvwl93ag0sh6e6v",
		"nibi1x5zknk8va44th5vjpg0fagf0lxx0rvurpmp8gs",
	}

	testCases := []struct {
		name            string
		contractsBefore []string
		msg             *sudo.MsgEditSudoers
		contractsAfter  []string
		shouldFail      bool
	}{
		{
			name: "happy",
			contractsBefore: []string{
				exampleAddrs[0],
			},
			msg: &sudo.MsgEditSudoers{
				Action: string(sudo.AddContracts),
				Contracts: []string{
					exampleAddrs[1],
					exampleAddrs[2],
				},
				Sender: root,
			},
			contractsAfter: []string{
				exampleAddrs[0],
				exampleAddrs[1],
				exampleAddrs[2],
			},
		},

		{
			name: "rotten address",
			contractsBefore: []string{
				exampleAddrs[0],
			},
			msg: &sudo.MsgEditSudoers{
				Action: string(sudo.AddContracts),
				Contracts: []string{
					exampleAddrs[1],
					"rotten address",
					exampleAddrs[2],
				},
				Sender: root,
			},
			shouldFail: true,
		},

		{
			name: "wrong action type",
			contractsBefore: []string{
				exampleAddrs[0],
			},
			msg: &sudo.MsgEditSudoers{
				Action: "not an action type",
				Sender: root,
			},
			shouldFail: true,
		},

		{
			name: "sent by non-sudo user",
			contractsBefore: []string{
				exampleAddrs[0],
			},
			msg: &sudo.MsgEditSudoers{
				Action: string(sudo.AddContracts),
				Sender: exampleAddrs[1],
				Contracts: []string{
					exampleAddrs[1],
					exampleAddrs[2],
				},
			},
			contractsAfter: []string{
				exampleAddrs[0],
				exampleAddrs[1],
				exampleAddrs[2],
			},
			shouldFail: true,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			_, k, ctx := setup()

			s.T().Log("Set starting contracts state")
			stateBefore := sudo.Sudoers{
				Root:      root,
				Contracts: tc.contractsBefore,
			}
			k.Sudoers.Set(ctx, stateBefore)
			gotStateBefore, err := k.Sudoers.Get(ctx)
			s.Require().NoError(err)
			s.Require().EqualValues(stateBefore, gotStateBefore)

			s.T().Log("Execute message")
			// Check via message handler directly
			msgServer := k
			res, err := msgServer.EditSudoers(sdk.WrapSDKContext(ctx), tc.msg)
			// Check via Keeper
			res2, err2 := k.AddContracts(sdk.WrapSDKContext(ctx), tc.msg)
			if tc.shouldFail {
				s.Require().Errorf(err, "resp: %s", res)
				s.Require().Errorf(err2, "resp: %s", res2)
				return
			}
			s.Require().NoError(err)

			s.T().Log("Check correctness of state updates")
			contractsAfter := set.New(tc.contractsAfter...)
			stateAfter, err := k.Sudoers.Get(ctx)
			s.Require().NoError(err)
			got := set.New(stateAfter.Contracts...)
			// Checking cardinality (length) and iterating to check if one set
			// contains the other is equivalent to set equality in math.
			s.EqualValues(contractsAfter.Len(), got.Len())
			for member := range got {
				s.True(contractsAfter.Has(member))
			}
		})
	}
}

func (s *Suite) TestKeeper_RemoveContracts() {
	root := "nibi1ggpg3vluy09qmfkgwsgkumhmmv2z44rdafn6qa"
	// root := "nibi1ggpg3vluy09qmfkgwsgkumhmmv2z44rd2vhrfw"
	exampleAddrs := []string{
		"nibi1zaavvzxez0elundtn32qnk9lkm8kmcsz44g7xl",
		"nibi1ah8gqrtjllhc5ld4rxgl4uglvwl93ag0sh6e6v",
		"nibi1x5zknk8va44th5vjpg0fagf0lxx0rvurpmp8gs",
	}

	for _, tc := range []struct {
		name            string
		contractsBefore []string
		msg             *sudo.MsgEditSudoers
		contractsAfter  []string
		shouldFail      bool
	}{
		{
			name: "happy",
			contractsBefore: []string{
				exampleAddrs[0],
				exampleAddrs[1],
				exampleAddrs[2],
			},
			msg: &sudo.MsgEditSudoers{
				Action: string(sudo.RemoveContracts),
				Contracts: []string{
					exampleAddrs[1],
					exampleAddrs[2],
				},
				Sender: root,
			},
			contractsAfter: []string{
				exampleAddrs[0],
			},
		},

		{
			name: "wrong action type",
			contractsBefore: []string{
				exampleAddrs[0],
			},
			msg: &sudo.MsgEditSudoers{
				Action: "not an action type",
				Sender: root,
			},
			shouldFail: true,
		},

		{
			name: "happy - no op",
			contractsBefore: []string{
				exampleAddrs[0],
				exampleAddrs[2],
			},
			msg: &sudo.MsgEditSudoers{
				Action: string(sudo.RemoveContracts),
				Contracts: []string{
					exampleAddrs[1],
				},
				Sender: root,
			},
			contractsAfter: []string{
				exampleAddrs[0],
				exampleAddrs[2],
			},
		},
	} {
		s.Run(tc.name, func() {
			_, k, ctx := setup()

			s.T().Log("Set starting contracts state")
			stateBefore := sudo.Sudoers{
				Root:      root,
				Contracts: tc.contractsBefore,
			}
			k.Sudoers.Set(ctx, stateBefore)
			gotStateBefore, err := k.Sudoers.Get(ctx)
			s.Require().NoError(err)
			s.Require().EqualValues(stateBefore, gotStateBefore)

			s.T().Log("Execute message")
			// Check via message handler directly
			msgServer := k
			res, err := msgServer.EditSudoers(ctx, tc.msg)
			// Check via Keeper
			res2, err2 := k.RemoveContracts(sdk.WrapSDKContext(ctx), tc.msg)
			if tc.shouldFail {
				s.Require().Errorf(err, "resp: %s", res)
				s.Require().Errorf(err2, "resp: %s", res2)
				return
			}

			s.T().Log("Check correctness of state updates")
			contractsAfter := set.New(tc.contractsAfter...)
			stateAfter, err := k.Sudoers.Get(ctx)
			s.Require().NoError(err)
			got := set.New(stateAfter.Contracts...)
			// Checking cardinality (length) and iterating to check if one set
			// contains the other is equivalent to set equality in math.
			s.EqualValues(contractsAfter.Len(), got.Len())
			for member := range got {
				s.True(contractsAfter.Has(member))
			}
		})
	}
}

func (s *Suite) TestEditZeroGasActors() {
	addrs := make([]sdk.AccAddress, 4)
	for idx, addrStr := range []string{
		"nibi1ze7y9qwdddejmy7jlw4cymqqlt2wh05yu7t8n7",
		"nibi1jr958gyp5598r5mx4ktcdlmx952gwk8zp85p30",
		"nibi1nmgpgr8l4t8pw9zqx9cltuymvz85wmw9kzd648",
		"nibi1em2mlkrkx0qsa6327tgvl3g0fh8a95hj4xx0f5",
	} {
		addrs[idx] = sdk.MustAccAddressFromBech32(addrStr)
	}

	testCases := testutil.FunctionTestCases{
		{
			Name: "outside permissions fails",
			Test: func() {
				_, k, ctx := setup()
				goCtx := sdk.WrapSDKContext(ctx)
				notSudoer := testutil.AccAddress()
				_, err := k.EditZeroGasActors(goCtx, &sudo.MsgEditZeroGasActors{
					Actors: sudo.ZeroGasActors{},
					Sender: notSudoer.String(),
				})
				s.Require().ErrorContains(err, "unauthorized: missing sudo permissions")
			},
		},

		{
			Name: "happy path",
			Test: func() {
				_, k, ctx := setup()
				goCtx := sdk.WrapSDKContext(ctx)

				resp, err := k.QueryZeroGasActors(goCtx, nil)
				s.NoError(err)
				s.Equal(sudo.DefaultZeroGasActors(), resp.Actors)

				newActorsWithDuplicates := sudo.ZeroGasActors{
					Senders: []string{
						addrs[0].String(),
						addrs[0].String(),
					},
					Contracts: []string{
						addrs[1].String(),
						addrs[2].String(),
						addrs[2].String(),
						addrs[3].String(),
					},
				}
				senderValid, err := k.GetRootAddr(ctx)
				s.Require().NoError(err)

				_, err = k.EditZeroGasActors(goCtx, &sudo.MsgEditZeroGasActors{
					Actors: newActorsWithDuplicates,
					Sender: senderValid.String(),
				})
				s.Require().NoError(err)

				resp, err = k.QueryZeroGasActors(goCtx, nil)
				s.NoError(err)
				s.Equal(sudo.ZeroGasActors{
					Senders: []string{addrs[0].String()},
					Contracts: []string{
						addrs[1].String(),
						addrs[2].String(),
						addrs[3].String(),
					},
				}, resp.Actors)
			},
		},

		{
			Name: "error with invalid gRPC message",
			Test: func() {
				_, k, ctx := setup()
				goCtx := sdk.WrapSDKContext(ctx)
				senderValid, err := k.GetRootAddr(ctx)
				s.Require().NoError(err)
				_, err = k.EditZeroGasActors(goCtx, &sudo.MsgEditZeroGasActors{
					Actors: sudo.ZeroGasActors{
						Contracts: []string{"0xNotAnAddr"},
					},
					Sender: senderValid.String(),
				})
				s.Require().ErrorContains(err, "could not parse address as Nibiru Bech32 or Ethereum hexadecimal")
			},
		},
	}
	testutil.RunFunctionTestSuite(&s.Suite, testCases)
}
