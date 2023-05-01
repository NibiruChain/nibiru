package testapp

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/cosmos/cosmos-sdk/simapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/libs/log"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"
	tmdb "github.com/tendermint/tm-db"

	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/common/denoms"
	inflationtypes "github.com/NibiruChain/nibiru/x/inflation/types"
)

func NewNibiruTestAppWithContext(appGenesis app.GenesisState) (*app.NibiruApp, sdk.Context) {
	app := NewNibiruTestApp(appGenesis)
	ctx := app.NewContext(false, tmproto.Header{
		Height: 1,
	})
	// app.BeginBlock(abci.RequestBeginBlock{Header: tmproto.Header{Height: 1}})

	app.OracleKeeper.SetPrice(ctx, asset.Registry.Pair(denoms.BTC, denoms.NUSD), sdk.NewDec(20000))
	app.OracleKeeper.SetPrice(ctx, "xxx:yyy", sdk.NewDec(20000))

	return app, ctx
}

// NewNibiruTestAppAndContext creates an 'app.NibiruApp' instance with an in-memory
// 'tmdb.MemDB' and fresh 'sdk.Context'.
func NewNibiruTestAppAndContext(shouldUseDefaultGenesis bool) (*app.NibiruApp, sdk.Context) {
	encoding := app.MakeTestEncodingConfig()
	var appGenesis app.GenesisState
	if shouldUseDefaultGenesis {
		appGenesis = app.NewDefaultGenesisState(encoding.Marshaler)
	}

	return NewNibiruTestAppWithContext(appGenesis)
}

// NewNibiruTestApp initializes a chain with the given genesis state to
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

	app := app.NewNibiruApp(
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

	app.InitChain(abci.RequestInitChain{
		ConsensusParams: simapp.DefaultConsensusParams,
		AppStateBytes:   stateBytes,
	})

	return app
}

// FundAccount is a utility function that funds an account by minting and
// sending the coins to the address. This should be used for testing purposes
// only!
func FundAccount(bankKeeper bankkeeper.Keeper, ctx sdk.Context, addr sdk.AccAddress, amounts sdk.Coins) error {
	if err := bankKeeper.MintCoins(ctx, inflationtypes.ModuleName, amounts); err != nil {
		return err
	}

	return bankKeeper.SendCoinsFromModuleToAccount(ctx, inflationtypes.ModuleName, addr, amounts)
}

// FundModuleAccount is a utility function that funds a module account by
// minting and sending the coins to the address. This should be used for testing
// purposes only!
func FundModuleAccount(bankKeeper bankkeeper.Keeper, ctx sdk.Context, recipientMod string, amounts sdk.Coins) error {
	if err := bankKeeper.MintCoins(ctx, inflationtypes.ModuleName, amounts); err != nil {
		return err
	}

	return bankKeeper.SendCoinsFromModuleToModule(ctx, inflationtypes.ModuleName, recipientMod, amounts)
}
