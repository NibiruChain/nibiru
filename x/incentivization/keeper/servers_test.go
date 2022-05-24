package keeper_test

import (
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/stretchr/testify/require"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"

	"github.com/NibiruChain/nibiru/x/incentivization/keeper"
	"github.com/NibiruChain/nibiru/x/incentivization/types"
	"github.com/NibiruChain/nibiru/x/testutil"
)

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
