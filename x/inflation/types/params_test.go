package types_test

import (
	"testing"

	sdkmath "cosmossdk.io/math"

	inflationtypes "github.com/NibiruChain/nibiru/v2/x/inflation/types"

	"github.com/stretchr/testify/require"
)

func TestParamsValidate(t *testing.T) {
	testCases := []struct {
		name     string
		params   inflationtypes.Params
		expError bool
	}{
		{
			"default",
			inflationtypes.DefaultParams(),
			false,
		},
		{
			"valid",
			inflationtypes.NewParams(
				inflationtypes.DefaultPolynomialFactors,
				inflationtypes.DefaultInflationDistribution,
				true,
				true,
				inflationtypes.DefaultEpochsPerPeriod,
				inflationtypes.DefaultPeriodsPerYear,
				inflationtypes.DefaultMaxPeriod,
			),
			false,
		},
		{
			"valid param literal",
			inflationtypes.Params{
				PolynomialFactors:     inflationtypes.DefaultPolynomialFactors,
				InflationDistribution: inflationtypes.DefaultInflationDistribution,
				InflationEnabled:      true,
				HasInflationStarted:   true,
				EpochsPerPeriod:       inflationtypes.DefaultEpochsPerPeriod,
				PeriodsPerYear:        inflationtypes.DefaultPeriodsPerYear,
			},
			false,
		},
		{
			"invalid - polynomial calculation - no coefficient",
			inflationtypes.Params{
				PolynomialFactors:     []sdkmath.LegacyDec{},
				InflationDistribution: inflationtypes.DefaultInflationDistribution,
				InflationEnabled:      true,
				HasInflationStarted:   true,
				EpochsPerPeriod:       inflationtypes.DefaultEpochsPerPeriod,
				PeriodsPerYear:        inflationtypes.DefaultPeriodsPerYear,
			},
			true,
		},
		{
			"invalid - inflation distribution - negative staking rewards",
			inflationtypes.Params{
				PolynomialFactors: inflationtypes.DefaultPolynomialFactors,
				InflationDistribution: inflationtypes.InflationDistribution{
					StakingRewards:    sdkmath.LegacyOneDec().Neg(),
					CommunityPool:     sdkmath.LegacyNewDecWithPrec(133333, 6),
					StrategicReserves: sdkmath.LegacyNewDecWithPrec(333333, 6),
				},
				InflationEnabled:    true,
				HasInflationStarted: true,
				EpochsPerPeriod:     inflationtypes.DefaultEpochsPerPeriod,
				PeriodsPerYear:      inflationtypes.DefaultPeriodsPerYear,
			},
			true,
		},
		{
			"invalid - inflation distribution - negative usage incentives",
			inflationtypes.Params{
				PolynomialFactors: inflationtypes.DefaultPolynomialFactors,
				InflationDistribution: inflationtypes.InflationDistribution{
					StakingRewards:    sdkmath.LegacyNewDecWithPrec(533334, 6),
					CommunityPool:     sdkmath.LegacyNewDecWithPrec(133333, 6),
					StrategicReserves: sdkmath.LegacyOneDec().Neg(),
				},
				InflationEnabled:    true,
				HasInflationStarted: true,
				EpochsPerPeriod:     inflationtypes.DefaultEpochsPerPeriod,
				PeriodsPerYear:      inflationtypes.DefaultPeriodsPerYear,
			},
			true,
		},
		{
			"invalid - inflation distribution - negative community pool rewards",
			inflationtypes.Params{
				PolynomialFactors: inflationtypes.DefaultPolynomialFactors,
				InflationDistribution: inflationtypes.InflationDistribution{
					StakingRewards:    sdkmath.LegacyNewDecWithPrec(533334, 6),
					CommunityPool:     sdkmath.LegacyOneDec().Neg(),
					StrategicReserves: sdkmath.LegacyNewDecWithPrec(333333, 6),
				},
				InflationEnabled:    true,
				HasInflationStarted: true,
				EpochsPerPeriod:     inflationtypes.DefaultEpochsPerPeriod,
				PeriodsPerYear:      inflationtypes.DefaultPeriodsPerYear,
			},
			true,
		},
		{
			"invalid - inflation distribution - total distribution ratio unequal 1",
			inflationtypes.Params{
				PolynomialFactors: inflationtypes.DefaultPolynomialFactors,
				InflationDistribution: inflationtypes.InflationDistribution{
					StakingRewards:    sdkmath.LegacyNewDecWithPrec(533333, 6),
					CommunityPool:     sdkmath.LegacyNewDecWithPrec(133333, 6),
					StrategicReserves: sdkmath.LegacyNewDecWithPrec(333333, 6),
				},
				InflationEnabled:    true,
				HasInflationStarted: true,
				EpochsPerPeriod:     inflationtypes.DefaultEpochsPerPeriod,
				PeriodsPerYear:      inflationtypes.DefaultPeriodsPerYear,
			},
			true,
		},
	}

	for _, tc := range testCases {
		tc := tc
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
