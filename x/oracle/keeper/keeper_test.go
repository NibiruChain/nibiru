package keeper

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/common/denoms"

	"github.com/NibiruChain/nibiru/x/oracle/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/staking"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

func TestParams(t *testing.T) {
	input := CreateTestInput(t)

	// Test default params setting
	input.OracleKeeper.SetParams(input.Ctx, types.DefaultParams())
	params := input.OracleKeeper.GetParams(input.Ctx)
	require.NotNil(t, params)

	// Test custom params setting
	votePeriod := uint64(10)
	voteThreshold := sdk.NewDecWithPrec(33, 2)
	oracleRewardBand := sdk.NewDecWithPrec(1, 2)
	slashFraction := sdk.NewDecWithPrec(1, 2)
	slashWindow := uint64(1000)
	minValidPerWindow := sdk.NewDecWithPrec(1, 4)
	whitelist := []asset.Pair{
		asset.Registry.Pair(denoms.BTC, denoms.NUSD),
		asset.Registry.Pair(denoms.ETH, denoms.NUSD),
	}

	// Should really test validateParams, but skipping because obvious
	newParams := types.Params{
		VotePeriod:        votePeriod,
		VoteThreshold:     voteThreshold,
		RewardBand:        oracleRewardBand,
		Whitelist:         whitelist,
		SlashFraction:     slashFraction,
		SlashWindow:       slashWindow,
		MinValidPerWindow: minValidPerWindow,
	}
	input.OracleKeeper.SetParams(input.Ctx, newParams)

	storedParams := input.OracleKeeper.GetParams(input.Ctx)
	require.NotNil(t, storedParams)
	require.Equal(t, storedParams, newParams)
}

func TestValidateFeeder(t *testing.T) {
	// initial setup
	input := CreateTestInput(t)
	addr, val := ValAddrs[0], ValPubKeys[0]
	addr1, val1 := ValAddrs[1], ValPubKeys[1]
	amt := sdk.TokensFromConsensusPower(100, sdk.DefaultPowerReduction)
	sh := staking.NewHandler(input.StakingKeeper)
	ctx := input.Ctx

	// Create 2 validators.
	_, err := sh(ctx, NewTestMsgCreateValidator(addr, val, amt))
	require.NoError(t, err)
	_, err = sh(ctx, NewTestMsgCreateValidator(addr1, val1, amt))
	require.NoError(t, err)
	staking.EndBlocker(ctx, input.StakingKeeper)

	require.Equal(
		t, input.BankKeeper.GetAllBalances(ctx, sdk.AccAddress(addr)),
		sdk.NewCoins(sdk.NewCoin(input.StakingKeeper.GetParams(ctx).BondDenom, InitTokens.Sub(amt))),
	)
	require.Equal(t, amt, input.StakingKeeper.Validator(ctx, addr).GetBondedTokens())
	require.Equal(
		t, input.BankKeeper.GetAllBalances(ctx, sdk.AccAddress(addr1)),
		sdk.NewCoins(sdk.NewCoin(input.StakingKeeper.GetParams(ctx).BondDenom, InitTokens.Sub(amt))),
	)
	require.Equal(t, amt, input.StakingKeeper.Validator(ctx, addr1).GetBondedTokens())

	require.NoError(t, input.OracleKeeper.ValidateFeeder(input.Ctx, sdk.AccAddress(addr), sdk.ValAddress(addr)))
	require.NoError(t, input.OracleKeeper.ValidateFeeder(input.Ctx, sdk.AccAddress(addr1), sdk.ValAddress(addr1)))

	// delegate works
	input.OracleKeeper.FeederDelegations.Insert(input.Ctx, addr, sdk.AccAddress(addr1))
	require.NoError(t, input.OracleKeeper.ValidateFeeder(input.Ctx, sdk.AccAddress(addr1), addr))
	require.Error(t, input.OracleKeeper.ValidateFeeder(input.Ctx, Addrs[2], addr))

	// only active validators can do oracle votes
	validator, found := input.StakingKeeper.GetValidator(input.Ctx, addr)
	require.True(t, found)
	validator.Status = stakingtypes.Unbonded
	input.StakingKeeper.SetValidator(input.Ctx, validator)
	require.Error(t, input.OracleKeeper.ValidateFeeder(input.Ctx, sdk.AccAddress(addr1), addr))
}
