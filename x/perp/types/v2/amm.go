package v2

import (
	fmt "fmt"
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/common"
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

// returns the amount of base reserve equivalent to the amount of quote reserve swapped
func (amm AMM) GetBaseReserveAmt(
	quoteReserveAmt sdk.Dec,
	dir Direction,
) (baseReserveDeltaAbs sdk.Dec, err error) {
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
	baseReserveDeltaAbs = baseReservesAfter.Sub(amm.BaseReserve).Abs()

	return baseReserveDeltaAbs, nil
}

// returns the amount of quote reserve equivalent to the amount of base asset given
func (amm AMM) GetQuoteReserveAmt(
	baseReserveAmt sdk.Dec,
	dir Direction,
) (quoteReserveDeltaAbs sdk.Dec, err error) {
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
	quoteReserveDeltaAbs = quoteReservesAfter.Sub(amm.QuoteReserve).Abs()

	return quoteReserveDeltaAbs, nil
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

func (amm *AMM) WithBias(bias sdk.Dec) *AMM {
	amm.Bias = bias
	return amm
}

func (amm *AMM) WithSqrtDepth(sqrtDepth sdk.Dec) *AMM {
	amm.SqrtDepth = sqrtDepth
	return amm
}

func (amm *AMM) SwapQuoteAsset(quoteAssetAmt sdk.Dec, dir Direction) (baseAssetDeltaAbs sdk.Dec, err error) {
	quoteReserveAmt := amm.FromQuoteAssetToReserve(quoteAssetAmt)
	baseReserveDeltaAbs, err := amm.GetBaseReserveAmt(quoteReserveAmt, dir)
	if err != nil {
		return sdk.Dec{}, err
	}

	if dir == Direction_LONG {
		amm.QuoteReserve = amm.QuoteReserve.Add(quoteReserveAmt)
		amm.BaseReserve = amm.BaseReserve.Sub(baseReserveDeltaAbs)
		amm.Bias = amm.Bias.Add(baseReserveDeltaAbs)
	} else if dir == Direction_SHORT {
		amm.QuoteReserve = amm.QuoteReserve.Sub(quoteReserveAmt)
		amm.BaseReserve = amm.BaseReserve.Add(baseReserveDeltaAbs)
		amm.Bias = amm.Bias.Sub(baseReserveDeltaAbs)
	}

	return baseReserveDeltaAbs, nil
}

func (amm *AMM) SwapBaseAsset(baseAssetAmt sdk.Dec, dir Direction) (quoteAssetDeltaAbs sdk.Dec, err error) {
	quoteReserveDeltaAbs, err := amm.GetQuoteReserveAmt(baseAssetAmt, dir)
	if err != nil {
		return sdk.Dec{}, err
	}

	if dir == Direction_LONG {
		amm.QuoteReserve = amm.QuoteReserve.Add(quoteReserveDeltaAbs)
		amm.BaseReserve = amm.BaseReserve.Sub(baseAssetAmt)
		amm.Bias = amm.Bias.Add(baseAssetAmt)
	} else if dir == Direction_SHORT {
		amm.QuoteReserve = amm.QuoteReserve.Sub(quoteReserveDeltaAbs)
		amm.BaseReserve = amm.BaseReserve.Add(baseAssetAmt)
		amm.Bias = amm.Bias.Sub(baseAssetAmt)
	}

	return amm.FromQuoteReserveToAsset(quoteReserveDeltaAbs), nil
}
