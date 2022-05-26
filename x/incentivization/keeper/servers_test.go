package keeper_test

import (
	"testing"
	"time"

	"github.com/cosmos/cosmos-sdk/simapp"

	"github.com/NibiruChain/nibiru/x/testutil/sample"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/stretchr/testify/require"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"

	"github.com/NibiruChain/nibiru/x/incentivization/keeper"
	"github.com/NibiruChain/nibiru/x/incentivization/types"
	"github.com/NibiruChain/nibiru/x/testutil"
)

func TestMsgServer_CreateIncentivizationProgram(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		app := testutil.NewTestApp(false)
		s := keeper.NewMsgServer(app.IncentivizationKeeper)
		ctx := app.NewContext(false, tmproto.Header{Time: time.Now()})

		addr := sample.AccAddress()
		funds := sdk.NewCoins(sdk.NewInt64Coin("test", 1000))
		require.NoError(t, simapp.FundAccount(app.BankKeeper, ctx, addr, funds))

		lockDuration := keeper.MinLockupDuration
		startTime := ctx.BlockTime().Add(10 * time.Second)

		// here we test the whole flow, custom start time + initial funds
		resp, err := s.CreateIncentivizationProgram(sdk.WrapSDKContext(ctx), &types.MsgCreateIncentivizationProgram{
			Sender:            addr.String(),
			LpDenom:           "denom",
			MinLockupDuration: &lockDuration,
			StartTime:         &startTime,
			Epochs:            keeper.MinEpochs,
			InitialFunds:      funds,
		})
		require.NoError(t, err)

		program, err := app.IncentivizationKeeper.IncentivizationProgramsState(ctx).Get(resp.ProgramId)
		require.NoError(t, err)

		escrowAddr, err := sdk.AccAddressFromBech32(program.EscrowAddress)
		require.NoError(t, err)

		escrowBalance := app.BankKeeper.GetAllBalances(ctx, escrowAddr)
		require.Equal(t, funds, escrowBalance)
	})
}

func TestMsgServer_FundIncentivizationProgram(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		app := testutil.NewTestApp(false)
		s := keeper.NewMsgServer(app.IncentivizationKeeper)
		ctx := app.NewContext(false, tmproto.Header{Time: time.Now()})

		addr := sample.AccAddress()
		// create program
		lockDuration := keeper.MinLockupDuration
		creationResp, err := s.CreateIncentivizationProgram(sdk.WrapSDKContext(ctx), &types.MsgCreateIncentivizationProgram{
			Sender:            addr.String(),
			LpDenom:           "denom",
			MinLockupDuration: &lockDuration,
			Epochs:            keeper.MinEpochs,
		})
		require.NoError(t, err)

		// fund account
		funds := sdk.NewCoins(sdk.NewInt64Coin("test", 1000))
		require.NoError(t, simapp.FundAccount(app.BankKeeper, ctx, addr, funds))
		// fund incentivization program
		_, err = s.FundIncentivizationProgram(sdk.WrapSDKContext(ctx), &types.MsgFundIncentivizationProgram{
			Sender: addr.String(),
			Id:     creationResp.ProgramId,
			Funds:  funds,
		})
		require.NoError(t, err)

		// assert the program was funded
		program, err := app.IncentivizationKeeper.IncentivizationProgramsState(ctx).Get(creationResp.ProgramId)
		require.NoError(t, err)

		escrowAddr, err := sdk.AccAddressFromBech32(program.EscrowAddress)
		require.NoError(t, err)

		escrowBalance := app.BankKeeper.GetAllBalances(ctx, escrowAddr)
		require.Equal(t, funds, escrowBalance)
	})
}

func TestQueryServer_IncentivizationProgram(t *testing.T) {
	app := testutil.NewTestApp(false)
	q := keeper.NewQueryServer(app.IncentivizationKeeper)
	ctx := app.NewContext(false, tmproto.Header{Time: time.Now()})

	// init
	program, err := app.IncentivizationKeeper.CreateIncentivizationProgram(ctx, "lp", 24*time.Hour, time.Now().Add(1*time.Second).UTC(), 100)
	require.NoError(t, err)

	resp, err := q.IncentivizationProgram(sdk.WrapSDKContext(ctx), &types.QueryIncentivizationProgramRequest{})
	require.NoError(t, err)
	require.Equal(t, program, resp.IncentivizationProgram)
}

func TestQueryServer_IncentivizationPrograms(t *testing.T) {
	app := testutil.NewTestApp(false)
	q := keeper.NewQueryServer(app.IncentivizationKeeper)
	ctx := app.NewContext(false, tmproto.Header{Time: time.Now()})

	// init
	_, err := app.IncentivizationKeeper.CreateIncentivizationProgram(ctx, "lp", 24*time.Hour, time.Now().Add(1*time.Second).UTC(), 100)
	require.NoError(t, err)

	program2, err := app.IncentivizationKeeper.CreateIncentivizationProgram(ctx, "lp", 24*time.Hour, time.Now().Add(1*time.Second).UTC(), 100)
	require.NoError(t, err)

	program3, err := app.IncentivizationKeeper.CreateIncentivizationProgram(ctx, "lp", 24*time.Hour, time.Now().Add(1*time.Second).UTC(), 100)
	require.NoError(t, err)

	// query
	resp, err := q.IncentivizationPrograms(sdk.WrapSDKContext(ctx), &types.QueryIncentivizationProgramsRequest{Pagination: &query.PageRequest{
		Offset: 1,
		Limit:  2,
	}})
	require.NoError(t, err)

	require.Equal(t, []*types.IncentivizationProgram{
		program2, program3,
	}, resp.IncentivizationPrograms)
}
