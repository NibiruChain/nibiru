package testapp

import (
	"encoding/json"
	"time"

	"cosmossdk.io/math"
	tmdb "github.com/cometbft/cometbft-db"
	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cometbft/cometbft/libs/log"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	auth "github.com/cosmos/cosmos-sdk/x/auth/types"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"

	"github.com/NibiruChain/nibiru/v2/app"
	"github.com/NibiruChain/nibiru/v2/app/appconst"
	"github.com/NibiruChain/nibiru/v2/x/common/asset"
	"github.com/NibiruChain/nibiru/v2/x/common/denoms"
	"github.com/NibiruChain/nibiru/v2/x/common/testutil"
	epochstypes "github.com/NibiruChain/nibiru/v2/x/epochs/types"
	inflationtypes "github.com/NibiruChain/nibiru/v2/x/inflation/types"
	sudotypes "github.com/NibiruChain/nibiru/v2/x/sudo/types"
)

// NewNibiruTestAppAndContext creates an 'app.NibiruApp' instance with an
// in-memory 'tmdb.MemDB' and fresh 'sdk.Context'.
func NewNibiruTestAppAndContext() (*app.NibiruApp, sdk.Context) {
	// Set up base app
	encoding := app.MakeEncodingConfig()
	var appGenesis app.GenesisState = app.ModuleBasics.DefaultGenesis(encoding.Codec)
	genModEpochs := epochstypes.DefaultGenesisFromTime(time.Now().UTC())

	// Set happy genesis: epochs
	appGenesis[epochstypes.ModuleName] = encoding.Codec.MustMarshalJSON(
		genModEpochs,
	)

	// Set happy genesis: sudo
	sudoGenesis := new(sudotypes.GenesisState)
	sudoGenesis.Sudoers = sudotypes.Sudoers{
		Root:      testutil.ADDR_SUDO_ROOT,
		Contracts: []string{testutil.ADDR_SUDO_ROOT},
	}
	appGenesis[sudotypes.ModuleName] = encoding.Codec.MustMarshalJSON(sudoGenesis)

	app := NewNibiruTestApp(appGenesis)
	ctx := NewContext(app)

	// Set defaults for certain modules.
	app.OracleKeeper.SetPrice(ctx, asset.Registry.Pair(denoms.BTC, denoms.NUSD), math.LegacyNewDec(20000))
	app.OracleKeeper.SetPrice(ctx, "xxx:yyy", math.LegacyNewDec(20000))

	return app, ctx
}

// NewContext: Returns a fresh sdk.Context corresponding to the given NibiruApp.
func NewContext(nibiru *app.NibiruApp) sdk.Context {
	blockHeader := tmproto.Header{
		Height: 1,
		Time:   time.Now().UTC(),
	}
	ctx := nibiru.NewContext(false, blockHeader)

	// Make sure there's a block proposer on the context.
	blockHeader.ProposerAddress = FirstBlockProposer(nibiru, ctx)
	ctx = ctx.WithBlockHeader(blockHeader)

	return ctx
}

func FirstBlockProposer(
	chain *app.NibiruApp, ctx sdk.Context,
) (proposerAddr sdk.ConsAddress) {
	maxQueryCount := uint32(10)
	valopers := chain.StakingKeeper.GetValidators(ctx, maxQueryCount)
	valAddrBz := valopers[0].GetOperator().Bytes()
	return sdk.ConsAddress(valAddrBz)
}

// SetDefaultSudoGenesis: Sets the sudo module genesis state to a valid
// default. See "DefaultSudoers".
func SetDefaultSudoGenesis(gen app.GenesisState) {
	encoding := app.MakeEncodingConfig()

	var sudoGen sudotypes.GenesisState
	encoding.Codec.MustUnmarshalJSON(gen[sudotypes.ModuleName], &sudoGen)
	if err := sudoGen.Validate(); err != nil {
		sudoGen.Sudoers = sudotypes.Sudoers{
			Root:      testutil.ADDR_SUDO_ROOT,
			Contracts: []string{testutil.ADDR_SUDO_ROOT},
		}
		gen[sudotypes.ModuleName] = encoding.Codec.MustMarshalJSON(&sudoGen)
	}
}

// NewNibiruTestApp initializes a chain with the given genesis state to
// creates an application instance ('app.NibiruApp'). This app uses an
// in-memory database ('tmdb.MemDB') and has logging disabled.
func NewNibiruTestApp(gen app.GenesisState, baseAppOptions ...func(*baseapp.BaseApp)) *app.NibiruApp {
	SetDefaultSudoGenesis(gen)

	app := app.NewNibiruApp(
		log.NewNopLogger(),
		tmdb.NewMemDB(),
		/*traceStore=*/ nil,
		/*loadLatest=*/ true,
		/*appOpts=*/ sims.EmptyAppOptions{},
		baseAppOptions...,
	)

	gen, err := GenesisStateWithSingleValidator(app.AppCodec(), gen)
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

// FundFeeCollector funds the module account that collects gas fees with some
// amount of "unibi", the gas token.
func FundFeeCollector(
	bk bankkeeper.Keeper, ctx sdk.Context, amount math.Int,
) error {
	return FundModuleAccount(
		bk,
		ctx,
		auth.FeeCollectorName,
		sdk.NewCoins(sdk.NewCoin(appconst.BondDenom, amount)),
	)
}
