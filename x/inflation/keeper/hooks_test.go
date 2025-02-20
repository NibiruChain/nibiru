package keeper_test

import (
	"fmt"
	"testing"
	"time"

	"cosmossdk.io/math"
	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/x/common/denoms"
	"github.com/NibiruChain/nibiru/x/common/testutil/testapp"
	epochstypes "github.com/NibiruChain/nibiru/x/epochs/types"
	"github.com/NibiruChain/nibiru/x/inflation/types"
)

// TestEpochIdentifierAfterEpochEnd: Ensures that the amount in the community
// pool after an epoch ends is greater than the amount before the epoch ends
// with the default module parameters.
func TestEpochIdentifierAfterEpochEnd(t *testing.T) {
	nibiruApp, ctx := testapp.NewNibiruTestAppAndContext()

	params := nibiruApp.InflationKeeper.GetParams(ctx)
	params.InflationEnabled = true
	nibiruApp.InflationKeeper.Params.Set(ctx, params)

	feePoolOld := nibiruApp.DistrKeeper.GetFeePool(ctx)
	nibiruApp.EpochsKeeper.AfterEpochEnd(ctx, epochstypes.DayEpochID, 1)
	feePoolNew := nibiruApp.DistrKeeper.GetFeePool(ctx)

	require.Greater(t, feePoolNew.CommunityPool.AmountOf(denoms.NIBI).BigInt().Uint64(),
		feePoolOld.CommunityPool.AmountOf(denoms.NIBI).BigInt().Uint64())
}

// TestPeriodChangesSkippedEpochsAfterEpochEnd: Tests whether current period and
// the number of skipped epochs are accurately updated and that skipped epochs
// are handled correctly.
func TestPeriodChangesSkippedEpochsAfterEpochEnd(t *testing.T) {
	nibiruApp, ctx := testapp.NewNibiruTestAppAndContext()
	inflationKeeper := nibiruApp.InflationKeeper

	testCases := []struct {
		name               string
		currentPeriod      uint64
		currentEpochNumber uint64
		epochIdentifier    string
		skippedEpochs      uint64
		InflationEnabled   bool
		periodChanges      bool
	}{
		{
			name:               "SkippedEpoch set DayEpochID disabledInflation",
			currentPeriod:      0,
			currentEpochNumber: 20, // so it's within range
			epochIdentifier:    epochstypes.DayEpochID,
			skippedEpochs:      19,
			InflationEnabled:   false,
			periodChanges:      false,
		},
		{
			name:               "SkippedEpoch set WeekEpochID disabledInflation ",
			currentPeriod:      0,
			currentEpochNumber: 20, // so it's within range
			epochIdentifier:    epochstypes.WeekEpochID,
			skippedEpochs:      19,
			InflationEnabled:   false,
			periodChanges:      false,
		},
		{
			name:               "[Period 0] disabledInflation",
			currentPeriod:      0,
			currentEpochNumber: 20, // so it's within range
			epochIdentifier:    epochstypes.DayEpochID,
			skippedEpochs:      19,
			InflationEnabled:   false,
			periodChanges:      false,
		},
		{
			name:               "[Period 0] period stays the same under epochs per period",
			currentPeriod:      0,
			currentEpochNumber: 29, // so it's within range
			epochIdentifier:    epochstypes.DayEpochID,
			skippedEpochs:      0,
			InflationEnabled:   true,
			periodChanges:      false,
		},
		{
			name:               "[Period 0] period changes once enough epochs have passed",
			currentPeriod:      0,
			currentEpochNumber: 30,
			epochIdentifier:    epochstypes.DayEpochID,
			skippedEpochs:      0,
			InflationEnabled:   true,
			periodChanges:      true,
		},
		{
			name:               "[Period 1] period stays the same under the epoch per period",
			currentPeriod:      1,
			currentEpochNumber: 59, // period change is at the end of epoch 59
			epochIdentifier:    epochstypes.DayEpochID,
			skippedEpochs:      0,
			InflationEnabled:   true,
			periodChanges:      false,
		},
		{
			name:               "[Period 1] period changes once enough epochs have passed",
			currentPeriod:      1,
			currentEpochNumber: 60,
			epochIdentifier:    epochstypes.DayEpochID,
			skippedEpochs:      0,
			InflationEnabled:   true,
			periodChanges:      true,
		},
		{
			name:               "[Period 0] with skipped epochs - period stays the same under epochs per period",
			currentPeriod:      0,
			currentEpochNumber: 30,
			epochIdentifier:    epochstypes.DayEpochID,
			skippedEpochs:      1,
			InflationEnabled:   true,
			periodChanges:      false,
		},
		{
			name:               "[Period 0] with skipped epochs - period changes once enough epochs have passed",
			currentPeriod:      0,
			currentEpochNumber: 40,
			epochIdentifier:    epochstypes.DayEpochID,
			skippedEpochs:      10,
			InflationEnabled:   true,
			periodChanges:      true,
		},
		{
			name:               "[Period 1] with skipped epochs - period stays the same under epochs per period",
			currentPeriod:      1,
			currentEpochNumber: 69,
			epochIdentifier:    epochstypes.DayEpochID,
			skippedEpochs:      10,
			InflationEnabled:   true,
			periodChanges:      false,
		},
		{
			name:               "[Period 1] with skipped epochs - period changes once enough epochs have passed",
			currentPeriod:      1,
			currentEpochNumber: 70,
			epochIdentifier:    epochstypes.DayEpochID,
			skippedEpochs:      10,
			InflationEnabled:   true,
			periodChanges:      true,
		},
	}
	for _, tc := range testCases {
		t.Run(fmt.Sprintf("Case %s", tc.name), func(t *testing.T) {
			params := inflationKeeper.GetParams(ctx)
			params.InflationEnabled = tc.InflationEnabled
			params.HasInflationStarted = tc.InflationEnabled
			inflationKeeper.Params.Set(ctx, params)

			inflationKeeper.NumSkippedEpochs.Set(ctx, tc.skippedEpochs)
			inflationKeeper.CurrentPeriod.Set(ctx, tc.currentPeriod)

			// original values
			prevSkippedEpochs := inflationKeeper.NumSkippedEpochs.Peek(ctx)
			prevPeriod := inflationKeeper.CurrentPeriod.Peek(ctx)
			originalProvision := inflationKeeper.GetEpochMintProvision(ctx)

			// Perform Epoch Hooks
			futureCtx := ctx.WithBlockTime(time.Now().Add(time.Minute))
			nibiruApp.EpochsKeeper.BeforeEpochStart(futureCtx, tc.epochIdentifier, tc.currentEpochNumber)
			nibiruApp.EpochsKeeper.AfterEpochEnd(futureCtx, tc.epochIdentifier, tc.currentEpochNumber)

			// new values
			newSkippedEpochs := inflationKeeper.NumSkippedEpochs.Peek(ctx)
			newPeriod := inflationKeeper.CurrentPeriod.Peek(ctx)

			if tc.periodChanges {
				newProvision := inflationKeeper.GetEpochMintProvision(ctx)

				expectedProvision := types.CalculateEpochMintProvision(
					inflationKeeper.GetParams(ctx),
					newPeriod,
				)

				require.Equal(t, expectedProvision, newProvision)
				// mint provisions will change
				require.NotEqual(t, newProvision, originalProvision)
				require.Equal(t, prevSkippedEpochs, newSkippedEpochs)
				require.Equal(t, prevPeriod+1, newPeriod)
			} else {
				require.Equal(t, prevPeriod, newPeriod, "period should not change but it did")
				if !tc.InflationEnabled && tc.epochIdentifier == epochstypes.DayEpochID {
					// Check for epochIdentifier for skippedEpoch increment
					require.EqualValues(t, prevSkippedEpochs+1, newSkippedEpochs)
				}
			}
		})
	}
}

func GetBalanceStaking(ctx sdk.Context, nibiruApp *app.NibiruApp) sdkmath.Int {
	return nibiruApp.BankKeeper.GetBalance(
		ctx,
		nibiruApp.AccountKeeper.GetModuleAddress(authtypes.FeeCollectorName),
		denoms.NIBI,
	).Amount
}

func TestManual(t *testing.T) {
	// This test is a manual test to check if the inflation is working as expected
	// We turn off inflation, then we turn it on and check if the balance is increasing
	// We turn it off again and check if the balance is not increasing
	// We turn it on again and check if the balance is increasing again with the correct amount

	nibiruApp, ctx := testapp.NewNibiruTestAppAndContext()
	inflationKeeper := nibiruApp.InflationKeeper

	params := inflationKeeper.GetParams(ctx)
	epochNumber := uint64(1)

	params.InflationEnabled = false
	params.HasInflationStarted = false
	params.EpochsPerPeriod = 30

	// y = 3 * x + 3 -> 3 nibi per epoch for period 0, 6 nibi per epoch for period 1
	params.PolynomialFactors = []math.LegacyDec{math.LegacyNewDec(3), math.LegacyNewDec(3)}
	params.InflationDistribution = types.InflationDistribution{
		CommunityPool:     math.LegacyZeroDec(),
		StakingRewards:    math.LegacyOneDec(),
		StrategicReserves: math.LegacyZeroDec(),
	}

	inflationKeeper.Params.Set(ctx, params)

	require.Equal(t, math.ZeroInt(), GetBalanceStaking(ctx, nibiruApp))

	for i := 0; i < 42069; i++ {
		inflationKeeper.Hooks().AfterEpochEnd(ctx, epochstypes.DayEpochID, epochNumber)
		epochNumber++
	}
	require.Equal(t, math.ZeroInt(), GetBalanceStaking(ctx, nibiruApp))
	require.EqualValues(t, uint64(0), inflationKeeper.CurrentPeriod.Peek(ctx))
	require.EqualValues(t, uint64(42069), inflationKeeper.NumSkippedEpochs.Peek(ctx))

	nibiruApp.EpochsKeeper.Epochs.Insert(ctx, epochstypes.DayEpochID, epochstypes.EpochInfo{
		Identifier:              epochstypes.DayEpochID,
		StartTime:               time.Now(),
		Duration:                0,
		CurrentEpoch:            42069,
		CurrentEpochStartTime:   time.Now(),
		EpochCountingStarted:    false,
		CurrentEpochStartHeight: 0,
	},
	)
	err := inflationKeeper.Sudo().ToggleInflation(ctx, true, testapp.DefaultSudoRoot())
	require.NoError(t, err)

	// Period 0 - inflate 3M NIBI over 30 epochs or 100k uNIBI per epoch
	for i := 0; i < 30; i++ {
		inflationKeeper.Hooks().AfterEpochEnd(ctx, epochstypes.DayEpochID, epochNumber)
		require.Equal(t, math.NewInt(100_000).Mul(math.NewInt(int64(i+1))), GetBalanceStaking(ctx, nibiruApp))
		epochNumber++
	}
	require.Equal(t, math.NewInt(3_000_000), GetBalanceStaking(ctx, nibiruApp))
	require.EqualValues(t, uint64(1), inflationKeeper.CurrentPeriod.Peek(ctx))
	require.EqualValues(t, uint64(42069), inflationKeeper.NumSkippedEpochs.Peek(ctx))

	err = inflationKeeper.Sudo().ToggleInflation(ctx, false, testapp.DefaultSudoRoot())
	require.NoError(t, err)

	for i := 0; i < 42069; i++ {
		inflationKeeper.Hooks().AfterEpochEnd(ctx, epochstypes.DayEpochID, epochNumber)
		epochNumber++
	}
	require.Equal(t, math.NewInt(3_000_000), GetBalanceStaking(ctx, nibiruApp))
	require.EqualValues(t, uint64(1), inflationKeeper.CurrentPeriod.Peek(ctx))
	require.EqualValues(t, uint64(84138), inflationKeeper.NumSkippedEpochs.Peek(ctx))

	err = inflationKeeper.Sudo().ToggleInflation(ctx, true, testapp.DefaultSudoRoot())
	require.NoError(t, err)

	// Period 1 - inflate 6M NIBI over 30 epochs or 200k uNIBI per epoch
	for i := 0; i < 30; i++ {
		inflationKeeper.Hooks().AfterEpochEnd(ctx, epochstypes.DayEpochID, epochNumber)
		require.Equal(t, math.NewInt(3_000_000).Add(math.NewInt(200_000).Mul(math.NewInt(int64(i+1)))), GetBalanceStaking(ctx, nibiruApp))
		epochNumber++
	}
	require.Equal(t, math.NewInt(9_000_000), GetBalanceStaking(ctx, nibiruApp))
	require.EqualValues(t, uint64(2), inflationKeeper.CurrentPeriod.Peek(ctx))
	require.EqualValues(t, uint64(84138), inflationKeeper.NumSkippedEpochs.Peek(ctx))

	require.EqualValues(t, uint64(1+2*42069+60), epochNumber)
}
