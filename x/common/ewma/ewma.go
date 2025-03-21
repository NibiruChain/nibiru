package ewma

import (
	"cosmossdk.io/math"
)

type MovingAverage interface {
	Add(math.LegacyDec)
	Value() math.LegacyDec
	Set(math.LegacyDec)
}

func NewMovingAverage(span math.LegacyDec) MovingAverage {
	return &variableEWMA{
		value: math.LegacyZeroDec(),
		decay: math.LegacyMustNewDecFromStr("2").Quo(span.Add(math.LegacyOneDec())),
	}
}

type variableEWMA struct {
	decay math.LegacyDec
	value math.LegacyDec
}

func (v *variableEWMA) Add(dec math.LegacyDec) {
	if v.value.IsZero() {
		v.value = dec

		return
	}

	// val = val * (1 - decay) + dec * decay
	v.value = v.value.Mul(math.LegacyOneDec().Sub(v.decay)).Add(dec.Mul(v.decay))
}

func (v *variableEWMA) Value() math.LegacyDec {
	return v.value
}

func (v *variableEWMA) Set(dec math.LegacyDec) {
	v.value = dec
}
