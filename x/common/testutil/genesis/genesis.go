package genesis

import (
	"time"

	"github.com/cosmos/cosmos-sdk/codec"
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
func NewTestGenesisState(appCodec codec.Codec) app.GenesisState {
	genState := app.ModuleBasics.DefaultGenesis(appCodec)

	// Set short voting period to allow fast gov proposals in tests
	var govGenState govtypes.GenesisState
	appCodec.MustUnmarshalJSON(genState[gov.ModuleName], &govGenState)
	*govGenState.Params.VotingPeriod = 20 * time.Second
	govGenState.Params.MinDeposit = sdk.NewCoins(sdk.NewInt64Coin(denoms.NIBI, 1e6)) // min deposit of 1 NIBI
	genState[gov.ModuleName] = appCodec.MustMarshalJSON(&govGenState)

	testapp.SetDefaultSudoGenesis(genState)

	return genState
}
