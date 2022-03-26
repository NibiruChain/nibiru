package testutil

import (
	"os"
	"path/filepath"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/MatrixDao/matrix/app"
	"github.com/cosmos/cosmos-sdk/simapp"
	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/libs/log"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"
	tmtypes "github.com/tendermint/tendermint/types"
	tmdb "github.com/tendermint/tm-db"
)

// New creates application instance with in-memory database and disabled logging.
func New() *app.App {
	userHomeDir, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}

	nodeHome := filepath.Join(userHomeDir, ".matrix")
	db := tmdb.NewMemDB()
	logger := log.NewNopLogger()

	encoding := app.MakeTestEncodingConfig()

	a := app.NewApp(logger, db, nil, true, map[int64]bool{}, nodeHome, 0, encoding,
		simapp.EmptyAppOptions{})

	// InitChain updates deliverState which is required when app.NewContext is called
	a.InitChain(abci.RequestInitChain{
		ConsensusParams: defaultConsensusParams,
		AppStateBytes:   []byte("{}"),
	})

	return a
}

func NewApp() (*app.App, sdk.Context) {
	newApp := New()
	ctx := newApp.NewContext(false, tmproto.Header{})

	return newApp, ctx
}

var defaultConsensusParams = &abci.ConsensusParams{
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
