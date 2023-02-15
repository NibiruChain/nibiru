package testapp

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
	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/common/denoms"
)

// NewNibiruTestAppAndContext creates an 'app.NibiruApp' instance with an in-memory
// 'tmdb.MemDB' and fresh 'sdk.Context'.
func NewNibiruTestAppAndContext(shouldUseDefaultGenesis bool) (*app.NibiruApp, sdk.Context) {
	encoding := simapp.MakeTestEncodingConfig()
	var appGenesis app.GenesisState
	if shouldUseDefaultGenesis {
		appGenesis = app.NewDefaultGenesisState(encoding.Marshaler)
	}

	newNibiruApp := NewNibiruTestApp(appGenesis)
	ctx := newNibiruApp.NewContext(false, tmproto.Header{})

	newNibiruApp.OracleKeeper.SetPrice(ctx, asset.Registry.Pair(denoms.BTC, denoms.NUSD), sdk.NewDec(20000))
	// newNibiruApp.OracleKeeper.SetPrice(ctx, asset.AssetRegistry.Pair(denoms.NIBI, denoms.NUSD), sdk.NewDec(10))
	newNibiruApp.OracleKeeper.SetPrice(ctx, "xxx:yyy", sdk.NewDec(20000))

	return newNibiruApp, ctx
}

// newNibiruTestApp initializes a chain with the given genesis state to
// creates an application instance ('app.NibiruApp'). This app uses an
// in-memory database ('tmdb.MemDB') and has logging disabled.
func NewNibiruTestApp(gen app.GenesisState) *app.NibiruApp {
	userHomeDir, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}

	nodeHome := filepath.Join(userHomeDir, ".nibid")
	db := tmdb.NewMemDB()
	logger := log.NewNopLogger()

	encoding := app.MakeTestEncodingConfig()

	nibiruApp := app.NewNibiruApp(
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
