package types_test

import (
	"testing"

	"github.com/MatrixDao/matrix/x/stablecoin/types"
	"github.com/stretchr/testify/require"
)

func TestGenesisState_Validate(t *testing.T) {
	for _, testCase := range []struct {
		description string
		genState    *types.GenesisState
		valid       bool
	}{
		{
			description: "default is valid",
			genState:    types.DefaultGenesis(),
			valid:       true,
		},
		{
			description: "valid genesis state",
			genState:    &types.GenesisState{

				// this line is used by starport scaffolding # types/genesis/validField
			},
			valid: true,
		},
		// this line is used by starport scaffolding # types/genesis/testcase
	} {
		t.Run(testCase.description, func(t *testing.T) {
			err := testCase.genState.Validate()
			if testCase.valid {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}
