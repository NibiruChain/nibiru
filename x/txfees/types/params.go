package types

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
)

// Parameter store keys.
var (
	KeyWhitelistedFeeTokenSetters = []byte("WhitelistedFeeTokenSetters")
)

// ParamTable for txfees module.
func ParamKeyTable() paramtypes.KeyTable {
	return paramtypes.NewKeyTable().RegisterParamSet(&Params{})
}

// DefaultParams are the default txfees module parameters.
func DefaultParams() Params {
	return Params{
		WhitelistedFeeTokenSetters: []string{},
	}
}

// validate params.
func (p Params) Validate() error {

	for _, a := range p.WhitelistedFeeTokenSetters {
		if _, err := sdk.AccAddressFromBech32(a); err != nil {
			return fmt.Errorf("invalid address")
		}
	}

	return nil
}

// Implements params.ParamSet.
func (p *Params) ParamSetPairs() paramtypes.ParamSetPairs {
	return paramtypes.ParamSetPairs{
		paramtypes.NewParamSetPair(KeyWhitelistedFeeTokenSetters, &p.WhitelistedFeeTokenSetters, validateAddressList),
	}
}

func validateAddressList(i interface{}) error {
	whitelist, ok := i.([]string)

	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	for _, a := range whitelist {
		if _, err := sdk.AccAddressFromBech32(a); err != nil {
			return fmt.Errorf("invalid address")
		}
	}

	return nil
}
