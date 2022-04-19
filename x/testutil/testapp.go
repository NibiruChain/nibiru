package testutil

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/NibiruChain/nibiru/app"
	"github.com/cosmos/cosmos-sdk/simapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/libs/log"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"
	tmtypes "github.com/tendermint/tendermint/types"
	tmdb "github.com/tendermint/tm-db"
)

// New creates application instance with in-memory database and disabled logging.
func New(shouldUseDefaultGenesis bool) *app.NibiruApp {
	userHomeDir, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}

	nodeHome := filepath.Join(userHomeDir, ".nibid")
	db := tmdb.NewMemDB()
	logger := log.NewNopLogger()

	encoding := app.MakeTestEncodingConfig()

	a := app.NewNibiruApp(
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

	var stateBytes []byte = []byte("{}")
	if shouldUseDefaultGenesis {
		genesisState := app.NewDefaultGenesisState(encoding.Marshaler)
		stateBytes, err = json.MarshalIndent(genesisState, "", " ")
		if err != nil {
			panic(err)
		}
	}

	// InitChain updates deliverState which is required when app.NewContext is called
	a.InitChain(abci.RequestInitChain{
		ConsensusParams: DefaultConsensusParams,
		AppStateBytes:   stateBytes,
	})

	return a
}

func NewNibiruApp(shouldUseDefaultGenesis bool) (*app.NibiruApp, sdk.Context) {
	newNibiruApp := New(shouldUseDefaultGenesis)
	ctx := newNibiruApp.NewContext(false, tmproto.Header{})

	return newNibiruApp, ctx
}

var DefaultConsensusParams = &abci.ConsensusParams{
	Block: &abci.BlockParams{
		MaxBytes: 200000,
		MaxGas:   2000000,
	},
	Evidence: &tmproto.EvidenceParams{
		MaxAgeNumBlocks: 302400,
		MaxAgeDuration:  504 * time.Hour, // 3 weeks is the max duration
		MaxBytes:        10000,
	},
	Validator: &tmproto.ValidatorParams{
		PubKeyTypes: []string{
			tmtypes.ABCIPubKeyTypeEd25519,
		},
	},
}
