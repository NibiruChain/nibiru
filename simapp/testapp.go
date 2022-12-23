package simapp

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/simapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/libs/log"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"
	tmdb "github.com/tendermint/tm-db"

	"github.com/NibiruChain/nibiru/x/common"
)

// NewTestNibiruApp creates an application instance ('app.NibiruApp') with an in-memory
// database ('tmdb.MemDB') and disabled logging. It either uses the application's
// default genesis state or a blank one.
func NewTestNibiruApp(shouldUseDefaultGenesis bool) *NibiruTestApp {
	encoding := simapp.MakeTestEncodingConfig()
	var appGenesis GenesisState
	if shouldUseDefaultGenesis {
		appGenesis = NewDefaultGenesisState(encoding.Marshaler)
	}
	return NewTestNibiruAppWithGenesis(appGenesis)
}

// NewTestNibiruAppAndContext creates an 'app.NibiruApp' instance with an in-memory
// 'tmdb.MemDB' and fresh 'sdk.Context'.
func NewTestNibiruAppAndContext(shouldUseDefaultGenesis bool) (*NibiruTestApp, sdk.Context) {
	newNibiruApp := NewTestNibiruApp(shouldUseDefaultGenesis)
	ctx := newNibiruApp.NewContext(false, tmproto.Header{})

	newNibiruApp.OracleKeeper.SetPrice(ctx, common.Pair_BTC_NUSD.String(), sdk.NewDec(20000))
	newNibiruApp.OracleKeeper.SetPrice(ctx, "xxx:yyy", sdk.NewDec(20000))

	return newNibiruApp, ctx
}

// NewTestNibiruAppWithGenesis initializes a chain with the given genesis state to
// creates an application instance ('app.NibiruApp'). This app uses an
// in-memory database ('tmdb.MemDB') and has logging disabled.
func NewTestNibiruAppWithGenesis(gen GenesisState) *NibiruTestApp {
	userHomeDir, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}

	nodeHome := filepath.Join(userHomeDir, ".nibid")
	db := tmdb.NewMemDB()
	logger := log.NewNopLogger()

	encoding := MakeTestEncodingConfig()

	nibiruApp := NewNibiruTestApp(
		logger,
		db,
		/*traceStore=*/ nil,
		/*loadLatest=*/ true,
		/*skipUpgradeHeights=*/ map[int64]bool{},
		/*homePath=*/ nodeHome,
		/*invCheckPeriod=*/ 0,
		/*encodingConfig=*/ encoding,
		/*appOpts=*/ simapp.EmptyAppOptions{},
	)

	stateBytes, err := json.MarshalIndent(gen, "", " ")
	if err != nil {
		panic(err)
	}

	nibiruApp.InitChain(abci.RequestInitChain{
		ConsensusParams: simapp.DefaultConsensusParams,
		AppStateBytes:   stateBytes,
	})

	return nibiruApp
}

// ----------------------------------------------------------------------------
// Genesis
// ----------------------------------------------------------------------------

const (
	GenOracleAddress = "nibi1zuxt7fvuxgj69mjxu3auca96zemqef5u2yemly"
)

/*
	NewTestGenesisStateFromDefault returns 'NewGenesisState' using the default

genesis as input. The blockchain genesis state is represented as a map from module
identifier strings to raw json messages.
*/
func NewTestGenesisStateFromDefault() GenesisState {
	encodingConfig := MakeTestEncodingConfig()
	codec := encodingConfig.Marshaler
	genState := NewDefaultGenesisState(codec)
	return NewTestGenesisState(codec, genState)
}

/*
NewTestGenesisState transforms 'inGenState' to add genesis parameter changes
that are well suited to integration testing, then returns the transformed genesis.
The blockchain genesis state is represented as a map from module identifier strings
to raw json messages.

Args:
- codec: Serializer for the module genesis state proto.Messages
- inGenState: Input genesis state before the custom test setup is applied
*/
func NewTestGenesisState(codec codec.Codec, inGenState GenesisState,
) (testGenState GenesisState) {
	testGenState = inGenState

	// Set short voting period to allow fast gov proposals in tests
	var govGenState govtypes.GenesisState
	codec.MustUnmarshalJSON(testGenState[govtypes.ModuleName], &govGenState)
	govGenState.VotingParams.VotingPeriod = time.Second * 20
	govGenState.DepositParams.MinDeposit = sdk.NewCoins(sdk.NewInt64Coin(common.DenomNIBI, 1*common.Precision)) // min deposit of 1 NIBI
	testGenState[govtypes.ModuleName] = codec.MustMarshalJSON(&govGenState)

	return testGenState
}
