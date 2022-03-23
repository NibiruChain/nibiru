package types

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func NewPool(
	pair string,
	tradeLimitRatio sdk.Dec,
	quoteAssetReserve sdk.Int,
	baseAssetReserve sdk.Int,
) *Pool {
	return &Pool{
		Pair:              pair,
		TradeLimitRatio:   tradeLimitRatio.String(),
		QuoteAssetReserve: quoteAssetReserve.String(),
		BaseAssetReserve:  baseAssetReserve.String(),
	}
}

// HasEnoughQuoteReserve returns true if there is enough quote reserve based on
// quoteReserve * tradeLimitRatio
func (p *Pool) HasEnoughQuoteReserve(quoteAmount sdk.Int) (bool, error) {
	quoteAssetReserve, ok := sdk.NewIntFromString(p.QuoteAssetReserve)
	if !ok {
		return false, fmt.Errorf("error with pool quote asset reserve value: %s",
			p.QuoteAssetReserve)
	}

	tradeLimitRatio, err := sdk.NewDecFromStr(p.TradeLimitRatio)
	if err != nil {
		return false, fmt.Errorf("error with pool trade limit ratio value: %s", p.TradeLimitRatio)
	}

	return quoteAssetReserve.ToDec().Mul(tradeLimitRatio).GTE(quoteAmount.ToDec()), nil
}

// GetBaseAmountByQuoteAmount returns the amount that you will get by specific quote amount
func GetBaseAmountByQuoteAmount(dir Direction, pool *Pool, quoteAmount sdk.Int) (sdk.Int, error) {
	if quoteAmount.IsZero() {
		return sdk.ZeroInt(), nil
	}

	_, err := GetPoolBaseAssetReserveAsInt(pool)
	if err != nil {
		return sdk.Int{}, err
	}

	quoteAssetReserve, err := GetPoolQuoteAssetReserveAsInt(pool)
	if err != nil {
		return sdk.Int{}, err
	}

	//_ := sdk.NewDecFromIntWithPrec(baseAssetReserve, 6).
	//	Mul(sdk.NewDecFromIntWithPrec(quoteAssetReserve, 6)) // x * y = k

	var quoteAssetAfter sdk.Int
	if dir == Direction_ADD_TO_AMM {
		quoteAssetAfter = quoteAssetReserve.Add(quoteAmount)
	} else {
		quoteAssetAfter = quoteAssetReserve.Sub(quoteAmount)
	}

	if quoteAssetAfter.Equal(sdk.ZeroInt()) {
		return sdk.Int{}, ErrQuoteReserveAtZero
	}

	return sdk.Int{}, nil
}

// GetPoolBaseAssetReserveAsInt returns the base asset reserve value from a pool as sdk.Int
func GetPoolBaseAssetReserveAsInt(pool *Pool) (sdk.Int, error) {
	baseAssetReserve, ok := sdk.NewIntFromString(pool.BaseAssetReserve)
	if !ok {
		return sdk.Int{}, fmt.Errorf("error with pool base asset reserve value: %s", pool.BaseAssetReserve)
	}

	return baseAssetReserve, nil
}

// GetPoolQuoteAssetReserveAsInt returns the quote asset reserve value from pool as sdk.Int
func GetPoolQuoteAssetReserveAsInt(pool *Pool) (sdk.Int, error) {
	quoteAssetReserve, ok := sdk.NewIntFromString(pool.QuoteAssetReserve)
	if !ok {
		return sdk.Int{}, fmt.Errorf("error with pool quote asset reserve value: %s", pool.QuoteAssetReserve)
	}

	return quoteAssetReserve, nil
}
