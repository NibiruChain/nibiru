package types

import (
	fmt "fmt"
	"testing"

	sdkmath "cosmossdk.io/math"

	"github.com/stretchr/testify/require"
)

// These numbers are for year n month 1
var ExpectedYearlyInflation = []sdkmath.LegacyDec{
	sdkmath.LegacyNewDec(193_333_719e6),
	sdkmath.LegacyNewDec(154_304_107e6),
	sdkmath.LegacyNewDec(123_153_673e6),
	sdkmath.LegacyNewDec(98_291_791e6),
	sdkmath.LegacyNewDec(78_448_949e6),
	sdkmath.LegacyNewDec(62_611_919e6),
	sdkmath.LegacyNewDec(49_972_019e6),
	sdkmath.LegacyNewDec(39_883_823e6),
}

// ExpectedTotalInflation is the total amount of NIBI tokens (in unibi) that
// should be minted via inflation for the network to reach its target supply.
// The value 800M is equivalent to:
// = (Community allocation of total supply) - (Community supply at start)
// = (60% of the total supply) - (Community supply at start)
// = (60% of 1.5 billion) - (Community supply at start)
// = 800 million NIBI
var ExpectedTotalInflation = sdkmath.LegacyNewDec(800_000_000e6)

func TestCalculateEpochMintProvision(t *testing.T) {
	params := DefaultParams()
	params.InflationEnabled = true

	period := uint64(0)
	totalInflation := sdkmath.LegacyZeroDec()

	// Only the first 8 years have inflation with default params, but we run
	// for 10 years expecting 0 inflation in the last 2 years.
	for year := uint64(0); year < 10; year++ {
		yearlyInflation := sdkmath.LegacyZeroDec()
		for month := uint64(0); month < 12; month++ {
			for day := uint64(0); day < 30; day++ {
				epochMintProvisions := CalculateEpochMintProvision(params, period)
				yearlyInflation = yearlyInflation.Add(epochMintProvisions)
			}
			period++
		}
		// Should be within 0.0098%
		if year < uint64(len(ExpectedYearlyInflation)) {
			require.NoError(t, withinRange(ExpectedYearlyInflation[year], yearlyInflation))
		} else {
			require.Equal(t, yearlyInflation, sdkmath.LegacyZeroDec())
		}
		totalInflation = totalInflation.Add(yearlyInflation)
	}
	require.NoError(t, withinRange(ExpectedTotalInflation, totalInflation))
}

func TestCalculateEpochMintProvisionInflationNotEnabled(t *testing.T) {
	params := DefaultParams()
	params.InflationEnabled = false

	epochId := uint64(0)
	totalInflation := sdkmath.LegacyZeroDec()

	// Only the first 8 years have inflation with default params, but we run
	// for 10 years expecting 0 inflation
	for year := uint64(0); year < 10; year++ {
		yearlyInflation := sdkmath.LegacyZeroDec()
		for month := uint64(0); month < 12; month++ {
			for day := uint64(0); day < 30; day++ {
				epochMintProvisions := CalculateEpochMintProvision(params, epochId)
				yearlyInflation = yearlyInflation.Add(epochMintProvisions)
			}
			epochId++
		}

		require.Equal(t, yearlyInflation, sdkmath.LegacyZeroDec())
		totalInflation = totalInflation.Add(yearlyInflation)
	}
	require.Equal(t, totalInflation, sdkmath.LegacyZeroDec())
}

func TestCalculateEpochMintProvision_ZeroEpochs(t *testing.T) {
	params := DefaultParams()
	params.EpochsPerPeriod = 0

	epochMintProvisions := CalculateEpochMintProvision(params, 1)
	require.Equal(t, epochMintProvisions, sdkmath.LegacyZeroDec())
}

// withinRange returns an error if the actual value is not within the expected value +/- tolerance
// tolerance is a percentage set to 0.01% by default
func withinRange(expected, actual sdkmath.LegacyDec) error {
	tolerance := sdkmath.LegacyNewDecWithPrec(1, 4)
	if expected.Sub(actual).Abs().Quo(expected).GT(tolerance) {
		tolerancePercent := tolerance.Mul(sdkmath.LegacyNewDec(100))
		return fmt.Errorf("expected %s to be within %s%% of %s", actual.String(), tolerancePercent.String(), expected.String())
	}
	return nil
}
