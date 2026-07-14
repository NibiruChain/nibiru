package keeper_test

import (
	"testing"
	"time"

	"cosmossdk.io/depinject"
	"cosmossdk.io/log"
	sdkmath "cosmossdk.io/math"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"

	bankkeeper "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/bank/keeper"
	banktestutil "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/bank/testutil"
	slashingkeeper "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/slashing/keeper"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/slashing/testutil"
	stakingkeeper "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/staking/keeper"
	stakingtypes "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/staking/types"

	distributionkeeper "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/distribution/keeper"

	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types"

	simtestutil "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/testutil/sims"
)

func TestSlashRedelegation(t *testing.T) {
	// setting up
	var stakingKeeper *stakingkeeper.Keeper
	var bankKeeper bankkeeper.Keeper
	var slashKeeper slashingkeeper.Keeper
	var distrKeeper distributionkeeper.Keeper

	app, err := simtestutil.Setup(depinject.Configs(
		depinject.Supply(log.NewNopLogger()),
		testutil.AppConfig,
	), &stakingKeeper, &bankKeeper, &slashKeeper, &distrKeeper)
	require.NoError(t, err)

	// get sdk context, staking msg server and bond denom
	ctx := app.NewContext(false, tmproto.Header{Height: app.LastBlockHeight() + 1})
	stakingMsgServer := stakingkeeper.NewMsgServerImpl(stakingKeeper)
	bondDenom := stakingKeeper.BondDenom(ctx)
	require.NoError(t, err)

	// evilVal will be slashed, goodVal won't be slashed
	evilValPubKey := secp256k1.GenPrivKey().PubKey()
	goodValPubKey := secp256k1.GenPrivKey().PubKey()

	// both test acc 1 and 2 delegated to evil val, both acc should be slashed when evil val is slashed
	// test acc 1 use the "undelegation after redelegation" trick (redelegate to good val and then undelegate) to avoid slashing
	// test acc 2 only undelegate from evil val
	testAcc1 := sdk.AccAddress([]byte("addr1_______________"))
	testAcc2 := sdk.AccAddress([]byte("addr2_______________"))

	// fund acc 1 and acc 2 //nolint:errcheck
	testCoins := sdk.NewCoins(sdk.NewCoin(bondDenom, stakingKeeper.TokensFromConsensusPower(ctx, 10))) //nolint:errcheck
	//nolint:errcheck
	banktestutil.FundAccount(bankKeeper, ctx, testAcc1, testCoins)
	//nolint:errcheck
	banktestutil.FundAccount(bankKeeper, ctx, testAcc2, testCoins)

	balance1Before := bankKeeper.GetBalance(ctx, testAcc1, bondDenom)
	balance2Before := bankKeeper.GetBalance(ctx, testAcc2, bondDenom)

	// assert acc 1 and acc 2 balance
	require.Equal(t, balance1Before.Amount.String(), testCoins[0].Amount.String())
	require.Equal(t, balance2Before.Amount.String(), testCoins[0].Amount.String())

	// creating evil val //nolint:errcheck
	evilValAddr := sdk.ValAddress(evilValPubKey.Address())
	//nolint:errcheck
	banktestutil.FundAccount(bankKeeper, ctx, sdk.AccAddress(evilValAddr), testCoins)
	createValMsg1, _ := stakingtypes.NewMsgCreateValidator(
		evilValAddr, evilValPubKey, testCoins[0], stakingtypes.Description{Details: "test"}, stakingtypes.NewCommissionRates(sdkmath.LegacyNewDecWithPrec(5, 1), sdkmath.LegacyNewDecWithPrec(5, 1), sdkmath.LegacyNewDec(0)), sdkmath.OneInt())
	_, err = stakingMsgServer.CreateValidator(ctx, createValMsg1)
	require.NoError(t, err)

	// creating good val //nolint:errcheck
	goodValAddr := sdk.ValAddress(goodValPubKey.Address())
	//nolint:errcheck
	banktestutil.FundAccount(bankKeeper, ctx, sdk.AccAddress(goodValAddr), testCoins)
	createValMsg2, _ := stakingtypes.NewMsgCreateValidator(
		goodValAddr, goodValPubKey, testCoins[0], stakingtypes.Description{Details: "test"}, stakingtypes.NewCommissionRates(sdkmath.LegacyNewDecWithPrec(5, 1), sdkmath.LegacyNewDecWithPrec(5, 1), sdkmath.LegacyNewDec(0)), sdkmath.OneInt())
	_, err = stakingMsgServer.CreateValidator(ctx, createValMsg2)
	require.NoError(t, err)

	// next block, commit height 2, move to height 3
	// acc 1 and acc 2 delegate to evil val
	ctx = simtestutil.NextBlock(app, ctx, time.Duration(1))
	require.NoError(t, err)

	// Acc 2 delegate
	delMsg := stakingtypes.NewMsgDelegate(testAcc2, evilValAddr, testCoins[0])
	_, err = stakingMsgServer.Delegate(ctx, delMsg)
	require.NoError(t, err)

	// Acc 1 delegate
	delMsg = stakingtypes.NewMsgDelegate(testAcc1, evilValAddr, testCoins[0])
	_, err = stakingMsgServer.Delegate(ctx, delMsg)
	require.NoError(t, err)

	// next block, commit height 3, move to height 4
	// with the new delegations, evil val increases in voting power and commit byzantine behaviour at height 4 consensus
	// at the same time, acc 1 and acc 2 withdraw delegation from evil val
	ctx = simtestutil.NextBlock(app, ctx, time.Duration(1))
	require.NoError(t, err)

	evilVal, found := stakingKeeper.GetValidator(ctx, evilValAddr)
	require.True(t, found)

	evilPower := stakingKeeper.TokensToConsensusPower(ctx, evilVal.Tokens)

	// Acc 1 redelegate from evil val to good val
	redelMsg := stakingtypes.NewMsgBeginRedelegate(testAcc1, evilValAddr, goodValAddr, testCoins[0])
	_, err = stakingMsgServer.BeginRedelegate(ctx, redelMsg)
	require.NoError(t, err)

	// Acc 1 undelegate from good val
	undelMsg := stakingtypes.NewMsgUndelegate(testAcc1, goodValAddr, testCoins[0])
	_, err = stakingMsgServer.Undelegate(ctx, undelMsg)
	require.NoError(t, err)

	// Acc 2 undelegate from evil val
	undelMsg = stakingtypes.NewMsgUndelegate(testAcc2, evilValAddr, testCoins[0])
	_, err = stakingMsgServer.Undelegate(ctx, undelMsg)
	require.NoError(t, err)

	// next block, commit height 4, move to height 5
	// Slash evil val for byzantine behaviour at height 4 consensus,
	// at which acc 1 and acc 2 still contributed to evil val voting power
	// even tho they undelegate at block 4, the valset update is applied after commited block 4 when height 4 consensus already passes
	ctx = simtestutil.NextBlock(app, ctx, time.Duration(1))
	require.NoError(t, err)

	// slash evil val with slash factor = 0.9, leaving only 10% of stake after slashing
	evilVal, _ = stakingKeeper.GetValidator(ctx, evilValAddr)
	evilValConsAddr, err := evilVal.GetConsAddr()
	require.NoError(t, err)

	slashKeeper.Slash(ctx, evilValConsAddr, sdkmath.LegacyMustNewDecFromStr("0.9"), evilPower, 4)

	// assert invariant to make sure we conduct slashing correctly
	_, stop := stakingkeeper.AllInvariants(stakingKeeper)(ctx)
	require.False(t, stop)

	_, stop = bankkeeper.AllInvariants(bankKeeper)(ctx)
	require.False(t, stop)

	_, stop = distributionkeeper.AllInvariants(distrKeeper)(ctx)
	require.False(t, stop)

	// one eternity later
	ctx = simtestutil.NextBlock(app, ctx, time.Duration(1000000000000000000))
	ctx = simtestutil.NextBlock(app, ctx, time.Duration(1))

	// confirm that account 1 and account 2 has been slashed, and the slash amount is correct
	balance1AfterSlashing := bankKeeper.GetBalance(ctx, testAcc1, bondDenom)
	balance2AfterSlashing := bankKeeper.GetBalance(ctx, testAcc2, bondDenom)

	require.Equal(t, balance1AfterSlashing.Amount.Mul(sdkmath.NewIntFromUint64(10)).String(), balance1Before.Amount.String())
	require.Equal(t, balance2AfterSlashing.Amount.Mul(sdkmath.NewIntFromUint64(10)).String(), balance2Before.Amount.String())
}
