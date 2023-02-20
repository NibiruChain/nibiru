package keeper_test

import (
	gocontext "context"
	"testing"
	"time"

	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/x/common/testutil/testapp"
	"github.com/NibiruChain/nibiru/x/epochs/keeper"

	"github.com/NibiruChain/nibiru/x/epochs/types"
)

func TestQueryEpochInfos(t *testing.T) {
	nibiruApp, ctx := testapp.NewNibiruTestAppAndContext(true)

	queryHelper := baseapp.NewQueryServerTestHelper(ctx, nibiruApp.InterfaceRegistry())
	types.RegisterQueryServer(queryHelper, keeper.NewQuerier(nibiruApp.EpochsKeeper))
	queryClient := types.NewQueryClient(queryHelper)

	chainStartTime := ctx.BlockTime()

	// Invalid param
	epochInfosResponse, err := queryClient.EpochInfos(gocontext.Background(), &types.QueryEpochsInfoRequest{})
	require.NoError(t, err)
	require.Len(t, epochInfosResponse.Epochs, 4)

	// check if EpochInfos are correct
	require.Equal(t, epochInfosResponse.Epochs[2].Identifier, "day")
	require.Equal(t, epochInfosResponse.Epochs[2].StartTime, chainStartTime)
	require.Equal(t, epochInfosResponse.Epochs[2].Duration, time.Hour*24)
	require.Equal(t, epochInfosResponse.Epochs[2].CurrentEpoch, uint64(0))
	require.Equal(t, epochInfosResponse.Epochs[2].CurrentEpochStartTime, chainStartTime)
	require.Equal(t, epochInfosResponse.Epochs[2].EpochCountingStarted, false)
	require.Equal(t, epochInfosResponse.Epochs[3].Identifier, "week")
	require.Equal(t, epochInfosResponse.Epochs[3].StartTime, chainStartTime)
	require.Equal(t, epochInfosResponse.Epochs[3].Duration, time.Hour*24*7)
	require.Equal(t, epochInfosResponse.Epochs[3].CurrentEpoch, uint64(0))
	require.Equal(t, epochInfosResponse.Epochs[3].CurrentEpochStartTime, chainStartTime)
	require.Equal(t, epochInfosResponse.Epochs[3].EpochCountingStarted, false)
}
