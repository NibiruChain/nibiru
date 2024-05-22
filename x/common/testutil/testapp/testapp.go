package testapp

import (
	"encoding/json"
	"time"

	"cosmossdk.io/math"
	tmdb "github.com/cometbft/cometbft-db"
	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cometbft/cometbft/libs/log"
	tmproto "github.com/cometbft/cometbft/types"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"

	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/app/appconst"
	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/common/denoms"
	"github.com/NibiruChain/nibiru/x/common/testutil"
	epochstypes "github.com/NibiruChain/nibiru/x/epochs/types"
	inflationtypes "github.com/NibiruChain/nibiru/x/inflation/types"
	sudotypes "github.com/NibiruChain/nibiru/x/sudo/types"
)

func init() {
	EnsureNibiruPrefix()
}

// NewNibiruTestAppAndContext creates an 'app.NibiruApp' instance with an
// in-memory 'tmdb.MemDB' and fresh 'sdk.Context'.
func NewNibiruTestAppAndContext() (*app.NibiruApp, sdk.Context) {
	// Prevent "invalid Bech32 prefix; expected nibi, got ...." error
	EnsureNibiruPrefix()

	// Set up base app
	encoding := app.MakeEncodingConfig()
	var appGenesis app.GenesisState = app.NewDefaultGenesisState(encoding.Codec)
	genModEpochs := epochstypes.DefaultGenesisFromTime(time.Now().UTC())

	// Set happy genesis: epochs
	appGenesis[epochstypes.ModuleName] = encoding.Codec.MustMarshalJSON(
		genModEpochs,
	)

	// Set happy genesis: sudo
	sudoGenesis := new(sudotypes.GenesisState)
	sudoGenesis.Sudoers = DefaultSudoers()
	appGenesis[sudotypes.ModuleName] = encoding.Codec.MustMarshalJSON(sudoGenesis)

	app := NewNibiruTestApp(appGenesis)
	ctx := NewContext(app)

	// Set defaults for certain modules.
	app.OracleKeeper.SetPrice(ctx, asset.Registry.Pair(denoms.BTC, denoms.NUSD), math.LegacyNewDec(20000))
	app.OracleKeeper.SetPrice(ctx, "xxx:yyy", math.LegacyNewDec(20000))
	app.SudoKeeper.Sudoers.Set(ctx, DefaultSudoers())

	return app, ctx
}

// NewContext: Returns a fresh sdk.Context corresponding to the given NibiruApp.
func NewContext(nibiru *app.NibiruApp) sdk.Context {
	return nibiru.NewContext(false, tmproto.Header{
		Height: 1,
		Time:   time.Now().UTC(),
	})
}

// DefaultSudoers: State for the x/sudo module for the default test app.
func DefaultSudoers() sudotypes.Sudoers {
	addr := DefaultSudoRoot().String()
	return sudotypes.Sudoers{
		Root:      addr,
		Contracts: []string{addr},
	}
}

func DefaultSudoRoot() sdk.AccAddress {
	return sdk.MustAccAddressFromBech32(testutil.ADDR_SUDO_ROOT)
}

// SetDefaultSudoGenesis: Sets the sudo module genesis state to a valid
// default. See "DefaultSudoers".
func SetDefaultSudoGenesis(gen app.GenesisState) {
	sudoGen := new(sudotypes.GenesisState)
	encoding := app.MakeEncodingConfig()
	encoding.Codec.MustUnmarshalJSON(gen[sudotypes.ModuleName], sudoGen)
	if err := sudoGen.Validate(); err != nil {
		sudoGen.Sudoers = DefaultSudoers()
		gen[sudotypes.ModuleName] = encoding.Codec.MustMarshalJSON(sudoGen)
	}
}

// NewNibiruTestAppAndZeroTimeCtx: Runs NewNibiruTestAppAndZeroTimeCtx with the
// block time set to time zero.
func NewNibiruTestAppAndContextAtTime(startTime time.Time) (*app.NibiruApp, sdk.Context) {
	app, _ := NewNibiruTestAppAndContext()
	ctx := NewContext(app).WithBlockTime(startTime)
	return app, ctx
}

// NewNibiruTestApp initializes a chain with the given genesis state to
// creates an application instance ('app.NibiruApp'). This app uses an
// in-memory database ('tmdb.MemDB') and has logging disabled.
func NewNibiruTestApp(gen app.GenesisState, baseAppOptions ...func(*baseapp.BaseApp)) *app.NibiruApp {
	db := tmdb.NewMemDB()
	logger := log.NewNopLogger()

	encoding := app.MakeEncodingConfig()
	SetDefaultSudoGenesis(gen)

	app := app.NewNibiruApp(
		logger,
		db,
		/*traceStore=*/ nil,
		/*loadLatest=*/ true,
		encoding,
		/*appOpts=*/ sims.EmptyAppOptions{},
		baseAppOptions...,
	)

	gen, err := GenesisStateWithSingleValidator(encoding.Codec, gen)
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
	nibiruPrefix := appconst.AccountAddressPrefix
	if csdkConfig.GetBech32AccountAddrPrefix() != nibiruPrefix {
		app.SetPrefixes(nibiruPrefix)
	}
}
