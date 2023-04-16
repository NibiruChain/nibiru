package keeper_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/NibiruChain/nibiru/x/common/denoms"
	"github.com/NibiruChain/nibiru/x/common/testutil/testapp"
	epochstypes "github.com/NibiruChain/nibiru/x/epochs/types"
	"github.com/NibiruChain/nibiru/x/inflation/types"
	"github.com/stretchr/testify/require"
)

func TestEpochIdentifierAfterEpochEnd(t *testing.T) {
	nibiruApp, ctx := testapp.NewNibiruTestAppAndContext(true)

	feePoolOld := nibiruApp.DistrKeeper.GetFeePool(ctx)
	nibiruApp.EpochsKeeper.AfterEpochEnd(ctx, epochstypes.DayEpochID, 1)
	feePoolNew := nibiruApp.DistrKeeper.GetFeePool(ctx)

	require.Greater(t, feePoolNew.CommunityPool.AmountOf(denoms.NIBI).BigInt().Uint64(),
		feePoolOld.CommunityPool.AmountOf(denoms.NIBI).BigInt().Uint64())
}

func TestPeriodChangesSkippedEpochsAfterEpochEnd(t *testing.T) {
	nibiruApp, ctx := testapp.NewNibiruTestAppAndContext(true)
	currentEpochPeriod := nibiruApp.InflationKeeper.EpochsPerPeriod(ctx)

	testCases := []struct {
		name             string
		currentPeriod    uint64
		height           uint64
		epochIdentifier  string
		skippedEpochs    uint64
		InflationEnabled bool
		periodChanges    bool
	}{
		{
			"SkippedEpoch set DayEpochID disabledInflation",
			0,
			currentEpochPeriod - 10, // so it's within range
			epochstypes.DayEpochID,
			0,
			false,
			false,
		},
		{
			"SkippedEpoch set WeekEpochID disabledInflation ",
			0,
			currentEpochPeriod - 10, // so it's within range
			epochstypes.WeekEpochID,
			0,
			false,
			false,
		},
		{
			"[Period 0] disabledInflation",
			0,
			currentEpochPeriod - 10, // so it's within range
			epochstypes.DayEpochID,
			0,
			false,
			false,
		},
		{
			"[Period 0] period stays the same under epochs per period",
			0,
			currentEpochPeriod - 10, // so it's within range
			epochstypes.DayEpochID,
			0,
			true,
			false,
		},
		{
			"[Period 0] period changes once enough epochs have passed",
			0,
			currentEpochPeriod + 1,
			epochstypes.DayEpochID,
			0,
			true,
			true,
		},
		{
			"[Period 1] period stays the same under the epoch per period",
			1,
			2*currentEpochPeriod - 1,
			epochstypes.DayEpochID,
			0,
			true,
			false,
		},
		{
			"[Period 1] period changes once enough epochs have passed",
			1,
			2*currentEpochPeriod + 1,
			epochstypes.DayEpochID,
			0,
			true,
			true,
		},
		{
			"[Period 0] with skipped epochs - period stays the same under epochs per period",
			0,
			currentEpochPeriod - 1,
			epochstypes.DayEpochID,
			10,
			true,
			false,
		},
		{
			"[Period 0] with skipped epochs - period stays the same under epochs per period",
			0,
			currentEpochPeriod + 1,
			epochstypes.DayEpochID,
			10,
			true,
			false,
		},
		{
			"[Period 0] with skipped epochs - period changes once enough epochs have passed",
			0,
			currentEpochPeriod + 11,
			epochstypes.DayEpochID,
			10,
			true,
			true,
		},
		{
			"[Period 1] with skipped epochs - period stays the same under epochs per period",
			1,
			2*currentEpochPeriod + 1,
			epochstypes.DayEpochID,
			10,
			true,
			false,
		},
		{
			"[Period 1] with skipped epochs - period changes once enough epochs have passed",
			1,
			2*currentEpochPeriod + 11,
			epochstypes.DayEpochID,
			10,
			true,
			true,
		},
	}
	for _, tc := range testCases {
		t.Run(fmt.Sprintf("Case %s", tc.name), func(t *testing.T) {
			params := nibiruApp.InflationKeeper.GetParams(ctx)
			params.InflationEnabled = tc.InflationEnabled
			nibiruApp.InflationKeeper.SetParams(ctx, params)

			nibiruApp.InflationKeeper.NumSkippedEpochs.Set(ctx, tc.skippedEpochs)
			nibiruApp.InflationKeeper.CurrentPeriod.Set(ctx, tc.currentPeriod)

			currentSkippedEpochs := nibiruApp.InflationKeeper.NumSkippedEpochs.Peek(ctx)
			currentPeriod := nibiruApp.InflationKeeper.CurrentPeriod.Peek(ctx)
			originalProvision := nibiruApp.InflationKeeper.GetEpochMintProvision(ctx)

			// Perform Epoch Hooks
			futureCtx := ctx.WithBlockTime(time.Now().Add(time.Minute))
			nibiruApp.EpochsKeeper.BeforeEpochStart(futureCtx, tc.epochIdentifier, tc.height)
			nibiruApp.EpochsKeeper.AfterEpochEnd(futureCtx, tc.epochIdentifier, tc.height)

			skippedEpochs := nibiruApp.InflationKeeper.NumSkippedEpochs.Peek(ctx)
			period := nibiruApp.InflationKeeper.CurrentPeriod.Peek(ctx)

			if tc.periodChanges {
				newProvision := nibiruApp.InflationKeeper.GetEpochMintProvision(ctx)
				expectedProvision := types.CalculateEpochMintProvision(
					nibiruApp.InflationKeeper.GetParams(ctx),
					period,
				)
				require.Equal(t, expectedProvision, newProvision)
				// mint provisions will change
				require.NotEqual(t, newProvision.BigInt().Uint64(), originalProvision.BigInt().Uint64())
				require.Equal(t, currentSkippedEpochs, skippedEpochs)
				require.Equal(t, currentPeriod+1, period)
			} else {
				require.Equal(t, currentPeriod, period)
				if !tc.InflationEnabled {
					// Check for epochIdentifier for skippedEpoch increment
					if tc.epochIdentifier == epochstypes.DayEpochID {
						require.Equal(t, currentSkippedEpochs+1, skippedEpochs)
					}
				}
			}
		})
	}
}
