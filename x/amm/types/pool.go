package types

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	ammv1 "github.com/MatrixDao/matrix/api/amm"
)

// PoolHasEnoughQuoteReserve returns true if there is enough quote reserve based on
// quoteReserve * tradeLimitRatio
func PoolHasEnoughQuoteReserve(pool *ammv1.Pool, quoteAmount sdk.Int) (bool, error) {
	quoteAssetReserve, ok := sdk.NewIntFromString(pool.QuoteAssetReserve)
	if !ok {
		return false, fmt.Errorf("error with pool quote asset reserve value: %s", pool.QuoteAssetReserve)
	}

	tradeLimitRatio, ok := sdk.NewIntFromString(pool.TradeLimitRatio)
	if !ok {
		return false, fmt.Errorf("error with pool trade limit ratio value: %s", pool.TradeLimitRatio)
	}

	return quoteAssetReserve.Mul(tradeLimitRatio).GTE(quoteAmount), nil
}
