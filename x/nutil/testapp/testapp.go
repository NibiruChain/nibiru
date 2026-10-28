package testapp

import (
	"encoding/json"
	"maps"
	"time"

	sdkmath "cosmossdk.io/math"
	tmdb "github.com/cometbft/cometbft-db"

	abci "github.com/NibiruChain/nibiru/v2/bft/abci/types"
	"github.com/NibiruChain/nibiru/v2/bft/libs/log"
	tmproto "github.com/NibiruChain/nibiru/v2/bft/proto/tendermint/types"
	cmttypes "github.com/NibiruChain/nibiru/v2/bft/types"

	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/codec"
	codectypes "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/codec/types"
	cryptocodec "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/crypto/codec"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/crypto/keys/secp256k1"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/testutil/mock"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/testutil/sims"
	sdk "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types"
	auth "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/auth/types"
	banktypes "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/bank/types"
	govtypes "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/gov/types"
	govtypesv1 "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/gov/types/v1"
	stakingtypes "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/staking/types"

	bankkeeper "github.com/NibiruChain/nibiru/v2/x/bank/keeper"

	"github.com/NibiruChain/nibiru/v2/app"
	"github.com/NibiruChain/nibiru/v2/app/appconst"
	"github.com/NibiruChain/nibiru/v2/x/epochs"
	"github.com/NibiruChain/nibiru/v2/x/mint"
	"github.com/NibiruChain/nibiru/v2/x/nutil/denoms"
	"github.com/NibiruChain/nibiru/v2/x/nutil/testutil"
	"github.com/NibiruChain/nibiru/v2/x/oracle/types"
	"github.com/NibiruChain/nibiru/v2/x/sudo"
)

// NewNibiruTestAppAndContext creates an 'app.NibiruApp' instance with an
// in-memory 'tmdb.MemDB' and fresh 'sdk.Context'.
func NewNibiruTestAppAndContext() (*app.NibiruApp, sdk.Context) {
	app, _ := NewNibiruTestApp(app.GenesisState{})
	ctx := NewContext(app)

	// Set defaults for certain modules.
	app.OracleKeeper.SetPrice(ctx, types.NewPair(denoms.BTC, denoms.NUSD), sdkmath.LegacyNewDec(20000))
	app.OracleKeeper.SetPrice(ctx, "xxx:yyy", sdkmath.LegacyNewDec(20000))

	return app, ctx
}

// NewNibiruTestAppAndContextPreXConsensus is NewNibiruTestAppAndContext
// without the genesis x-consensus contract (see
// NewNibiruTestAppPreXConsensus). Use it for tests that assume an empty Wasm
// genesis, such as code ID 1 being the first code they store.
func NewNibiruTestAppAndContextPreXConsensus() (*app.NibiruApp, sdk.Context) {
	app, _ := NewNibiruTestAppPreXConsensus()
	ctx := NewContext(app)
	app.OracleKeeper.SetPrice(ctx, types.NewPair(denoms.BTC, denoms.NUSD), sdkmath.LegacyNewDec(20000))
	app.OracleKeeper.SetPrice(ctx, "xxx:yyy", sdkmath.LegacyNewDec(20000))
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

	var sudoGen sudo.GenesisState
	encoding.Codec.MustUnmarshalJSON(gen[sudo.ModuleName], &sudoGen)
	if err := sudoGen.Validate(); err != nil {
		sudoGen.Sudoers = sudo.Sudoers{
			Root:  testutil.ADDR_SUDO_ROOT,
			Roles: []sudo.RoleMembers{{Role: sudo.RoleWasmDeployer, Members: []string{testutil.ADDR_SUDO_ROOT}}},
		}
		gen[sudo.ModuleName] = encoding.Codec.MustMarshalJSON(&sudoGen)
	}
}

// NewNibiruTestApp initializes a chain with the given genesis state to
// creates an application instance ('app.NibiruApp'). This app uses an
// in-memory database ('tmdb.MemDB') and has logging disabled.
func NewNibiruTestApp(customGenesisOverride app.GenesisState) (
	nibiruApp *app.NibiruApp, gen app.GenesisState,
) {
	return NewNibiruTestAppWithDB(tmdb.NewMemDB(), customGenesisOverride)
}

// NewNibiruTestAppWithDB is NewNibiruTestApp on a caller-owned database, so a
// test can commit blocks, then open a second app instance on the same
// database to simulate a node restart.
func NewNibiruTestAppWithDB(db tmdb.DB, customGenesisOverride app.GenesisState) (
	nibiruApp *app.NibiruApp, gen app.GenesisState,
) {
	return newNibiruTestApp(db, customGenesisOverride, sims.DefaultConsensusParams)
}

// NewNibiruTestAppPreXConsensus returns a chain as it looked before the
// v2.22 upgrade: the x/consensus store holds sims.DefaultConsensusParams, like
// every test app, but no x-consensus contract exists and the Wasm store is
// empty. Use it for upgrade tests, and for Wasm tests that assume an empty
// Wasm genesis (code ID 1 free, no contracts).
//
// InitChain sends no consensus params, which is the only case where genesis
// skips the contract (keeper.ErrNoStoredParams); the params are stored
// right after, in the same uncommitted genesis state.
func NewNibiruTestAppPreXConsensus() (
	nibiruApp *app.NibiruApp, gen app.GenesisState,
) {
	nibiruApp, gen = newNibiruTestApp(tmdb.NewMemDB(), app.GenesisState{}, nil)
	nibiruApp.ConsensusKeeper.Set(
		NewContext(nibiruApp), sims.DefaultConsensusParams)
	return nibiruApp, gen
}

// newNibiruTestApp builds the test app on db and runs InitChain with the
// given consensus params (nil sends none).
func newNibiruTestApp(
	db tmdb.DB,
	customGenesisOverride app.GenesisState,
	consensusParams *tmproto.ConsensusParams,
) (nibiruApp *app.NibiruApp, gen app.GenesisState) {
	app := app.NewNibiruApp(
		log.NewNopLogger(),
		db,
		/*traceStore=*/ nil,
		/*loadLatest=*/ true,
		/*appOpts=*/ sims.EmptyAppOptions{},
	)

	// configure genesis from default
	gen = app.DefaultGenesis()

	// Set happy genesis: epochs
	genModEpochs := epochs.DefaultGenesisFromTime(time.Now().UTC())
	gen[epochs.ModuleName] = app.AppCodec().MustMarshalJSON(
		genModEpochs,
	)

	// Set happy genesis: sudo
	sudoGenesis := sudo.GenesisState{
		Sudoers: sudo.Sudoers{
			Root:  testutil.ADDR_SUDO_ROOT,
			Roles: []sudo.RoleMembers{{Role: sudo.RoleWasmDeployer, Members: []string{testutil.ADDR_SUDO_ROOT}}},
		},
	}
	gen[sudo.ModuleName] = app.AppCodec().MustMarshalJSON(&sudoGenesis)

	// Set happy genesis: gov
	// Set short voting period to allow fast gov proposals in tests
	var govGenesis govtypesv1.GenesisState
	app.AppCodec().MustUnmarshalJSON(gen[govtypes.ModuleName], &govGenesis)
	*govGenesis.Params.VotingPeriod = 20 * time.Second
	govGenesis.Params.MinDeposit = sdk.NewCoins(sdk.NewInt64Coin(appconst.DENOM_UNIBI, 1e6)) // min deposit of 1 NIBI
	gen[govtypes.ModuleName] = app.AppCodec().MustMarshalJSON(&govGenesis)

	maps.Copy(gen, customGenesisOverride)

	gen, err := GenesisStateWithSingleValidator(app.AppCodec(), gen)
	if err != nil {
		panic(err)
	}

	stateBytes, err := json.MarshalIndent(gen, "", " ")
	if err != nil {
		panic(err)
	}

	// Real genesis always has a time, and x-consensus is instantiated during
	// InitChain, where Wasm requires a positive block time.
	app.InitChain(abci.RequestInitChain{
		Time:            time.Now().UTC(),
		ConsensusParams: consensusParams,
		AppStateBytes:   stateBytes,
	})

	return app, gen
}

// FundAccount is a utility function that funds an account by minting and
// sending the coins to the address. This should be used for testing purposes
// only!
func FundAccount(
	bankKeeper bankkeeper.Keeper, ctx sdk.Context, addr sdk.AccAddress,
	amounts sdk.Coins,
) error {
	if err := bankKeeper.MintCoins(ctx, mint.ModuleName, amounts); err != nil {
		return err
	}

	return bankKeeper.SendCoinsFromModuleToAccount(ctx, mint.ModuleName, addr, amounts)
}

// FundModuleAccount is a utility function that funds a module account by
// minting and sending the coins to the address. This should be used for testing
// purposes only!
func FundModuleAccount(
	bankKeeper bankkeeper.Keeper, ctx sdk.Context,
	recipientMod string, amounts sdk.Coins,
) error {
	if err := bankKeeper.MintCoins(ctx, mint.ModuleName, amounts); err != nil {
		return err
	}

	return bankKeeper.SendCoinsFromModuleToModule(ctx, mint.ModuleName, recipientMod, amounts)
}

// FundFeeCollector funds the module account that collects gas fees with some
// amount of "unibi", the gas token.
func FundFeeCollector(
	bk bankkeeper.Keeper, ctx sdk.Context, amount sdkmath.Int,
) error {
	return FundModuleAccount(
		bk,
		ctx,
		auth.FeeCollectorName,
		sdk.NewCoins(sdk.NewCoin(appconst.DENOM_UNIBI, amount)),
	)
}

// GenesisStateWithSingleValidator initializes GenesisState with a single validator and genesis accounts
// that also act as delegators.
func GenesisStateWithSingleValidator(codec codec.Codec, genesisState app.GenesisState) (app.GenesisState, error) {
	privVal := mock.NewPV()
	pubKey, err := privVal.GetPubKey()
	if err != nil {
		return nil, err
	}

	// create validator set with single validator
	validator := cmttypes.NewValidator(pubKey, 1)
	valSet := cmttypes.NewValidatorSet([]*cmttypes.Validator{validator})

	// generate genesis account
	senderPrivKey := secp256k1.GenPrivKey()
	acc := auth.NewBaseAccount(senderPrivKey.PubKey().Address().Bytes(), senderPrivKey.PubKey(), 0, 0)
	authGenesis := auth.NewGenesisState(auth.DefaultParams(), []auth.GenesisAccount{acc})
	genesisState[auth.ModuleName] = codec.MustMarshalJSON(authGenesis)

	// add genesis account balance
	var bankGenesis banktypes.GenesisState
	codec.MustUnmarshalJSON(genesisState[banktypes.ModuleName], &bankGenesis)
	bankGenesis.Balances = append(bankGenesis.Balances, banktypes.Balance{
		Address: acc.GetAddress().String(),
		Coins:   sdk.NewCoins(sdk.NewCoin(appconst.DENOM_UNIBI, sdkmath.NewIntFromUint64(1e14))),
	})

	genesisState, err = genesisStateWithValSet(codec, genesisState, valSet, []auth.GenesisAccount{acc}, bankGenesis.Balances...)
	if err != nil {
		return nil, err
	}

	return genesisState, nil
}

func genesisStateWithValSet(
	cdc codec.Codec,
	genesisState app.GenesisState,
	valSet *cmttypes.ValidatorSet, genAccs []auth.GenesisAccount,
	balances ...banktypes.Balance,
) (app.GenesisState, error) {
	validators := make([]stakingtypes.Validator, 0, len(valSet.Validators))
	delegations := make([]stakingtypes.Delegation, 0, len(valSet.Validators))

	for _, val := range valSet.Validators {
		pk, err := cryptocodec.FromTmPubKeyInterface(val.PubKey)
		if err != nil {
			return nil, err
		}
		pkAny, err := codectypes.NewAnyWithValue(pk)
		if err != nil {
			return nil, err
		}
		validator := stakingtypes.Validator{
			OperatorAddress:   sdk.ValAddress(val.Address).String(),
			ConsensusPubkey:   pkAny,
			Jailed:            false,
			Status:            stakingtypes.Bonded,
			Tokens:            sdk.DefaultPowerReduction,
			DelegatorShares:   sdkmath.LegacyOneDec(),
			Description:       stakingtypes.Description{},
			UnbondingHeight:   int64(0),
			UnbondingTime:     time.Unix(0, 0).UTC(),
			Commission:        stakingtypes.NewCommission(sdkmath.LegacyZeroDec(), sdkmath.LegacyZeroDec(), sdkmath.LegacyZeroDec()),
			MinSelfDelegation: sdkmath.ZeroInt(),
		}
		validators = append(validators, validator)
		delegations = append(delegations, stakingtypes.NewDelegation(genAccs[0].GetAddress(), val.Address.Bytes(), sdkmath.LegacyOneDec()))
	}
	// set validators and delegations
	genesisState[stakingtypes.ModuleName] = cdc.MustMarshalJSON(
		stakingtypes.NewGenesisState(stakingtypes.DefaultParams(), validators, delegations),
	)

	totalSupply := sdk.NewCoins()
	for _, b := range balances {
		// add genesis acc tokens to total supply
		totalSupply = totalSupply.Add(b.Coins...)
	}

	for range delegations {
		// add delegated tokens to total supply
		totalSupply = totalSupply.Add(sdk.NewCoin(appconst.DENOM_UNIBI, sdk.DefaultPowerReduction))
	}

	// add bonded amount to bonded pool module account
	balances = append(balances, banktypes.Balance{
		Address: auth.NewModuleAddress(stakingtypes.BondedPoolName).String(),
		Coins:   sdk.Coins{sdk.NewCoin(appconst.DENOM_UNIBI, sdk.DefaultPowerReduction)},
	})

	// update total supply
	genesisState[banktypes.ModuleName] = cdc.MustMarshalJSON(
		banktypes.NewGenesisState(
			banktypes.DefaultParams(),
			balances,
			totalSupply,
			[]banktypes.Metadata{},
			[]banktypes.SendEnabled{},
		),
	)

	return genesisState, nil
}
