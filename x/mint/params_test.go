package mint_test

import (
	"testing"

	sdkmath "cosmossdk.io/math"

	"github.com/NibiruChain/nibiru/v2/x/mint"

	"github.com/stretchr/testify/require"
)

func TestParamsValidate(t *testing.T) {
	testCases := []struct {
		name     string
		params   mint.Params
		expError bool
	}{
		{
			"default",
			mint.DefaultParams(),
			false,
		},
		{
			"valid",
			mint.NewParams(
				mint.DefaultPolynomialFactors,
				mint.DefaultInflationDistribution,
				true,
				true,
				mint.DefaultEpochsPerPeriod,
				mint.DefaultPeriodsPerYear,
				mint.DefaultMaxPeriod,
			),
			false,
		},
		{
			"valid param literal",
			mint.Params{
				PolynomialFactors:     mint.DefaultPolynomialFactors,
				InflationDistribution: mint.DefaultInflationDistribution,
				InflationEnabled:      true,
				HasInflationStarted:   true,
				EpochsPerPeriod:       mint.DefaultEpochsPerPeriod,
				PeriodsPerYear:        mint.DefaultPeriodsPerYear,
			},
			false,
		},
		{
			"invalid - polynomial calculation - no coefficient",
			mint.Params{
				PolynomialFactors:     []sdkmath.LegacyDec{},
				InflationDistribution: mint.DefaultInflationDistribution,
				InflationEnabled:      true,
				HasInflationStarted:   true,
				EpochsPerPeriod:       mint.DefaultEpochsPerPeriod,
				PeriodsPerYear:        mint.DefaultPeriodsPerYear,
			},
			true,
		},
		{
			"invalid - inflation distribution - negative staking rewards",
			mint.Params{
				PolynomialFactors: mint.DefaultPolynomialFactors,
				InflationDistribution: mint.InflationDistribution{
					StakingRewards:    sdkmath.LegacyOneDec().Neg(),
					CommunityPool:     sdkmath.LegacyNewDecWithPrec(133333, 6),
					StrategicReserves: sdkmath.LegacyNewDecWithPrec(333333, 6),
				},
				InflationEnabled:    true,
				HasInflationStarted: true,
				EpochsPerPeriod:     mint.DefaultEpochsPerPeriod,
				PeriodsPerYear:      mint.DefaultPeriodsPerYear,
			},
			true,
		},
		{
			"invalid - inflation distribution - negative usage incentives",
			mint.Params{
				PolynomialFactors: mint.DefaultPolynomialFactors,
				InflationDistribution: mint.InflationDistribution{
					StakingRewards:    sdkmath.LegacyNewDecWithPrec(533334, 6),
					CommunityPool:     sdkmath.LegacyNewDecWithPrec(133333, 6),
					StrategicReserves: sdkmath.LegacyOneDec().Neg(),
				},
				InflationEnabled:    true,
				HasInflationStarted: true,
				EpochsPerPeriod:     mint.DefaultEpochsPerPeriod,
				PeriodsPerYear:      mint.DefaultPeriodsPerYear,
			},
			true,
		},
		{
			"invalid - inflation distribution - negative community pool rewards",
			mint.Params{
				PolynomialFactors: mint.DefaultPolynomialFactors,
				InflationDistribution: mint.InflationDistribution{
					StakingRewards:    sdkmath.LegacyNewDecWithPrec(533334, 6),
					CommunityPool:     sdkmath.LegacyOneDec().Neg(),
					StrategicReserves: sdkmath.LegacyNewDecWithPrec(333333, 6),
				},
				InflationEnabled:    true,
				HasInflationStarted: true,
				EpochsPerPeriod:     mint.DefaultEpochsPerPeriod,
				PeriodsPerYear:      mint.DefaultPeriodsPerYear,
			},
			true,
		},
		{
			"invalid - inflation distribution - total distribution ratio unequal 1",
			mint.Params{
				PolynomialFactors: mint.DefaultPolynomialFactors,
				InflationDistribution: mint.InflationDistribution{
					StakingRewards:    sdkmath.LegacyNewDecWithPrec(533333, 6),
					CommunityPool:     sdkmath.LegacyNewDecWithPrec(133333, 6),
					StrategicReserves: sdkmath.LegacyNewDecWithPrec(333333, 6),
				},
				InflationEnabled:    true,
				HasInflationStarted: true,
				EpochsPerPeriod:     mint.DefaultEpochsPerPeriod,
				PeriodsPerYear:      mint.DefaultPeriodsPerYear,
			},
			true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.params.Validate()

			if tc.expError {
				require.Error(t, err, tc.name)
			} else {
				require.NoError(t, err, tc.name)
			}
		})
	}
}
