package genesis

import (
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/x/common/denoms"
)

/*
	NewTestGenesisState returns 'NewGenesisState' using the default

genesis as input. The blockchain genesis state is represented as a map from module
identifier strings to raw json messages.
*/
func NewTestGenesisState() app.GenesisState {
	encodingConfig := app.MakeTestEncodingConfig()
	codec := encodingConfig.Marshaler
	genState := app.NewDefaultGenesisState(codec)

	// Set short voting period to allow fast gov proposals in tests
	var govGenState govtypes.GenesisState
	codec.MustUnmarshalJSON(genState[govtypes.ModuleName], &govGenState)
	govGenState.VotingParams.VotingPeriod = time.Second * 20
	govGenState.DepositParams.MinDeposit = sdk.NewCoins(sdk.NewInt64Coin(denoms.NIBI, 1_000_000)) // min deposit of 1 NIBI
	genState[govtypes.ModuleName] = codec.MustMarshalJSON(&govGenState)

	return genState
}
