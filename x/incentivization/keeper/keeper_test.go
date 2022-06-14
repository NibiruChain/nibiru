package keeper_test

import (
	"testing"
	"time"

	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"

	"github.com/NibiruChain/nibiru/x/incentivization/types"

	"github.com/cosmos/cosmos-sdk/simapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/x/incentivization/keeper"
	testutilapp "github.com/NibiruChain/nibiru/x/testutil/app"
	"github.com/NibiruChain/nibiru/x/testutil/sample"
)

func TestKeeper_CreateIncentivizationProgram(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		app, ctx := testutilapp.NewNibiruApp(false)

		createdProgram, err := app.IncentivizationKeeper.CreateIncentivizationProgram(ctx, "denom", 48*time.Hour, ctx.BlockTime(), 1000)
		require.NoError(t, err)

		gotProgram, err := app.IncentivizationKeeper.IncentivizationProgramsState(ctx).Get(0)
		require.NoError(t, err)
		require.Equal(t, createdProgram, gotProgram)

		require.Equal(t, uint64(0), createdProgram.Id)
		require.Equal(t, authtypes.NewModuleAddress(keeper.NewEscrowAccountName(0)).String(), createdProgram.EscrowAddress)
	})
	t.Run("min lockup duration too low", func(t *testing.T) {
		app, ctx := testutilapp.NewNibiruApp(false)

		_, err := app.IncentivizationKeeper.CreateIncentivizationProgram(ctx, "denom", 1*time.Second, ctx.BlockTime(), keeper.MinEpochs)

		require.ErrorIs(t, err, types.ErrMinLockupDurationTooLow)
	})

	t.Run("epochs lower than minimum", func(t *testing.T) {
		app, ctx := testutilapp.NewNibiruApp(false)

		_, err := app.IncentivizationKeeper.CreateIncentivizationProgram(ctx, "denom", 48*time.Hour, ctx.BlockTime(), keeper.MinEpochs-1)

		require.ErrorIs(t, err, types.ErrEpochsTooLow)
	})

	t.Run("start time before block time", func(t *testing.T) {
		app := testutilapp.NewTestApp(false)
		ctx := app.NewContext(false, tmproto.Header{Time: time.Now()})
		_, err := app.IncentivizationKeeper.CreateIncentivizationProgram(ctx, "denom", 48*time.Hour, ctx.BlockTime().Add(-1*time.Second), keeper.MinEpochs+1)

		require.ErrorIs(t, err, types.ErrStartTimeInPast)
	})
}

func TestKeeper_FundIncentivizationProgram(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		app, ctx := testutilapp.NewNibiruApp(false)
		addr := sample.AccAddress()
		fundingAmount := sdk.NewCoins(sdk.NewInt64Coin("test", 100))
		require.NoError(t, simapp.FundAccount(app.BankKeeper, ctx, addr, fundingAmount))

		createdProgram, err := app.IncentivizationKeeper.CreateIncentivizationProgram(ctx, "denom", 48*time.Hour, ctx.BlockTime(), 1000)
		require.NoError(t, err)

		err = app.IncentivizationKeeper.FundIncentivizationProgram(ctx, createdProgram.Id, addr, fundingAmount)
		require.NoError(t, err)

		escrowAddr, err := sdk.AccAddressFromBech32(createdProgram.EscrowAddress)
		require.NoError(t, err)

		balance := app.BankKeeper.GetAllBalances(ctx, escrowAddr)
		require.Equal(t, balance, fundingAmount)
	})
}
