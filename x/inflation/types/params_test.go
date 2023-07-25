package types

import (
	"testing"

	storetypes "github.com/cosmos/cosmos-sdk/store/types"

	"github.com/NibiruChain/nibiru/app/codec"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	paramstypes "github.com/cosmos/cosmos-sdk/x/params/types"
)

func TestParamsValidate(t *testing.T) {
	testCases := []struct {
		name     string
		params   Params
		expError bool
	}{
		{
			"default",
			DefaultParams(),
			false,
		},
		{
			"valid",
			NewParams(
				DefaultExponentialCalculation,
				DefaultInflationDistribution,
				true,
				DefaultEpochsPerPeriod,
			),
			false,
		},
		{
			"valid param literal",
			Params{
				ExponentialCalculation: DefaultExponentialCalculation,
				InflationDistribution:  DefaultInflationDistribution,
				InflationEnabled:       true,
				EpochsPerPeriod:        DefaultEpochsPerPeriod,
			},
			false,
		},
		{
			"invalid - exponential calculation - negative A",
			Params{
				ExponentialCalculation: ExponentialCalculation{
					A: sdk.NewDec(int64(-1)),
					R: sdk.NewDecWithPrec(5, 1),
					C: sdk.NewDec(int64(9_375_000)),
				},
				InflationDistribution: DefaultInflationDistribution,
				InflationEnabled:      true,
				EpochsPerPeriod:       DefaultEpochsPerPeriod,
			},
			true,
		},
		{
			"invalid - exponential calculation - R greater than 1",
			Params{
				ExponentialCalculation: ExponentialCalculation{
					A: sdk.NewDec(int64(300_000_000)),
					R: sdk.NewDecWithPrec(5, 0),
					C: sdk.NewDec(int64(9_375_000)),
				},
				InflationDistribution: DefaultInflationDistribution,
				InflationEnabled:      true,
				EpochsPerPeriod:       DefaultEpochsPerPeriod,
			},
			true,
		},
		{
			"invalid - exponential calculation - negative R",
			Params{
				ExponentialCalculation: ExponentialCalculation{
					A: sdk.NewDec(int64(300_000_000)),
					R: sdk.NewDecWithPrec(-5, 1),
					C: sdk.NewDec(int64(9_375_000)),
				},
				InflationDistribution: DefaultInflationDistribution,
				InflationEnabled:      true,
				EpochsPerPeriod:       DefaultEpochsPerPeriod,
			},
			true,
		},
		{
			"invalid - exponential calculation - negative C",
			Params{
				ExponentialCalculation: ExponentialCalculation{
					A: sdk.NewDec(int64(300_000_000)),
					R: sdk.NewDecWithPrec(5, 1),
					C: sdk.NewDec(int64(-9_375_000)),
				},
				InflationDistribution: DefaultInflationDistribution,
				InflationEnabled:      true,
				EpochsPerPeriod:       DefaultEpochsPerPeriod,
			},
			true,
		},
		{
			"invalid - inflation distribution - negative staking rewards",
			Params{
				ExponentialCalculation: DefaultExponentialCalculation,
				InflationDistribution: InflationDistribution{
					StakingRewards:    sdk.OneDec().Neg(),
					CommunityPool:     sdk.NewDecWithPrec(133333, 6),
					StrategicReserves: sdk.NewDecWithPrec(333333, 6),
				},
				InflationEnabled: true,
				EpochsPerPeriod:  DefaultEpochsPerPeriod,
			},
			true,
		},
		{
			"invalid - inflation distribution - negative usage incentives",
			Params{
				ExponentialCalculation: DefaultExponentialCalculation,
				InflationDistribution: InflationDistribution{
					StakingRewards:    sdk.NewDecWithPrec(533334, 6),
					CommunityPool:     sdk.NewDecWithPrec(133333, 6),
					StrategicReserves: sdk.OneDec().Neg(),
				},
				InflationEnabled: true,
				EpochsPerPeriod:  DefaultEpochsPerPeriod,
			},
			true,
		},
		{
			"invalid - inflation distribution - negative community pool rewards",
			Params{
				ExponentialCalculation: DefaultExponentialCalculation,
				InflationDistribution: InflationDistribution{
					StakingRewards:    sdk.NewDecWithPrec(533334, 6),
					CommunityPool:     sdk.OneDec().Neg(),
					StrategicReserves: sdk.NewDecWithPrec(333333, 6),
				},
				InflationEnabled: true,
				EpochsPerPeriod:  DefaultEpochsPerPeriod,
			},
			true,
		},
		{
			"invalid - inflation distribution - total distribution ratio unequal 1",
			Params{
				ExponentialCalculation: DefaultExponentialCalculation,
				InflationDistribution: InflationDistribution{
					StakingRewards:    sdk.NewDecWithPrec(533333, 6),
					CommunityPool:     sdk.NewDecWithPrec(133333, 6),
					StrategicReserves: sdk.NewDecWithPrec(333333, 6),
				},
				InflationEnabled: true,
				EpochsPerPeriod:  DefaultEpochsPerPeriod,
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

type ParamKeyTableTestSuite struct {
	suite.Suite
}

func (s *ParamKeyTableTestSuite) TestParamKeyTable() {
	encCfg := codec.MakeEncodingConfig()
	cdc := encCfg.Marshaler
	amino := encCfg.Amino

	storeKey := storetypes.NewKVStoreKey("mockStoreKey")
	transientStoreKey := storetypes.NewTransientStoreKey("mockTransientKey")

	var keyTable paramstypes.KeyTable
	s.Require().NotPanics(func() {
		keyTable = ParamKeyTable()
	})
	s.Require().NotPanics(func() {
		subspace := paramstypes.NewSubspace(
			cdc,
			amino,
			storeKey, transientStoreKey, "inflationsubspace",
		)
		subspace.WithKeyTable(keyTable)
	})
}
