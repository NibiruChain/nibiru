package v1

import (
	fmt "fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
)

var _ paramtypes.ParamSet = (*Params)(nil)

// ParamKeyTable the param key table for launch module
func ParamKeyTable() paramtypes.KeyTable {
	return paramtypes.NewKeyTable().RegisterParamSet(&Params{})
}

// ParamSetPairs get the params.ParamSet
func (p *Params) ParamSetPairs() paramtypes.ParamSetPairs {
	return paramtypes.ParamSetPairs{
		paramtypes.NewParamSetPair(
			[]byte("Stopped"),
			&p.Stopped,
			validateStopped,
		),
		paramtypes.NewParamSetPair(
			[]byte("MaintenanceMarginRatio"),
			&p.MaintenanceMarginRatio,
			validateMaintenanceMarginRatio,
		),
	}
}

// DefaultParams returns the default parameters for the x/perp module.
func DefaultParams() Params {
	return Params{
		Stopped:                true,
		MaintenanceMarginRatio: sdk.OneInt(),
	}
}

// Validate validates the set of params
func (p *Params) Validate() error {
	err := validateStopped(p.Stopped)
	if err != nil {
		return err
	}

	err = validateMaintenanceMarginRatio(p.MaintenanceMarginRatio)
	if err != nil {
		return err
	}

	return err
}

func validateStopped(i interface{}) error {
	_, ok := i.(bool)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}
	return nil
}

func validateMaintenanceMarginRatio(i interface{}) error {
	_, ok := i.(sdk.Int)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}
	return nil
}
