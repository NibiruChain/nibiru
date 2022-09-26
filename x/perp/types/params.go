package types

import (
	fmt "fmt"
	"time"

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
			[]byte("FeePoolFeeRatio"),
			&p.FeePoolFeeRatio,
			validatePercentageRatio,
		),
		paramtypes.NewParamSetPair(
			[]byte("EcosystemFundFeeRatio"),
			&p.EcosystemFundFeeRatio,
			validatePercentageRatio,
		),
		paramtypes.NewParamSetPair(
			[]byte("LiquidationFeeRatio"),
			&p.LiquidationFeeRatio,
			validatePercentageRatio,
		),
		paramtypes.NewParamSetPair(
			[]byte("PartialLiquidationRatio"),
			&p.PartialLiquidationRatio,
			validatePercentageRatio,
		),
		paramtypes.NewParamSetPair(
			[]byte("FundingRateInterval"),
			&p.FundingRateInterval,
			validateFundingRateInterval,
		),
		paramtypes.NewParamSetPair(
			[]byte("TwapLookbackWindow"),
			&p.TwapLookbackWindow,
			validateTwapLookbackWindow,
		),
		paramtypes.NewParamSetPair(
			[]byte("WhitelistedLiquidators"),
			&p.WhitelistedLiquidators,
			validateAddress,
		),
	}
}

// NewParams creates a new Params instance
func NewParams(
	stopped bool,
	feePoolFeeRatio sdk.Dec,
	ecosystemFundFeeRatio sdk.Dec,
	liquidationFeeRatio sdk.Dec,
	partialLiquidationRatio sdk.Dec,
	fundingRateInterval string,
	twapLookbackWindow time.Duration,
) Params {
	return Params{
		Stopped:                 stopped,
		FeePoolFeeRatio:         feePoolFeeRatio,
		EcosystemFundFeeRatio:   ecosystemFundFeeRatio,
		LiquidationFeeRatio:     liquidationFeeRatio,
		PartialLiquidationRatio: partialLiquidationRatio,
		FundingRateInterval:     fundingRateInterval,
		TwapLookbackWindow:      twapLookbackWindow,
	}
}

// DefaultParams returns the default parameters for the x/perp module.
func DefaultParams() Params {
	return NewParams(
		/* stopped */ false,
		/* feePoolFeeRatio */ sdk.MustNewDecFromStr("0.001"), // 10 bps
		/* ecosystemFundFeeRatio */ sdk.MustNewDecFromStr("0.001"), // 10 bps
		/* liquidationFee */ sdk.MustNewDecFromStr("0.025"), // 250 bps
		/* partialLiquidationRatio */ sdk.MustNewDecFromStr("0.25"),
		/* epochIdentifier */ "30 min",
		/* twapLookbackWindow */ 15*time.Minute,
	)
}

// Validate validates the set of params
func (p *Params) Validate() error {
	err := validateStopped(p.Stopped)
	if err != nil {
		return err
	}

	err = validatePercentageRatio(p.LiquidationFeeRatio)
	if err != nil {
		return err
	}

	err = validatePercentageRatio(p.FeePoolFeeRatio)
	if err != nil {
		return err
	}

	err = validatePercentageRatio(p.PartialLiquidationRatio)
	if err != nil {
		return err
	}

	return validatePercentageRatio(p.EcosystemFundFeeRatio)
}

func validatePercentageRatio(i interface{}) error {
	ratio, ok := i.(sdk.Dec)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	if ratio.IsNil() {
		return fmt.Errorf("invalid nil decimal")
	}

	if ratio.GT(sdk.OneDec()) {
		return fmt.Errorf("ratio is above max value(1.00): %s", ratio.String())
	} else if ratio.IsNegative() {
		return fmt.Errorf("ratio is negative: %s", ratio.String())
	}

	return nil
}

func validateFundingRateInterval(i interface{}) error {
	_, err := getAsString(i)
	if err != nil {
		return err
	}
	return nil
}

func validateStopped(i interface{}) error {
	_, ok := i.(bool)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}
	return nil
}

func getAsString(i interface{}) (string, error) {
	value, ok := i.(string)
	if !ok {
		return "invalid", fmt.Errorf("invalid parameter type: %T", i)
	}
	return value, nil
}

func validateTwapLookbackWindow(i interface{}) error {
	val, ok := i.(time.Duration)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}
	if val <= 0 {
		return fmt.Errorf("twap lookback window must be positive, current value is %s", val.String())
	}
	return nil
}

func validateAddress(i interface{}) error {
	val, ok := i.([]string)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}
	for _, addr := range val {
		_, err := sdk.AccAddressFromBech32(addr)
		if err != nil {
			return err
		}
	}
	return nil
}
