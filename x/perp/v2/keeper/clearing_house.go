package keeper

import (
	"errors"
	"fmt"
	"time"

	"github.com/NibiruChain/collections"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/perp/v2/types"
)

// OpenPosition opens a position on the selected pair.
//
// args:
//   - ctx: cosmos-sdk context
//   - pair: pair to open position on
//   - dir: direction the user is taking
//   - traderAddr: address of the trader
//   - quoteAssetAmt: amount of quote asset to open position with
//   - leverage: leverage to open position with
//   - baseAmtLimit: minimum base asset amount to open position with
//
// ret:
//   - positionResp: contains the result of the open position and the new position
//   - err: error
func (k Keeper) OpenPosition(
	ctx sdk.Context,
	pair asset.Pair,
	dir types.Direction,
	traderAddr sdk.AccAddress,
	quoteAssetAmt sdk.Int,
	leverage sdk.Dec,
	baseAmtLimit sdk.Dec,
) (positionResp *types.PositionResp, err error) {
	market, err := k.Markets.Get(ctx, pair)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", types.ErrPairNotFound, pair)
	}

	if !market.Enabled {
		return nil, fmt.Errorf("%w: %s", types.ErrMarketNotEnabled, pair)
	}

	amm, err := k.AMMs.Get(ctx, pair)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", types.ErrPairNotFound, pair)
	}

	err = checkOpenPositionRequirements(market, quoteAssetAmt, leverage)
	if err != nil {
		return nil, err
	}

	position, err := k.Positions.Get(ctx, collections.Join(pair, traderAddr))
	isNewPosition := errors.Is(err, collections.ErrNotFound)
	if isNewPosition {
		position = types.ZeroPosition(ctx, pair, traderAddr)
	} else if err != nil && !isNewPosition {
		return nil, err
	}

	sameSideLong := position.Size_.IsPositive() && dir == types.Direction_LONG
	sameSideShort := position.Size_.IsNegative() && dir == types.Direction_SHORT

	var updatedAMM *types.AMM
	var openSideMatchesPosition = sameSideLong || sameSideShort
	if isNewPosition || openSideMatchesPosition {
		updatedAMM, positionResp, err = k.increasePosition(
			ctx,
			market,
			amm,
			position,
			dir,
			/* openNotional */ leverage.MulInt(quoteAssetAmt),
			/* minPositionSize */ baseAmtLimit,
			/* leverage */ leverage)
		if err != nil {
			return nil, err
		}
	} else {
		quoteAssetAmtToDec := sdk.NewDecFromInt(quoteAssetAmt)
		updatedAMM, positionResp, err = k.openReversePosition(
			ctx,
			market,
			amm,
			position,
			/* quoteAssetAmount */ quoteAssetAmtToDec,
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

// increases a position by increasedNotional amount in margin units.
// Calculates the amount of margin required given the leverage parameter.
// Recalculates the remaining margin after applying a funding payment.
// Does not realize PnL.
//
// For example, a long position with position notional value of 150 NUSD and unrealized PnL of 50 NUSD
// could increase their position by 30 NUSD using 10x leverage.
// This would be:
//   - 3 NUSD as margin requirement
//   - new open notional value of 130 NUSD
//   - new position notional value of 150 NUSD
//   - unrealized PnL remains unchanged at 50 NUSD
//   - remaining margin is calculated by applying the funding payment
//
// args:
//   - ctx: sdk.Context
//   - market: the perp market
//   - amm: the amm reserves
//   - currentPosition: the current position
//   - dir: the direction the user is taking
//   - increasedNotional: the amount of notional the user is increasing by, must be positive
//   - baseAmtLimit: the user-specified limit on the base reserves
//   - leverage: the leverage the user is taking
//
// returns:
//   - updatedAMM: the updated AMM reserves
//   - positionResp: updated position information
//   - err: error
func (k Keeper) increasePosition(
	ctx sdk.Context,
	market types.Market,
	amm types.AMM,
	currentPosition types.Position,
	dir types.Direction,
	increasedNotional sdk.Dec, // unsigned
	baseAmtLimit sdk.Dec, // unsigned
	leverage sdk.Dec, // unsigned
) (updatedAMM *types.AMM, positionResp *types.PositionResp, err error) {
	positionResp = &types.PositionResp{}
	marginIncrease := increasedNotional.Quo(leverage)
	fundingPayment := FundingPayment(currentPosition, market.LatestCumulativePremiumFraction) // signed
	remainingMargin := currentPosition.Margin.Add(marginIncrease).Sub(fundingPayment)         // signed

	updatedAMM, baseAssetDeltaAbs, err := k.SwapQuoteAsset(
		ctx,
		market,
		amm,
		dir,
		increasedNotional,
		baseAmtLimit,
	)
	if err != nil {
		return nil, nil, err
	}

	if dir == types.Direction_LONG {
		positionResp.ExchangedPositionSize = baseAssetDeltaAbs
	} else if dir == types.Direction_SHORT {
		positionResp.ExchangedPositionSize = baseAssetDeltaAbs.Neg()
	}

	positionNotional, err := PositionNotionalSpot(amm, currentPosition)
	if err != nil {
		return nil, nil, err
	}

	positionResp.ExchangedNotionalValue = increasedNotional
	positionResp.PositionNotional = positionNotional.Add(increasedNotional)
	positionResp.RealizedPnl = sdk.ZeroDec()
	positionResp.MarginToVault = marginIncrease
	positionResp.FundingPayment = fundingPayment
	positionResp.BadDebt = sdk.MinDec(sdk.ZeroDec(), remainingMargin).Abs()
	positionResp.Position = types.Position{
		TraderAddress:                   currentPosition.TraderAddress,
		Pair:                            currentPosition.Pair,
		Size_:                           currentPosition.Size_.Add(positionResp.ExchangedPositionSize),
		Margin:                          sdk.MaxDec(sdk.ZeroDec(), remainingMargin).Abs(),
		OpenNotional:                    currentPosition.OpenNotional.Add(increasedNotional),
		LatestCumulativePremiumFraction: market.LatestCumulativePremiumFraction,
		LastUpdatedBlockNumber:          ctx.BlockHeight(),
	}
	positionResp.UnrealizedPnlAfter = UnrealizedPnl(positionResp.Position, positionResp.PositionNotional)

	return updatedAMM, positionResp, nil
}

// decreases a position by decreasedNotional amount in margin units.
// Calculates the amount of margin required given the leverage parameter.
// Recalculates the remaining margin after applying a funding payment.
//
// args:
//   - ctx: sdk.Context
//   - market: the perp market
//   - amm: the amm reserves
//   - currentPosition: the current position
//   - decreasedNotional: the amount of notional the user is decreasing by
//   - baseAmtLimit: the user-specified limit on the base reserves
//   - skipFluctuationLimitCheck: whether to skip the fluctuation limit check
//
// returns:
//   - updatedAMM: the updated AMM reserves
//   - positionResp: updated position information
//   - err: error
func (k Keeper) openReversePosition(
	ctx sdk.Context,
	market types.Market,
	amm types.AMM,
	currentPosition types.Position,
	quoteAssetAmount sdk.Dec,
	leverage sdk.Dec,
	baseAmtLimit sdk.Dec,
) (updatedAMM *types.AMM, positionResp *types.PositionResp, err error) {
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

// Decreases a position by decreasedNotional amount in margin units.
// Realizes PnL and calculates remaining margin after applying a funding payment.
//
// For example, a long position with position notional value of 150 NUSD and PnL of 50 NUSD
// could decrease their position by 30 NUSD. This would realize a PnL of 10 NUSD (50NUSD * 30/150)
// and update their margin (old margin + realized PnL - funding payment).
// Their new position notional value would be 120 NUSD and their position size would
// shrink by 20%.
//
// args:
//   - ctx: cosmos-sdk context
//   - market: the perp market
//   - amm: the amm reserves
//   - currentPosition: the current position
//   - decreasedNotional: the amount of notional the user is decreasing by
//   - baseAmtLimit: the user-specified limit on the base reserves
//   - skipFluctuationLimitCheck: whether to skip the fluctuation limit check
//
// returns:
//   - updatedAMM: the updated AMM reserves
//   - positionResp: updated position information
//   - err: error
func (k Keeper) decreasePosition(
	ctx sdk.Context,
	market types.Market,
	amm types.AMM,
	currentPosition types.Position,
	decreasedNotional sdk.Dec,
	baseAmtLimit sdk.Dec,
	skipFluctuationLimitCheck bool,
) (updatedAMM *types.AMM, positionResp *types.PositionResp, err error) {
	if currentPosition.Size_.IsZero() {
		return nil, nil, fmt.Errorf("current position size is zero, nothing to decrease")
	}

	var dir types.Direction
	if currentPosition.Size_.IsPositive() {
		dir = types.Direction_SHORT
	} else {
		dir = types.Direction_LONG
	}

	positionResp = &types.PositionResp{
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

	if dir == types.Direction_LONG {
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

	positionResp.Position = types.Position{
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

// Closes a position and realizes PnL and funding payments.
// Opens a position in the opposite direction if there is notional value remaining.
// Errors out if the provided notional value is not greater than the existing position's notional value.
// Errors out if there is bad debt.
//
// args:
//   - ctx: cosmos-sdk context
//   - market: the perp market
//   - amm: the amm reserves
//   - existingPosition: the existing position
//   - quoteAssetAmount: the amount of quote asset to close
//   - leverage: the leverage to open the new position with
//   - baseAmtLimit: the user-specified limit on the base reserves
//
// returns:
//   - updatedAMM: the updated AMM reserves
//   - positionResp: updated position information
//   - err: error
func (k Keeper) closeAndOpenReversePosition(
	ctx sdk.Context,
	market types.Market,
	amm types.AMM,
	existingPosition types.Position,
	quoteAssetAmount sdk.Dec,
	leverage sdk.Dec,
	baseAmtLimit sdk.Dec,
) (updatedAMM *types.AMM, positionResp *types.PositionResp, err error) {
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

	var increasePositionResp *types.PositionResp
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

		var sideToTake types.Direction
		// flipped since we are going against the current position
		if existingPosition.Size_.IsPositive() {
			sideToTake = types.Direction_SHORT
		} else {
			sideToTake = types.Direction_LONG
		}

		newPosition := types.ZeroPosition(
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

		positionResp = &types.PositionResp{
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

// checkOpenPositionRequirements checks the minimum requirements to open a position.
//
// - Checks that quote asset is not zero.
// - Checks that leverage is not zero.
// - Checks that leverage is below requirement.
//
// args:
// - market: the market where the position will be opened
// - quoteAssetAmt: the amount of quote asset
// - leverage: the amount of leverage to take, as sdk.Dec
//
// returns:
// - error: if any of the requirements is not met
func checkOpenPositionRequirements(market types.Market, quoteAssetAmt sdk.Int, userLeverage sdk.Dec) error {
	if !quoteAssetAmt.IsPositive() {
		return types.ErrInputQuoteAmtNegative
	}

	if !userLeverage.IsPositive() {
		return types.ErrUserLeverageNegative
	}

	if userLeverage.GT(market.MaxLeverage) {
		return types.ErrLeverageIsTooHigh
	}

	return nil
}

// afterPositionUpdate is called when a position has been updated.
func (k Keeper) afterPositionUpdate(
	ctx sdk.Context,
	market types.Market,
	amm types.AMM,
	traderAddr sdk.AccAddress,
	positionResp types.PositionResp,
) (err error) {
	// check bad debt
	if !positionResp.BadDebt.IsZero() {
		return fmt.Errorf("bad debt must be zero to prevent attacker from leveraging it")
	}

	// check price fluctuation
	if err := k.checkPriceFluctuationLimitRatio(ctx, market, amm); err != nil {
		return err
	}

	if !positionResp.Position.Size_.IsZero() {
		spotNotional, err := PositionNotionalSpot(amm, positionResp.Position)
		if err != nil {
			return err
		}
		twapNotional, err := k.PositionNotionalTWAP(ctx, positionResp.Position, market.TwapLookbackWindow)
		if err != nil {
			return err
		}
		var preferredPositionNotional sdk.Dec
		if positionResp.Position.Size_.IsPositive() {
			preferredPositionNotional = sdk.MaxDec(spotNotional, twapNotional)
		} else {
			preferredPositionNotional = sdk.MinDec(spotNotional, twapNotional)
		}

		marginRatio := MarginRatio(positionResp.Position, preferredPositionNotional, market.LatestCumulativePremiumFraction)
		if marginRatio.LT(market.MaintenanceMarginRatio) {
			return types.ErrMarginRatioTooLow
		}
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

	if !positionResp.Position.Size_.IsZero() {
		k.Positions.Insert(ctx, collections.Join(market.Pair, traderAddr), positionResp.Position)
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

	return ctx.EventManager().EmitTypedEvent(&types.PositionChangedEvent{
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

// transfers the fee to the exchange fee pool
//
// args:
// - ctx: the cosmos-sdk context
// - pair: the trading pair
// - trader: the trader's address
// - positionNotional: the position's notional value
//
// returns:
// - fees: the fees to be transferred
// - err: error if any
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

	feeToExchangeFeePool := m.ExchangeFeeRatio.Mul(positionNotional).RoundInt()
	if feeToExchangeFeePool.IsPositive() {
		if err = k.BankKeeper.SendCoinsFromAccountToModule(
			ctx,
			/* from */ trader,
			/* to */ types.FeePoolModuleAccount,
			/* coins */ sdk.NewCoins(
				sdk.NewCoin(
					pair.QuoteDenom(),
					feeToExchangeFeePool,
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

	return feeToExchangeFeePool.Add(feeToEcosystemFund), nil
}

// checks that the mark price of the pool does not violate the fluctuation limit
//
// args:
//   - ctx: the cosmos-sdk context
//   - market: the perp market
//   - amm: the amm reserves
//
// returns:
//   - err: error if any
func (k Keeper) checkPriceFluctuationLimitRatio(ctx sdk.Context, market types.Market, amm types.AMM) error {
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

	if amm.MarkPrice().GT(snapshotUpperLimit) || amm.MarkPrice().LT(snapshotLowerLimit) {
		return types.ErrOverFluctuationLimit.Wrapf("candidate mark price %s is not within the fluctuation limit [%s, %s]",
			amm.MarkPrice(), snapshotLowerLimit, snapshotUpperLimit)
	}

	return nil
}

// ClosePosition closes a position entirely and transfers the remaining margin back to the user.
// Errors if the position has bad debt.
//
// args:
//   - ctx: the cosmos-sdk context
//   - pair: the pair of the position
//   - traderAddr: the address of the trader
//
// returns:
//   - positionResp: response object containing information about the position change
//   - err: error if any
func (k Keeper) ClosePosition(ctx sdk.Context, pair asset.Pair, traderAddr sdk.AccAddress) (*types.PositionResp, error) {
	position, err := k.Positions.Get(ctx, collections.Join(pair, traderAddr))
	if err != nil {
		return nil, err
	}

	market, err := k.Markets.Get(ctx, pair)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", types.ErrPairNotFound, pair)
	}

	amm, err := k.AMMs.Get(ctx, pair)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", types.ErrPairNotFound, pair)
	}

	updatedAMM, positionResp, err := k.closePositionEntirely(
		ctx,
		market,
		amm,
		position,
		/* quoteAssetAmountLimit */ sdk.ZeroDec(),
	)
	if err != nil {
		return nil, err
	}

	if positionResp.BadDebt.IsPositive() {
		if err = k.realizeBadDebt(
			ctx,
			market,
			positionResp.BadDebt.RoundInt(),
		); err != nil {
			return nil, err
		}

		return positionResp, nil
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

// Closes a position and realizes PnL and funding payments.
// Does not error out if there is bad debt, that is for callers to decide.
//
// args:
//   - ctx: cosmos-sdk context
//   - market: the perp market
//   - amm: the amm reserves
//   - currentPosition: the existing position
//   - quoteAssetAmountLimit: the user-specified limit on the quote asset reserves
//   - skipFluctuationLimitCheck: whether to skip the fluctuation check
//
// returns:
//   - updatedAMM: updated AMM reserves
//   - positionResp: response object containing information about the position change
//   - err: error
func (k Keeper) closePositionEntirely(
	ctx sdk.Context,
	market types.Market,
	amm types.AMM,
	currentPosition types.Position,
	quoteAssetAmountLimit sdk.Dec,
) (updatedAMM *types.AMM, positionResp *types.PositionResp, err error) {
	if currentPosition.Size_.IsZero() {
		return nil, nil, fmt.Errorf("zero position size")
	}

	trader, err := sdk.AccAddressFromBech32(currentPosition.TraderAddress)
	if err != nil {
		return nil, nil, err
	}

	positionResp = &types.PositionResp{
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

	var sideToTake types.Direction
	// flipped since we are going against the current position
	if currentPosition.Size_.IsPositive() {
		sideToTake = types.Direction_SHORT
	} else {
		sideToTake = types.Direction_LONG
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
	positionResp.Position = types.Position{
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
