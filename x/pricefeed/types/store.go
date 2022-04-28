package types

import (
	"fmt"

	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
)

// Parameter keys
var (
	KeyPairs     = []byte("Pairs")
	DefaultPairs = []Pair{}
)

// NewParams creates a new AssetParams object
func NewParams(markets []Pair) Params {
	return Params{
		Pairs: markets,
	}
}

// DefaultParams default params for pricefeed
func DefaultParams() Params {
	return NewParams(DefaultPairs)
}

// ParamKeyTable Key declaration for parameters
func ParamKeyTable() paramtypes.KeyTable {
	return paramtypes.NewKeyTable().RegisterParamSet(&Params{})
}

// ParamSetPairs implements the ParamSet interface and returns all the key/value pairs
// pairs of pricefeed module's parameters.
func (p *Params) ParamSetPairs() paramtypes.ParamSetPairs {
	return paramtypes.ParamSetPairs{
		paramtypes.NewParamSetPair(KeyPairs, &p.Pairs, validatePairParams),
	}
}

// Validate ensure that params have valid values
func (p Params) Validate() error {
	return validatePairParams(p.Pairs)
}

func validatePairParams(i interface{}) error {
	markets, ok := i.(Pairs)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	return markets.Validate()
}
