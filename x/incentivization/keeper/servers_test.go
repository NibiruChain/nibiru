package keeper_test

import (
	"github.com/NibiruChain/nibiru/x/incentivization/keeper"
	"github.com/NibiruChain/nibiru/x/incentivization/types"
	"github.com/NibiruChain/nibiru/x/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/stretchr/testify/require"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"
	"testing"
	"time"
)

func TestQueryServer_IncentivizationProgram(t *testing.T) {
	app := testutil.NewTestApp(false)
	q := keeper.NewQueryServer(app.IncentivizationKeeper)
	ctx := app.NewContext(false, tmproto.Header{Time: time.Now()})

	// init
	program, err := app.IncentivizationKeeper.CreateIncentivizationProgram(ctx, "lp", 24*time.Hour, time.Now().Add(1*time.Second), 100)
	require.NoError(t, err)
	program.StartTime = program.StartTime.UTC()

	resp, err := q.IncentivizationProgram(sdk.WrapSDKContext(ctx), &types.QueryIncentivizationProgramRequest{})
	require.NoError(t, err)
	require.Equal(t, program, resp.IncentivizationProgram)
}

func TestQueryServer_IncentivizationPrograms(t *testing.T) {
	app := testutil.NewTestApp(false)
	q := keeper.NewQueryServer(app.IncentivizationKeeper)
	ctx := app.NewContext(false, tmproto.Header{Time: time.Now()})

	// init
	program1, err := app.IncentivizationKeeper.CreateIncentivizationProgram(ctx, "lp", 24*time.Hour, time.Now().Add(1*time.Second), 100)
	require.NoError(t, err)
	program1.StartTime = program1.StartTime.UTC()

	program2, err := app.IncentivizationKeeper.CreateIncentivizationProgram(ctx, "lp", 24*time.Hour, time.Now().Add(1*time.Second), 100)
	require.NoError(t, err)
	program2.StartTime = program2.StartTime.UTC()

	program3, err := app.IncentivizationKeeper.CreateIncentivizationProgram(ctx, "lp", 24*time.Hour, time.Now().Add(1*time.Second), 100)
	require.NoError(t, err)
	program3.StartTime = program3.StartTime.UTC()

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
