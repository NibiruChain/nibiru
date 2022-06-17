package keeper

import (
	"errors"
	"fmt"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/perp/types"
	vpooltypes "github.com/NibiruChain/nibiru/x/vpool/types"
)

// TODO test: OpenPosition | https://github.com/NibiruChain/nibiru/issues/299
func (k Keeper) OpenPosition(
	ctx sdk.Context,
	pair common.AssetPair,
	side types.Side,
	traderAddr sdk.AccAddress,
	quoteAssetAmount sdk.Int,
	leverage sdk.Dec,
	baseAssetAmountLimit sdk.Dec,
) (err error) {
	ctx.BlockTime().UnixMilli()
	err = k.requireVpool(ctx, pair)
	if err != nil {
		return err
	}
	// require params
	params := k.GetParams(ctx)
	// TODO: missing checks

	position, err := k.GetPosition(ctx, pair, traderAddr)
	var isNewPosition bool = errors.Is(err, types.ErrPositionNotFound)
	if isNewPosition {
		position = types.ZeroPosition(ctx, pair, traderAddr)
		k.SetPosition(ctx, pair, traderAddr, position)
	} else if err != nil && !isNewPosition {
		return err
	}

	var positionResp *types.PositionResp
	sameSideLong := position.Size_.IsPositive() && side == types.Side_BUY
	sameSideShort := position.Size_.IsNegative() && side == types.Side_SELL
	var openSideMatchesPosition = sameSideLong || sameSideShort
	switch {
	case isNewPosition || openSideMatchesPosition:
		// increase position case

		positionResp, err = k.increasePosition(
			ctx,
			*position,
			side,
			/* openNotional */ leverage.MulInt(quoteAssetAmount),
			/* minPositionSize */ baseAssetAmountLimit,
			/* leverage */ leverage)
		if err != nil {
			return err
		}

	// everything else decreases the position
	default:
		positionResp, err = k.openReversePosition(
			ctx,
			*position,
			/* quoteAssetAmount */ quoteAssetAmount.ToDec(),
			/* leverage */ leverage,
			/* baseAssetAmountLimit */ baseAssetAmountLimit,
			/* canOverFluctuationLimit */ false,
		)
		if err != nil {
			return err
		}
	}

	return k.afterPositionUpdate(ctx, pair, traderAddr, params, isNewPosition, *positionResp)
}

// afterPositionUpdate is called when a position has been updated.
func (k Keeper) afterPositionUpdate(
	ctx sdk.Context,
	pair common.AssetPair,
	traderAddr sdk.AccAddress,
	params types.Params,
	isNewPosition bool,
	positionResp types.PositionResp,
) (err error) {
	// update position in state
	k.SetPosition(ctx, pair, traderAddr, positionResp.Position)

	if !isNewPosition && !positionResp.Position.Size_.IsZero() {
		marginRatio, err := k.GetMarginRatio(
			ctx,
			*positionResp.Position,
			types.MarginCalculationPriceOption_MAX_PNL,
		)
		if err != nil {
			return err
		}
		if err = requireMoreMarginRatio(marginRatio, params.MaintenanceMarginRatio, true); err != nil {
			return err
		}
	}

	if !positionResp.BadDebt.IsZero() {
		return fmt.Errorf(
			"bad debt must be zero to prevent attacker from leveraging it")
	}

	// transfer trader <=> vault
	marginToVaultInt := positionResp.MarginToVault.RoundInt()
	switch {
	case marginToVaultInt.IsPositive():
		coinToSend := sdk.NewCoin(pair.GetQuoteTokenDenom(), marginToVaultInt)
		if err = k.BankKeeper.SendCoinsFromAccountToModule(
			ctx, traderAddr, types.VaultModuleAccount, sdk.NewCoins(coinToSend)); err != nil {
			return err
		}
	case marginToVaultInt.IsNegative():
		coinToSend := sdk.NewCoin(pair.GetQuoteTokenDenom(), marginToVaultInt.Abs())
		if err = k.BankKeeper.SendCoinsFromModuleToAccount(
			ctx, types.VaultModuleAccount, traderAddr, sdk.NewCoins(coinToSend)); err != nil {
			return err
		}
	}

	transferredFee, err := k.transferFee(
		ctx, pair, traderAddr, positionResp.ExchangedQuoteAssetAmount)
	if err != nil {
		return err
	}

	spotPrice, err := k.VpoolKeeper.GetSpotPrice(ctx, pair)
	if err != nil {
		return err
	}

	// calculate positionNotional (it's different depends on long or short side)
	// long: unrealizedPnl = positionNotional - openNotional => positionNotional = openNotional + unrealizedPnl
	// short: unrealizedPnl = openNotional - positionNotional => positionNotional = openNotional - unrealizedPnl
	var positionNotional sdk.Dec = sdk.ZeroDec()
	if positionResp.Position.Size_.IsPositive() {
		positionNotional = positionResp.Position.OpenNotional.Add(positionResp.UnrealizedPnlAfter)
	} else if positionResp.Position.Size_.IsNegative() {
		positionNotional = positionResp.Position.OpenNotional.Sub(positionResp.UnrealizedPnlAfter)
	}

	return ctx.EventManager().EmitTypedEvent(&types.PositionChangedEvent{
		TraderAddress:         traderAddr.String(),
		Pair:                  pair.String(),
		Margin:                sdk.NewCoin(pair.GetQuoteTokenDenom(), positionResp.Position.Margin.RoundInt()),
		PositionNotional:      positionNotional,
		ExchangedPositionSize: positionResp.ExchangedPositionSize,
		TransactionFee:        sdk.NewCoin(pair.GetQuoteTokenDenom(), transferredFee),
		PositionSize:          positionResp.Position.Size_,
		RealizedPnl:           positionResp.RealizedPnl,
		UnrealizedPnlAfter:    positionResp.UnrealizedPnlAfter,
		BadDebt:               positionResp.BadDebt,
		LiquidationPenalty:    sdk.ZeroDec(),
		SpotPrice:             spotPrice,
		FundingPayment:        positionResp.FundingPayment,
		BlockHeight:           ctx.BlockHeight(),
		BlockTimeMs:           ctx.BlockTime().UnixMilli(),
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
  - baseAssetAmountLimit: the limit on the base asset amount to make sure the trader doesn't get screwed, in base asset units
  - leverage: the amount of leverage to take, as sdk.Dec

ret:
  - positionResp: contains the result of the increase position and the new position
  - err: error
*/
func (k Keeper) increasePosition(
	ctx sdk.Context,
	currentPosition types.Position,
	side types.Side,
	increasedNotional sdk.Dec,
	baseAssetAmountLimit sdk.Dec,
	leverage sdk.Dec,
) (positionResp *types.PositionResp, err error) {
	positionResp = &types.PositionResp{}

	positionResp.ExchangedPositionSize, err = k.swapQuoteForBase(
		ctx,
		currentPosition.GetAssetPair(),
		side,
		increasedNotional,
		baseAssetAmountLimit,
		false,
	)
	if err != nil {
		return nil, err
	}

	increaseMarginRequirement := increasedNotional.Quo(leverage)

	remaining, err := k.CalcRemainMarginWithFundingPayment(
		ctx,
		currentPosition,
		increaseMarginRequirement,
	)
	if err != nil {
		return nil, err
	}

	_, unrealizedPnL, err := k.getPositionNotionalAndUnrealizedPnL(
		ctx,
		currentPosition,
		types.PnLCalcOption_SPOT_PRICE,
	)
	if err != nil {
		return nil, err
	}

	positionResp.ExchangedQuoteAssetAmount = increasedNotional
	positionResp.UnrealizedPnlAfter = unrealizedPnL
	positionResp.RealizedPnl = sdk.ZeroDec()
	positionResp.MarginToVault = increaseMarginRequirement
	positionResp.FundingPayment = remaining.FundingPayment
	positionResp.BadDebt = remaining.BadDebt
	positionResp.Position = &types.Position{
		TraderAddress:                       currentPosition.TraderAddress,
		Pair:                                currentPosition.Pair,
		Size_:                               currentPosition.Size_.Add(positionResp.ExchangedPositionSize),
		Margin:                              remaining.Margin,
		OpenNotional:                        currentPosition.OpenNotional.Add(increasedNotional),
		LastUpdateCumulativePremiumFraction: remaining.LatestCumulativePremiumFraction,
		BlockNumber:                         ctx.BlockHeight(),
	}

	k.Logger(ctx).Debug("increase_position",
		"positionResp",
		positionResp.String(),
	)

	return positionResp, nil
}

// getLatestCumulativePremiumFraction returns the last cumulative premium fraction recorded for the
// specific pair.
func (k Keeper) getLatestCumulativePremiumFraction(
	ctx sdk.Context, pair common.AssetPair,
) (sdk.Dec, error) {
	pairMetadata, err := k.PairMetadata().Get(ctx, pair)
	if err != nil {
		k.Logger(ctx).Error(
			err.Error(),
			"pair",
			pair.String(),
		)
		return sdk.Dec{}, err
	}
	// this should never fail
	return pairMetadata.CumulativePremiumFractions[len(pairMetadata.CumulativePremiumFractions)-1], nil
}

/*
Calculates position notional value and unrealized PnL. Lets the caller pick
either spot price, TWAP, or ORACLE to use for calculation.

args:
  - ctx: cosmos-sdk context
  - position: the trader's position
  - pnlCalcOption: SPOT or TWAP or ORACLE

Returns:
  - positionNotional: the position's notional value as sdk.Dec (signed)
  - unrealizedPnl: the position's unrealized profits and losses (PnL) as sdk.Dec (signed)
		For LONG positions, this is positionNotional - openNotional
		For SHORT positions, this is openNotional - positionNotional
*/
func (k Keeper) getPositionNotionalAndUnrealizedPnL(
	ctx sdk.Context,
	currentPosition types.Position,
	pnlCalcOption types.PnLCalcOption,
) (positionNotional sdk.Dec, unrealizedPnL sdk.Dec, err error) {
	positionSizeAbs := currentPosition.Size_.Abs()
	if positionSizeAbs.IsZero() {
		return sdk.ZeroDec(), sdk.ZeroDec(), nil
	}

	var baseAssetDirection vpooltypes.Direction
	if currentPosition.Size_.IsPositive() {
		// LONG
		baseAssetDirection = vpooltypes.Direction_ADD_TO_POOL
	} else {
		// SHORT
		baseAssetDirection = vpooltypes.Direction_REMOVE_FROM_POOL
	}

	switch pnlCalcOption {
	case types.PnLCalcOption_TWAP:
		positionNotional, err = k.VpoolKeeper.GetBaseAssetTWAP(
			ctx,
			currentPosition.GetAssetPair(),
			baseAssetDirection,
			positionSizeAbs,
			/*lookbackInterval=*/ 15*time.Minute,
		)
		if err != nil {
			k.Logger(ctx).Error(err.Error(), "calc_option", pnlCalcOption.String())
			return sdk.ZeroDec(), sdk.ZeroDec(), err
		}
	case types.PnLCalcOption_SPOT_PRICE:
		positionNotional, err = k.VpoolKeeper.GetBaseAssetPrice(
			ctx,
			currentPosition.GetAssetPair(),
			baseAssetDirection,
			positionSizeAbs,
		)
		if err != nil {
			k.Logger(ctx).Error(err.Error(), "calc_option", pnlCalcOption.String())
			return sdk.ZeroDec(), sdk.ZeroDec(), err
		}
	case types.PnLCalcOption_ORACLE:
		oraclePrice, err := k.VpoolKeeper.GetUnderlyingPrice(
			ctx, currentPosition.GetAssetPair())
		if err != nil {
			k.Logger(ctx).Error(err.Error(), "calc_option", pnlCalcOption.String())
			return sdk.ZeroDec(), sdk.ZeroDec(), err
		}
		positionNotional = oraclePrice.Mul(positionSizeAbs)
	default:
		panic("unrecognized pnl calc option: " + pnlCalcOption.String())
	}

	if positionNotional.Equal(currentPosition.OpenNotional) {
		// if position notional and open notional are the same, then early return
		return positionNotional, sdk.ZeroDec(), nil
	}

	if currentPosition.Size_.IsPositive() {
		// LONG
		unrealizedPnL = positionNotional.Sub(currentPosition.OpenNotional)
	} else {
		// SHORT
		unrealizedPnL = currentPosition.OpenNotional.Sub(positionNotional)
	}

	k.Logger(ctx).Debug("get_position_notional_and_unrealized_pnl",
		"position",
		currentPosition.String(),
		"position_notional",
		positionNotional.String(),
		"unrealized_pnl",
		unrealizedPnL.String(),
	)

	return positionNotional, unrealizedPnL, nil
}

// TODO test: openReversePosition | https://github.com/NibiruChain/nibiru/issues/299
func (k Keeper) openReversePosition(
	ctx sdk.Context,
	currentPosition types.Position,
	quoteAssetAmount sdk.Dec,
	leverage sdk.Dec,
	baseAssetAmountLimit sdk.Dec,
	canOverFluctuationLimit bool,
) (positionResp *types.PositionResp, err error) {
	notionalToDecreaseBy := leverage.Mul(quoteAssetAmount)
	currentPositionNotional, _, err := k.getPositionNotionalAndUnrealizedPnL(
		ctx,
		currentPosition,
		types.PnLCalcOption_SPOT_PRICE,
	)
	if err != nil {
		return nil, err
	}

	if currentPositionNotional.GT(notionalToDecreaseBy) {
		// position reduction
		return k.decreasePosition(
			ctx,
			currentPosition,
			notionalToDecreaseBy,
			baseAssetAmountLimit,
			canOverFluctuationLimit,
		)
	} else {
		// close and reverse
		return k.closeAndOpenReversePosition(
			ctx,
			currentPosition,
			quoteAssetAmount,
			leverage,
			baseAssetAmountLimit,
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
  - baseAssetAmountLimit: the limit on the base asset amount to make sure the trader doesn't get screwed, in base asset units
  - canOverFluctuationLimit: whether or not the position change can go over the fluctuation limit

ret:
  - positionResp: contains the result of the decrease position and the new position
  - err: error
*/
// TODO(https://github.com/NibiruChain/nibiru/issues/403): implement fluctuation limit check
func (k Keeper) decreasePosition(
	ctx sdk.Context,
	currentPosition types.Position,
	decreasedNotional sdk.Dec,
	baseAssetAmountLimit sdk.Dec,
	canOverFluctuationLimit bool,
) (positionResp *types.PositionResp, err error) {
	positionResp = &types.PositionResp{
		RealizedPnl:   sdk.ZeroDec(),
		MarginToVault: sdk.ZeroDec(),
	}

	currentPositionNotional, currentUnrealizedPnL, err := k.
		getPositionNotionalAndUnrealizedPnL(
			ctx,
			currentPosition,
			types.PnLCalcOption_SPOT_PRICE,
		)
	if err != nil {
		return nil, err
	}

	var sideToTake types.Side
	// flipped since we are going against the current position
	if currentPosition.Size_.IsPositive() {
		sideToTake = types.Side_SELL
	} else {
		sideToTake = types.Side_BUY
	}

	positionResp.ExchangedPositionSize, err = k.swapQuoteForBase(
		ctx,
		currentPosition.GetAssetPair(),
		sideToTake,
		decreasedNotional,
		baseAssetAmountLimit,
		canOverFluctuationLimit,
	)
	if err != nil {
		return nil, err
	}

	if !currentPosition.Size_.IsZero() {
		positionResp.RealizedPnl = currentUnrealizedPnL.Mul(
			positionResp.ExchangedPositionSize.Abs().
				Quo(currentPosition.Size_.Abs()),
		)
	}

	remaining, err := k.CalcRemainMarginWithFundingPayment(
		ctx,
		currentPosition,
		positionResp.RealizedPnl,
	)
	if err != nil {
		return nil, err
	}

	positionResp.BadDebt = remaining.BadDebt
	positionResp.FundingPayment = remaining.FundingPayment
	positionResp.UnrealizedPnlAfter = currentUnrealizedPnL.Sub(positionResp.RealizedPnl)
	positionResp.ExchangedQuoteAssetAmount = decreasedNotional

	// calculate openNotional (it's different depends on long or short side)
	// long: unrealizedPnl = positionNotional - openNotional => openNotional = positionNotional - unrealizedPnl
	// short: unrealizedPnl = openNotional - positionNotional => openNotional = positionNotional + unrealizedPnl
	// positionNotional = oldPositionNotional - notionalValueToDecrease
	var remainOpenNotional sdk.Dec
	if currentPosition.Size_.IsPositive() {
		remainOpenNotional = currentPositionNotional.
			Sub(decreasedNotional).
			Sub(positionResp.UnrealizedPnlAfter)
	} else {
		remainOpenNotional = currentPositionNotional.
			Sub(decreasedNotional).
			Add(positionResp.UnrealizedPnlAfter)
	}

	if remainOpenNotional.IsNegative() {
		panic("value of open notional < 0")
	}

	positionResp.Position = &types.Position{
		TraderAddress:                       currentPosition.TraderAddress,
		Pair:                                currentPosition.Pair,
		Size_:                               currentPosition.Size_.Add(positionResp.ExchangedPositionSize),
		Margin:                              remaining.Margin,
		OpenNotional:                        remainOpenNotional,
		LastUpdateCumulativePremiumFraction: remaining.LatestCumulativePremiumFraction,
		BlockNumber:                         ctx.BlockHeight(),
	}

	k.Logger(ctx).Debug("decrease_position",
		"positionResp",
		positionResp.String(),
	)

	return positionResp, nil
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
  - baseAssetAmountLimit: limit on the base asset movement to ensure trader doesn't get screwed

ret:
  - positionResp: response object containing information about the position change
  - err: error
*/
func (k Keeper) closeAndOpenReversePosition(
	ctx sdk.Context,
	existingPosition types.Position,
	quoteAssetAmount sdk.Dec,
	leverage sdk.Dec,
	baseAssetAmountLimit sdk.Dec,
) (positionResp *types.PositionResp, err error) {
	trader, err := sdk.AccAddressFromBech32(existingPosition.TraderAddress)
	if err != nil {
		return nil, err
	}

	closePositionResp, err := k.closePositionEntirely(
		ctx,
		existingPosition,
		sdk.ZeroDec(),
	)
	if err != nil {
		return nil, err
	}

	if closePositionResp.BadDebt.IsPositive() {
		return nil, fmt.Errorf("underwater position")
	}

	reverseNotionalValue := leverage.Mul(quoteAssetAmount)
	remainingReverseNotionalValue := reverseNotionalValue.Sub(
		closePositionResp.ExchangedQuoteAssetAmount)

	if remainingReverseNotionalValue.IsNegative() {
		// should never happen as this should also be checked in the caller
		return nil, fmt.Errorf(
			"provided quote asset amount and leverage not large enough to close position. need %s but got %s",
			closePositionResp.ExchangedQuoteAssetAmount.String(), reverseNotionalValue.String())
	} else if remainingReverseNotionalValue.IsPositive() {
		updatedBaseAssetAmountLimit := baseAssetAmountLimit
		if baseAssetAmountLimit.IsPositive() {
			updatedBaseAssetAmountLimit = baseAssetAmountLimit.
				Sub(closePositionResp.ExchangedPositionSize.Abs())
		}
		if updatedBaseAssetAmountLimit.IsNegative() {
			return nil, fmt.Errorf(
				"position size changed by greater than the specified base limit: %s",
				baseAssetAmountLimit.String(),
			)
		}

		var sideToTake types.Side
		// flipped since we are going against the current position
		if existingPosition.Size_.IsPositive() {
			sideToTake = types.Side_SELL
		} else {
			sideToTake = types.Side_BUY
		}

		newPosition := types.ZeroPosition(
			ctx,
			existingPosition.GetAssetPair(),
			trader,
		)
		increasePositionResp, err := k.increasePosition(
			ctx,
			*newPosition,
			sideToTake,
			remainingReverseNotionalValue,
			updatedBaseAssetAmountLimit,
			leverage,
		)
		if err != nil {
			return nil, err
		}
		positionResp = &types.PositionResp{
			Position:                  increasePositionResp.Position,
			ExchangedQuoteAssetAmount: closePositionResp.ExchangedQuoteAssetAmount.Add(increasePositionResp.ExchangedQuoteAssetAmount),
			BadDebt:                   closePositionResp.BadDebt.Add(increasePositionResp.BadDebt),
			ExchangedPositionSize:     closePositionResp.ExchangedPositionSize.Add(increasePositionResp.ExchangedPositionSize),
			FundingPayment:            closePositionResp.FundingPayment.Add(increasePositionResp.FundingPayment),
			RealizedPnl:               closePositionResp.RealizedPnl.Add(increasePositionResp.RealizedPnl),
			MarginToVault:             closePositionResp.MarginToVault.Add(increasePositionResp.MarginToVault),
			UnrealizedPnlAfter:        sdk.ZeroDec(),
		}
	} else {
		// case where remaining open notional == 0
		positionResp = closePositionResp
	}

	k.Logger(ctx).Debug("close_and_open_reverse_position",
		"positionResp",
		positionResp.String(),
	)

	return positionResp, nil
}

/*
Closes a position and realizes PnL and funding payments.
Does not error out if there is bad debt, that is for callers to decide.

args:
  - ctx: cosmos-sdk context
  - currentPosition: current position
  - quoteAssetAmountLimit: a limit on quote asset to ensure trader doesn't get screwed

ret:
  - positionResp: response object containing information about the position change
  - err: error
*/
func (k Keeper) closePositionEntirely(
	ctx sdk.Context,
	currentPosition types.Position,
	quoteAssetAmountLimit sdk.Dec,
) (positionResp *types.PositionResp, err error) {
	if currentPosition.Size_.IsZero() {
		return nil, fmt.Errorf("zero position size")
	}

	trader, err := sdk.AccAddressFromBech32(currentPosition.TraderAddress)
	if err != nil {
		return nil, err
	}

	positionResp = &types.PositionResp{
		UnrealizedPnlAfter:    sdk.ZeroDec(),
		ExchangedPositionSize: currentPosition.Size_.Neg(),
	}

	// calculate unrealized PnL
	_, unrealizedPnL, err := k.getPositionNotionalAndUnrealizedPnL(
		ctx,
		currentPosition,
		types.PnLCalcOption_SPOT_PRICE,
	)
	if err != nil {
		return nil, err
	}

	positionResp.RealizedPnl = unrealizedPnL

	// calculate remaining margin with funding payment
	remaining, err := k.CalcRemainMarginWithFundingPayment(
		ctx, currentPosition, unrealizedPnL)
	if err != nil {
		return nil, err
	}

	positionResp.BadDebt = remaining.BadDebt
	positionResp.FundingPayment = remaining.FundingPayment
	positionResp.MarginToVault = remaining.Margin.Neg()

	var baseAssetDirection vpooltypes.Direction
	if currentPosition.Size_.IsPositive() {
		baseAssetDirection = vpooltypes.Direction_ADD_TO_POOL
	} else {
		baseAssetDirection = vpooltypes.Direction_REMOVE_FROM_POOL
	}

	exchangedQuoteAssetAmount, err := k.VpoolKeeper.SwapBaseForQuote(
		ctx,
		currentPosition.GetAssetPair(),
		baseAssetDirection,
		currentPosition.Size_.Abs(),
		quoteAssetAmountLimit,
	)
	if err != nil {
		return nil, err
	}

	positionResp.ExchangedQuoteAssetAmount = exchangedQuoteAssetAmount
	positionResp.Position = &types.Position{
		TraderAddress:                       currentPosition.TraderAddress,
		Pair:                                currentPosition.Pair,
		Size_:                               sdk.ZeroDec(),
		Margin:                              sdk.ZeroDec(),
		OpenNotional:                        sdk.ZeroDec(),
		LastUpdateCumulativePremiumFraction: remaining.LatestCumulativePremiumFraction,
		BlockNumber:                         ctx.BlockHeight(),
	}

	if err = k.ClearPosition(
		ctx,
		currentPosition.GetAssetPair(),
		trader,
	); err != nil {
		return nil, err
	}

	k.Logger(ctx).Debug("close_position_entirely",
		"positionResp",
		positionResp.String(),
	)

	return positionResp, nil
}

// ClosePosition gets the current position, and calls OpenPosition to open a reverse position with amount equal to the current open notional.
func (k Keeper) ClosePosition(ctx sdk.Context, pair common.AssetPair, addr sdk.AccAddress) (*types.PositionResp, error) {
	position, err := k.GetPosition(ctx, pair, addr)
	if err != nil {
		return nil, err
	}

	positionNotional, _, err := k.getPositionNotionalAndUnrealizedPnL(ctx, *position, types.PnLCalcOption_SPOT_PRICE)
	if err != nil {
		return nil, err
	}

	positionResp, err := k.openReversePosition(
		ctx,
		*position,
		positionNotional,
		/* leverage */ sdk.OneDec(),
		/* baseLimit */ sdk.ZeroDec(),
		/* canOverFluctuationLimit */ false,
	)
	if err != nil {
		return nil, err
	}

	if err = k.afterPositionUpdate(
		ctx,
		pair,
		addr,
		k.GetParams(ctx),
		/* isNewPosition */ false,
		*positionResp,
	); err != nil {
		return nil, err
	}

	return positionResp, nil
}

// TODO test: transferFee | https://github.com/NibiruChain/nibiru/issues/299
func (k Keeper) transferFee(
	ctx sdk.Context, pair common.AssetPair, trader sdk.AccAddress,
	positionNotional sdk.Dec,
) (sdk.Int, error) {
	toll, spread := k.CalcPerpTxFee(ctx, positionNotional)

	hasToll := toll.IsPositive()
	hasSpread := spread.IsPositive()

	if !hasToll && !hasSpread {
		return sdk.ZeroInt(), nil
	}

	var err error
	if hasSpread {
		spreadCoins := sdk.NewCoins(sdk.NewCoin(pair.GetQuoteTokenDenom(), spread))
		err = k.BankKeeper.SendCoinsFromAccountToModule(
			ctx, trader, types.PerpEFModuleAccount, spreadCoins)
		if err != nil {
			return sdk.Int{}, err
		}
	}
	if hasToll {
		tollCoins := sdk.NewCoins(sdk.NewCoin(pair.GetQuoteTokenDenom(), toll))
		err = k.BankKeeper.SendCoinsFromAccountToModule(
			ctx, trader, types.FeePoolModuleAccount, tollCoins)
		if err != nil {
			return sdk.Int{}, err
		}
	}

	return toll.Add(spread), nil
}

/*
Calculates both position notional value and unrealized PnL based on
both spot price and TWAP, and lets the caller pick which one based on MAX or MIN.

args:
  - ctx: cosmos-sdk context
  - position: the trader's position
  - pnlPreferenceOption: MAX or MIN

Returns:
  - positionNotional: the position's notional value as sdk.Dec (signed)
  - unrealizedPnl: the position's unrealized profits and losses (PnL) as sdk.Dec (signed)
		For LONG positions, this is positionNotional - openNotional
		For SHORT positions, this is openNotional - positionNotional
*/
func (k Keeper) getPreferencePositionNotionalAndUnrealizedPnL(
	ctx sdk.Context,
	position types.Position,
	pnLPreferenceOption types.PnLPreferenceOption,
) (positionNotional sdk.Dec, unrealizedPnl sdk.Dec, err error) {
	spotPositionNotional, spotPricePnl, err := k.getPositionNotionalAndUnrealizedPnL(
		ctx,
		position,
		types.PnLCalcOption_SPOT_PRICE,
	)
	if err != nil {
		k.Logger(ctx).Error(
			err.Error(),
			"calc_option",
			types.PnLCalcOption_SPOT_PRICE.String(),
			"preference_option",
			pnLPreferenceOption.String(),
		)
		return sdk.Dec{}, sdk.Dec{}, err
	}

	twapPositionNotional, twapPricePnL, err := k.getPositionNotionalAndUnrealizedPnL(
		ctx,
		position,
		types.PnLCalcOption_TWAP,
	)
	if err != nil {
		k.Logger(ctx).Error(
			err.Error(),
			"calc_option",
			types.PnLCalcOption_TWAP.String(),
			"preference_option",
			pnLPreferenceOption.String(),
		)
		return sdk.Dec{}, sdk.Dec{}, err
	}

	switch pnLPreferenceOption {
	case types.PnLPreferenceOption_MAX:
		positionNotional = sdk.MaxDec(spotPositionNotional, twapPositionNotional)
		unrealizedPnl = sdk.MaxDec(spotPricePnl, twapPricePnL)
	case types.PnLPreferenceOption_MIN:
		positionNotional = sdk.MinDec(spotPositionNotional, twapPositionNotional)
		unrealizedPnl = sdk.MinDec(spotPricePnl, twapPricePnL)
	default:
		panic("invalid pnl preference option " + pnLPreferenceOption.String())
	}

	return positionNotional, unrealizedPnl, nil
}

/*
Trades quoteAssets in exchange for baseAssets.
The quote asset is a stablecoin like NUSD.
The base asset is a crypto asset like BTC or ETH.

args:
  - ctx: cosmos-sdk context
  - pair: a token pair like BTC:NUSD
  - dir: either add or remove from pool
  - quoteAssetAmount: the amount of quote asset being traded
  - baseAmountLimit: a limiter to ensure the trader doesn't get screwed by slippage
  - canOverFluctuationLimit: whether or not to check if the swapped amount is over the fluctuation limit. Currently unused.

ret:
  - baseAssetAmount: the amount of base asset swapped
  - err: error
*/
func (k Keeper) swapQuoteForBase(
	ctx sdk.Context,
	pair common.AssetPair,
	side types.Side,
	quoteAssetAmount sdk.Dec,
	baseAssetLimit sdk.Dec,
	canOverFluctuationLimit bool,
) (baseAmount sdk.Dec, err error) {
	var quoteAssetDirection vpooltypes.Direction
	if side == types.Side_BUY {
		quoteAssetDirection = vpooltypes.Direction_ADD_TO_POOL
	} else {
		// side == types.Side_SELL
		quoteAssetDirection = vpooltypes.Direction_REMOVE_FROM_POOL
	}

	baseAmount, err = k.VpoolKeeper.SwapQuoteForBase(
		ctx, pair, quoteAssetDirection, quoteAssetAmount, baseAssetLimit)
	if err != nil {
		k.Logger(ctx).Error(
			err.Error(),
			"pair",
			pair.String(),
			"side",
			side.String(),
			"quoteAssetAmount",
			quoteAssetAmount.String(),
			"baseAssetLimit",
			baseAssetLimit.String(),
		)
		return sdk.Dec{}, err
	}

	k.Logger(ctx).Debug("swap_quote_for_base",
		"side",
		side.String(),
		"baseAmt",
		baseAmount.Abs(),
	)

	if side == types.Side_BUY {
		return baseAmount, nil
	} else {
		// side == types.Side_SELL
		return baseAmount.Neg(), nil
	}
}
