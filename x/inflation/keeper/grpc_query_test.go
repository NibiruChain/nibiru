package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/x/common/testutil/testapp"
	"github.com/NibiruChain/nibiru/x/inflation/types"
)

func TestPeriod(t *testing.T) {
	nibiruApp, ctx := testapp.NewNibiruTestAppAndContext(true)

	resp, err := nibiruApp.InflationKeeper.Period(sdk.WrapSDKContext(ctx), &types.QueryPeriodRequest{})

	require.NoError(t, err)
	assert.Equal(t, uint64(0), resp.Period)

	nibiruApp.InflationKeeper.CurrentPeriod.Next(ctx)

	resp, err = nibiruApp.InflationKeeper.Period(sdk.WrapSDKContext(ctx), &types.QueryPeriodRequest{})
	require.NoError(t, err)
	assert.Equal(t, uint64(1), resp.Period)
}

func SkippedEpochs(t *testing.T) {
	nibiruApp, ctx := testapp.NewNibiruTestAppAndContext(true)

	resp, err := nibiruApp.InflationKeeper.SkippedEpochs(sdk.WrapSDKContext(ctx), &types.QuerySkippedEpochsRequest{})

	require.NoError(t, err)
	assert.Equal(t, uint64(0), resp.SkippedEpochs)

	nibiruApp.InflationKeeper.NumSkippedEpochs.Next(ctx)

	resp, err = nibiruApp.InflationKeeper.SkippedEpochs(sdk.WrapSDKContext(ctx), &types.QuerySkippedEpochsRequest{})
	require.NoError(t, err)
	assert.Equal(t, uint64(1), resp.SkippedEpochs)
}
