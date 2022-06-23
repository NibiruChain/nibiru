package testapp

import (
	"encoding/json"
	"os"
	"path/filepath"

	"time"

	"github.com/cosmos/cosmos-sdk/simapp"
	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/libs/log"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"
	tmdb "github.com/tendermint/tm-db"

	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/x/common"
	pricefeedtypes "github.com/NibiruChain/nibiru/x/pricefeed/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/cosmos-sdk/codec"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
)

// New creates application instance with in-memory database and disabled logging.
func NewTestApp(shouldUseDefaultGenesis bool) *app.NibiruApp {
	userHomeDir, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}

	nodeHome := filepath.Join(userHomeDir, ".nibid")
	db := tmdb.NewMemDB()
	logger := log.NewNopLogger()

	encoding := app.MakeTestEncodingConfig()

	testApp := app.NewNibiruApp(
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

	var stateBytes = []byte("{}")
	if shouldUseDefaultGenesis {
		genesisState := app.NewDefaultGenesisState(encoding.Marshaler)
		stateBytes, err = json.MarshalIndent(genesisState, "", " ")
		if err != nil {
			panic(err)
		}
	}

	// InitChain updates deliverState which is required when app.NewContext is called
	testApp.InitChain(abci.RequestInitChain{
		ConsensusParams: simapp.DefaultConsensusParams,
		AppStateBytes:   stateBytes,
	})

	return testApp
}

/* NewNibiruApp creates an 'app.NibiruApp' instance with an in-memory
   'tmdb.MemDB' and fresh 'sdk.Context'. */
func NewNibiruApp(shouldUseDefaultGenesis bool) (*app.NibiruApp, sdk.Context) {
	newNibiruApp := NewTestApp(shouldUseDefaultGenesis)
	ctx := newNibiruApp.NewContext(false, tmproto.Header{})

	return newNibiruApp, ctx
}

func NewTestAppWithGenesis(gen app.GenesisState) *app.NibiruApp {
	userHomeDir, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}

	nodeHome := filepath.Join(userHomeDir, ".nibid")
	db := tmdb.NewMemDB()
	logger := log.NewNopLogger()

	encoding := app.MakeTestEncodingConfig()

	testApp := app.NewNibiruApp(
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

	testApp.InitChain(abci.RequestInitChain{
		ConsensusParams: simapp.DefaultConsensusParams,
		AppStateBytes:   stateBytes,
	})

	return testApp
}

// ----------------------------------------------------------------------------
// Genesis
// ----------------------------------------------------------------------------

const (
	GenOracleAddress  = "nibi1zuxt7fvuxgj69mjxu3auca96zemqef5u2yemly"
	GenOracleMnemonic = "kit soon capital dry sadness balance rival embark behind coast online struggle deer crush hospital during man monkey prison action custom wink utility arrive"
)

func NewTestGenesisStateFromDefault() app.GenesisState {
	encodingConfig := app.MakeTestEncodingConfig()
	codec := encodingConfig.Marshaler
	genState := app.NewDefaultGenesisState(codec)
	return NewTestGenesisState(codec, genState)
}

func NewTestGenesisState(codec codec.Codec, inGenState app.GenesisState,
) (testGenState app.GenesisState) {

	testGenState = inGenState

	// Set short voting period to allow fast gov proposals in tests
	var govGenState govtypes.GenesisState
	codec.MustUnmarshalJSON(testGenState[govtypes.ModuleName], &govGenState)
	govGenState.VotingParams.VotingPeriod = time.Second * 20
	govGenState.DepositParams.MinDeposit = sdk.NewCoins(
		sdk.NewInt64Coin(common.DenomGov, 1_000_000)) // min deposit of 1 NIBI
	bz := codec.MustMarshalJSON(&govGenState)
	testGenState[govtypes.ModuleName] = bz

	// pricefeed genesis state
	pfGenState := pricefeedtypes.GenesisState{}
	codec.MustUnmarshalJSON(testGenState[pricefeedtypes.ModuleName], &pfGenState)
	pfGenState = *GenesisPricefeed()
	bz = codec.MustMarshalJSON(&pfGenState)
	testGenState[pricefeedtypes.ModuleName] = bz

	return testGenState
}

// GenesisPricefeed returns an x/pricefeed GenesisState to specify the module parameters.
func GenesisPricefeed() *pricefeedtypes.GenesisState {
	oracle := sdk.MustAccAddressFromBech32(GenOracleAddress)
	oracles := []sdk.AccAddress{oracle}

	var gen pricefeedtypes.GenesisState
	pairs := pricefeedtypes.DefaultPairs
	gen.Params.Pairs = pairs
	gen.PostedPrices = []pricefeedtypes.PostedPrice{
		{
			PairID:        pairs[0].String(), // PairGovStable
			OracleAddress: oracle,
			Price:         sdk.NewDec(10),
			Expiry:        time.Now().Add(1 * time.Hour),
		},
		{
			PairID:        pairs[1].String(), // PairCollStable
			OracleAddress: oracle,
			Price:         sdk.OneDec(),
			Expiry:        time.Now().Add(1 * time.Hour),
		},
	}
	gen.GenesisOracles = oracles

	return &gen
}
