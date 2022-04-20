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
		Token0Reserve:         quoteAssetReserve.String(),
		Token1Reserve:         baseAssetReserve.String(),
		FluctuationLimitRatio: fluctuationLimitRatio.String(),
	}
}

// HasEnoughQuoteReserve returns true if there is enough quote reserve based on
// quoteReserve * tradeLimitRatio
func (p *Pool) HasEnoughQuoteReserve(quoteAmount sdk.Int) (bool, error) {
	quoteAssetReserve, ok := sdk.NewIntFromString(p.Token0Reserve)
	if !ok {
		return false, fmt.Errorf("error with pool quote asset reserve value: %s",
			p.Token0Reserve)
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

	baseAssetReserve, err := p.GetPoolToken1ReserveAsInt()
	if err != nil {
		return sdk.Int{}, err
	}

	quoteAssetReserve, err := p.GetPoolToken0ReserveAsInt()
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

// GetPoolToken1ReserveAsInt returns the base asset reserve value from a pool as sdk.Int
func (p *Pool) GetPoolToken1ReserveAsInt() (sdk.Int, error) {
	baseAssetReserve, ok := sdk.NewIntFromString(p.Token1Reserve)
	if !ok {
		return sdk.Int{}, fmt.Errorf("error with pool base asset reserve value: %s", p.Token1Reserve)
	}

	return baseAssetReserve, nil
}

// GetPoolToken0ReserveAsInt returns the quote asset reserve value from pool as sdk.Int
func (p *Pool) GetPoolToken0ReserveAsInt() (sdk.Int, error) {
	quoteAssetReserve, ok := sdk.NewIntFromString(p.Token0Reserve)
	if !ok {
		return sdk.Int{}, fmt.Errorf("error with pool quote asset reserve value: %s", p.Token0Reserve)
	}

	return quoteAssetReserve, nil
}

// IncreaseToken0Reserve increases the quote reserve by amount
func (p *Pool) IncreaseToken0Reserve(amount sdk.Int) {
	quoteAssetReserve, _ := p.GetPoolToken0ReserveAsInt()
	p.Token0Reserve = quoteAssetReserve.Add(amount).String()
}

// DecreaseToken1Reserve decreases the base reserve by amount
func (p *Pool) DecreaseToken1Reserve(amount sdk.Int) {
	baseAssetReserve, _ := p.GetPoolToken1ReserveAsInt()
	p.Token1Reserve = baseAssetReserve.Sub(amount).String()
}

// DecreaseToken0Reserve descreases the quote asset reserve by amount
func (p *Pool) DecreaseToken0Reserve(amount sdk.Int) {
	quoteAssetReserve, _ := p.GetPoolToken0ReserveAsInt()
	p.Token0Reserve = quoteAssetReserve.Sub(amount).String()
}

func (p *Pool) IncreaseToken1Reserve(amount sdk.Int) {
	baseAssetReserve, _ := p.GetPoolToken1ReserveAsInt()
	p.Token1Reserve = baseAssetReserve.Add(amount).String()
}
