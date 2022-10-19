package types

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// HasEnoughQuoteReserve returns true if there is enough quote reserve based on
// quoteReserve * tradeLimitRatio
func (p *VPool) HasEnoughQuoteReserve(quoteAmount sdk.Dec) bool {
	return p.QuoteAssetReserve.Mul(p.TradeLimitRatio).GTE(quoteAmount)
}

// HasEnoughBaseReserve returns true if there is enough base reserve based on
// baseReserve * tradeLimitRatio
func (p *VPool) HasEnoughBaseReserve(baseAmount sdk.Dec) bool {
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
func (p *VPool) GetBaseAmountByQuoteAmount(
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
func (p *VPool) GetQuoteAmountByBaseAmount(
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
	quoteAmount = quoteAssetsAfter.Sub(p.QuoteAssetReserve).Abs()

	return quoteAmount, nil
}

// IncreaseBaseAssetReserve increases the quote reserve by amount
func (p *VPool) IncreaseBaseAssetReserve(amount sdk.Dec) {
	p.BaseAssetReserve = p.BaseAssetReserve.Add(amount)
}

// DecreaseBaseAssetReserve descreases the quote asset reserve by amount
func (p *VPool) DecreaseBaseAssetReserve(amount sdk.Dec) {
	p.BaseAssetReserve = p.BaseAssetReserve.Sub(amount)
}

func (p *VPool) IncreaseQuoteAssetReserve(amount sdk.Dec) {
	p.QuoteAssetReserve = p.QuoteAssetReserve.Add(amount)
}

// DecreaseQuoteAssetReserve decreases the base reserve by amount
func (p *VPool) DecreaseQuoteAssetReserve(amount sdk.Dec) {
	p.QuoteAssetReserve = p.QuoteAssetReserve.Sub(amount)
}

// ValidateReserves checks that reserves are positive.
func (p *VPool) ValidateReserves() error {
	if !p.QuoteAssetReserve.IsPositive() || !p.BaseAssetReserve.IsPositive() {
		return ErrNonPositiveReserves.Wrap("pool: " + p.String())
	} else {
		return nil
	}
}

func (m *VPool) Validate() error {
	if err := m.Pair.Validate(); err != nil {
		return fmt.Errorf("invalid asset pair: %w", err)
	}

	// trade limit ratio always between 0 and 1
	if m.TradeLimitRatio.LT(sdk.ZeroDec()) || m.TradeLimitRatio.GT(sdk.OneDec()) {
		return fmt.Errorf("trade limit ratio must be 0 <= ratio <= 1")
	}

	// quote asset reserve always > 0
	if !m.QuoteAssetReserve.IsPositive() {
		return fmt.Errorf("quote asset reserve must be > 0")
	}

	// base asset reserve always > 0
	if !m.BaseAssetReserve.IsPositive() {
		return fmt.Errorf("base asset reserve must be > 0")
	}

	// fluctuation limit ratio between 0 and 1
	if m.FluctuationLimitRatio.LT(sdk.ZeroDec()) || m.FluctuationLimitRatio.GT(sdk.OneDec()) {
		return fmt.Errorf("fluctuation limit ratio must be 0 <= ratio <= 1")
	}

	// max oracle spread ratio between 0 and 1
	if m.MaxOracleSpreadRatio.LT(sdk.ZeroDec()) || m.MaxOracleSpreadRatio.GT(sdk.OneDec()) {
		return fmt.Errorf("max oracle spread ratio must be 0 <= ratio <= 1")
	}

	if m.MaintenanceMarginRatio.LT(sdk.ZeroDec()) || m.MaintenanceMarginRatio.GT(sdk.OneDec()) {
		return fmt.Errorf("maintenance margin ratio ratio must be 0 <= ratio <= 1")
	}

	if m.MaxLeverage.LTE(sdk.ZeroDec()) {
		return fmt.Errorf("Max leverage must be > 0")
	}

	if sdk.OneDec().Quo(m.MaxLeverage).LT(m.MaintenanceMarginRatio) {
		return fmt.Errorf("margin ratio opened with max leverage position will be lower than Maintenance margin ratio")
	}

	return nil
}

// GetMarkPrice returns the price of the asset.
func (p VPool) GetMarkPrice() sdk.Dec {
	if p.BaseAssetReserve.IsNil() || p.BaseAssetReserve.IsZero() ||
		p.QuoteAssetReserve.IsNil() || p.QuoteAssetReserve.IsZero() {
		return sdk.ZeroDec()
	}

	return p.QuoteAssetReserve.Quo(p.BaseAssetReserve)
}

/*
IsOverFluctuationLimitInRelationWithSnapshot compares the updated pool's spot price with the current spot price.

If the fluctuation limit ratio is zero, then the fluctuation limit check is skipped.

args:
  - pool: the updated vpool
  - snapshot: the snapshot to compare against

ret:
  - bool: true if the fluctuation limit is violated. false otherwise
*/
func (p VPool) IsOverFluctuationLimitInRelationWithSnapshot(snapshot ReserveSnapshot) bool {
	if p.FluctuationLimitRatio.IsZero() {
		return false
	}

	markPrice := p.GetMarkPrice()
	snapshotUpperLimit := snapshot.GetUpperMarkPriceFluctuationLimit(p.FluctuationLimitRatio)
	snapshotLowerLimit := snapshot.GetLowerMarkPriceFluctuationLimit(p.FluctuationLimitRatio)

	if markPrice.GT(snapshotUpperLimit) || markPrice.LT(snapshotLowerLimit) {
		return true
	}

	return false
}

/*
IsOverSpreadLimit compares the current mark price of the vpool
to the underlying's index price.
It panics if you provide it with a pair that doesn't exist in the state.

args:
  - indexPrice: the index price we want to compare.

ret:
  - bool: whether or not the price has deviated from the oracle price beyond a spread ratio
*/
func (p VPool) IsOverSpreadLimit(indexPrice sdk.Dec) bool {
	return p.GetMarkPrice().Sub(indexPrice).
		Quo(indexPrice).Abs().GTE(p.MaxOracleSpreadRatio)
}
