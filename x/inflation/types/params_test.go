package types_test

import (
	"testing"

	inflationtypes "github.com/NibiruChain/nibiru/x/inflation/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
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
				PolynomialFactors:     []sdk.Dec{},
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
					StakingRewards:    sdk.OneDec().Neg(),
					CommunityPool:     sdk.NewDecWithPrec(133333, 6),
					StrategicReserves: sdk.NewDecWithPrec(333333, 6),
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
					StakingRewards:    sdk.NewDecWithPrec(533334, 6),
					CommunityPool:     sdk.NewDecWithPrec(133333, 6),
					StrategicReserves: sdk.OneDec().Neg(),
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
					StakingRewards:    sdk.NewDecWithPrec(533334, 6),
					CommunityPool:     sdk.OneDec().Neg(),
					StrategicReserves: sdk.NewDecWithPrec(333333, 6),
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
					StakingRewards:    sdk.NewDecWithPrec(533333, 6),
					CommunityPool:     sdk.NewDecWithPrec(133333, 6),
					StrategicReserves: sdk.NewDecWithPrec(333333, 6),
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
