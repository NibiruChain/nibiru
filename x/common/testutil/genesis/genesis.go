package genesis

import (
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	gov "github.com/cosmos/cosmos-sdk/x/gov/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types/v1"

	"github.com/NibiruChain/nibiru/v2/app"
	"github.com/NibiruChain/nibiru/v2/x/common/denoms"
	"github.com/NibiruChain/nibiru/v2/x/common/testutil/testapp"
)

/*
	NewTestGenesisState returns 'NewGenesisState' using the default

genesis as input. The blockchain genesis state is represented as a map from module
identifier strings to raw json messages.
*/
func NewTestGenesisState(encodingConfig app.EncodingConfig) app.GenesisState {
	codec := encodingConfig.Codec
	genState := app.NewDefaultGenesisState(codec)

	// Set short voting period to allow fast gov proposals in tests
	var govGenState govtypes.GenesisState
	codec.MustUnmarshalJSON(genState[gov.ModuleName], &govGenState)
	*govGenState.Params.VotingPeriod = time.Second * 20
	govGenState.Params.MinDeposit = sdk.NewCoins(sdk.NewInt64Coin(denoms.NIBI, 1_000_000)) // min deposit of 1 NIBI
	genState[gov.ModuleName] = codec.MustMarshalJSON(&govGenState)

	testapp.SetDefaultSudoGenesis(genState)

	return genState
}
