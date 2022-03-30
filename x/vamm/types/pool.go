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
	fluctuationLimitRatio sdk.Dec,
) *Pool {
	return &Pool{
		Pair:                  pair,
		TradeLimitRatio:       tradeLimitRatio.String(),
		QuoteAssetReserve:     quoteAssetReserve.String(),
		BaseAssetReserve:      baseAssetReserve.String(),
		FluctuationLimitRatio: fluctuationLimitRatio.String(),
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
func (p *Pool) GetBaseAmountByQuoteAmount(dir Direction, quoteAmount sdk.Int) (sdk.Int, error) {
	if quoteAmount.IsZero() {
		return sdk.ZeroInt(), nil
	}

	baseAssetReserve, err := p.GetPoolBaseAssetReserveAsInt()
	if err != nil {
		return sdk.Int{}, err
	}

	quoteAssetReserve, err := p.GetPoolQuoteAssetReserveAsInt()
	if err != nil {
		return sdk.Int{}, err
	}

	invariant := quoteAssetReserve.ToDec().Mul(baseAssetReserve.ToDec()) // x * y = k

	var quoteAssetAfter sdk.Dec
	if dir == Direction_ADD_TO_AMM {
		quoteAssetAfter = quoteAssetReserve.ToDec().Add(quoteAmount.ToDec())
	} else {
		quoteAssetAfter = quoteAssetReserve.ToDec().Sub(quoteAmount.ToDec())
	}

	if quoteAssetAfter.Equal(sdk.ZeroDec()) {
		return sdk.Int{}, ErrQuoteReserveAtZero
	}

	baseAssetAfter := invariant.Quo(quoteAssetAfter)
	baseAssetBought := baseAssetAfter.Sub(baseAssetReserve.ToDec()).Abs()

	if !invariant.TruncateInt().Mod(quoteAssetAfter.TruncateInt()).Equal(sdk.ZeroInt()) {
		if dir == Direction_ADD_TO_AMM {
			baseAssetBought = baseAssetBought.Sub(sdk.OneDec())
		} else {
			baseAssetBought = baseAssetBought.Add(sdk.OneDec())
		}
	}

	return baseAssetBought.TruncateInt(), nil
}

// GetPoolBaseAssetReserveAsInt returns the base asset reserve value from a pool as sdk.Int
func (p *Pool) GetPoolBaseAssetReserveAsInt() (sdk.Int, error) {
	baseAssetReserve, ok := sdk.NewIntFromString(p.BaseAssetReserve)
	if !ok {
		return sdk.Int{}, fmt.Errorf("error with pool base asset reserve value: %s", p.BaseAssetReserve)
	}

	return baseAssetReserve, nil
}

// GetPoolQuoteAssetReserveAsInt returns the quote asset reserve value from pool as sdk.Int
func (p *Pool) GetPoolQuoteAssetReserveAsInt() (sdk.Int, error) {
	quoteAssetReserve, ok := sdk.NewIntFromString(p.QuoteAssetReserve)
	if !ok {
		return sdk.Int{}, fmt.Errorf("error with pool quote asset reserve value: %s", p.QuoteAssetReserve)
	}

	return quoteAssetReserve, nil
}

// IncreaseQuoteAssetReserve increases the quote reserve by amount
func (p *Pool) IncreaseQuoteAssetReserve(amount sdk.Int) {
	quoteAssetReserve, _ := p.GetPoolQuoteAssetReserveAsInt()
	p.QuoteAssetReserve = quoteAssetReserve.Add(amount).String()
}

// DecreaseBaseAssetReserve decreases the base reserve by amount
func (p *Pool) DecreaseBaseAssetReserve(amount sdk.Int) {
	baseAssetReserve, _ := p.GetPoolBaseAssetReserveAsInt()
	p.BaseAssetReserve = baseAssetReserve.Sub(amount).String()
}

// DecreaseQuoteAssetReserve descreases the quote asset reserve by amount
func (p *Pool) DecreaseQuoteAssetReserve(amount sdk.Int) {
	quoteAssetReserve, _ := p.GetPoolQuoteAssetReserveAsInt()
	p.QuoteAssetReserve = quoteAssetReserve.Sub(amount).String()
}

func (p *Pool) IncreaseBaseAssetReserve(amount sdk.Int) {
	baseAssetReserve, _ := p.GetPoolBaseAssetReserveAsInt()
	p.BaseAssetReserve = baseAssetReserve.Add(amount).String()
}
