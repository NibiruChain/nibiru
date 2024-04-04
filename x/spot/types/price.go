package types

import (
	"cosmossdk.io/math"
)

// CalcSpotPrice calculates the spot price based on weight.
// spotPrice = (BalanceIn / WeightIn) / (BalanceOut / WeightOut)
func (pool Pool) CalcSpotPrice(tokenIn, tokenOut string) (math.LegacyDec, error) {
	_, poolAssetIn, err := pool.getPoolAssetAndIndex(tokenIn)
	if err != nil {
		return math.LegacyDec{}, err
	}

	_, poolAssetOut, err := pool.getPoolAssetAndIndex(tokenOut)
	if err != nil {
		return math.LegacyDec{}, err
	}

	weightedBalanceIn := math.LegacyNewDecFromInt(poolAssetIn.Token.Amount).Quo(math.LegacyNewDecFromInt(poolAssetIn.Weight))
	weightedBalanceOut := math.LegacyNewDecFromInt(poolAssetOut.Token.Amount).Quo(math.LegacyNewDecFromInt(poolAssetOut.Weight))

	return weightedBalanceIn.Quo(weightedBalanceOut), nil
}
