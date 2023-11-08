package types

import (
	"errors"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	paramstypes "github.com/cosmos/cosmos-sdk/x/params/types"
)

var (
	KeyInflationEnabled      = []byte("InflationEnabled")
	KeyPolynomialFactors     = []byte("PolynomialFactors")
	KeyInflationDistribution = []byte("InflationDistribution")
	KeyEpochsPerPeriod       = []byte("EpochsPerPeriod")
	KeyMaxPeriod             = []byte("MaxPeriod")
)

var (
	DefaultInflation         = true
	DefaultPolynomialFactors = []sdk.Dec{
		sdk.MustNewDecFromStr("-0.00014903"),
		sdk.MustNewDecFromStr("0.07527647"),
		sdk.MustNewDecFromStr("-19.11742154"),
		sdk.MustNewDecFromStr("3170.0969905"),
		sdk.MustNewDecFromStr("-339271.31060432"),
		sdk.MustNewDecFromStr("18063678.8582418"),
	}
	DefaultInflationDistribution = InflationDistribution{
		StakingRewards:    sdk.NewDecWithPrec(27_8, 3),  // 27.8%
		CommunityPool:     sdk.NewDecWithPrec(62_20, 4), // 62.20%
		StrategicReserves: sdk.NewDecWithPrec(10, 2),    // 10%
	}
	DefaultEpochsPerPeriod = uint64(30)
	DefaultMaxPeriod       = uint64(8 * 12 * 30) // 8 years with 360 days per year
)

func NewParams(
	polynomialCalculation []sdk.Dec,
	inflationDistribution InflationDistribution,
	inflationEnabled bool,
	epochsPerPeriod,
	maxPeriod uint64,
) Params {
	return Params{
		PolynomialFactors:     polynomialCalculation,
		InflationDistribution: inflationDistribution,
		InflationEnabled:      inflationEnabled,
		EpochsPerPeriod:       epochsPerPeriod,
		MaxPeriod:             maxPeriod,
	}
}

// default minting module parameters
func DefaultParams() Params {
	return Params{
		PolynomialFactors:     DefaultPolynomialFactors,
		InflationDistribution: DefaultInflationDistribution,
		InflationEnabled:      DefaultInflation,
		EpochsPerPeriod:       DefaultEpochsPerPeriod,
		MaxPeriod:             DefaultMaxPeriod,
	}
}

// ParamKeyTable returns the parameter key table.
func ParamKeyTable() paramstypes.KeyTable {
	return paramstypes.NewKeyTable().RegisterParamSet(&Params{})
}

var _ paramstypes.ParamSet = (*Params)(nil)

// ParamSetPairs returns all the of key, value type, and validation function
// for each module parameter. ParamSetPairs implements the ParamSet interface.
func (p *Params) ParamSetPairs() paramstypes.ParamSetPairs {
	return paramstypes.ParamSetPairs{
		paramstypes.NewParamSetPair(KeyInflationEnabled, &p.InflationEnabled, validateBool),
		paramstypes.NewParamSetPair(KeyPolynomialFactors, &p.PolynomialFactors, validatePolynomialFactors),
		paramstypes.NewParamSetPair(KeyInflationDistribution, &p.InflationDistribution, validateInflationDistribution),
		paramstypes.NewParamSetPair(KeyEpochsPerPeriod, &p.EpochsPerPeriod, validateUint64),
		paramstypes.NewParamSetPair(KeyMaxPeriod, &p.MaxPeriod, validateUint64),
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

func (p Params) Validate() error {
	if err := validateEpochsPerPeriod(p.EpochsPerPeriod); err != nil {
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

	return validateBool(p.InflationEnabled)
}
