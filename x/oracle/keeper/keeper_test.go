package keeper

import (
	"testing"

	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"

	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/stretchr/testify/require"
)

func TestValidateFeeder(t *testing.T) {
	// initial setup
	input := CreateTestFixture(t)
	addr, val := ValAddrs[0], ValPubKeys[0]
	addr1, val1 := ValAddrs[1], ValPubKeys[1]
	amt := sdk.TokensFromConsensusPower(100, sdk.DefaultPowerReduction)
	sh := stakingkeeper.NewMsgServerImpl(&input.StakingKeeper)
	ctx := input.Ctx

	// Create 2 validators.
	_, err := sh.CreateValidator(ctx, NewTestMsgCreateValidator(addr, val, amt))
	require.NoError(t, err)
	_, err = sh.CreateValidator(ctx, NewTestMsgCreateValidator(addr1, val1, amt))
	require.NoError(t, err)
	_, err = input.StakingKeeper.EndBlocker(ctx)
	require.NoError(t, err)

	params, err := input.StakingKeeper.GetParams(ctx)
	require.NoError(t, err)
	require.Equal(
		t, input.BankKeeper.GetAllBalances(ctx, sdk.AccAddress(addr)),
		sdk.NewCoins(sdk.NewCoin(params.BondDenom, InitTokens.Sub(amt))),
	)
	validator, err := input.StakingKeeper.GetValidator(ctx, addr)
	require.NoError(t, err)
	require.Equal(t, amt, validator.GetBondedTokens())
	require.Equal(
		t, input.BankKeeper.GetAllBalances(ctx, sdk.AccAddress(addr1)),
		sdk.NewCoins(sdk.NewCoin(params.BondDenom, InitTokens.Sub(amt))),
	)
	require.Equal(t, amt, validator.GetBondedTokens())

	require.NoError(t, input.OracleKeeper.ValidateFeeder(input.Ctx, sdk.AccAddress(addr), sdk.ValAddress(addr)))
	require.NoError(t, input.OracleKeeper.ValidateFeeder(input.Ctx, sdk.AccAddress(addr1), sdk.ValAddress(addr1)))

	// delegate works
	input.OracleKeeper.FeederDelegations.Insert(input.Ctx, addr, sdk.AccAddress(addr1))
	require.NoError(t, input.OracleKeeper.ValidateFeeder(input.Ctx, sdk.AccAddress(addr1), addr))
	require.Error(t, input.OracleKeeper.ValidateFeeder(input.Ctx, Addrs[2], addr))

	// only active validators can do oracle votes
	validator.Status = stakingtypes.Unbonded
	input.StakingKeeper.SetValidator(input.Ctx, validator)
	require.Error(t, input.OracleKeeper.ValidateFeeder(input.Ctx, sdk.AccAddress(addr1), addr))
}
