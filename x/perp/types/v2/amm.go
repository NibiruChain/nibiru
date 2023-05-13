package v2

import (
	fmt "fmt"
	"math/big"

	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/common/asset"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (amm AMM) Validate() error {
	if amm.BaseReserve.LTE(sdk.ZeroDec()) {
		return fmt.Errorf("init pool token supply must be > 0")
	}

	if amm.QuoteReserve.LTE(sdk.ZeroDec()) {
		return fmt.Errorf("init pool token supply must be > 0")
	}

	if amm.PriceMultiplier.LTE(sdk.ZeroDec()) {
		return fmt.Errorf("init price multiplier must be > 0")
	}

	if amm.SqrtDepth.LTE(sdk.ZeroDec()) {
		return fmt.Errorf("init sqrt depth must be > 0")
	}

	return nil
}

// ValidateReserves checks that reserves are positive.
func (amm AMM) ValidateReserves() error {
	if !amm.QuoteReserve.IsPositive() || !amm.BaseReserve.IsPositive() {
		return ErrNonPositiveReserves.Wrap("pool: " + amm.String())
	}
	return nil
}

// ValidateLiquidityDepth checks that reserves are positive.
func (amm AMM) ValidateLiquidityDepth() error {
	computedSqrtDepth, err := amm.ComputeSqrtDepth()
	if err != nil {
		return err
	}

	if !amm.SqrtDepth.IsPositive() {
		return ErrLiquidityDepth.Wrap(
			"liq depth must be positive. pool: " + amm.String())
	} else if !amm.SqrtDepth.Sub(computedSqrtDepth).Abs().LTE(sdk.NewDec(1)) {
		return ErrLiquidityDepth.Wrap(
			"computed sqrt and current sqrt are mismatched. pool: " + amm.String())
	} else {
		return nil
	}
}

// returns the amount of quote reserve equivalent to the amount of quote asset given
func (amm AMM) FromQuoteAssetToReserve(quoteAsset sdk.Dec) sdk.Dec {
	return quoteAsset.Quo(amm.PriceMultiplier)
}

// returns the amount of quote asset equivalent to the amount of quote reserve given
func (amm AMM) FromQuoteReserveToAsset(quoteReserve sdk.Dec) sdk.Dec {
	return quoteReserve.Mul(amm.PriceMultiplier)
}

// Returns the amount of base reserve equivalent to the amount of base asset given
//
// args:
// - quoteReserveAmt: the amount of quote reserve before the trade, must be positive
// - dir: the direction of the trade
//
// returns:
// - baseReserveDelta: the amount of base reserve after the trade
// - err: error
//
// NOTE: baseReserveDelta is always positive
func (amm AMM) GetBaseReserveAmt(
	quoteReserveAmt sdk.Dec,
	dir Direction,
) (baseReserveDelta sdk.Dec, err error) {
	if quoteReserveAmt.LTE(sdk.ZeroDec()) {
		return sdk.ZeroDec(), nil
	}

	invariant := amm.QuoteReserve.Mul(amm.BaseReserve) // x * y = k

	var quoteReservesAfter sdk.Dec
	if dir == Direction_LONG {
		quoteReservesAfter = amm.QuoteReserve.Add(quoteReserveAmt)
	} else {
		quoteReservesAfter = amm.QuoteReserve.Sub(quoteReserveAmt)
	}

	if quoteReservesAfter.LTE(sdk.ZeroDec()) {
		return sdk.Dec{}, ErrQuoteReserveAtZero
	}

	baseReservesAfter := invariant.Quo(quoteReservesAfter)
	baseReserveDelta = baseReservesAfter.Sub(amm.BaseReserve).Abs()

	return baseReserveDelta, nil
}

// returns the amount of quote reserve equivalent to the amount of base asset given
//
// args:
// - baseReserveAmt: the amount of base reserves to trade, must be positive
// - dir: the direction of the trade
//
// returns:
// - quoteReserveDelta: the amount of quote reserve after the trade
// - err: error
//
// NOTE: quoteReserveDelta is always positive
func (amm AMM) GetQuoteReserveAmt(
	baseReserveAmt sdk.Dec,
	dir Direction,
) (quoteReserveDelta sdk.Dec, err error) {
	if baseReserveAmt.LTE(sdk.ZeroDec()) {
		return sdk.ZeroDec(), nil
	}

	invariant := amm.QuoteReserve.Mul(amm.BaseReserve) // x * y = k

	var baseReservesAfter sdk.Dec
	if dir == Direction_LONG {
		baseReservesAfter = amm.BaseReserve.Sub(baseReserveAmt)
	} else {
		baseReservesAfter = amm.BaseReserve.Add(baseReserveAmt)
	}

	if baseReservesAfter.LTE(sdk.ZeroDec()) {
		return sdk.Dec{}, ErrBaseReserveAtZero.Wrapf(
			"base assets below zero after trying to swap %s base assets",
			baseReserveAmt.String(),
		)
	}

	quoteReservesAfter := invariant.Quo(baseReservesAfter)
	quoteReserveDelta = quoteReservesAfter.Sub(amm.QuoteReserve).Abs()

	return quoteReserveDelta, nil
}

// Returns the instantaneous mark price of the trading pair
func (amm AMM) MarkPrice() sdk.Dec {
	if amm.BaseReserve.IsNil() || amm.BaseReserve.IsZero() ||
		amm.QuoteReserve.IsNil() || amm.QuoteReserve.IsZero() {
		return sdk.ZeroDec()
	}

	return amm.QuoteReserve.Quo(amm.BaseReserve).Mul(amm.PriceMultiplier)
}

// Returns the sqrt k of the reserves
func (amm AMM) ComputeSqrtDepth() (sqrtDepth sdk.Dec, err error) {
	mul := new(big.Int).Mul(amm.BaseReserve.BigInt(), amm.BaseReserve.BigInt())

	chopped := common.ChopPrecisionAndRound(mul)
	if chopped.BitLen() > common.MaxDecBitLen {
		return sdk.Dec{}, ErrLiquidityDepthOverflow
	}

	liqDepth := amm.QuoteReserve.Mul(amm.BaseReserve)
	return common.SqrtDec(liqDepth)
}

func (amm *AMM) WithBaseReserve(baseReserve sdk.Dec) *AMM {
	amm.BaseReserve = baseReserve
	return amm
}

func (amm *AMM) WithQuoteReserve(quoteReserve sdk.Dec) *AMM {
	amm.QuoteReserve = quoteReserve
	return amm
}

func (amm *AMM) WithPriceMultiplier(priceMultiplier sdk.Dec) *AMM {
	amm.PriceMultiplier = priceMultiplier
	return amm
}

func (amm *AMM) WithTotalLong(totalLong sdk.Dec) *AMM {
	amm.TotalLong = totalLong
	return amm
}

func (amm *AMM) WithTotalShort(totalShort sdk.Dec) *AMM {
	amm.TotalShort = totalShort
	return amm
}

func (amm *AMM) WithSqrtDepth(sqrtDepth sdk.Dec) *AMM {
	amm.SqrtDepth = sqrtDepth
	return amm
}

// SwapQuoteAsset swaps base asset for quote asset
//
// args:
// - quoteAssetAmt: amount of base asset to swap, must be positive
// - dir: direction the user takes
//
// returns:
// - baseAssetDelta: amount of base asset received
// - err: error
//
// NOTE: baseAssetDelta is always positive
func (amm *AMM) SwapQuoteAsset(quoteAssetAmt sdk.Dec, dir Direction) (baseAssetDelta sdk.Dec, err error) {
	quoteReserveAmt := amm.FromQuoteAssetToReserve(quoteAssetAmt)
	baseReserveDelta, err := amm.GetBaseReserveAmt(quoteReserveAmt, dir)
	if err != nil {
		return sdk.Dec{}, err
	}

	if dir == Direction_LONG {
		amm.QuoteReserve = amm.QuoteReserve.Add(quoteReserveAmt)
		amm.BaseReserve = amm.BaseReserve.Sub(baseReserveDelta)
		amm.TotalLong = amm.TotalLong.Add(baseReserveDelta)
	} else if dir == Direction_SHORT {
		amm.QuoteReserve = amm.QuoteReserve.Sub(quoteReserveAmt)
		amm.BaseReserve = amm.BaseReserve.Add(baseReserveDelta)
		amm.TotalShort = amm.TotalShort.Add(baseReserveDelta)
	}

	return baseReserveDelta, nil
}

// SwapBaseAsset swaps base asset for quote asset
//
// args:
//   - baseAssetAmt: amount of base asset to swap, must be positive
//   - dir: direction of swap
//
// returns:
//   - quoteAssetDelta: amount of quote asset received
//   - err: error if any
//
// Note: quoteAssetDelta is always positive
func (amm *AMM) SwapBaseAsset(baseAssetAmt sdk.Dec, dir Direction) (quoteAssetDelta sdk.Dec, err error) {
	quoteReserveDelta, err := amm.GetQuoteReserveAmt(baseAssetAmt, dir)
	if err != nil {
		return sdk.Dec{}, err
	}

	if dir == Direction_LONG {
		amm.QuoteReserve = amm.QuoteReserve.Add(quoteReserveDelta)
		amm.BaseReserve = amm.BaseReserve.Sub(baseAssetAmt)
		amm.TotalLong = amm.TotalLong.Add(baseAssetAmt)
	} else if dir == Direction_SHORT {
		amm.QuoteReserve = amm.QuoteReserve.Sub(quoteReserveDelta)
		amm.BaseReserve = amm.BaseReserve.Add(baseAssetAmt)
		amm.TotalShort = amm.TotalShort.Add(baseAssetAmt)
	}

	return amm.FromQuoteReserveToAsset(quoteReserveDelta), nil
}

func (amm *AMM) WithPair(pair asset.Pair) *AMM {
	amm.Pair = pair
	return amm
}

/*
GetBias returns the bias of the market in the base asset. It's the net amount of base assets for longs minus the net
amount of base assets for shorts.
*/
func (amm *AMM) GetBias() (bias sdk.Dec) {
	return amm.TotalLong.Sub(amm.TotalShort)
}

/*
GetRepegCost provides the cost of re-pegging the pool to a new candidate peg multiplier.
*/
func (amm AMM) GetRepegCost(newPriceMultiplier sdk.Dec) (cost sdk.Dec, err error) {
	if !newPriceMultiplier.IsPositive() {
		return sdk.Dec{}, ErrNonPositivePegMultiplier
	}

	bias := amm.GetBias()

	if bias.IsZero() {
		return sdk.ZeroDec(), nil
	}

	var dir Direction
	if bias.IsPositive() {
		dir = Direction_SHORT
	} else {
		dir = Direction_LONG
	}

	biasInQuoteReserve, err := amm.SwapBaseAsset(bias.Abs(), dir)
	if err != nil {
		return
	}

	cost = biasInQuoteReserve.Mul(newPriceMultiplier.Sub(amm.PriceMultiplier))

	if bias.IsNegative() {
		cost = cost.Neg()
	}

	return cost, nil
}

/*
GetSwapInvariantUpdateCost returns the cost of updating the invariant of the pool
*/
func (amm AMM) GetSwapInvariantUpdateCost(swapInvariantMultiplier sdk.Dec) (cost sdk.Dec, err error) {
	quoteReserveBefore, err := amm.GetMarketTotalQuoteReserves()
	if err != nil {
		return
	}

	newMarket, err := amm.UpdateSwapInvariant(swapInvariantMultiplier)
	if err != nil {
		return
	}

	quoteReserveAfter, err := newMarket.GetMarketTotalQuoteReserves()
	if err != nil {
		return
	}

	cost = amm.FromQuoteReserveToAsset(quoteReserveAfter.Sub(quoteReserveBefore))
	return
}

/* UpdateSwapInvariant creates a new market object with an updated swap invariant */
func (amm AMM) UpdateSwapInvariant(swapInvariantMultiplier sdk.Dec) (newAMM AMM, err error) {
	if swapInvariantMultiplier.IsNil() {
		return AMM{}, ErrNilSwapInvariantMutliplier
	}

	if !swapInvariantMultiplier.IsPositive() {
		return AMM{}, ErrNonPositiveSwapInvariantMutliplier
	}

	// k = x * y
	// newK = (cx) * (cy) = c^2 xy = c^2 k
	// newPrice = (c y) / (c x) = y / x = price | unchanged price
	swapInvariant := amm.BaseReserve.Mul(amm.QuoteReserve)
	newSwapInvariant := swapInvariant.Mul(swapInvariantMultiplier)

	// Change the swap invariant while holding price constant.
	// Multiplying by the same factor to both of the reserves won't affect price.
	cSquared := newSwapInvariant.Quo(swapInvariant)
	c, err := common.SqrtDec(cSquared)
	if err != nil {
		return
	}

	newBaseAmount := c.Mul(amm.BaseReserve)
	newQuoteAmount := c.Mul(amm.QuoteReserve)
	newSqrtDepth := common.MustSqrtDec(newBaseAmount.Mul(newQuoteAmount))

	newAMM = AMM{
		Pair:            amm.Pair,
		BaseReserve:     newBaseAmount,
		QuoteReserve:    newQuoteAmount,
		SqrtDepth:       newSqrtDepth,
		TotalLong:       amm.TotalLong,
		TotalShort:      amm.TotalShort,
		PriceMultiplier: amm.PriceMultiplier,
	}

	return newAMM, newAMM.Validate()
}

/*
GetMarketTotalQuoteReserves returns the total value of the quote reserve in the market between short and long (sum of
open notional values)
*/
func (amm AMM) GetMarketTotalQuoteReserves() (totalQuoteReserve sdk.Dec, err error) {
	longQuoteReserve, err := amm.GetQuoteReserveAmt(amm.TotalLong, Direction_SHORT)
	if err != nil {
		return sdk.Dec{}, err
	}
	shortQuoteReserve, err := amm.GetQuoteReserveAmt(amm.TotalShort, Direction_LONG)
	if err != nil {
		return sdk.Dec{}, err
	}

	return longQuoteReserve.Add(shortQuoteReserve), nil
}
