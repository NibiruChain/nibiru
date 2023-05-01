package keeper

import (
	"errors"
	"fmt"
	"time"

	"github.com/NibiruChain/collections"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/perp/types"
	v2types "github.com/NibiruChain/nibiru/x/perp/types/v2"
)

/*
OpenPosition opens a position on the selected pair.

args:
  - ctx: cosmos-sdk context
  - pair: the pair where the position will be opened
  - side: whether the position in the BUY or SELL direction
  - traderAddr: the address of the trader who opens the position
  - quoteAssetAmount: the amount of quote asset
  - leverage: the amount of leverage to take, as sdk.Dec
  - baseAmtLimit: the limit on the base asset amount to make sure the trader doesn't get screwed, in base asset units

ret:
  - positionResp: contains the result of the open position and the new position
  - err: error
*/
func (k Keeper) OpenPosition(
	ctx sdk.Context,
	pair asset.Pair,
	side v2types.Direction,
	traderAddr sdk.AccAddress,
	quoteAssetAmt sdk.Int,
	leverage sdk.Dec,
	baseAmtLimit sdk.Dec,
) (positionResp *v2types.PositionResp, err error) {
	market, err := k.Markets.Get(ctx, pair)
	if err != nil {
		return nil, types.ErrPairNotFound
	}

	amm, err := k.AMMs.Get(ctx, pair)
	if err != nil {
		return nil, types.ErrPairNotFound
	}

	err = k.checkOpenPositionRequirements(market, quoteAssetAmt, leverage)
	if err != nil {
		return nil, err
	}

	position, err := k.Positions.Get(ctx, collections.Join(pair, traderAddr))
	isNewPosition := errors.Is(err, collections.ErrNotFound)
	if isNewPosition {
		position = v2types.ZeroPosition(ctx, pair, traderAddr)
	} else if err != nil && !isNewPosition {
		return nil, err
	}

	sameSideLong := position.Size_.IsPositive() && side == v2types.Direction_LONG
	sameSideShort := position.Size_.IsNegative() && side == v2types.Direction_SHORT

	var updatedAMM *v2types.AMM
	var openSideMatchesPosition = sameSideLong || sameSideShort
	if isNewPosition || openSideMatchesPosition {
		updatedAMM, positionResp, err = k.increasePosition(
			ctx,
			market,
			amm,
			position,
			side,
			/* openNotional */ leverage.MulInt(quoteAssetAmt),
			/* minPositionSize */ baseAmtLimit,
			/* leverage */ leverage)
		if err != nil {
			return nil, err
		}
	} else {
		updatedAMM, positionResp, err = k.openReversePosition(
			ctx,
			market,
			amm,
			position,
			/* quoteAssetAmount */ quoteAssetAmt.ToDec(),
			/* leverage */ leverage,
			/* baseAmtLimit */ baseAmtLimit,
		)
		if err != nil {
			return nil, err
		}
	}

	if err = k.afterPositionUpdate(ctx, market, *updatedAMM, traderAddr, *positionResp); err != nil {
		return nil, err
	}

	return positionResp, nil
}

// checkOpenPositionRequirements checks the minimum requirements to open a position.
//
// - Checks that quote asset is not zero.
// - Checks that leverage is not zero.
// - Checks that leverage is below requirement.
func (k Keeper) checkOpenPositionRequirements(market v2types.Market, quoteAssetAmt sdk.Int, leverage sdk.Dec) error {
	if quoteAssetAmt.IsZero() {
		return types.ErrQuoteAmountIsZero
	}

	if leverage.IsZero() {
		return types.ErrLeverageIsZero
	}

	if leverage.GT(market.MaxLeverage) {
		return types.ErrLeverageIsTooHigh
	}

	return nil
}

// afterPositionUpdate is called when a position has been updated.
func (k Keeper) afterPositionUpdate(
	ctx sdk.Context,
	market v2types.Market,
	amm v2types.AMM,
	traderAddr sdk.AccAddress,
	positionResp v2types.PositionResp,
) (err error) {
	if !positionResp.Position.Size_.IsZero() {
		k.Positions.Insert(ctx, collections.Join(market.Pair, traderAddr), *positionResp.Position)
	}

	if !positionResp.BadDebt.IsZero() {
		return fmt.Errorf("bad debt must be zero to prevent attacker from leveraging it")
	}

	if !positionResp.Position.Size_.IsZero() {
		spotNotional, err := PositionNotionalSpot(amm, *positionResp.Position)
		if err != nil {
			return err
		}
		twapNotional, err := k.PositionNotionalTWAP(ctx, *positionResp.Position, market.TwapLookbackWindow)
		if err != nil {
			return err
		}
		positionNotional := sdk.MaxDec(spotNotional, twapNotional)

		marginRatio, err := MarginRatio(*positionResp.Position, positionNotional, market.LatestCumulativePremiumFraction)
		if err != nil {
			return err
		}

		if marginRatio.LT(market.MaintenanceMarginRatio) {
			return types.ErrMarginRatioTooLow
		}
	}

	// check price fluctuation
	err = k.checkPriceFluctuationLimitRatio(ctx, market, amm)
	if err != nil {
		return err
	}

	// transfer trader <=> vault
	marginToVault := positionResp.MarginToVault.RoundInt()
	switch {
	case marginToVault.IsPositive():
		coinToSend := sdk.NewCoin(market.Pair.QuoteDenom(), marginToVault)
		if err = k.BankKeeper.SendCoinsFromAccountToModule(
			ctx, traderAddr, types.VaultModuleAccount, sdk.NewCoins(coinToSend)); err != nil {
			return err
		}
	case marginToVault.IsNegative():
		if err = k.Withdraw(ctx, market, traderAddr, marginToVault.Abs()); err != nil {
			return err
		}
	}

	transferredFee, err := k.transferFee(ctx, market.Pair, traderAddr, positionResp.ExchangedNotionalValue)
	if err != nil {
		return err
	}

	// calculate positionNotional (it's different depends on long or short side)
	// long: unrealizedPnl = positionNotional - openNotional => positionNotional = openNotional + unrealizedPnl
	// short: unrealizedPnl = openNotional - positionNotional => positionNotional = openNotional - unrealizedPnl
	positionNotional := sdk.ZeroDec()
	if positionResp.Position.Size_.IsPositive() {
		positionNotional = positionResp.Position.OpenNotional.Add(positionResp.UnrealizedPnlAfter)
	} else if positionResp.Position.Size_.IsNegative() {
		positionNotional = positionResp.Position.OpenNotional.Sub(positionResp.UnrealizedPnlAfter)
	}

	return ctx.EventManager().EmitTypedEvent(&v2types.PositionChangedEvent{
		TraderAddress:      traderAddr.String(),
		Pair:               market.Pair,
		Margin:             sdk.NewCoin(market.Pair.QuoteDenom(), positionResp.Position.Margin.RoundInt()),
		PositionNotional:   positionNotional,
		ExchangedNotional:  positionResp.ExchangedNotionalValue,
		ExchangedSize:      positionResp.ExchangedPositionSize,
		TransactionFee:     sdk.NewCoin(market.Pair.QuoteDenom(), transferredFee),
		PositionSize:       positionResp.Position.Size_,
		RealizedPnl:        positionResp.RealizedPnl,
		UnrealizedPnlAfter: positionResp.UnrealizedPnlAfter,
		BadDebt:            sdk.NewCoin(market.Pair.QuoteDenom(), positionResp.BadDebt.RoundInt()),
		FundingPayment:     positionResp.FundingPayment,
		BlockHeight:        ctx.BlockHeight(),
		BlockTimeMs:        ctx.BlockTime().UnixMilli(),
	})
}

/*
increases a position by increasedNotional amount in margin units.
Calculates the amount of margin required given the leverage parameter.
Recalculates the remaining margin after applying a funding payment.
Does not realize PnL.

For example, a long position with position notional value of 150 NUSD and unrealized PnL of 50 NUSD
could increase their position by 30 NUSD using 10x leverage.
This would be:
  - 3 NUSD as margin requirement
  - new open notional value of 130 NUSD
  - new position notional value of 150 NUSD
  - unrealized PnL remains unchanged at 50 NUSD
  - remaining margin is calculated by applying the funding payment

args:
  - ctx: cosmos-sdk context
  - currentPosition: the current position
  - side: whether the position is increasing in the BUY or SELL direction
  - increasedNotional: the notional value to increase the position by, in margin units
  - baseAmtLimit: the limit on the base asset amount to make sure the trader doesn't get screwed, in base asset units
  - leverage: the amount of leverage to take, as sdk.Dec

ret:
  - positionResp: contains the result of the increase position and the new position
  - err: error
*/
func (k Keeper) increasePosition(
	ctx sdk.Context,
	market v2types.Market,
	amm v2types.AMM,
	currentPosition v2types.Position,
	side v2types.Direction,
	increasedNotional sdk.Dec,
	baseAmtLimit sdk.Dec,
	leverage sdk.Dec,
) (updatedAMM *v2types.AMM, positionResp *v2types.PositionResp, err error) {
	positionResp = &v2types.PositionResp{}

	updatedAMM, baseAssetDeltaAbs, err := k.SwapQuoteAsset(
		ctx,
		market,
		amm,
		side,
		increasedNotional,
		baseAmtLimit,
	)
	if err != nil {
		return nil, nil, err
	}

	marginDelta := increasedNotional.Quo(leverage)
	fundingPayment := FundingPayment(currentPosition, market.LatestCumulativePremiumFraction)
	remainingMarginSigned := currentPosition.Margin.Add(marginDelta).Sub(fundingPayment)

	positionNotional, err := PositionNotionalSpot(amm, currentPosition)
	if err != nil {
		return nil, nil, err
	}
	unrealizedPnl := UnrealizedPnl(currentPosition, positionNotional)

	if side == v2types.Direction_LONG {
		positionResp.ExchangedPositionSize = baseAssetDeltaAbs
	} else if side == v2types.Direction_SHORT {
		positionResp.ExchangedPositionSize = baseAssetDeltaAbs.Neg()
	}

	positionResp.ExchangedNotionalValue = increasedNotional
	positionResp.PositionNotional = positionNotional.Add(increasedNotional)
	positionResp.UnrealizedPnlAfter = unrealizedPnl
	positionResp.RealizedPnl = sdk.ZeroDec()
	positionResp.MarginToVault = marginDelta
	positionResp.FundingPayment = fundingPayment
	positionResp.BadDebt = sdk.MinDec(sdk.ZeroDec(), remainingMarginSigned).Abs()
	positionResp.Position = &v2types.Position{
		TraderAddress:                   currentPosition.TraderAddress,
		Pair:                            currentPosition.Pair,
		Size_:                           currentPosition.Size_.Add(positionResp.ExchangedPositionSize),
		Margin:                          sdk.MaxDec(sdk.ZeroDec(), remainingMarginSigned).Abs(),
		OpenNotional:                    currentPosition.OpenNotional.Add(increasedNotional),
		LatestCumulativePremiumFraction: market.LatestCumulativePremiumFraction,
		LastUpdatedBlockNumber:          ctx.BlockHeight(),
	}

	return updatedAMM, positionResp, nil
}

// TODO test: openReversePosition | https://github.com/NibiruChain/nibiru/issues/299
func (k Keeper) openReversePosition(
	ctx sdk.Context,
	market v2types.Market,
	amm v2types.AMM,
	currentPosition v2types.Position,
	quoteAssetAmount sdk.Dec,
	leverage sdk.Dec,
	baseAmtLimit sdk.Dec,
) (updatedAMM *v2types.AMM, positionResp *v2types.PositionResp, err error) {
	notionalToDecreaseBy := leverage.Mul(quoteAssetAmount)
	currentPositionNotional, err := PositionNotionalSpot(amm, currentPosition)
	if err != nil {
		return nil, nil, err
	}

	if currentPositionNotional.GT(notionalToDecreaseBy) {
		// position reduction
		return k.decreasePosition(
			ctx,
			market,
			amm,
			currentPosition,
			notionalToDecreaseBy,
			baseAmtLimit,
			/* skipFluctuationLimitCheck */ false,
		)
	} else {
		// close and reverse
		return k.closeAndOpenReversePosition(
			ctx,
			market,
			amm,
			currentPosition,
			quoteAssetAmount,
			leverage,
			baseAmtLimit,
		)
	}
}

/*
Decreases a position by decreasedNotional amount in margin units.
Realizes PnL and calculates remaining margin after applying a funding payment.

For example, a long position with position notional value of 150 NUSD and PnL of 50 NUSD
could decrease their position by 30 NUSD. This would realize a PnL of 10 NUSD (50NUSD * 30/150)
and update their margin (old margin + realized PnL - funding payment).
Their new position notional value would be 120 NUSD and their position size would
shrink by 20%.

args:
  - ctx: cosmos-sdk context
  - currentPosition: the current position
  - decreasedNotional: the notional value to decrease the position by, in margin units
  - baseAmtLimit: the limit on the base asset amount to make sure the trader doesn't get screwed, in base asset units
  - skipFluctuationLimitCheck: whether or not the position change can go over the fluctuation limit

ret:
  - positionResp: contains the result of the decrease position and the new position
  - err: error
*/
func (k Keeper) decreasePosition(
	ctx sdk.Context,
	market v2types.Market,
	amm v2types.AMM,
	currentPosition v2types.Position,
	decreasedNotional sdk.Dec,
	baseAmtLimit sdk.Dec,
	skipFluctuationLimitCheck bool,
) (updatedAMM *v2types.AMM, positionResp *v2types.PositionResp, err error) {
	if currentPosition.Size_.IsZero() {
		return nil, nil, fmt.Errorf("current position size is zero, nothing to decrease")
	}

	var dir v2types.Direction
	if currentPosition.Size_.IsPositive() {
		dir = v2types.Direction_SHORT
	} else {
		dir = v2types.Direction_LONG
	}

	positionResp = &v2types.PositionResp{
		MarginToVault: sdk.ZeroDec(),
	}

	currentPositionNotional, err := PositionNotionalSpot(amm, currentPosition)
	if err != nil {
		return nil, nil, err
	}
	currentUnrealizedPnl := UnrealizedPnl(currentPosition, currentPositionNotional)

	updatedAMM, baseAssetDeltaAbs, err := k.SwapQuoteAsset(
		ctx,
		market,
		amm,
		dir,
		decreasedNotional,
		baseAmtLimit,
	)
	if err != nil {
		return nil, nil, err
	}

	if dir == v2types.Direction_LONG {
		positionResp.ExchangedPositionSize = baseAssetDeltaAbs
	} else {
		positionResp.ExchangedPositionSize = baseAssetDeltaAbs.Neg()
	}

	positionResp.RealizedPnl = currentUnrealizedPnl.Mul(
		positionResp.ExchangedPositionSize.Abs().
			Quo(currentPosition.Size_.Abs()),
	)

	fundingPayment := FundingPayment(currentPosition, market.LatestCumulativePremiumFraction)
	remainingMargin := currentPosition.Margin.Add(positionResp.RealizedPnl).Sub(fundingPayment)

	positionResp.BadDebt = sdk.MinDec(sdk.ZeroDec(), remainingMargin).Abs()
	positionResp.FundingPayment = fundingPayment
	positionResp.UnrealizedPnlAfter = currentUnrealizedPnl.Sub(positionResp.RealizedPnl)
	positionResp.ExchangedNotionalValue = decreasedNotional
	positionResp.PositionNotional = currentPositionNotional.Sub(decreasedNotional)

	// calculate openNotional (it's different depends on long or short side)
	// long: unrealizedPnl = positionNotional - openNotional => openNotional = positionNotional - unrealizedPnl
	// short: unrealizedPnl = openNotional - positionNotional => openNotional = positionNotional + unrealizedPnl
	// positionNotional = oldPositionNotional - notionalValueToDecrease
	var remainOpenNotional sdk.Dec
	if currentPosition.Size_.IsPositive() {
		remainOpenNotional = positionResp.PositionNotional.Sub(positionResp.UnrealizedPnlAfter)
	} else {
		remainOpenNotional = positionResp.PositionNotional.Add(positionResp.UnrealizedPnlAfter)
	}

	if remainOpenNotional.IsNegative() {
		return nil, nil, fmt.Errorf("value of open notional < 0")
	}

	positionResp.Position = &v2types.Position{
		TraderAddress:                   currentPosition.TraderAddress,
		Pair:                            currentPosition.Pair,
		Size_:                           currentPosition.Size_.Add(positionResp.ExchangedPositionSize),
		Margin:                          sdk.MaxDec(sdk.ZeroDec(), remainingMargin).Abs(),
		OpenNotional:                    remainOpenNotional,
		LatestCumulativePremiumFraction: market.LatestCumulativePremiumFraction,
		LastUpdatedBlockNumber:          ctx.BlockHeight(),
	}

	return updatedAMM, positionResp, nil
}

/*
Closes a position and realizes PnL and funding payments.
Opens a position in the opposite direction if there is notional value remaining.
Errors out if the provided notional value is not greater than the existing position's notional value.
Errors out if there is bad debt.

args:
  - ctx: cosmos-sdk context
  - existingPosition: current position
  - quoteAssetAmount: the amount of notional value to move by. Must be greater than the existingPosition's notional value.
  - leverage: the amount of leverage to take
  - baseAmtLimit: limit on the base asset movement to ensure trader doesn't get screwed
  - skipFluctuationLimitCheck: whether or not to skip the fluctuation limit check

ret:
  - positionResp: response object containing information about the position change
  - err: error
*/
func (k Keeper) closeAndOpenReversePosition(
	ctx sdk.Context,
	market v2types.Market,
	amm v2types.AMM,
	existingPosition v2types.Position,
	quoteAssetAmount sdk.Dec,
	leverage sdk.Dec,
	baseAmtLimit sdk.Dec,
) (updatedAMM *v2types.AMM, positionResp *v2types.PositionResp, err error) {
	trader, err := sdk.AccAddressFromBech32(existingPosition.TraderAddress)
	if err != nil {
		return nil, nil, err
	}

	updatedAMM, closePositionResp, err := k.closePositionEntirely(
		ctx,
		market,
		amm,
		existingPosition,
		/* quoteAssetAmountLimit */ sdk.ZeroDec(),
		/* skipFluctuationLimitCheck */ false,
	)
	if err != nil {
		return nil, nil, err
	}

	if closePositionResp.BadDebt.IsPositive() {
		return nil, nil, fmt.Errorf("underwater position")
	}

	reverseNotionalValue := leverage.Mul(quoteAssetAmount)
	remainingReverseNotionalValue := reverseNotionalValue.Sub(
		closePositionResp.ExchangedNotionalValue)

	var increasePositionResp *v2types.PositionResp
	if remainingReverseNotionalValue.IsNegative() {
		// should never happen as this should also be checked in the caller
		return nil, nil, fmt.Errorf(
			"provided quote asset amount and leverage not large enough to close position. need %s but got %s",
			closePositionResp.ExchangedNotionalValue.String(), reverseNotionalValue.String())
	} else if remainingReverseNotionalValue.IsPositive() {
		updatedBaseAmtLimit := baseAmtLimit
		if baseAmtLimit.IsPositive() {
			updatedBaseAmtLimit = baseAmtLimit.Sub(closePositionResp.ExchangedPositionSize.Abs())
		}
		if updatedBaseAmtLimit.IsNegative() {
			return nil, nil, fmt.Errorf(
				"position size changed by greater than the specified base limit: %s",
				baseAmtLimit.String(),
			)
		}

		var sideToTake v2types.Direction
		// flipped since we are going against the current position
		if existingPosition.Size_.IsPositive() {
			sideToTake = v2types.Direction_SHORT
		} else {
			sideToTake = v2types.Direction_LONG
		}

		newPosition := v2types.ZeroPosition(
			ctx,
			existingPosition.Pair,
			trader,
		)
		updatedAMM, increasePositionResp, err = k.increasePosition(
			ctx,
			market,
			*updatedAMM,
			newPosition,
			sideToTake,
			remainingReverseNotionalValue,
			updatedBaseAmtLimit,
			leverage,
		)
		if err != nil {
			return nil, nil, err
		}

		positionResp = &v2types.PositionResp{
			Position:               increasePositionResp.Position,
			PositionNotional:       increasePositionResp.PositionNotional,
			ExchangedNotionalValue: closePositionResp.ExchangedNotionalValue.Add(increasePositionResp.ExchangedNotionalValue),
			BadDebt:                closePositionResp.BadDebt.Add(increasePositionResp.BadDebt),
			ExchangedPositionSize:  closePositionResp.ExchangedPositionSize.Add(increasePositionResp.ExchangedPositionSize),
			FundingPayment:         closePositionResp.FundingPayment.Add(increasePositionResp.FundingPayment),
			RealizedPnl:            closePositionResp.RealizedPnl.Add(increasePositionResp.RealizedPnl),
			MarginToVault:          closePositionResp.MarginToVault.Add(increasePositionResp.MarginToVault),
			UnrealizedPnlAfter:     sdk.ZeroDec(),
		}
	} else {
		// case where remaining open notional == 0
		positionResp = closePositionResp
	}

	return updatedAMM, positionResp, nil
}

/*
Closes a position and realizes PnL and funding payments.
Does not error out if there is bad debt, that is for callers to decide.

args:
  - ctx: cosmos-sdk context
  - currentPosition: current position
  - quoteAssetAmountLimit: a limit on quote asset to ensure trader doesn't get screwed
  - skipFluctuationLimitCheck: whether or not to skip the fluctuation limit check

ret:
  - positionResp: response object containing information about the position change
  - err: error
*/
func (k Keeper) closePositionEntirely(
	ctx sdk.Context,
	market v2types.Market,
	amm v2types.AMM,
	currentPosition v2types.Position,
	quoteAssetAmountLimit sdk.Dec,
	skipFluctuationLimitCheck bool,
) (updatedAMM *v2types.AMM, positionResp *v2types.PositionResp, err error) {
	if currentPosition.Size_.IsZero() {
		return nil, nil, fmt.Errorf("zero position size")
	}

	trader, err := sdk.AccAddressFromBech32(currentPosition.TraderAddress)
	if err != nil {
		return nil, nil, err
	}

	positionResp = &v2types.PositionResp{
		UnrealizedPnlAfter:    sdk.ZeroDec(),
		ExchangedPositionSize: currentPosition.Size_.Neg(),
		PositionNotional:      sdk.ZeroDec(),
	}

	// calculate unrealized PnL
	positionNotional, err := PositionNotionalSpot(amm, currentPosition)
	if err != nil {
		return nil, nil, err
	}
	unrealizedPnl := UnrealizedPnl(currentPosition, positionNotional)

	positionResp.RealizedPnl = unrealizedPnl
	// calculate remaining margin with funding payment
	fundingPayment := FundingPayment(currentPosition, market.LatestCumulativePremiumFraction)
	remainingMargin := currentPosition.Margin.Add(unrealizedPnl).Sub(fundingPayment)

	if remainingMargin.IsPositive() {
		positionResp.BadDebt = sdk.ZeroDec()
		positionResp.MarginToVault = remainingMargin.Neg()
	} else {
		positionResp.BadDebt = remainingMargin.Abs()
		positionResp.MarginToVault = sdk.ZeroDec()
	}

	positionResp.FundingPayment = fundingPayment

	var sideToTake v2types.Direction
	// flipped since we are going against the current position
	if currentPosition.Size_.IsPositive() {
		sideToTake = v2types.Direction_SHORT
	} else {
		sideToTake = v2types.Direction_LONG
	}
	updatedAMM, exchangedNotionalValue, err := k.SwapBaseAsset(
		ctx,
		market,
		amm,
		sideToTake,
		currentPosition.Size_.Abs(),
		quoteAssetAmountLimit,
	)
	if err != nil {
		return nil, nil, err
	}

	positionResp.ExchangedNotionalValue = exchangedNotionalValue
	positionResp.Position = &v2types.Position{
		TraderAddress:                   currentPosition.TraderAddress,
		Pair:                            currentPosition.Pair,
		Size_:                           sdk.ZeroDec(),
		Margin:                          sdk.ZeroDec(),
		OpenNotional:                    sdk.ZeroDec(),
		LatestCumulativePremiumFraction: market.LatestCumulativePremiumFraction,
		LastUpdatedBlockNumber:          ctx.BlockHeight(),
	}

	err = k.Positions.Delete(ctx, collections.Join(currentPosition.Pair, trader))
	if err != nil {
		return nil, nil, err
	}

	return updatedAMM, positionResp, nil
}

/*
ClosePosition closes a position entirely and transfers the remaining margin back to the user.
Errors if the position has bad debt.

args:
  - ctx: the cosmos-sdk context
  - pair: the trading pair
  - traderAddr: the trader's address

ret:
  - positionResp: the response containing the updated position and applied funding payment, bad debt, PnL
  - err: error if any
*/
func (k Keeper) ClosePosition(ctx sdk.Context, pair asset.Pair, traderAddr sdk.AccAddress) (*v2types.PositionResp, error) {
	position, err := k.Positions.Get(ctx, collections.Join(pair, traderAddr))
	if err != nil {
		return nil, err
	}

	market, err := k.Markets.Get(ctx, pair)
	if err != nil {
		return nil, types.ErrPairNotFound
	}

	amm, err := k.AMMs.Get(ctx, pair)
	if err != nil {
		return nil, err
	}

	updatedAMM, positionResp, err := k.closePositionEntirely(
		ctx,
		market,
		amm,
		position,
		/* quoteAssetAmountLimit */ sdk.ZeroDec(),
		/* skipFluctuationLimitCheck */ false,
	)
	if err != nil {
		return nil, err
	}

	if positionResp.BadDebt.IsPositive() {
		return nil, fmt.Errorf("underwater position")
	}

	if err = k.afterPositionUpdate(
		ctx,
		market,
		*updatedAMM,
		traderAddr,
		*positionResp,
	); err != nil {
		return nil, err
	}

	return positionResp, nil
}

func (k Keeper) transferFee(
	ctx sdk.Context,
	pair asset.Pair,
	trader sdk.AccAddress,
	positionNotional sdk.Dec,
) (fees sdk.Int, err error) {
	m, err := k.Markets.Get(ctx, pair)
	if err != nil {
		return sdk.Int{}, err
	}

	feeToFeePool := m.ExchangeFeeRatio.Mul(positionNotional).RoundInt()
	if feeToFeePool.IsPositive() {
		if err = k.BankKeeper.SendCoinsFromAccountToModule(
			ctx,
			/* from */ trader,
			/* to */ types.FeePoolModuleAccount,
			/* coins */ sdk.NewCoins(
				sdk.NewCoin(
					pair.QuoteDenom(),
					feeToFeePool,
				),
			),
		); err != nil {
			return sdk.Int{}, err
		}
	}

	feeToEcosystemFund := m.EcosystemFundFeeRatio.Mul(positionNotional).RoundInt()
	if feeToEcosystemFund.IsPositive() {
		if err = k.BankKeeper.SendCoinsFromAccountToModule(
			ctx,
			/* from */ trader,
			/* to */ types.PerpEFModuleAccount,
			/* coins */ sdk.NewCoins(
				sdk.NewCoin(
					pair.QuoteDenom(),
					feeToEcosystemFund,
				),
			),
		); err != nil {
			return sdk.Int{}, err
		}
	}

	return feeToFeePool.Add(feeToEcosystemFund), nil
}

/*
Check's that a pool that we're about to save to state does not violate the fluctuation limit.
Always tries to check against a snapshot from a previous block. If one doesn't exist, then it just uses the current snapshot.
This should run prior to updating the snapshot, otherwise it will compare the currently updated market to itself.

args:
  - ctx: the cosmos-sdk context
  - pool: the updated market

ret:
  - err: error if any
*/
func (k Keeper) checkPriceFluctuationLimitRatio(ctx sdk.Context, market v2types.Market, amm v2types.AMM) error {
	if market.PriceFluctuationLimitRatio.IsZero() {
		// early return to avoid expensive state operations
		return nil
	}

	it := k.ReserveSnapshots.Iterate(ctx, collections.PairRange[asset.Pair, time.Time]{}.Prefix(amm.Pair).Descending())
	defer it.Close()

	if !it.Valid() {
		return fmt.Errorf("error getting last snapshot number for pair %s", amm.Pair)
	}

	snapshotMarkPrice := it.Value().Amm.MarkPrice()
	snapshotUpperLimit := snapshotMarkPrice.Mul(sdk.OneDec().Add(market.PriceFluctuationLimitRatio))
	snapshotLowerLimit := snapshotMarkPrice.Mul(sdk.OneDec().Sub(market.PriceFluctuationLimitRatio))

	if amm.MarkPrice().GT(snapshotUpperLimit) || snapshotMarkPrice.LT(snapshotLowerLimit) {
		return v2types.ErrOverFluctuationLimit
	}

	return nil
}
