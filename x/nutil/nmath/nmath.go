package nmath

import "math/big"

// BigMin returns the smaller of x or y
func BigMin(x, y *big.Int) *big.Int {
	if x == nil || x.Cmp(y) > 0 {
		return y
	}
	return x
}

// BigMax returns the larger of x or y
func BigMax(x, y *big.Int) *big.Int {
	if x == nil || x.Cmp(y) < 0 {
		return y
	}
	return x
}
