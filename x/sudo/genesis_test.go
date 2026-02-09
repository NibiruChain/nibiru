package sudo_test

import (
	"github.com/NibiruChain/nibiru/v2/eth"
	"github.com/NibiruChain/nibiru/v2/x/nutil/testutil"
	"github.com/NibiruChain/nibiru/v2/x/sudo"
)

// TestGenesisState_Validate tests the GenesisState.Validate() function with comprehensive test cases.
// This test verifies that the validation logic correctly identifies valid and invalid genesis states,
// covering all error paths in the Validate() function.
func (s *Suite) TestGenesisState_Validate() {
	// Generate valid test addresses
	_, addrs := testutil.PrivKeyAddressPairs(5)
	addrStrs := make([]string, len(addrs))
	for idx, addr := range addrs {
		addrStrs[idx] = addr.String()
	}

	testCases := []struct {
		name     string
		genState *sudo.GenesisState
		wantErr  string
	}{
		{
			name: "valid - complete state",
			genState: &sudo.GenesisState{
				Sudoers: sudo.Sudoers{
					Root:      addrStrs[0],
					Contracts: []string{addrStrs[1], addrStrs[2]},
				},
				ZeroGasActors: &sudo.ZeroGasActors{
					Senders:   []string{addrStrs[3]},
					Contracts: []string{eth.NibiruAddrToEthAddr(addrs[4]).Hex()},
				},
			},
		},
		{
			name: "valid - minimal (root only)",
			genState: &sudo.GenesisState{
				Sudoers: sudo.Sudoers{
					Root:      addrStrs[0],
					Contracts: []string{},
				},
			},
		},
		{
			name: "valid - with ZeroGasActors only",
			genState: &sudo.GenesisState{
				Sudoers: sudo.Sudoers{
					Root:      addrStrs[0],
					Contracts: []string{},
				},
				ZeroGasActors: &sudo.ZeroGasActors{
					Senders:   []string{addrStrs[1]},
					Contracts: []string{eth.NibiruAddrToEthAddr(addrs[2]).Hex()},
				},
			},
		},
		{
			name: "invalid - empty root",
			genState: &sudo.GenesisState{
				Sudoers: sudo.Sudoers{
					Root:      "",
					Contracts: []string{addrStrs[1]},
				},
			},
			wantErr: "root addr",
		},
		{
			name: "invalid - nil contracts",
			genState: &sudo.GenesisState{
				Sudoers: sudo.Sudoers{
					Root:      addrStrs[0],
					Contracts: nil,
				},
			},
			wantErr: "nil contract state",
		},
		{
			name: "invalid - bad root address",
			genState: &sudo.GenesisState{
				Sudoers: sudo.Sudoers{
					Root:      "invalid",
					Contracts: []string{},
				},
			},
			wantErr: "root addr",
		},
		{
			name: "invalid - bad contract address",
			genState: &sudo.GenesisState{
				Sudoers: sudo.Sudoers{
					Root:      addrStrs[0],
					Contracts: []string{"invalid"},
				},
			},
			wantErr: "contract addr",
		},
		{
			name: "invalid - ZeroGasActors bad sender",
			genState: &sudo.GenesisState{
				Sudoers: sudo.Sudoers{
					Root:      addrStrs[0],
					Contracts: []string{},
				},
				ZeroGasActors: &sudo.ZeroGasActors{
					Senders: []string{"invalid"},
				},
			},
			wantErr: "ZeroGasActors stateless validation error",
		},
		{
			name: "invalid - ZeroGasActors bad contract",
			genState: &sudo.GenesisState{
				Sudoers: sudo.Sudoers{
					Root:      addrStrs[0],
					Contracts: []string{},
				},
				ZeroGasActors: &sudo.ZeroGasActors{
					Contracts: []string{"0xBAD"},
				},
			},
			wantErr: "ZeroGasActors stateless validation error",
		},
		{
			name: "invalid - ZeroGasActors with empty senders and contracts",
			genState: &sudo.GenesisState{
				Sudoers: sudo.Sudoers{
					Root:      addrStrs[0],
					Contracts: []string{},
				},
				ZeroGasActors: &sudo.ZeroGasActors{
					Senders:   []string{},
					Contracts: []string{},
				},
			},
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			err := tc.genState.Validate()
			if len(tc.wantErr) != 0 {
				s.Require().ErrorContains(err, tc.wantErr)
				return
			}
			s.Require().NoError(err)
		})
	}
}
