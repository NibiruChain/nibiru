package testutil

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/cosmos/cosmos-sdk/simapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/libs/log"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"
	tmdb "github.com/tendermint/tm-db"

	"github.com/NibiruChain/nibiru/app"
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
