package types

import (
	fmt "fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"

	"github.com/NibiruChain/nibiru/x/common"
)

var _ paramtypes.ParamSet = (*Params)(nil)

// ParamKeyTable the param key table for launch module
func ParamKeyTable() paramtypes.KeyTable {
	return paramtypes.NewKeyTable().RegisterParamSet(&Params{})
}

// NewParams creates a new Params instance
func NewParams(
	collRatio sdk.Dec,
	feeRatio sdk.Dec,
	efFeeRatio sdk.Dec,
	bonusRateRecoll sdk.Dec,
	distrEpochIdentifier string,
	adjustmentStep sdk.Dec,
	priceLowerBound sdk.Dec,
	priceUpperBound sdk.Dec,
	isCollateralRatioValid bool,
) Params {
	million := sdk.NewDec(1 * common.TO_MICRO)
	collRatioInt := collRatio.Mul(million).RoundInt().Int64()
	feeRationInt := feeRatio.Mul(million).RoundInt().Int64()
	efFeeRatioInt := efFeeRatio.Mul(million).RoundInt().Int64()
	bonusRateRecollInt := bonusRateRecoll.Mul(million).RoundInt().Int64()

	adjustmentStepInt := adjustmentStep.Mul(million).RoundInt().Int64()
	priceLowerBoundInt := priceLowerBound.Mul(million).RoundInt().Int64()
	priceUpperBoundInt := priceUpperBound.Mul(million).RoundInt().Int64()

	return Params{
		CollRatio:              collRatioInt,
		FeeRatio:               feeRationInt,
		EfFeeRatio:             efFeeRatioInt,
		BonusRateRecoll:        bonusRateRecollInt,
		DistrEpochIdentifier:   distrEpochIdentifier,
		AdjustmentStep:         adjustmentStepInt,
		PriceLowerBound:        priceLowerBoundInt,
		PriceUpperBound:        priceUpperBoundInt,
		IsCollateralRatioValid: isCollateralRatioValid,
	}
}

// DefaultParams returns a default set of parameters
func DefaultParams() Params {
	genesisCollRatio := sdk.OneDec()
	feeRatio := sdk.MustNewDecFromStr("0.002")
	efFeeRatio := sdk.MustNewDecFromStr("0.5")
	bonusRateRecoll := sdk.MustNewDecFromStr("0.002")
	distrEpochIdentifier := "15 min"
	adjustmentStep := sdk.MustNewDecFromStr("0.0025")
	priceLowerBound := sdk.MustNewDecFromStr("0.9999")
	priceUpperBound := sdk.MustNewDecFromStr("1.0001")
	isCollateralRatioValid := false // Will be valid once we start posting prices and updating the collateral

	return NewParams(genesisCollRatio, feeRatio, efFeeRatio, bonusRateRecoll, distrEpochIdentifier,
		adjustmentStep,
		priceLowerBound,
		priceUpperBound, isCollateralRatioValid)
}

// ParamSetPairs get the params.ParamSet
func (p *Params) ParamSetPairs() paramtypes.ParamSetPairs {
	return paramtypes.ParamSetPairs{
		paramtypes.NewParamSetPair(
			[]byte("CollRatio"),
			&p.CollRatio,
			validateCollRatio,
		),
		paramtypes.NewParamSetPair(
			[]byte("FeeRatio"),
			&p.FeeRatio,
			validateFeeRatio,
		),
		paramtypes.NewParamSetPair(
			[]byte("EfFeeRatio"),
			&p.EfFeeRatio,
			validateEfFeeRatio,
		),
		paramtypes.NewParamSetPair(
			[]byte("BonusRateRecoll"),
			&p.BonusRateRecoll,
			validateBonusRateRecoll,
		),
		paramtypes.NewParamSetPair(
			[]byte("DistrEpochIdentifier"),
			&p.DistrEpochIdentifier,
			validateDistrEpochIdentifier,
		),
		paramtypes.NewParamSetPair(
			[]byte("AdjustmentStep"),
			&p.AdjustmentStep,
			validateAdjustmentStep,
		),
		paramtypes.NewParamSetPair(
			[]byte("PriceLowerBound"),
			&p.PriceLowerBound,
			validatePriceLowerBound,
		),
		paramtypes.NewParamSetPair(
			[]byte("PriceUpperBound"),
			&p.PriceUpperBound,
			validatePriceUpperBound,
		),
		paramtypes.NewParamSetPair(
			[]byte("IsCollateralRatioValid"),
			&p.IsCollateralRatioValid,
			validateIsCollateralRatioValid,
		),
	}
}

// Validate validates the set of params
func (p *Params) Validate() error {
	err := validateCollRatio(p.CollRatio)
	if err != nil {
		return err
	}

	err = validateFeeRatio(p.FeeRatio)
	if err != nil {
		return err
	}

	return validateEfFeeRatio(p.EfFeeRatio)
}

func (p *Params) GetFeeRatioAsDec() sdk.Dec {
	return sdk.NewIntFromUint64(uint64(p.FeeRatio)).
		ToDec().Quo(sdk.MustNewDecFromStr("1000000"))
}

func (p *Params) GetCollRatioAsDec() sdk.Dec {
	return sdk.NewIntFromUint64(uint64(p.CollRatio)).
		ToDec().Quo(sdk.MustNewDecFromStr("1000000"))
}

func (p *Params) GetEfFeeRatioAsDec() sdk.Dec {
	return sdk.NewIntFromUint64(uint64(p.EfFeeRatio)).
		ToDec().Quo(sdk.MustNewDecFromStr("1000000"))
}

func (p *Params) GetBonusRateRecollAsDec() sdk.Dec {
	return sdk.NewIntFromUint64(uint64(p.BonusRateRecoll)).
		ToDec().Quo(sdk.MustNewDecFromStr("1000000"))
}

func (p *Params) GetAdjustmentStepAsDec() sdk.Dec {
	return sdk.NewIntFromUint64(uint64(p.AdjustmentStep)).
		ToDec().Quo(sdk.MustNewDecFromStr("1000000"))
}

func (p *Params) GetPriceLowerBoundAsDec() sdk.Dec {
	return sdk.NewIntFromUint64(uint64(p.PriceLowerBound)).
		ToDec().Quo(sdk.MustNewDecFromStr("1000000"))
}

func (p *Params) GetPriceUpperBoundAsDec() sdk.Dec {
	return sdk.NewIntFromUint64(uint64(p.PriceUpperBound)).
		ToDec().Quo(sdk.MustNewDecFromStr("1000000"))
}

func validateCollRatio(i interface{}) error {
	collRatio, err := getAsInt64(i)
	if err != nil {
		return err
	}

	if collRatio > 1*common.TO_MICRO {
		return fmt.Errorf("collateral ratio is above max value(1e6): %d", collRatio)
	} else if collRatio < 0 {
		return fmt.Errorf("collateral Ratio is negative: %d", collRatio)
	} else {
		return nil
	}
}

func validateIsCollateralRatioValid(i interface{}) error {
	_, ok := i.(bool)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}
	return nil
}

func validateBonusRateRecoll(i interface{}) error {
	bonusRateRecoll, err := getAsInt64(i)
	if err != nil {
		return err
	}

	if bonusRateRecoll > 1*common.TO_MICRO {
		return fmt.Errorf("collateral Ratio is above max value(1e6): %d", bonusRateRecoll)
	} else if bonusRateRecoll < 0 {
		return fmt.Errorf("collateral Ratio is negative: %d", bonusRateRecoll)
	} else {
		return nil
	}
}

func validateFeeRatio(i interface{}) error {
	feeRatio, err := getAsInt64(i)
	if err != nil {
		return err
	}

	if feeRatio > 1*common.TO_MICRO {
		return fmt.Errorf("fee ratio is above max value(1e6): %d", feeRatio)
	} else if feeRatio < 0 {
		return fmt.Errorf("fee ratio is negative: %d", feeRatio)
	} else {
		return nil
	}
}

func validateEfFeeRatio(i interface{}) error {
	efFeeRatio, err := getAsInt64(i)
	if err != nil {
		return err
	}

	if efFeeRatio > 1*common.TO_MICRO {
		return fmt.Errorf("stable EF fee ratio is above max value(1e6): %d", efFeeRatio)
	} else if efFeeRatio < 0 {
		return fmt.Errorf("stable EF fee ratio is negative: %d", efFeeRatio)
	} else {
		return nil
	}
}

func validateDistrEpochIdentifier(i interface{}) error {
	_, err := getString(i)
	if err != nil {
		return err
	}
	return nil
}

func validateAdjustmentStep(i interface{}) error {
	adjustmentStep, err := getAsInt64(i)
	if err != nil {
		return err
	}

	if adjustmentStep > 1*common.TO_MICRO {
		return fmt.Errorf("AdjustmentStep is above max value(1e6): %d", adjustmentStep)
	} else if adjustmentStep < 0 {
		return fmt.Errorf("AdjustmentStep is negative: %d", adjustmentStep)
	} else {
		return nil
	}
}

func validatePriceLowerBound(i interface{}) error {
	priceLowerBound, err := getAsInt64(i)
	if err != nil {
		return err
	}

	if priceLowerBound > 1*common.TO_MICRO {
		return fmt.Errorf("PriceLowerBound is above max value(1e6): %d", priceLowerBound)
	} else if priceLowerBound < 0 {
		return fmt.Errorf("PriceLowerBound is negative: %d", priceLowerBound)
	} else {
		return nil
	}
}

func validatePriceUpperBound(i interface{}) error {
	priceUpperBound, err := getAsInt64(i)
	if err != nil {
		return err
	}

	if priceUpperBound > 2*common.TO_MICRO {
		return fmt.Errorf("PriceUpperBound is above max value(1e6): %d", priceUpperBound)
	} else if priceUpperBound < 0 {
		return fmt.Errorf("PriceUpperBound is negative: %d", priceUpperBound)
	} else {
		return nil
	}
}

func getString(i interface{}) (string, error) {
	value, ok := i.(string)
	if !ok {
		return "invalid", fmt.Errorf("invalid parameter type: %T", i)
	}
	return value, nil
}

func getAsInt64(i interface{}) (int64, error) {
	value, ok := i.(int64)
	if !ok {
		return 0, fmt.Errorf("invalid parameter type: %T", i)
	}
	return value, nil
}
