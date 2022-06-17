package types

import (
	"fmt"

	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"

	"github.com/NibiruChain/nibiru/x/common"
)

// Parameter keys
var (
	DefaultPairs = common.AssetPairs{
		{Token0: common.CollDenom, Token1: common.StableDenom},
		{Token0: common.GovDenom, Token1: common.StableDenom},
	}
)

// NewParams creates a new AssetParams object
func NewParams(
	pairs []string,
) Params {
	return Params{
		Pairs: pairs,
	}
}

// DefaultParams default params for pricefeed
func DefaultParams() Params {
	return NewParams(DefaultPairs.Strings())
}

// ParamKeyTable Key declaration for parameters
func ParamKeyTable() paramtypes.KeyTable {
	return paramtypes.NewKeyTable().RegisterParamSet(&Params{})
}

// ParamSetPairs implements the ParamSet interface and returns all the key/value
// pairs of pricefeed module's parameters.
func (p *Params) ParamSetPairs() paramtypes.ParamSetPairs {
	return paramtypes.ParamSetPairs{
		paramtypes.NewParamSetPair([]byte("Pairs"), &p.Pairs, validateParamPairs),
	}
}

// Validate ensure that params have valid values
func (p Params) Validate() error {
	err := validateParamPairs(p.Pairs)
	if err != nil {
		return err
	}
	return nil
}

func validateParamPairs(i interface{}) error {
	pairs, ok := i.([]string)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}
	for _, pairStr := range pairs {
		_, err := common.NewAssetPairFromStr(pairStr)
		if err != nil {
			return err
		}
	}

	return nil
}
