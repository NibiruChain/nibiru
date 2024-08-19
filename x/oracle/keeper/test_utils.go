// nolint
package keeper

import (
	"testing"
	"time"

	"cosmossdk.io/math"
	"github.com/NibiruChain/nibiru/v2/x/common/denoms"
	"github.com/NibiruChain/nibiru/v2/x/oracle/types"
	"github.com/NibiruChain/nibiru/v2/x/sudo"
	sudokeeper "github.com/NibiruChain/nibiru/v2/x/sudo/keeper"
	sudotypes "github.com/NibiruChain/nibiru/v2/x/sudo/types"
	dbm "github.com/cometbft/cometbft-db"
	"github.com/cometbft/cometbft/crypto"
	"github.com/cometbft/cometbft/crypto/secp256k1"
	"github.com/cometbft/cometbft/libs/log"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/cosmos/cosmos-sdk/std"
	"github.com/cosmos/cosmos-sdk/store"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	"github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/cosmos/cosmos-sdk/types/module/testutil"
	"github.com/cosmos/cosmos-sdk/x/auth"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	"github.com/cosmos/cosmos-sdk/x/auth/tx"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/cosmos/cosmos-sdk/x/bank"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	distr "github.com/cosmos/cosmos-sdk/x/distribution"
	distrkeeper "github.com/cosmos/cosmos-sdk/x/distribution/keeper"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/cosmos/cosmos-sdk/x/params"
	paramstypes "github.com/cosmos/cosmos-sdk/x/params/types"
	slashingkeeper "github.com/cosmos/cosmos-sdk/x/slashing/keeper"
	slashingtypes "github.com/cosmos/cosmos-sdk/x/slashing/types"
	"github.com/cosmos/cosmos-sdk/x/staking"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/stretchr/testify/require"
)

const faucetAccountName = "faucet"

// ModuleBasics nolint
var ModuleBasics = module.NewBasicManager(
	auth.AppModuleBasic{},
	bank.AppModuleBasic{},
	distr.AppModuleBasic{},
	staking.AppModuleBasic{},
	params.AppModuleBasic{},
	sudo.AppModuleBasic{},
)

// MakeTestCodec nolint
func MakeTestCodec(t *testing.T) codec.Codec {
	return MakeEncodingConfig(t).Codec
}

// MakeEncodingConfig nolint
func MakeEncodingConfig(_ *testing.T) testutil.TestEncodingConfig {
	amino := codec.NewLegacyAmino()
	interfaceRegistry := codectypes.NewInterfaceRegistry()
	codec := codec.NewProtoCodec(interfaceRegistry)
	txCfg := tx.NewTxConfig(codec, tx.DefaultSignModes)

	std.RegisterInterfaces(interfaceRegistry)
	std.RegisterLegacyAminoCodec(amino)

	ModuleBasics.RegisterLegacyAminoCodec(amino)
	ModuleBasics.RegisterInterfaces(interfaceRegistry)
	types.RegisterLegacyAminoCodec(amino)
	types.RegisterInterfaces(interfaceRegistry)
	return testutil.TestEncodingConfig{
		InterfaceRegistry: interfaceRegistry,
		Codec:             codec,
		TxConfig:          txCfg,
		Amino:             amino,
	}
}

// Test addresses
var (
	ValPubKeys = sims.CreateTestPubKeys(5)

	pubKeys = []crypto.PubKey{
		secp256k1.GenPrivKey().PubKey(),
		secp256k1.GenPrivKey().PubKey(),
		secp256k1.GenPrivKey().PubKey(),
		secp256k1.GenPrivKey().PubKey(),
		secp256k1.GenPrivKey().PubKey(),
	}

	Addrs = []sdk.AccAddress{
		sdk.AccAddress(pubKeys[0].Address()),
		sdk.AccAddress(pubKeys[1].Address()),
		sdk.AccAddress(pubKeys[2].Address()),
		sdk.AccAddress(pubKeys[3].Address()),
		sdk.AccAddress(pubKeys[4].Address()),
	}

	ValAddrs = []sdk.ValAddress{
		sdk.ValAddress(pubKeys[0].Address()),
		sdk.ValAddress(pubKeys[1].Address()),
		sdk.ValAddress(pubKeys[2].Address()),
		sdk.ValAddress(pubKeys[3].Address()),
		sdk.ValAddress(pubKeys[4].Address()),
	}

	InitTokens = sdk.TokensFromConsensusPower(200, sdk.DefaultPowerReduction)
	InitCoins  = sdk.NewCoins(sdk.NewCoin(denoms.NIBI, InitTokens))

	OracleDecPrecision = 8
)

// TestFixture nolint
type TestFixture struct {
	Ctx           sdk.Context
	Cdc           *codec.LegacyAmino
	AccountKeeper authkeeper.AccountKeeper
	BankKeeper    bankkeeper.Keeper
	OracleKeeper  Keeper
	StakingKeeper stakingkeeper.Keeper
	DistrKeeper   distrkeeper.Keeper
	SudoKeeper    types.SudoKeeper
}

// CreateTestFixture nolint
// Creates a base app, with 5 accounts,
func CreateTestFixture(t *testing.T) TestFixture {
	keyAcc := sdk.NewKVStoreKey(authtypes.StoreKey)
	keyBank := sdk.NewKVStoreKey(banktypes.StoreKey)
	keyParams := sdk.NewKVStoreKey(paramstypes.StoreKey)
	tKeyParams := sdk.NewTransientStoreKey(paramstypes.TStoreKey)
	keyOracle := sdk.NewKVStoreKey(types.StoreKey)
	keyStaking := sdk.NewKVStoreKey(stakingtypes.StoreKey)
	keySlashing := sdk.NewKVStoreKey(slashingtypes.StoreKey)
	keyDistr := sdk.NewKVStoreKey(distrtypes.StoreKey)
	keySudo := sdk.NewKVStoreKey(sudotypes.StoreKey)

	govModuleAddr := authtypes.NewModuleAddress(govtypes.ModuleName).String()

	db := dbm.NewMemDB()
	ms := store.NewCommitMultiStore(db)
	ctx := sdk.NewContext(ms, tmproto.Header{Time: time.Now().UTC(), Height: 1}, false, log.NewNopLogger())
	encodingConfig := MakeEncodingConfig(t)
	appCodec, legacyAmino := encodingConfig.Codec, encodingConfig.Amino

	ms.MountStoreWithDB(keyAcc, storetypes.StoreTypeIAVL, db)
	ms.MountStoreWithDB(keyBank, storetypes.StoreTypeIAVL, db)
	ms.MountStoreWithDB(tKeyParams, storetypes.StoreTypeTransient, db)
	ms.MountStoreWithDB(keyParams, storetypes.StoreTypeIAVL, db)
	ms.MountStoreWithDB(keyOracle, storetypes.StoreTypeIAVL, db)
	ms.MountStoreWithDB(keyStaking, storetypes.StoreTypeIAVL, db)
	ms.MountStoreWithDB(keyDistr, storetypes.StoreTypeIAVL, db)

	require.NoError(t, ms.LoadLatestVersion())

	blackListAddrs := map[string]bool{
		authtypes.FeeCollectorName:     true,
		stakingtypes.NotBondedPoolName: true,
		stakingtypes.BondedPoolName:    true,
		distrtypes.ModuleName:          true,
		faucetAccountName:              true,
	}

	maccPerms := map[string][]string{
		faucetAccountName:              {authtypes.Minter},
		authtypes.FeeCollectorName:     nil,
		stakingtypes.NotBondedPoolName: {authtypes.Burner, authtypes.Staking},
		stakingtypes.BondedPoolName:    {authtypes.Burner, authtypes.Staking},
		distrtypes.ModuleName:          nil,
		types.ModuleName:               nil,
	}

	accountKeeper := authkeeper.NewAccountKeeper(
		appCodec,
		keyAcc,
		authtypes.ProtoBaseAccount,
		maccPerms,
		sdk.GetConfig().GetBech32AccountAddrPrefix(),
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)
	bankKeeper := bankkeeper.NewBaseKeeper(
		appCodec,
		keyBank,
		accountKeeper,
		blackListAddrs,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)

	totalSupply := sdk.NewCoins(sdk.NewCoin(denoms.NIBI, InitTokens.MulRaw(int64(len(Addrs)*10))))
	bankKeeper.MintCoins(ctx, faucetAccountName, totalSupply)

	stakingKeeper := stakingkeeper.NewKeeper(
		appCodec,
		keyStaking,
		accountKeeper,
		bankKeeper,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)
	stakingParams := stakingtypes.DefaultParams()
	stakingParams.BondDenom = denoms.NIBI
	stakingKeeper.SetParams(ctx, stakingParams)

	slashingKeeper := slashingkeeper.NewKeeper(appCodec, legacyAmino, keySlashing, stakingKeeper, govModuleAddr)

	distrKeeper := distrkeeper.NewKeeper(
		appCodec,
		keyDistr,
		accountKeeper, bankKeeper, stakingKeeper,
		authtypes.FeeCollectorName,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)

	distrKeeper.SetFeePool(ctx, distrtypes.InitialFeePool())
	distrParams := distrtypes.DefaultParams()
	distrParams.CommunityTax = math.LegacyNewDecWithPrec(2, 2)
	distrParams.BaseProposerReward = math.LegacyNewDecWithPrec(1, 2)
	distrParams.BonusProposerReward = math.LegacyNewDecWithPrec(4, 2)
	distrKeeper.SetParams(ctx, distrParams)
	stakingKeeper.SetHooks(stakingtypes.NewMultiStakingHooks(distrKeeper.Hooks()))

	feeCollectorAcc := authtypes.NewEmptyModuleAccount(authtypes.FeeCollectorName)
	notBondedPool := authtypes.NewEmptyModuleAccount(stakingtypes.NotBondedPoolName, authtypes.Burner, authtypes.Staking)
	bondPool := authtypes.NewEmptyModuleAccount(stakingtypes.BondedPoolName, authtypes.Burner, authtypes.Staking)
	distrAcc := authtypes.NewEmptyModuleAccount(distrtypes.ModuleName)
	oracleAcc := authtypes.NewEmptyModuleAccount(types.ModuleName, authtypes.Minter)

	bankKeeper.SendCoinsFromModuleToModule(ctx, faucetAccountName, stakingtypes.NotBondedPoolName, sdk.NewCoins(sdk.NewCoin(denoms.NIBI, InitTokens.MulRaw(int64(len(Addrs))))))

	sudoKeeper := sudokeeper.NewKeeper(appCodec, keySudo)
	sudoAcc := authtypes.NewEmptyModuleAccount(sudotypes.ModuleName)

	accountKeeper.SetModuleAccount(ctx, feeCollectorAcc)
	accountKeeper.SetModuleAccount(ctx, bondPool)
	accountKeeper.SetModuleAccount(ctx, notBondedPool)
	accountKeeper.SetModuleAccount(ctx, distrAcc)
	accountKeeper.SetModuleAccount(ctx, oracleAcc)
	accountKeeper.SetModuleAccount(ctx, sudoAcc)

	for _, addr := range Addrs {
		accountKeeper.SetAccount(ctx, authtypes.NewBaseAccountWithAddress(addr))
		err := bankKeeper.SendCoinsFromModuleToAccount(ctx, faucetAccountName, addr, InitCoins)
		require.NoError(t, err)
	}

	keeper := NewKeeper(
		appCodec,
		keyOracle,
		accountKeeper,
		bankKeeper,
		distrKeeper,
		stakingKeeper,
		slashingKeeper,
		sudoKeeper,
		distrtypes.ModuleName,
	)

	defaults := types.DefaultParams()

	for _, pair := range defaults.Whitelist {
		keeper.WhitelistedPairs.Insert(ctx, pair)
	}

	keeper.Params.Set(ctx, defaults)

	return TestFixture{
		ctx, legacyAmino, accountKeeper, bankKeeper,
		keeper,
		*stakingKeeper,
		distrKeeper,
		sudoKeeper,
	}
}

// NewTestMsgCreateValidator test msg creator
func NewTestMsgCreateValidator(
	address sdk.ValAddress, pubKey cryptotypes.PubKey, amt sdk.Int,
) *stakingtypes.MsgCreateValidator {
	commission := stakingtypes.NewCommissionRates(math.LegacyZeroDec(), math.LegacyZeroDec(), math.LegacyZeroDec())
	msg, _ := stakingtypes.NewMsgCreateValidator(
		address, pubKey, sdk.NewCoin(denoms.NIBI, amt),
		stakingtypes.Description{}, commission, math.OneInt(),
	)

	return msg
}

// FundAccount is a utility function that funds an account by minting and
// sending the coins to the address. This should be used for testing purposes
// only!
func FundAccount(input TestFixture, addr sdk.AccAddress, amounts sdk.Coins) error {
	if err := input.BankKeeper.MintCoins(input.Ctx, faucetAccountName, amounts); err != nil {
		return err
	}

	return input.BankKeeper.SendCoinsFromModuleToAccount(input.Ctx, faucetAccountName, addr, amounts)
}

func AllocateRewards(t *testing.T, input TestFixture, rewards sdk.Coins, votePeriods uint64) {
	require.NoError(t, input.BankKeeper.MintCoins(input.Ctx, faucetAccountName, rewards))
	require.NoError(t, input.OracleKeeper.AllocateRewards(input.Ctx, faucetAccountName, rewards, votePeriods))
}

var (
	testStakingAmt   = sdk.TokensFromConsensusPower(10, sdk.DefaultPowerReduction)
	testExchangeRate = math.LegacyNewDec(1700)
)

func Setup(t *testing.T) (TestFixture, types.MsgServer) {
	fixture := CreateTestFixture(t)

	params, _ := fixture.OracleKeeper.Params.Get(fixture.Ctx)

	params.VotePeriod = 1
	params.SlashWindow = 100
	fixture.OracleKeeper.Params.Set(fixture.Ctx, params)

	params, _ = fixture.OracleKeeper.Params.Get(fixture.Ctx)

	h := NewMsgServerImpl(fixture.OracleKeeper)
	sh := stakingkeeper.NewMsgServerImpl(&fixture.StakingKeeper)

	// Validator created
	_, err := sh.CreateValidator(fixture.Ctx, NewTestMsgCreateValidator(ValAddrs[0], ValPubKeys[0], testStakingAmt))
	require.NoError(t, err)
	_, err = sh.CreateValidator(fixture.Ctx, NewTestMsgCreateValidator(ValAddrs[1], ValPubKeys[1], testStakingAmt))
	require.NoError(t, err)
	_, err = sh.CreateValidator(fixture.Ctx, NewTestMsgCreateValidator(ValAddrs[2], ValPubKeys[2], testStakingAmt))
	require.NoError(t, err)
	_, err = sh.CreateValidator(fixture.Ctx, NewTestMsgCreateValidator(ValAddrs[3], ValPubKeys[3], testStakingAmt))
	require.NoError(t, err)
	_, err = sh.CreateValidator(fixture.Ctx, NewTestMsgCreateValidator(ValAddrs[4], ValPubKeys[4], testStakingAmt))
	require.NoError(t, err)
	staking.EndBlocker(fixture.Ctx, &fixture.StakingKeeper)

	return fixture, h
}

func MakeAggregatePrevoteAndVote(
	t *testing.T,
	input TestFixture,
	msgServer types.MsgServer,
	height int64,
	rates types.ExchangeRateTuples,
	valIdx int,
) {
	salt := "1"
	ratesStr, err := rates.ToString()
	require.NoError(t, err)
	hash := types.GetAggregateVoteHash(salt, ratesStr, ValAddrs[valIdx])

	prevoteMsg := types.NewMsgAggregateExchangeRatePrevote(hash, Addrs[valIdx], ValAddrs[valIdx])
	_, err = msgServer.AggregateExchangeRatePrevote(sdk.WrapSDKContext(input.Ctx.WithBlockHeight(height)), prevoteMsg)
	require.NoError(t, err)

	voteMsg := types.NewMsgAggregateExchangeRateVote(salt, ratesStr, Addrs[valIdx], ValAddrs[valIdx])
	_, err = msgServer.AggregateExchangeRateVote(sdk.WrapSDKContext(input.Ctx.WithBlockHeight(height+1)), voteMsg)
	require.NoError(t, err)
}
