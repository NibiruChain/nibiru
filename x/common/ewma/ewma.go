package ewma

import (
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type MovingAverage interface {
	Add(sdk.Dec)
	Value() sdk.Dec
	Set(sdk.Dec)
}

func NewMovingAverage(span sdk.Dec) MovingAverage {
	return &variableEWMA{
		value: math.LegacyZeroDec(),
		decay: math.LegacyMustNewDecFromStr("2").Quo(span.Add(math.LegacyOneDec())),
	}
}

type variableEWMA struct {
	decay sdk.Dec
	value sdk.Dec
}

func (v *variableEWMA) Add(dec sdk.Dec) {
	if v.value.IsZero() {
		v.value = dec

		return
	}

	// val = val * (1 - decay) + dec * decay
	v.value = v.value.Mul(math.LegacyOneDec().Sub(v.decay)).Add(dec.Mul(v.decay))
}

func (v *variableEWMA) Value() sdk.Dec {
	return v.value
}

func (v *variableEWMA) Set(dec sdk.Dec) {
	v.value = dec
}
