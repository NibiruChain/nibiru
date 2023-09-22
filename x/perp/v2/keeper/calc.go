package keeper

import (
	"fmt"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"

	types "github.com/NibiruChain/nibiru/x/perp/v2/types"
)

// PositionNotionalSpot returns the position's notional value based on the spot price.
func PositionNotionalSpot(amm types.AMM, position types.Position) (positionNotional sdk.Dec, err error) {
	// we want to know the price if the user closes their position
	// e.g. if the user has positive size, we want to short
	if position.Size_.IsNil() {
		return sdk.Dec{}, fmt.Errorf("input base amt is nil")
	}

	var dir types.Direction
	if position.Size_.IsPositive() {
		dir = types.Direction_SHORT
	} else {
		dir = types.Direction_LONG
	}

	quoteReserve, err := amm.GetQuoteReserveAmt(position.Size_.Abs(), dir)
	if err != nil {
		return sdk.Dec{}, err
	}
	return amm.FromQuoteReserveToAsset(quoteReserve), nil
}

// PositionNotionalTWAP returns the position's notional value based on the TWAP price.
func (k Keeper) PositionNotionalTWAP(ctx sdk.Context,
	position types.Position,
	twapLookbackWindow time.Duration,
) (positionNotional sdk.Dec, err error) {
	// we want to know the price if the user closes their position
	// e.g. if the user has positive size, we want to short
	var dir types.Direction
	if position.Size_.IsPositive() {
		dir = types.Direction_SHORT
	} else {
		dir = types.Direction_LONG
	}

	return k.CalcTwap(
		ctx,
		position.Pair,
		types.TwapCalcOption_BASE_ASSET_SWAP,
		dir,
		position.Size_.Abs(),
		/*lookbackInterval=*/ twapLookbackWindow,
	)
}

// UnrealizedPnl calculates the unrealized profits and losses (PnL) of a position.
func UnrealizedPnl(position types.Position, positionNotional sdk.Dec) (unrealizedPnlSigned sdk.Dec) {
	if position.Size_.IsPositive() {
		// LONG
		return positionNotional.Sub(position.OpenNotional)
	} else {
		// SHORT
		return position.OpenNotional.Sub(positionNotional)
	}
}

// MarginRatio Given a position and it's notional value, returns the margin ratio.
func MarginRatio(
	position types.Position,
	positionNotional sdk.Dec,
	marketLatestCumulativePremiumFraction sdk.Dec,
) sdk.Dec {
	if position.Size_.IsZero() || positionNotional.IsZero() {
		return sdk.ZeroDec()
	}

	unrealizedPnl := UnrealizedPnl(position, positionNotional)
	fundingPayment := FundingPayment(position, marketLatestCumulativePremiumFraction)
	remainingMargin := position.Margin.Add(unrealizedPnl).Sub(fundingPayment)

	return remainingMargin.Quo(positionNotional)
}

// FundingPayment calculates the funding payment of a position.
//
// args:
//   - position: the position to calculate funding payment for
//   - marketLatestCumulativePremiumFraction: the latest cumulative premium fraction of the market
//
// returns:
//   - fundingPayment: the funding payment of the position, signed
func FundingPayment(position types.Position, marketLatestCumulativePremiumFraction sdk.Dec) sdk.Dec {
	return marketLatestCumulativePremiumFraction.
		Sub(position.LatestCumulativePremiumFraction).
		Mul(position.Size_)
}
