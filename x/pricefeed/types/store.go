package types

import (
	"fmt"
	"time"

	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"

	"github.com/NibiruChain/nibiru/x/common"
)

// Parameter keys
var (
	DefaultPairs = common.AssetPairs{
		common.PairGovStable,
		common.PairCollStable,
		common.PairBTCStable,
		common.PairETHStable,
	}
	DefaultLookbackWindow = 15 * time.Minute
)

// NewParams creates a new AssetParams object
func NewParams(
	pairs common.AssetPairs,
	twapLookbackWindow time.Duration,
) Params {
	return Params{
		Pairs:              pairs,
		TwapLookbackWindow: twapLookbackWindow,
	}
}

// DefaultParams default params for pricefeed
func DefaultParams() Params {
	return NewParams(DefaultPairs, DefaultLookbackWindow)
}

// ParamKeyTable Key declaration for parameters
func ParamKeyTable() paramtypes.KeyTable {
	return paramtypes.NewKeyTable().RegisterParamSet(&Params{})
}

// ParamSetPairs implements the ParamSet interface and returns all the key/value
// pairs of pricefeed module's parameters.
func (p *Params) ParamSetPairs() paramtypes.ParamSetPairs {
	return paramtypes.ParamSetPairs{
		paramtypes.NewParamSetPair(
			[]byte("Pairs"), &p.Pairs, validateParamPairs,
		),
		paramtypes.NewParamSetPair(
			[]byte("TwapLookbackWindow"), &p.TwapLookbackWindow, validateTwapLookbackWindow,
		),
	}
}

// Validate ensure that params have valid values
func (p Params) Validate() error {
	err := validateParamPairs(p.Pairs)
	if err != nil {
		return err
	}
	err = validateTwapLookbackWindow(p.TwapLookbackWindow)
	if err != nil {
		return err
	}
	return nil
}

func validateParamPairs(i interface{}) error {
	pairs, ok := i.([]common.AssetPair)
	if !ok {
		return fmt.Errorf("invalid parameter type for pairs: %T", i)
	}
	for _, pair := range pairs {
		if err := pair.Validate(); err != nil {
			return err
		}
	}

	return nil
}

func validateTwapLookbackWindow(i interface{}) error {
	d, ok := i.(time.Duration)
	if !ok {
		return fmt.Errorf("invalid parameter type for twap lookback window: %T", i)
	}
	if d < 0 {
		return fmt.Errorf("invalid twapLookbackWindow, negative value is not allowed: %s", d)
	}
	return nil
}
