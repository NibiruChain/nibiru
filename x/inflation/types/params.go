package types

import (
	"errors"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	paramstypes "github.com/cosmos/cosmos-sdk/x/params/types"
)

var (
	KeyInflationEnabled       = []byte("InflationEnabled")
	KeyExponentialCalculation = []byte("ExponentialCalculation")
	KeyInflationDistribution  = []byte("InflationDistribution")
	KeyEpochsPerPeriod        = []byte("EpochsPerPeriod")
)

var (
	DefaultInflation              = true
	DefaultExponentialCalculation = ExponentialCalculation{
		A: sdk.NewDec(int64(405_000_000)),
		R: sdk.NewDecWithPrec(50, 2), // 50%
		C: sdk.NewDecWithPrec(395_507_8125, 4),
	}
	DefaultInflationDistribution = InflationDistribution{
		StakingRewards:    sdk.NewDecWithPrec(27_8, 3),  // 27.8%
		CommunityPool:     sdk.NewDecWithPrec(62_20, 4), // 62.20%
		StrategicReserves: sdk.NewDecWithPrec(10, 2),    // 10%
	}
	DefaultEpochsPerPeriod = uint64(365)
)

func NewParams(
	exponentialCalculation ExponentialCalculation,
	inflationDistribution InflationDistribution,
	inflationEnabled bool,
	epochsPerPeriod uint64,
) Params {
	return Params{
		ExponentialCalculation: exponentialCalculation,
		InflationDistribution:  inflationDistribution,
		InflationEnabled:       inflationEnabled,
		EpochsPerPeriod:        epochsPerPeriod,
	}
}

// default minting module parameters
func DefaultParams() Params {
	return Params{
		ExponentialCalculation: DefaultExponentialCalculation,
		InflationDistribution:  DefaultInflationDistribution,
		InflationEnabled:       DefaultInflation,
		EpochsPerPeriod:        DefaultEpochsPerPeriod,
	}
}

// ParamKeyTable returns the parameter key table.
func ParamKeyTable() paramstypes.KeyTable {
	return paramstypes.NewKeyTable().RegisterParamSet(&Params{})
}

// ParamSetPairs implements the ParamSet interface and returns all the key/value pairs
// pairs of oracle module's parameters.
func (p *Params) ParamSetPairs() paramstypes.ParamSetPairs {
	return paramstypes.ParamSetPairs{
		paramstypes.NewParamSetPair(KeyInflationEnabled, &p.InflationEnabled, validateBool),
		paramstypes.NewParamSetPair(KeyExponentialCalculation, &p.ExponentialCalculation, validateExponentialCalculation),
		paramstypes.NewParamSetPair(KeyInflationDistribution, &p.InflationDistribution, validateInflationDistribution),
		paramstypes.NewParamSetPair(KeyEpochsPerPeriod, &p.EpochsPerPeriod, validateUint64),
	}
}

func validateExponentialCalculation(i interface{}) error {
	v, ok := i.(ExponentialCalculation)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	// validate initial value
	if v.A.IsNegative() {
		return fmt.Errorf("initial value cannot be negative")
	}

	// validate reduction factor
	if v.R.GT(sdk.NewDec(1)) {
		return fmt.Errorf("reduction factor cannot be greater than 1")
	}

	if v.R.IsNegative() {
		return fmt.Errorf("reduction factor cannot be negative")
	}

	// validate long term inflation
	if v.C.IsNegative() {
		return fmt.Errorf("long term inflation cannot be negative")
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

func validateInt64(i interface{}) error {
	_, ok := i.(int64)
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
	if err := validateExponentialCalculation(p.ExponentialCalculation); err != nil {
		return err
	}
	if err := validateInflationDistribution(p.InflationDistribution); err != nil {
		return err
	}

	return validateBool(p.InflationEnabled)
}
