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
	if !ok {
		return false, fmt.Errorf("error with pool trade limit ratio value: %s", pool.TradeLimitRatio)
	}

	tradeLimitRatioDec := sdk.NewDecFromIntWithPrec(tradeLimitRatio, 6)
	return quoteAssetReserve.ToDec().Mul(tradeLimitRatioDec).GTE(quoteAmount.ToDec()), nil
}

// GetBaseAmountByQuoteAmount returns the amount that you will get by specific quote amount
func GetBaseAmountByQuoteAmount(dir ammv1.Direction, pool *ammv1.Pool, quoteAmount sdk.Int) (sdk.Int, error) {
	if quoteAmount.IsZero() {
		return sdk.ZeroInt(), nil
	}

	baseAssetReserve, err := GetPoolBaseAssetReserveAsInt(pool)
	if err != nil {
		return sdk.Int{}, err
	}

	quoteAssetReserve, err := GetPoolQuoteAssetReserveAsInt(pool)
	if err != nil {
		return sdk.Int{}, err
	}

	//invariant := sdk.NewDecFromIntWithPrec(baseAssetReserve, 6).
	//	Mul(sdk.NewDecFromIntWithPrec(quoteAssetReserve, 6)) // x * y = k

	var quoteAssetAfter sdk.Int
	if dir == ammv1.Direction_ADD_TO_AMM {
		quoteAssetAfter = quoteAssetReserve.Add(quoteAmount)
	} else {
		quoteAssetAfter = quoteAssetReserve.Sub(quoteAmount)
	}

	if quoteAssetAfter.Equal(sdk.ZeroInt()) {
		return sdk.Int{}, fmt.Errorf("quote asset after is zero")
	}

	return sdk.Int{}, nil
}

// GetPoolBaseAssetReserveAsInt returns the base asset reserve value from a pool as sdk.Int
func GetPoolBaseAssetReserveAsInt(pool *ammv1.Pool) (sdk.Int, error) {
	baseAssetReserve, ok := sdk.NewIntFromString(pool.BaseAssetReserve)
	if !ok {
		return sdk.Int{}, fmt.Errorf("error with pool base asset reserve value: %s", pool.BaseAssetReserve)
	}

	return baseAssetReserve, nil
}

// GetPoolQuoteAssetReserveAsInt returns the quote asset reserve value from pool as sdk.Int
func GetPoolQuoteAssetReserveAsInt(pool *ammv1.Pool) (sdk.Int, error) {
	quoteAssetReserve, ok := sdk.NewIntFromString(pool.QuoteAssetReserve)
	if !ok {
		return sdk.Int{}, fmt.Errorf("error with pool quote asset reserve value: %s", pool.QuoteAssetReserve)
	}

	return quoteAssetReserve, nil
}
