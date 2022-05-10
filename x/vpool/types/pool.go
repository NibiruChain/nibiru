package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func NewPool(
	pair string,
	tradeLimitRatio sdk.Dec,
	quoteAssetReserve sdk.Int,
	baseAssetReserve sdk.Int,
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
func (p *Pool) HasEnoughQuoteReserve(quoteAmount sdk.Int) bool {
	return p.QuoteAssetReserve.ToDec().Mul(p.TradeLimitRatio).GTE(quoteAmount.ToDec())
}

// GetBaseAmountByQuoteAmount returns the amount that you will get by specific quote amount
func (p *Pool) GetBaseAmountByQuoteAmount(dir Direction, quoteAmount sdk.Int) (sdk.Int, error) {
	if quoteAmount.IsZero() {
		return sdk.ZeroInt(), nil
	}

	invariant := p.QuoteAssetReserve.ToDec().Mul(p.BaseAssetReserve.ToDec()) // x * y = k

	var quoteAssetsAfter sdk.Dec
	if dir == Direction_ADD_TO_POOL {
		quoteAssetsAfter = p.QuoteAssetReserve.Add(quoteAmount).ToDec()
	} else {
		quoteAssetsAfter = p.QuoteAssetReserve.Sub(quoteAmount).ToDec()
	}

	if quoteAssetsAfter.LTE(sdk.ZeroDec()) {
		return sdk.Int{}, ErrQuoteReserveAtZero
	}

	baseAssetsAfter := invariant.Quo(quoteAssetsAfter)
	baseAssetsTransferred := p.BaseAssetReserve.ToDec().Sub(baseAssetsAfter).Abs()

	// protocol always gives out less base assets to long traders
	// and gives more base asset debt to short traders (i.e. requires more base asset repayment from short traders)
	if dir == Direction_ADD_TO_POOL {
		return baseAssetsTransferred.TruncateInt(), nil
	} else {
		return baseAssetsTransferred.Ceil().TruncateInt(), nil
	}
}

// IncreaseBaseAssetReserve increases the quote reserve by amount
func (p *Pool) IncreaseBaseAssetReserve(amount sdk.Int) {
	p.BaseAssetReserve = p.BaseAssetReserve.Add(amount)
}

// DecreaseBaseAssetReserve descreases the quote asset reserve by amount
func (p *Pool) DecreaseBaseAssetReserve(amount sdk.Int) {
	p.BaseAssetReserve = p.BaseAssetReserve.Sub(amount)
}

func (p *Pool) IncreaseQuoteAssetReserve(amount sdk.Int) {
	p.QuoteAssetReserve = p.QuoteAssetReserve.Add(amount)
}

// DecreaseQuoteAssetReserve decreases the base reserve by amount
func (p *Pool) DecreaseQuoteAssetReserve(amount sdk.Int) {
	p.QuoteAssetReserve = p.QuoteAssetReserve.Sub(amount)
}
