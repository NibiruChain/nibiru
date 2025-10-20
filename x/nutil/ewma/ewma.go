package ewma

import (
	sdkmath "cosmossdk.io/math"
)

type MovingAverage interface {
	Add(sdkmath.LegacyDec)
	Value() sdkmath.LegacyDec
	Set(sdkmath.LegacyDec)
}

func NewMovingAverage(span sdkmath.LegacyDec) MovingAverage {
	return &variableEWMA{
		value: sdkmath.LegacyZeroDec(),
		decay: sdkmath.LegacyMustNewDecFromStr("2").Quo(span.Add(sdkmath.LegacyOneDec())),
	}
}

type variableEWMA struct {
	decay sdkmath.LegacyDec
	value sdkmath.LegacyDec
}

func (v *variableEWMA) Add(dec sdkmath.LegacyDec) {
	if v.value.IsZero() {
		v.value = dec

		return
	}

	// val = val * (1 - decay) + dec * decay
	v.value = v.value.Mul(sdkmath.LegacyOneDec().Sub(v.decay)).Add(dec.Mul(v.decay))
}

func (v *variableEWMA) Value() sdkmath.LegacyDec {
	return v.value
}

func (v *variableEWMA) Set(dec sdkmath.LegacyDec) {
	v.value = dec
}
