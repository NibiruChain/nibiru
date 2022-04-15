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
func NewParams(
	collRatio sdk.Dec, feeRatio sdk.Dec, efFeeRatio sdk.Dec, bonusRateRecoll sdk.Dec,
) Params {
	million := sdk.NewDec(1_000_000)
	collRatioInt := collRatio.Mul(million).RoundInt()
	feeRationInt := feeRatio.Mul(million).RoundInt()
	efFeeRatioInt := efFeeRatio.Mul(million).RoundInt()
	bonusRateRecollInt := bonusRateRecoll.Mul(million).RoundInt()

	return Params{
		CollRatio:       collRatioInt.Int64(),
		FeeRatio:        feeRationInt.Int64(),
		EfFeeRatio:      efFeeRatioInt.Int64(),
		BonusRateRecoll: bonusRateRecollInt.Int64(),
	}
}

// DefaultParams returns a default set of parameters
func DefaultParams() Params {
	genesisCollRatio := sdk.OneDec()
	feeRatio := sdk.MustNewDecFromStr("0.002")
	efFeeRatio := sdk.MustNewDecFromStr("0.5")
	bonusRateRecoll := sdk.MustNewDecFromStr("0.002")

	return NewParams(genesisCollRatio, feeRatio, efFeeRatio, bonusRateRecoll)
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

func validateCollRatio(i interface{}) error {
	collRatio, err := getAsInt64(i)
	if err != nil {
		return err
	}

	if collRatio > 1_000_000 {
		return fmt.Errorf("collateral ratio is above max value(1e6): %d", collRatio)
	} else if collRatio < 0 {
		return fmt.Errorf("collateral Ratio is negative: %d", collRatio)
	} else {
		return nil
	}
}

func validateBonusRateRecoll(i interface{}) error {
	bonusRateRecoll, err := getAsInt64(i)
	if err != nil {
		return err
	}

	if bonusRateRecoll > 1_000_000 {
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

	if feeRatio > 1_000_000 {
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

	if efFeeRatio > 1_000_000 {
		return fmt.Errorf("stable EF fee ratio is above max value(1e6): %d", efFeeRatio)
	} else if efFeeRatio < 0 {
		return fmt.Errorf("stable EF fee ratio is negative: %d", efFeeRatio)
	} else {
		return nil
	}
}

func getAsInt64(i interface{}) (int64, error) {
	value, ok := i.(int64)
	if !ok {
		return 0, fmt.Errorf("invalid parameter type: %T", i)
	}
	return value, nil
}
