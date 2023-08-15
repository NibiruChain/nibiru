package testapp

import (
	"encoding/json"
	"time"

	tmdb "github.com/cometbft/cometbft-db"
	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cometbft/cometbft/libs/log"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	"github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"

	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/common/denoms"
	epochstypes "github.com/NibiruChain/nibiru/x/epochs/types"
	inflationtypes "github.com/NibiruChain/nibiru/x/inflation/types"
)

// NewNibiruTestAppAndContext creates an 'app.NibiruApp' instance with an
// in-memory 'tmdb.MemDB' and fresh 'sdk.Context'.
func NewNibiruTestAppAndContext() (*app.NibiruApp, sdk.Context) {
	encoding := app.MakeEncodingConfigAndRegister()
	var appGenesis app.GenesisState = app.NewDefaultGenesisState(encoding.Marshaler)
	genModEpochs := epochstypes.DefaultGenesisFromTime(time.Now().UTC())
	appGenesis[epochstypes.ModuleName] = encoding.Marshaler.MustMarshalJSON(
		genModEpochs,
	)
	app := NewNibiruTestApp(appGenesis)
	ctx := NewContext(app)

	app.OracleKeeper.SetPrice(ctx, asset.Registry.Pair(denoms.BTC, denoms.NUSD), sdk.NewDec(20000))
	app.OracleKeeper.SetPrice(ctx, "xxx:yyy", sdk.NewDec(20000))

	return app, ctx
}

func NewContext(nibiru *app.NibiruApp) sdk.Context {
	return nibiru.NewContext(false, tmproto.Header{
		Height: 1,
		Time:   time.Now().UTC(),
	})
}

// NewNibiruTestAppAndZeroTimeCtx: Runs NewNibiruTestAppAndZeroTimeCtx with the
// block time set to time zero.
func NewNibiruTestAppAndContextAtTime(startTime time.Time) (*app.NibiruApp, sdk.Context) {
	app, ctx := NewNibiruTestAppAndContext()
	ctx = NewContext(app).WithBlockTime(startTime)
	return app, ctx
}

// NewNibiruTestApp initializes a chain with the given genesis state to
// creates an application instance ('app.NibiruApp'). This app uses an
// in-memory database ('tmdb.MemDB') and has logging disabled.
func NewNibiruTestApp(gen app.GenesisState) *app.NibiruApp {
	db := tmdb.NewMemDB()
	logger := log.NewNopLogger()

	encoding := app.MakeEncodingConfigAndRegister()
	app := app.NewNibiruApp(
		logger,
		db,
		/*traceStore=*/ nil,
		/*loadLatest=*/ true,
		encoding,
		/*appOpts=*/ sims.EmptyAppOptions{},
	)

	gen, err := GenesisStateWithSingleValidator(encoding.Marshaler, gen)
	if err != nil {
		panic(err)
	}

	stateBytes, err := json.MarshalIndent(gen, "", " ")
	if err != nil {
		panic(err)
	}

	app.InitChain(abci.RequestInitChain{
		ConsensusParams: sims.DefaultConsensusParams,
		AppStateBytes:   stateBytes,
	})

	return app
}

// FundAccount is a utility function that funds an account by minting and
// sending the coins to the address. This should be used for testing purposes
// only!
func FundAccount(
	bankKeeper bankkeeper.Keeper, ctx sdk.Context, addr sdk.AccAddress,
	amounts sdk.Coins,
) error {
	if err := bankKeeper.MintCoins(ctx, inflationtypes.ModuleName, amounts); err != nil {
		return err
	}

	return bankKeeper.SendCoinsFromModuleToAccount(ctx, inflationtypes.ModuleName, addr, amounts)
}

// FundModuleAccount is a utility function that funds a module account by
// minting and sending the coins to the address. This should be used for testing
// purposes only!
func FundModuleAccount(
	bankKeeper bankkeeper.Keeper, ctx sdk.Context,
	recipientMod string, amounts sdk.Coins,
) error {
	if err := bankKeeper.MintCoins(ctx, inflationtypes.ModuleName, amounts); err != nil {
		return err
	}

	return bankKeeper.SendCoinsFromModuleToModule(ctx, inflationtypes.ModuleName, recipientMod, amounts)
}

// EnsureNibiruPrefix sets the account address prefix to Nibiru's rather than
// the default from the Cosmos-SDK, guaranteeing that tests will work with nibi
// addresses rather than cosmos ones (for Gaia).
func EnsureNibiruPrefix() {
	csdkConfig := sdk.GetConfig()
	nibiruPrefix := app.AccountAddressPrefix
	if csdkConfig.GetBech32AccountAddrPrefix() != nibiruPrefix {
		app.SetPrefixes(nibiruPrefix)
	}
}
