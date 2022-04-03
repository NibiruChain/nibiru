package types

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

// NewParams creates a new Params instance
func NewParams(collRatio sdk.Dec) Params {
	collRatioInt := collRatio.Mul(sdk.MustNewDecFromStr("1000000")).RoundInt()

	// TODO: Verify collRatio is an integer in a test.
	return Params{CollRatio: collRatioInt.Int64()}
}

// DefaultParams returns a default set of parameters
func DefaultParams() Params {
	genesisCollRatio := sdk.MustNewDecFromStr("1")
	return NewParams(genesisCollRatio)
}

// ParamSetPairs get the params.ParamSet
func (p *Params) ParamSetPairs() paramtypes.ParamSetPairs {
	return paramtypes.ParamSetPairs{
		paramtypes.NewParamSetPair(
			[]byte("CollRatio"),
			&p.CollRatio,
			validateCollRatio,
		),
	}
}

// Validate validates the set of params
func (p Params) Validate() error {
	return validateCollRatio(p.CollRatio)
}

func validateCollRatio(i interface{}) error {
	collRatio, ok := i.(int64)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	if collRatio > 1_000_000 {
		return fmt.Errorf("collateral Ratio is above max value(1e6): %o", collRatio)
	} else {
		return nil
	}
}
