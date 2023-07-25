package keeper_test

import (
	"testing"
	"time"

	"github.com/NibiruChain/nibiru/x/sudo/keeper"

	"github.com/NibiruChain/nibiru/x/sudo/types"

	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/app/codec"
	"github.com/NibiruChain/nibiru/x/common/set"
	"github.com/NibiruChain/nibiru/x/common/testutil"
	"github.com/NibiruChain/nibiru/x/common/testutil/testapp"
	"github.com/NibiruChain/nibiru/x/sudo"
)

func init() {
	useNibiAccPrefix()
}
func setup() (*app.NibiruApp, sdk.Context) {
	genState := app.NewDefaultGenesisState(codec.DefaultEncoding().Marshaler)
	nibiru := testapp.NewNibiruTestApp(genState)
	ctx := nibiru.NewContext(false, tmproto.Header{
		Height:  1,
		ChainID: "nibiru-sudonet-1",
		Time:    time.Now().UTC(),
	})
	return nibiru, ctx
}

func useNibiAccPrefix() {
	accountAddressPrefix := "nibi"
	accountPubKeyPrefix := accountAddressPrefix + "pub"
	validatorAddressPrefix := accountAddressPrefix + "valoper"
	validatorPubKeyPrefix := accountAddressPrefix + "valoperpub"
	consNodeAddressPrefix := accountAddressPrefix + "valcons"
	consNodePubKeyPrefix := accountAddressPrefix + "valconspub"
	config := sdk.GetConfig()
	config.SetBech32PrefixForAccount(accountAddressPrefix, accountPubKeyPrefix)
	config.SetBech32PrefixForValidator(validatorAddressPrefix, validatorPubKeyPrefix)
	config.SetBech32PrefixForConsensusNode(consNodeAddressPrefix, consNodePubKeyPrefix)
}

func TestGenesis(t *testing.T) {
	for _, testCase := range []struct {
		name     string
		genState *types.GenesisState
		panic    bool
		empty    bool
	}{
		{
			name:     "default genesis (empty)",
			genState: sudo.DefaultGenesis(),
			empty:    true,
		},
		{
			name: "happy genesis with contracts",
			genState: &types.GenesisState{
				Sudoers: types.Sudoers{
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
			genState: &types.GenesisState{
				Sudoers: types.Sudoers{
					Root:      "root",
					Contracts: []string{"contract"},
				}},
			panic: true,
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			// Setup
			nibiru, ctx := setup()

			// InitGenesis
			if testCase.panic {
				require.Panics(t, func() {
					sudo.InitGenesis(ctx, nibiru.SudoKeeper, *testCase.genState)
				})
				return
			}
			require.NotPanics(t, func() {
				sudo.InitGenesis(ctx, nibiru.SudoKeeper, *testCase.genState)
			})

			// ExportGenesis
			got := sudo.ExportGenesis(ctx, nibiru.SudoKeeper)
			require.NotNil(t, got)

			// Validate
			if testCase.empty {
				// We only run this when we expect empty or null values.
				// Otherwise, it resets the fields of the struct.
				testutil.Fill(got)
			}
			require.EqualValues(t, *testCase.genState, *got)

			// Validate with AppModule
			cdc := types.ModuleCdc
			require.Panics(t, func() {
				// failing case
				appModule := sudo.AppModule{}
				_ = appModule.ExportGenesis(ctx, cdc)
			})
			appModule := sudo.NewAppModule(cdc, nibiru.SudoKeeper)
			jsonBz := appModule.ExportGenesis(ctx, cdc)
			err := appModule.ValidateGenesis(cdc, nil, jsonBz)
			require.NoErrorf(t, err, "exportedGenesis: %s", jsonBz)
		})
	}
}

func TestSudo_AddContracts(t *testing.T) {
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
		t.Run(tc.name, func(t *testing.T) {
			_, _ = setup()
			root := testutil.AccAddress().String()
			sudoers := keeper.Sudoers{
				Root:      root,
				Contracts: set.New(tc.start...),
			}

			newContractsState, err := sudoers.AddContracts(tc.delta)
			if tc.shouldError {
				require.Error(t, err)
				return
			}
			require.NoErrorf(t, err, "newState: %s", newContractsState.ToSlice())
		})
	}
}

func TestMsgServer_ChangeRoot(t *testing.T) {
	app, ctx := setup()

	_, err := app.SudoKeeper.Sudoers.Get(ctx)
	require.NoError(t, err)

	actualRoot := testutil.AccAddress().String()
	newRoot := testutil.AccAddress().String()
	fakeRoot := testutil.AccAddress().String()

	app.SudoKeeper.Sudoers.Set(ctx, types.Sudoers{
		Root: actualRoot,
	})

	// try to change root with non-root account
	msgServer := keeper.NewMsgServer(app.SudoKeeper)
	_, err = msgServer.ChangeRoot(
		sdk.WrapSDKContext(ctx),
		&types.MsgChangeRoot{Sender: fakeRoot, NewRoot: newRoot},
	)
	require.EqualError(t, err, "unauthorized: missing sudo permissions")

	// try to change root with root account
	_, err = msgServer.ChangeRoot(
		sdk.WrapSDKContext(ctx),
		&types.MsgChangeRoot{Sender: actualRoot, NewRoot: newRoot},
	)
	require.NoError(t, err)

	// check that root has changed
	sudoers, err := app.SudoKeeper.Sudoers.Get(ctx)
	require.NoError(t, err)

	require.Equal(t, newRoot, sudoers.Root)
}

func TestSudo_FromPbSudoers(t *testing.T) {
	for _, tc := range []struct {
		name string
		in   types.Sudoers
		out  keeper.Sudoers
	}{
		{
			name: "empty",
			in:   types.Sudoers{},
			out: keeper.Sudoers{
				Root:      "",
				Contracts: set.Set[string]{},
			},
		},
		{
			name: "happy",
			in:   types.Sudoers{Root: "root", Contracts: []string{"contractA", "contractB"}},
			out: keeper.Sudoers{
				Root:      "root",
				Contracts: set.New[string]("contractA", "contractB"),
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			out := keeper.SudoersFromPb(tc.in)
			assert.EqualValues(t, tc.out.Contracts, out.Contracts)
			assert.EqualValues(t, tc.out.Root, out.Root)

			pbSudoers := out.ToPb()
			for _, contract := range tc.in.Contracts {
				assert.True(t, set.New(pbSudoers.Contracts...).Has(contract))
			}
		})
	}
}

func TestKeeper_AddContracts(t *testing.T) {
	root := "nibi1ggpg3vluy09qmfkgwsgkumhmmv2z44rdafn6qa"
	exampleAddrs := []string{
		"nibi1zaavvzxez0elundtn32qnk9lkm8kmcsz44g7xl",
		"nibi1ah8gqrtjllhc5ld4rxgl4uglvwl93ag0sh6e6v",
		"nibi1x5zknk8va44th5vjpg0fagf0lxx0rvurpmp8gs",
	}

	testCases := []struct {
		name            string
		contractsBefore []string
		msg             *types.MsgEditSudoers
		contractsAfter  []string
		shouldFail      bool
	}{
		{
			name: "happy",
			contractsBefore: []string{
				exampleAddrs[0],
			},
			msg: &types.MsgEditSudoers{
				Action: string(types.AddContracts),
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
			msg: &types.MsgEditSudoers{
				Action: string(types.AddContracts),
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
			msg: &types.MsgEditSudoers{
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
			msg: &types.MsgEditSudoers{
				Action: string(types.AddContracts),
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
		t.Run(tc.name, func(t *testing.T) {
			nibiru, ctx := setup()
			k := nibiru.SudoKeeper

			t.Log("Set starting contracts state")
			stateBefore := types.Sudoers{
				Root:      root,
				Contracts: tc.contractsBefore,
			}
			k.Sudoers.Set(ctx, stateBefore)
			gotStateBefore, err := k.Sudoers.Get(ctx)
			require.NoError(t, err)
			require.EqualValues(t, stateBefore, gotStateBefore)

			t.Log("Execute message")
			// Check via message handler directly
			msgServer := keeper.NewMsgServer(k)
			res, err := msgServer.EditSudoers(sdk.WrapSDKContext(ctx), tc.msg)
			// Check via Keeper
			res2, err2 := k.AddContracts(sdk.WrapSDKContext(ctx), tc.msg)
			if tc.shouldFail {
				require.Errorf(t, err, "resp: %s", res)
				require.Errorf(t, err2, "resp: %s", res2)
				return
			}
			require.NoError(t, err)

			t.Log("Check correctness of state updates")
			contractsAfter := set.New(tc.contractsAfter...)
			stateAfter, err := k.Sudoers.Get(ctx)
			require.NoError(t, err)
			got := set.New(stateAfter.Contracts...)
			// Checking cardinality (length) and iterating to check if one set
			// contains the other is equivalent to set equality in math.
			assert.EqualValues(t, contractsAfter.Len(), got.Len())
			for member := range got {
				assert.True(t, contractsAfter.Has(member))
			}
		})
	}
}

type DummyMsg struct {
}

var _ sdk.Msg = (*DummyMsg)(nil)

func (dm DummyMsg) GetSigners() []sdk.AccAddress { return []sdk.AccAddress{} }
func (dm DummyMsg) ValidateBasic() error         { return nil }
func (dm *DummyMsg) Reset()                      {}
func (dm *DummyMsg) ProtoMessage()               {}
func (dm *DummyMsg) String() string              { return "dummy" }

func TestKeeper_RemoveContracts(t *testing.T) {
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
		msg             *types.MsgEditSudoers
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
			msg: &types.MsgEditSudoers{
				Action: string(types.RemoveContracts),
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
			msg: &types.MsgEditSudoers{
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
			msg: &types.MsgEditSudoers{
				Action: string(types.RemoveContracts),
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
		t.Run(tc.name, func(t *testing.T) {
			nibiru, ctx := setup()
			k := nibiru.SudoKeeper

			t.Log("Set starting contracts state")
			stateBefore := types.Sudoers{
				Root:      root,
				Contracts: tc.contractsBefore,
			}
			k.Sudoers.Set(ctx, stateBefore)
			gotStateBefore, err := k.Sudoers.Get(ctx)
			require.NoError(t, err)
			require.EqualValues(t, stateBefore, gotStateBefore)

			t.Log("Execute message")
			// Check via message handler directly
			msgServer := keeper.NewMsgServer(k)
			res, err := msgServer.EditSudoers(ctx, tc.msg)
			// Check via Keeper
			res2, err2 := k.RemoveContracts(sdk.WrapSDKContext(ctx), tc.msg)
			if tc.shouldFail {
				require.Errorf(t, err, "resp: %s", res)
				require.Errorf(t, err2, "resp: %s", res2)
				return
			}

			t.Log("Check correctness of state updates")
			contractsAfter := set.New(tc.contractsAfter...)
			stateAfter, err := k.Sudoers.Get(ctx)
			require.NoError(t, err)
			got := set.New(stateAfter.Contracts...)
			// Checking cardinality (length) and iterating to check if one set
			// contains the other is equivalent to set equality in math.
			assert.EqualValues(t, contractsAfter.Len(), got.Len())
			for member := range got {
				assert.True(t, contractsAfter.Has(member))
			}
		})
	}
}
