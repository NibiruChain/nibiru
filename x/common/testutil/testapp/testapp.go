package testapp

import (
	"encoding/json"
	"maps"
	"time"

	"cosmossdk.io/math"
	tmdb "github.com/cometbft/cometbft-db"
	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cometbft/cometbft/libs/log"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	tmtypes "github.com/cometbft/cometbft/types"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	"github.com/cosmos/cosmos-sdk/testutil/mock"
	"github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	auth "github.com/cosmos/cosmos-sdk/x/auth/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	govtypesv1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	"github.com/NibiruChain/nibiru/v2/app"
	nibiruapp "github.com/NibiruChain/nibiru/v2/app"
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
	app, _ := NewNibiruTestApp(app.GenesisState{})
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
func NewNibiruTestApp(customGenesisOverride app.GenesisState) (
	nibiruApp *app.NibiruApp, gen app.GenesisState,
) {
	app := app.NewNibiruApp(
		log.NewNopLogger(),
		tmdb.NewMemDB(),
		/*traceStore=*/ nil,
		/*loadLatest=*/ true,
		/*appOpts=*/ sims.EmptyAppOptions{},
	)

	// configure genesis from default
	gen = app.DefaultGenesis()

	// Set happy genesis: epochs
	genModEpochs := epochstypes.DefaultGenesisFromTime(time.Now().UTC())
	gen[epochstypes.ModuleName] = app.AppCodec().MustMarshalJSON(
		genModEpochs,
	)

	// Set happy genesis: sudo
	sudoGenesis := sudotypes.GenesisState{
		Sudoers: sudotypes.Sudoers{
			Root:      testutil.ADDR_SUDO_ROOT,
			Contracts: []string{testutil.ADDR_SUDO_ROOT},
		},
	}
	gen[sudotypes.ModuleName] = app.AppCodec().MustMarshalJSON(&sudoGenesis)

	// Set happy genesis: gov
	// Set short voting period to allow fast gov proposals in tests
	var govGenesis govtypesv1.GenesisState
	app.AppCodec().MustUnmarshalJSON(gen[govtypes.ModuleName], &govGenesis)
	*govGenesis.Params.VotingPeriod = 20 * time.Second
	govGenesis.Params.MinDeposit = sdk.NewCoins(sdk.NewInt64Coin(denoms.NIBI, 1e6)) // min deposit of 1 NIBI
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

	app.InitChain(abci.RequestInitChain{
		ConsensusParams: sims.DefaultConsensusParams,
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

// GenesisStateWithSingleValidator initializes GenesisState with a single validator and genesis accounts
// that also act as delegators.
func GenesisStateWithSingleValidator(codec codec.Codec, genesisState nibiruapp.GenesisState) (nibiruapp.GenesisState, error) {
	privVal := mock.NewPV()
	pubKey, err := privVal.GetPubKey()
	if err != nil {
		return nil, err
	}

	// create validator set with single validator
	validator := tmtypes.NewValidator(pubKey, 1)
	valSet := tmtypes.NewValidatorSet([]*tmtypes.Validator{validator})

	// generate genesis account
	senderPrivKey := secp256k1.GenPrivKey()
	acc := authtypes.NewBaseAccount(senderPrivKey.PubKey().Address().Bytes(), senderPrivKey.PubKey(), 0, 0)
	authGenesis := authtypes.NewGenesisState(authtypes.DefaultParams(), []authtypes.GenesisAccount{acc})
	genesisState[authtypes.ModuleName] = codec.MustMarshalJSON(authGenesis)

	// add genesis account balance
	var bankGenesis banktypes.GenesisState
	codec.MustUnmarshalJSON(genesisState[banktypes.ModuleName], &bankGenesis)
	bankGenesis.Balances = append(bankGenesis.Balances, banktypes.Balance{
		Address: acc.GetAddress().String(),
		Coins:   sdk.NewCoins(sdk.NewCoin(appconst.BondDenom, math.NewIntFromUint64(1e14))),
	})

	genesisState, err = genesisStateWithValSet(codec, genesisState, valSet, []authtypes.GenesisAccount{acc}, bankGenesis.Balances...)
	if err != nil {
		return nil, err
	}

	return genesisState, nil
}

func genesisStateWithValSet(
	cdc codec.Codec,
	genesisState nibiruapp.GenesisState,
	valSet *tmtypes.ValidatorSet, genAccs []authtypes.GenesisAccount,
	balances ...banktypes.Balance,
) (nibiruapp.GenesisState, error) {
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
			DelegatorShares:   math.LegacyOneDec(),
			Description:       stakingtypes.Description{},
			UnbondingHeight:   int64(0),
			UnbondingTime:     time.Unix(0, 0).UTC(),
			Commission:        stakingtypes.NewCommission(math.LegacyZeroDec(), math.LegacyZeroDec(), math.LegacyZeroDec()),
			MinSelfDelegation: math.ZeroInt(),
		}
		validators = append(validators, validator)
		delegations = append(delegations, stakingtypes.NewDelegation(genAccs[0].GetAddress(), val.Address.Bytes(), math.LegacyOneDec()))
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
		totalSupply = totalSupply.Add(sdk.NewCoin(appconst.BondDenom, sdk.DefaultPowerReduction))
	}

	// add bonded amount to bonded pool module account
	balances = append(balances, banktypes.Balance{
		Address: authtypes.NewModuleAddress(stakingtypes.BondedPoolName).String(),
		Coins:   sdk.Coins{sdk.NewCoin(appconst.BondDenom, sdk.DefaultPowerReduction)},
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
