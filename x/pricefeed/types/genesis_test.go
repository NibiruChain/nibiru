package types

import (
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	tmtypes "github.com/tendermint/tendermint/types"

	"github.com/NibiruChain/nibiru/x/common"
)

func TestGenesisState_Validate(t *testing.T) {
	mockPrivKey := tmtypes.NewMockPV()
	pubkey, err := mockPrivKey.GetPubKey()
	require.NoError(t, err)
	addr := sdk.AccAddress(pubkey.Address())
	now := time.Now()

	examplePairs := common.NewAssetPairs("xrp:bnb")
	for _, tc := range []struct {
		desc     string
		genState *GenesisState
		valid    bool
	}{
		{
			desc:     "default is valid",
			genState: DefaultGenesis(),
			valid:    true,
		},
		{
			desc:     "valid genesis state - empty",
			genState: &GenesisState{},
			valid:    true,
		},
		{
			desc: "valid genesis state - full",
			genState: NewGenesisState(
				NewParams(
					/*pairs=*/ examplePairs,
				),
				[]PostedPrice{NewPostedPrice(examplePairs[0], addr, sdk.OneDec(), now)},
			),
			valid: true,
		},
		{
			desc: "invalid posted price - no valid pairs",
			genState: NewGenesisState(
				NewParams(
					/*pairs=*/ common.AssetPairs{},
				),
				[]PostedPrice{NewPostedPrice(examplePairs[0], nil, sdk.OneDec(), now)},
			),
			valid: false,
		},
		{
			desc: "duplicated posted price at same timestamp - invalid",
			genState: NewGenesisState(
				NewParams(
					/*pairs=*/ common.AssetPairs{},
				),
				[]PostedPrice{
					NewPostedPrice(examplePairs[0], addr, sdk.OneDec(), now),
					NewPostedPrice(examplePairs[0], addr, sdk.OneDec(), now),
				},
			),
			valid: false,
		},

		// this line is used by starport scaffolding # types/genesis/testcase
	} {
		t.Run(tc.desc, func(t *testing.T) {
			err := tc.genState.Validate()
			if tc.valid {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}
