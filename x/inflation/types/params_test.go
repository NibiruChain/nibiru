package types

import (
	"testing"

	"cosmossdk.io/depinject"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/store"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	dbm "github.com/cometbft/cometbft-db"
	paramstestutil "github.com/cosmos/cosmos-sdk/x/params/testutil"
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

	cdc   codec.Codec
	amino *codec.LegacyAmino
	db    *dbm.MemDB
	ms    storetypes.CommitMultiStore
	key   *storetypes.KVStoreKey
	tkey  *storetypes.TransientStoreKey
}

func (s *ParamKeyTableTestSuite) SetupTest() {
	s.key = sdk.NewKVStoreKey("storekey")
	s.tkey = sdk.NewTransientStoreKey("transientstorekey")

	db := dbm.NewMemDB()
	ms := store.NewCommitMultiStore(db)
	ms.MountStoreWithDB(s.key, storetypes.StoreTypeIAVL, db)
	ms.MountStoreWithDB(s.tkey, storetypes.StoreTypeTransient, db)
	s.NoError(ms.LoadLatestVersion())

	cdc := new(codec.Codec)
	legacyAmino := new(codec.LegacyAmino)
	err := depinject.Inject(paramstestutil.AppConfig,
		cdc,
		legacyAmino,
	)
	s.NoError(err)
}

func (s *ParamKeyTableTestSuite) TestParamKeyTable() {
	var keyTable paramstypes.KeyTable
	s.Require().NotPanics(func() {
		keyTable = ParamKeyTable()
	})
	s.Require().NotPanics(func() {
		subspace := paramstypes.NewSubspace(
			s.cdc, s.amino, s.key, s.tkey, "inflationsubspace",
		)
		subspace.WithKeyTable(keyTable)
	})
}
