package types

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	ammv1 "github.com/MatrixDao/matrix/api/amm"
)

func NewPool(
	pair string,
	tradeLimitRatio sdk.Int,
	quoteAssetReserve sdk.Int,
	baseAssetReserve sdk.Int,
) *ammv1.Pool {
	return &ammv1.Pool{
		Pair:              pair,
		TradeLimitRatio:   tradeLimitRatio.String(),
		QuoteAssetReserve: quoteAssetReserve.String(),
		BaseAssetReserve:  baseAssetReserve.String(),
	}
}

// PoolHasEnoughQuoteReserve returns true if there is enough quote reserve based on
// quoteReserve * tradeLimitRatio
func PoolHasEnoughQuoteReserve(pool *ammv1.Pool, quoteAmount sdk.Int) (bool, error) {
	quoteAssetReserve, ok := sdk.NewIntFromString(pool.QuoteAssetReserve)
	if !ok {
		return false, fmt.Errorf("error with pool quote asset reserve value: %s", pool.QuoteAssetReserve)
	}

	tradeLimitRatio, ok := sdk.NewIntFromString(pool.TradeLimitRatio)
	tradeLimitRatioDec := sdk.NewDecFromIntWithPrec(tradeLimitRatio, 6)
	if !ok {
		return false, fmt.Errorf("error with pool trade limit ratio value: %s", pool.TradeLimitRatio)
	}

	return quoteAssetReserve.ToDec().Mul(tradeLimitRatioDec).GTE(quoteAmount.ToDec()), nil
}

// GetBaseAmountByQuoteAmount returns the amount that you will get by specific quote amount
func GetBaseAmountByQuoteAmount(dir ammv1.Direction, pool *ammv1.Pool, quoteAmount sdk.Int) sdk.Int {
	return sdk.ZeroInt()
}
