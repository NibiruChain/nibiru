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
func NewParams(collRatio sdk.Dec, feeRatio sdk.Dec, efFeeRatio sdk.Dec) Params {
	sixthPower := sdk.MustNewDecFromStr("1000000")
	collRatioInt := collRatio.Mul(sixthPower).RoundInt()
	feeRationInt := feeRatio.Mul(sixthPower).RoundInt()
	efFeeRatioInt := efFeeRatio.Mul(sixthPower).RoundInt()

	return Params{
		CollRatio:  collRatioInt.Int64(),
		FeeRatio:   feeRationInt.Int64(),
		EfFeeRatio: efFeeRatioInt.Int64(),
	}
}

// DefaultParams returns a default set of parameters
func DefaultParams() Params {
	genesisCollRatio := sdk.OneDec()
	feeRatio := sdk.MustNewDecFromStr("0.002")
	efFeeRatio := sdk.MustNewDecFromStr("0.5")

	return NewParams(genesisCollRatio, feeRatio, efFeeRatio)
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

func validateCollRatio(i interface{}) error {
	collRatio, err := getAsInt64(i)
	if err != nil {
		return err
	}

	if collRatio > 1_000_000 {
		return fmt.Errorf("collateral Ratio is above max value(1e6): %d", collRatio)
	} else if collRatio < 0 {
		return fmt.Errorf("collateral Ratio is negative: %d", collRatio)
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
		return fmt.Errorf("fee Ratio is above max value(1e6): %d", feeRatio)
	} else {
		return nil
	}
}

func validateEfFeeRatio(i interface{}) error {
	feeRatio, err := getAsInt64(i)
	if err != nil {
		return err
	}

	if feeRatio > 1_000_000 {
		return fmt.Errorf("ecosystem fund fee Ratio is above max value(1e6): %d", feeRatio)
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
