package types

import (
	fmt "fmt"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

// These numbers are for year n month 1
var ExpectedYearlyInflation = []sdk.Dec{
	sdk.NewDec(195_895_391_000_000),
	sdk.NewDec(156_348_637_000_000),
	sdk.NewDec(124_785_459_000_000),
	sdk.NewDec(99_594_157_000_000),
	sdk.NewDec(79_488_398_000_000),
	sdk.NewDec(63_441_527_000_000),
	sdk.NewDec(50_634_148_000_000),
	sdk.NewDec(40_412_283_000_000),
}

// ExpectedTotalInflation is the total amount of NIBI tokens (in unibi) that
// should be minted via inflation for the network to reach its target supply.
// The value 810.6 million is equivalent to:
// = (Community allocation of total supply) - (Community supply at start)
// = (60% of the total supply) - (Community supply at start)
// = (60% of 1.5 billion) - (Community supply at start)
// = 810.6 million NIBI
var ExpectedTotalInflation = sdk.NewDec(810_600_000_000_000)

func TestCalculateEpochMintProvision(t *testing.T) {
	params := DefaultParams()
	params.InflationEnabled = true

	epochId := uint64(0)
	totalInflation := sdk.ZeroDec()

	// Only the first 8 years have inflation with default params but we run
	// for 10 years expecting 0 inflation in the last 2 years.
	for year := uint64(0); year < 10; year++ {
		yearlyInflation := sdk.ZeroDec()
		for month := uint64(0); month < 12; month++ {
			for day := uint64(0); day < 30; day++ {
				epochMintProvisions := CalculateEpochMintProvision(params, epochId)
				yearlyInflation = yearlyInflation.Add(epochMintProvisions)
			}
			epochId++
		}
		// Should be within 0.0098%
		if year < uint64(len(ExpectedYearlyInflation)) {
			require.NoError(t, withingRange(yearlyInflation, ExpectedYearlyInflation[year]))
		} else {
			require.Equal(t, yearlyInflation, sdk.ZeroDec())
		}
		totalInflation = totalInflation.Add(yearlyInflation)
	}
	require.NoError(t, withingRange(totalInflation, ExpectedTotalInflation))
}

func TestCalculateEpochMintProvision_ZeroEpochs(t *testing.T) {
	params := DefaultParams()
	params.EpochsPerPeriod = 0

	epochMintProvisions := CalculateEpochMintProvision(params, 1)
	require.Equal(t, epochMintProvisions, sdk.ZeroDec())
}

// withingRange returns an error if the actual value is not within the expected value +/- tolerance
// tolerance is a percentage set to 0.01% by default
func withingRange(expected, actual sdk.Dec) error {
	tolerance := sdk.NewDecWithPrec(1, 4)
	is_within := expected.Sub(actual).Abs().Quo(expected).LTE(tolerance)
	if !is_within {
		tolerancePercent := tolerance.Mul(sdk.NewDec(100))
		return fmt.Errorf("expected %s to be within %s%% of %s", actual.String(), tolerancePercent.String(), expected.String())
	}
	return nil
}
