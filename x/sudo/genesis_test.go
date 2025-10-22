package sudo_test

import (
	"github.com/NibiruChain/nibiru/v2/x/nutil/testutil"
	"github.com/NibiruChain/nibiru/v2/x/sudo"
)

// TestGenesisState_Validate tests the GenesisState.Validate() function with comprehensive test cases.
// This test verifies that the validation logic correctly identifies valid and invalid genesis states,
// covering all error paths in the Validate() function.
func (s *Suite) TestGenesisState_Validate() {
	// Generate valid test addresses
	validAddr := testutil.AccAddress().String()
	validAddr1 := testutil.AccAddress().String()
	validAddr2 := testutil.AccAddress().String()
	validAddr3 := testutil.AccAddress().String()
	validAddr4 := testutil.AccAddress().String()

	testCases := []struct {
		name        string
		genState    *sudo.GenesisState
		wantErr     bool
		errContains string
	}{
		{
			name: "valid - complete state",
			genState: &sudo.GenesisState{
				Sudoers: sudo.Sudoers{
					Root:      validAddr,
					Contracts: []string{validAddr1, validAddr2},
				},
				ZeroGasActors: &sudo.ZeroGasActors{
					Senders:   []string{validAddr3},
					Contracts: []string{validAddr4},
				},
			},
			wantErr: false,
		},
		{
			name: "valid - minimal (root only)",
			genState: &sudo.GenesisState{
				Sudoers: sudo.Sudoers{
					Root:      validAddr,
					Contracts: []string{},
				},
			},
			wantErr: false,
		},
		{
			name: "valid - with ZeroGasActors only",
			genState: &sudo.GenesisState{
				Sudoers: sudo.Sudoers{
					Root:      validAddr,
					Contracts: []string{},
				},
				ZeroGasActors: &sudo.ZeroGasActors{
					Senders:   []string{validAddr1},
					Contracts: []string{validAddr2},
				},
			},
			wantErr: false,
		},
		{
			name: "invalid - empty root",
			genState: &sudo.GenesisState{
				Sudoers: sudo.Sudoers{
					Root:      "",
					Contracts: []string{validAddr1},
				},
			},
			wantErr:     true,
			errContains: "root addr",
		},
		{
			name: "invalid - nil contracts",
			genState: &sudo.GenesisState{
				Sudoers: sudo.Sudoers{
					Root:      validAddr,
					Contracts: nil,
				},
			},
			wantErr:     true,
			errContains: "nil contract state",
		},
		{
			name: "invalid - bad root address",
			genState: &sudo.GenesisState{
				Sudoers: sudo.Sudoers{
					Root:      "invalid",
					Contracts: []string{},
				},
			},
			wantErr:     true,
			errContains: "root addr",
		},
		{
			name: "invalid - bad contract address",
			genState: &sudo.GenesisState{
				Sudoers: sudo.Sudoers{
					Root:      validAddr,
					Contracts: []string{"invalid"},
				},
			},
			wantErr:     true,
			errContains: "contract addr",
		},
		{
			name: "invalid - ZeroGasActors bad sender",
			genState: &sudo.GenesisState{
				Sudoers: sudo.Sudoers{
					Root:      validAddr,
					Contracts: []string{},
				},
				ZeroGasActors: &sudo.ZeroGasActors{
					Senders: []string{"invalid"},
				},
			},
			wantErr:     true,
			errContains: "ZeroGasActors stateless validation error",
		},
		{
			name: "invalid - ZeroGasActors bad contract",
			genState: &sudo.GenesisState{
				Sudoers: sudo.Sudoers{
					Root:      validAddr,
					Contracts: []string{},
				},
				ZeroGasActors: &sudo.ZeroGasActors{
					Contracts: []string{"0xBAD"},
				},
			},
			wantErr:     true,
			errContains: "ZeroGasActors stateless validation error",
		},
		{
			name: "invalid - ZeroGasActors with empty senders and contracts",
			genState: &sudo.GenesisState{
				Sudoers: sudo.Sudoers{
					Root:      validAddr,
					Contracts: []string{},
				},
				ZeroGasActors: &sudo.ZeroGasActors{
					Senders:   []string{},
					Contracts: []string{},
				},
			},
			wantErr: false, // Empty ZeroGasActors should be valid
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			err := tc.genState.Validate()
			if tc.wantErr {
				s.Require().Error(err)
				if tc.errContains != "" {
					s.Require().Contains(err.Error(), tc.errContains)
				}
			} else {
				s.Require().NoError(err)
			}
		})
	}
}
