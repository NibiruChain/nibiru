package types

import (
	"errors"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

var (
	KeyInflationEnabled      = []byte("InflationEnabled")
	KeyHasInflationStarted   = []byte("HasInflationStarted")
	KeyPolynomialFactors     = []byte("PolynomialFactors")
	KeyInflationDistribution = []byte("InflationDistribution")
	KeyEpochsPerPeriod       = []byte("EpochsPerPeriod")
	KeyPeriodsPerYear        = []byte("PeriodsPerYear")
	KeyMaxPeriod             = []byte("MaxPeriod")
)

var (
	DefaultInflation         = false
	DefaultPolynomialFactors = []sdk.Dec{
		sdk.MustNewDecFromStr("-0.00014851"),
		sdk.MustNewDecFromStr("0.07501001"),
		sdk.MustNewDecFromStr("-19.04980404"),
		sdk.MustNewDecFromStr("3158.89014745"),
		sdk.MustNewDecFromStr("-338072.13773281"),
		sdk.MustNewDecFromStr("17999834.05992003"),
	}
	DefaultInflationDistribution = InflationDistribution{
		CommunityPool:     sdk.NewDecWithPrec(35_142714, 8), // 35.142714%
		StakingRewards:    sdk.NewDecWithPrec(27_855672, 8), // 27.855672%
		StrategicReserves: sdk.NewDecWithPrec(37_001614, 8), // 37.001614%
	}
	DefaultEpochsPerPeriod = uint64(30)
	DefaultPeriodsPerYear  = uint64(12)
	DefaultMaxPeriod       = uint64(8 * 12) // 8 years with 360 days per year
)

func NewParams(
	polynomialCalculation []sdk.Dec,
	inflationDistribution InflationDistribution,
	inflationEnabled bool,
	hasInflationStarted bool,
	epochsPerPeriod,
	periodsPerYear,
	maxPeriod uint64,
) Params {
	return Params{
		PolynomialFactors:     polynomialCalculation,
		InflationDistribution: inflationDistribution,
		InflationEnabled:      inflationEnabled,
		HasInflationStarted:   hasInflationStarted,
		EpochsPerPeriod:       epochsPerPeriod,
		PeriodsPerYear:        periodsPerYear,
		MaxPeriod:             maxPeriod,
	}
}

// default inflation module parameters
func DefaultParams() Params {
	return Params{
		PolynomialFactors:     DefaultPolynomialFactors,
		InflationDistribution: DefaultInflationDistribution,
		InflationEnabled:      DefaultInflation,
		HasInflationStarted:   DefaultInflation,
		EpochsPerPeriod:       DefaultEpochsPerPeriod,
		PeriodsPerYear:        DefaultPeriodsPerYear,
		MaxPeriod:             DefaultMaxPeriod,
	}
}

func validatePolynomialFactors(i interface{}) error {
	v, ok := i.([]sdk.Dec)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	if len(v) == 0 {
		return errors.New("polynomial factors cannot be empty")
	}
	return nil
}

func validateInflationDistribution(i interface{}) error {
	v, ok := i.(InflationDistribution)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	if v.StakingRewards.IsNegative() {
		return errors.New("staking distribution ratio must not be negative")
	}

	if v.CommunityPool.IsNegative() {
		return errors.New("community pool distribution ratio must not be negative")
	}

	if v.StrategicReserves.IsNegative() {
		return errors.New("pool incentives distribution ratio must not be negative")
	}

	totalProportions := v.StakingRewards.Add(v.StrategicReserves).Add(v.CommunityPool)
	if !totalProportions.Equal(sdk.OneDec()) {
		return errors.New("total distributions ratio should be 1")
	}

	return nil
}

func validateBool(i interface{}) error {
	_, ok := i.(bool)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	return nil
}

func validateUint64(i interface{}) error {
	_, ok := i.(uint64)
	if !ok {
		return fmt.Errorf("invalid genesis state type: %T", i)
	}
	return nil
}

func validateEpochsPerPeriod(i interface{}) error {
	val, ok := i.(uint64)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	if val <= 0 {
		return fmt.Errorf("epochs per period must be positive: %d", val)
	}

	return nil
}

func validatePeriodsPerYear(i interface{}) error {
	val, ok := i.(uint64)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	if val <= 0 {
		return fmt.Errorf("periods per year must be positive: %d", val)
	}

	return nil
}

func (p Params) Validate() error {
	if err := validateEpochsPerPeriod(p.EpochsPerPeriod); err != nil {
		return err
	}
	if err := validatePeriodsPerYear(p.PeriodsPerYear); err != nil {
		return err
	}
	if err := validatePolynomialFactors(p.PolynomialFactors); err != nil {
		return err
	}
	if err := validateInflationDistribution(p.InflationDistribution); err != nil {
		return err
	}
	if err := validateUint64(p.MaxPeriod); err != nil {
		return err
	}
	if err := validateBool(p.HasInflationStarted); err != nil {
		return err
	}

	return validateBool(p.InflationEnabled)
}
