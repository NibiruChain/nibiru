package types_test

import (
	fmt "fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/x/common/testutil"
	"github.com/NibiruChain/nibiru/x/tokenfactory/types"
)

func TestDenomStr_Validate(t *testing.T) {
	testCases := []struct {
		denom   types.DenomStr
		wantErr string
	}{
		{"tf/creator123/subdenom", ""},
		{"tf//subdenom", "empty creator"},
		{"tf/creator123/", "empty subdenom"},
		{"creator123/subdenom", "invalid number of sections"},
		{"tf/creator123/subdenom/extra", "invalid number of sections"},
		{"/creator123/subdenom", "missing denom prefix"},
	}

	for _, tc := range testCases {
		t.Run(string(tc.denom), func(t *testing.T) {
			tfDenom, err := tc.denom.ToStruct()

			if tc.wantErr != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tfDenom.Denom(), tc.denom)
			assert.Equal(t, tfDenom.String(), string(tc.denom))

			assert.NoError(t, tfDenom.Validate())
			assert.NotPanics(t, func() {
				_ = tfDenom.DefaultBankMetadata()
				_ = tc.denom.MustToStruct()
			})

			assert.NoError(t, types.GenesisDenom{
				Denom:             tc.denom.String(),
				AuthorityMetadata: types.DenomAuthorityMetadata{},
			}.Validate())
		})
	}
}

func TestModuleParamsValidate(t *testing.T) {
	params := types.DefaultModuleParams()
	require.NoError(t, params.Validate())

	params.DenomCreationGasConsume = 0
	require.Error(t, params.Validate())
}

func TestGenesisState(t *testing.T) {
	var happyGenDenoms []types.GenesisDenom
	for i := 0; i < 5; i++ {
		creator := testutil.AccAddress()
		lettersIdx := i * 2
		happyGenDenoms = append(happyGenDenoms, types.GenesisDenom{
			Denom: types.TFDenom{
				Creator:  creator.String(),
				Subdenom: testutil.Latin.Letters[lettersIdx : lettersIdx+4],
			}.String(),
			AuthorityMetadata: types.DenomAuthorityMetadata{
				Admin: creator.String(),
			},
		})
	}

	for idx, tc := range []struct {
		name     string
		genState types.GenesisState
		wantErr  string
	}{
		{name: "default", wantErr: "", genState: *types.DefaultGenesis()},
		{name: "sad: params", wantErr: types.ErrInvalidModuleParams.Error(),
			genState: types.GenesisState{
				Params: types.ModuleParams{
					DenomCreationGasConsume: 0,
				},
				FactoryDenoms: happyGenDenoms,
			},
		},
		{name: "sad: duplicate",
			wantErr: "duplicate denom",
			genState: types.GenesisState{
				Params: types.DefaultModuleParams(),
				FactoryDenoms: []types.GenesisDenom{
					happyGenDenoms[0], happyGenDenoms[0], happyGenDenoms[1],
				},
			},
		},
		{name: "sad: invalid admin",
			wantErr: types.ErrInvalidAdmin.Error(),
			genState: types.GenesisState{
				Params: types.DefaultModuleParams(),
				FactoryDenoms: []types.GenesisDenom{
					happyGenDenoms[0],
					{
						Denom: happyGenDenoms[1].Denom,
						AuthorityMetadata: types.DenomAuthorityMetadata{
							Admin: "not_an_address",
						},
					},
				},
			},
		},

		{name: "sad: invalid genesis denom",
			wantErr: types.ErrInvalidGenesis.Error(),
			genState: types.GenesisState{
				Params: types.DefaultModuleParams(),
				FactoryDenoms: []types.GenesisDenom{
					happyGenDenoms[0],
					{
						Denom: "sad denom",
						AuthorityMetadata: types.DenomAuthorityMetadata{
							Admin: happyGenDenoms[1].AuthorityMetadata.Admin,
						},
					},
				},
			},
		},
	} {
		t.Run(fmt.Sprintf("%v %s", idx, tc.name), func(t *testing.T) {
			err := tc.genState.Validate()
			if tc.wantErr != "" {
				assert.Error(t, err)
				require.Contains(t, err.Error(), tc.wantErr)
				return
			}
			require.NoError(t, err)
		})
	}
}
