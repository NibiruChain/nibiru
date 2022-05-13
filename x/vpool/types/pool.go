package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func NewPool(
	pair string,
	tradeLimitRatio sdk.Dec,
	quoteAssetReserve sdk.Dec,
	baseAssetReserve sdk.Dec,
	fluctuationLimitRatio sdk.Dec,
) *Pool {
	return &Pool{
		Pair:                  pair,
		BaseAssetReserve:      baseAssetReserve,
		QuoteAssetReserve:     quoteAssetReserve,
		TradeLimitRatio:       tradeLimitRatio,
		FluctuationLimitRatio: fluctuationLimitRatio,
	}
}

// HasEnoughQuoteReserve returns true if there is enough quote reserve based on
// quoteReserve * tradeLimitRatio
func (p *Pool) HasEnoughQuoteReserve(quoteAmount sdk.Dec) bool {
	return p.QuoteAssetReserve.Mul(p.TradeLimitRatio).GTE(quoteAmount)
}

// HasEnoughBaseReserve returns true if there is enough base reserve based on
// baseReserve * tradeLimitRatio
func (p *Pool) HasEnoughBaseReserve(baseAmount sdk.Dec) bool {
	return p.BaseAssetReserve.Mul(p.TradeLimitRatio).GTE(baseAmount)
}

/*
GetBaseAmountByQuoteAmount returns the amount of base asset you will get out
by giving a specified amount of quote asset

args:
  - dir: add to pool or remove from pool
  - quoteAmount: the amount of quote asset to add to/remove from the pool

ret:
  - baseAmountOut: the amount of base assets required to make this hypothetical swap
				   always an absolute value
  - err: error
*/
func (p *Pool) GetBaseAmountByQuoteAmount(
	dir Direction,
	quoteAmount sdk.Dec,
) (baseAmount sdk.Dec, err error) {
	if quoteAmount.IsZero() {
		return sdk.ZeroDec(), nil
	}

	invariant := p.QuoteAssetReserve.Mul(p.BaseAssetReserve) // x * y = k

	var quoteAssetsAfter sdk.Dec
	if dir == Direction_ADD_TO_POOL {
		quoteAssetsAfter = p.QuoteAssetReserve.Add(quoteAmount)
	} else {
		quoteAssetsAfter = p.QuoteAssetReserve.Sub(quoteAmount)
	}

	if quoteAssetsAfter.LTE(sdk.ZeroDec()) {
		return sdk.Dec{}, ErrQuoteReserveAtZero
	}

	baseAssetsAfter := invariant.Quo(quoteAssetsAfter)
	baseAmount = baseAssetsAfter.Sub(p.BaseAssetReserve).Abs()

	return baseAmount, nil
}

/*
GetQuoteAmountByBaseAmount returns the amount of quote asset you will get out
by giving a specified amount of base asset

args:
  - dir: add to pool or remove from pool
  - baseAmount: the amount of base asset to add to/remove from the pool

ret:
  - quoteAmountOut: the amount of quote assets required to make this hypothetical swap
					always an absolute value
  - err: error
*/
func (p *Pool) GetQuoteAmountByBaseAmount(
	dir Direction, baseAmount sdk.Dec,
) (quoteAmount sdk.Dec, err error) {
	if baseAmount.IsZero() {
		return sdk.ZeroDec(), nil
	}

	invariant := p.QuoteAssetReserve.Mul(p.BaseAssetReserve) // x * y = k

	var baseAssetsAfter sdk.Dec
	if dir == Direction_ADD_TO_POOL {
		baseAssetsAfter = p.BaseAssetReserve.Add(baseAmount)
	} else {
		baseAssetsAfter = p.BaseAssetReserve.Sub(baseAmount)
	}

	if baseAssetsAfter.LTE(sdk.ZeroDec()) {
		return sdk.Dec{}, ErrBaseReserveAtZero.Wrapf(
			"base assets below zero after trying to swap %s base assets",
			baseAmount.String(),
		)
	}

	quoteAssetsAfter := invariant.Quo(baseAssetsAfter)
	quoteAssetsTransferred := quoteAssetsAfter.Sub(p.QuoteAssetReserve).Abs()

	return quoteAssetsTransferred, nil
}

// IncreaseBaseAssetReserve increases the quote reserve by amount
func (p *Pool) IncreaseBaseAssetReserve(amount sdk.Dec) {
	p.BaseAssetReserve = p.BaseAssetReserve.Add(amount)
}

// DecreaseBaseAssetReserve descreases the quote asset reserve by amount
func (p *Pool) DecreaseBaseAssetReserve(amount sdk.Dec) {
	p.BaseAssetReserve = p.BaseAssetReserve.Sub(amount)
}

func (p *Pool) IncreaseQuoteAssetReserve(amount sdk.Dec) {
	p.QuoteAssetReserve = p.QuoteAssetReserve.Add(amount)
}

// DecreaseQuoteAssetReserve decreases the base reserve by amount
func (p *Pool) DecreaseQuoteAssetReserve(amount sdk.Dec) {
	p.QuoteAssetReserve = p.QuoteAssetReserve.Sub(amount)
}
