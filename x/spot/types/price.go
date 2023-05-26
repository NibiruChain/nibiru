package types

import sdk "github.com/cosmos/cosmos-sdk/types"

// CalcSpotPrice calculates the spot price based on weight.
// spotPrice = (BalanceIn / WeightIn) / (BalanceOut / WeightOut)
func (pool Pool) CalcSpotPrice(tokenIn, tokenOut string) (sdk.Dec, error) {
	_, poolAssetIn, err := pool.getPoolAssetAndIndex(tokenIn)
	if err != nil {
		return sdk.Dec{}, err
	}

	_, poolAssetOut, err := pool.getPoolAssetAndIndex(tokenOut)
	if err != nil {
		return sdk.Dec{}, err
	}

	weightedBalanceIn := sdk.NewDecFromInt(poolAssetIn.Token.Amount).Quo(sdk.NewDecFromInt(poolAssetIn.Weight))
	weightedBalanceOut := sdk.NewDecFromInt(poolAssetOut.Token.Amount).Quo(sdk.NewDecFromInt(poolAssetOut.Weight))

	return weightedBalanceIn.Quo(weightedBalanceOut), nil
}
