package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/collections"

	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/perp/v2/types"
)

func (k Keeper) MultiLiquidate(
	ctx sdk.Context, liquidator sdk.AccAddress, liquidationRequests []*types.MsgMultiLiquidate_Liquidation,
) ([]*types.MsgMultiLiquidateResponse_LiquidationResponse, error) {
	resp := make([]*types.MsgMultiLiquidateResponse_LiquidationResponse, len(liquidationRequests))

	var allFailed bool = true

	for i, req := range liquidationRequests {
		traderAddr := sdk.MustAccAddressFromBech32(req.Trader)
		cachedCtx, commit := ctx.CacheContext()
		liquidatorFee, perpEfFee, err := k.liquidate(cachedCtx, liquidator, req.Pair, traderAddr)

		if err != nil {
			resp[i] = &types.MsgMultiLiquidateResponse_LiquidationResponse{
				Success: false,
				Error:   err.Error(),
			}
		} else {
			allFailed = false
			resp[i] = &types.MsgMultiLiquidateResponse_LiquidationResponse{
				Success:       true,
				LiquidatorFee: liquidatorFee,
				PerpEfFee:     perpEfFee,
			}

			ctx.EventManager().EmitEvents(cachedCtx.EventManager().Events())
			commit()
		}
	}

	if allFailed {
		return resp, types.ErrAllLiquidationsFailed.Wrapf("%d liquidations failed", len(liquidationRequests))
	}

	return resp, nil
}

/*
	liquidate allows to liquidate the trader position if the margin is below the

required margin maintenance ratio.

args:
  - liquidator: the liquidator who is executing the liquidation
  - pair: the asset pair
  - trader: the trader who owns the position being liquidated

ret:
  - liquidatorFee: the amount of coins given to the liquidator
  - perpEcosystemFundFee: the amount of coins given to the ecosystem fund
  - err: error
*/
func (k Keeper) liquidate(
	ctx sdk.Context,
	liquidator sdk.AccAddress,
	pair asset.Pair,
	trader sdk.AccAddress,
) (liquidatorFee sdk.Coin, ecosystemFundFee sdk.Coin, err error) {
	market, err := k.Markets.Get(ctx, pair)
	if err != nil {
		_ = ctx.EventManager().EmitTypedEvent(&types.LiquidationFailedEvent{
			Pair:       pair,
			Trader:     trader.String(),
			Liquidator: liquidator.String(),
			Reason:     types.LiquidationFailedEvent_NONEXISTENT_PAIR,
		})
		return sdk.Coin{}, sdk.Coin{}, types.ErrPairNotFound
	}

	amm, err := k.AMMs.Get(ctx, pair)
	if err != nil {
		_ = ctx.EventManager().EmitTypedEvent(&types.LiquidationFailedEvent{
			Pair:       pair,
			Trader:     trader.String(),
			Liquidator: liquidator.String(),
			Reason:     types.LiquidationFailedEvent_NONEXISTENT_PAIR,
		})
		return sdk.Coin{}, sdk.Coin{}, types.ErrPairNotFound
	}

	position, err := k.Positions.Get(ctx, collections.Join(pair, trader))
	if err != nil {
		_ = ctx.EventManager().EmitTypedEvent(&types.LiquidationFailedEvent{
			Pair:       pair,
			Trader:     trader.String(),
			Liquidator: liquidator.String(),
			Reason:     types.LiquidationFailedEvent_NONEXISTENT_POSITION,
		})
		return
	}

	spotNotional, err := PositionNotionalSpot(amm, position)
	if err != nil {
		return
	}
	twapNotional, err := k.PositionNotionalTWAP(ctx, position, market.TwapLookbackWindow)
	if err != nil {
		return
	}

	// give the user the preferred position notional
	var preferredPositionNotional sdk.Dec
	if position.Size_.IsPositive() {
		preferredPositionNotional = sdk.MaxDec(spotNotional, twapNotional)
	} else {
		preferredPositionNotional = sdk.MinDec(spotNotional, twapNotional)
	}

	marginRatio := MarginRatio(position, preferredPositionNotional, market.LatestCumulativePremiumFraction)
	if marginRatio.GTE(market.MaintenanceMarginRatio) {
		_ = ctx.EventManager().EmitTypedEvent(&types.LiquidationFailedEvent{
			Pair:       pair,
			Trader:     trader.String(),
			Liquidator: liquidator.String(),
			Reason:     types.LiquidationFailedEvent_POSITION_HEALTHY,
		})
		return sdk.Coin{}, sdk.Coin{}, types.ErrPositionHealthy
	}

	spotMarginRatio := MarginRatio(position, spotNotional, market.LatestCumulativePremiumFraction)
	var liquidationResponse types.LiquidateResp
	if spotMarginRatio.GTE(market.LiquidationFeeRatio) {
		liquidationResponse, err = k.executePartialLiquidation(ctx, market, amm, liquidator, &position)
	} else {
		liquidationResponse, err = k.executeFullLiquidation(ctx, market, amm, liquidator, &position)
	}
	if err != nil {
		return
	}

	liquidatorFee = sdk.NewCoin(
		pair.QuoteDenom(),
		liquidationResponse.FeeToLiquidator,
	)

	ecosystemFundFee = sdk.NewCoin(
		pair.QuoteDenom(),
		liquidationResponse.FeeToPerpEcosystemFund,
	)

	return liquidatorFee, ecosystemFundFee, nil
}

/*
executeFullLiquidation Fully liquidates a position. It is assumed that the margin ratio has already been
checked prior to calling this method.

args:
  - ctx: cosmos-sdk context
  - liquidator: the liquidator's address
  - position: the position to liquidate

ret:
  - liquidationResp: a response object containing the results of the liquidation
  - err: error
*/
func (k Keeper) executeFullLiquidation(
	ctx sdk.Context, market types.Market, amm types.AMM, liquidator sdk.AccAddress, position *types.Position,
) (liquidationResp types.LiquidateResp, err error) {
	traderAddr, err := sdk.AccAddressFromBech32(position.TraderAddress)
	if err != nil {
		return types.LiquidateResp{}, err
	}

	_, positionResp, err := k.closePositionEntirely(
		ctx,
		market,
		amm,
		/* currentPosition */ *position,
		/* quoteAssetAmountLimit */ sdk.ZeroDec(),
	)
	if err != nil {
		return types.LiquidateResp{}, err
	}

	remainMargin := positionResp.MarginToVault.Abs()

	feeToLiquidator := market.LiquidationFeeRatio.
		Mul(positionResp.ExchangedNotionalValue).
		QuoInt64(2)
	totalBadDebt := positionResp.BadDebt

	if feeToLiquidator.GT(remainMargin) {
		// if the remainMargin is not enough for liquidationFee, count it as bad debt
		totalBadDebt = totalBadDebt.Add(feeToLiquidator.Sub(remainMargin))
		remainMargin = sdk.ZeroDec()
	} else {
		// Otherwise, the remaining margin will be transferred to ecosystemFund
		remainMargin = remainMargin.Sub(feeToLiquidator)
	}

	// Realize bad debt
	if totalBadDebt.IsPositive() {
		if err = k.realizeBadDebt(
			ctx,
			market,
			totalBadDebt.RoundInt(),
		); err != nil {
			return types.LiquidateResp{}, err
		}
	}

	feeToPerpEcosystemFund := sdk.ZeroDec()
	if remainMargin.IsPositive() {
		feeToPerpEcosystemFund = remainMargin
	}

	liquidationResp = types.LiquidateResp{
		BadDebt:                totalBadDebt.RoundInt(),
		FeeToLiquidator:        feeToLiquidator.RoundInt(),
		FeeToPerpEcosystemFund: feeToPerpEcosystemFund.RoundInt(),
		Liquidator:             liquidator.String(),
		PositionResp:           positionResp,
	}
	err = k.distributeLiquidateRewards(ctx, market, liquidationResp)
	if err != nil {
		return types.LiquidateResp{}, err
	}

	_ = ctx.EventManager().EmitTypedEvent(&types.PositionLiquidatedEvent{
		Pair:                  position.Pair,
		TraderAddress:         traderAddr.String(),
		ExchangedQuoteAmount:  positionResp.ExchangedNotionalValue,
		ExchangedPositionSize: positionResp.ExchangedPositionSize,
		LiquidatorAddress:     liquidator.String(),
		FeeToLiquidator:       sdk.NewCoin(position.Pair.QuoteDenom(), feeToLiquidator.RoundInt()),
		FeeToEcosystemFund:    sdk.NewCoin(position.Pair.QuoteDenom(), feeToPerpEcosystemFund.RoundInt()),
		BadDebt:               sdk.NewCoin(position.Pair.QuoteDenom(), totalBadDebt.RoundInt()),
		Margin:                sdk.NewCoin(position.Pair.QuoteDenom(), liquidationResp.PositionResp.Position.Margin.RoundInt()),
		PositionNotional:      liquidationResp.PositionResp.PositionNotional,
		PositionSize:          liquidationResp.PositionResp.Position.Size_,
		UnrealizedPnl:         liquidationResp.PositionResp.UnrealizedPnlAfter,
		MarkPrice:             amm.MarkPrice(),
		BlockHeight:           ctx.BlockHeight(),
		BlockTimeMs:           ctx.BlockTime().UnixMilli(),
	})

	return liquidationResp, err
}

// executePartialLiquidation partially liquidates a position
func (k Keeper) executePartialLiquidation(
	ctx sdk.Context, market types.Market, amm types.AMM, liquidator sdk.AccAddress, currentPosition *types.Position,
) (types.LiquidateResp, error) {
	traderAddr, err := sdk.AccAddressFromBech32(currentPosition.TraderAddress)
	if err != nil {
		return types.LiquidateResp{}, err
	}

	var dir types.Direction
	if currentPosition.Size_.IsPositive() {
		dir = types.Direction_SHORT
	} else {
		dir = types.Direction_LONG
	}

	quoteReserveDelta, err := amm.GetQuoteReserveAmt(currentPosition.Size_.Mul(market.PartialLiquidationRatio), dir)
	if err != nil {
		return types.LiquidateResp{}, err
	}
	quoteAssetDelta := amm.FromQuoteReserveToAsset(quoteReserveDelta)

	_, positionResp, err := k.decreasePosition(
		/* ctx */ ctx,
		market,
		amm,
		/* currentPosition */ *currentPosition,
		/* quoteAssetAmount */ quoteAssetDelta,
		/* baseAmtLimit */ sdk.ZeroDec(),
		/* skipFluctuationLimitCheck */ true,
	)
	if err != nil {
		return types.LiquidateResp{}, err
	}

	// Remove the liquidation fee from the margin of the position
	liquidationFeeAmount := quoteAssetDelta.Mul(market.LiquidationFeeRatio)
	positionResp.Position.Margin = positionResp.Position.Margin.Sub(liquidationFeeAmount)
	k.Positions.Insert(ctx, collections.Join(positionResp.Position.Pair, traderAddr), *positionResp.Position)

	// Compute splits for the liquidation fee
	feeToLiquidator := liquidationFeeAmount.QuoInt64(2)
	feeToPerpEcosystemFund := liquidationFeeAmount.Sub(feeToLiquidator)

	liquidationResponse := types.LiquidateResp{
		BadDebt:                sdk.ZeroInt(),
		FeeToLiquidator:        feeToLiquidator.RoundInt(),
		FeeToPerpEcosystemFund: feeToPerpEcosystemFund.RoundInt(),
		Liquidator:             liquidator.String(),
		PositionResp:           positionResp,
	}
	err = k.distributeLiquidateRewards(ctx, market, liquidationResponse)
	if err != nil {
		return types.LiquidateResp{}, err
	}

	_ = ctx.EventManager().EmitTypedEvent(&types.PositionLiquidatedEvent{
		Pair:                  currentPosition.Pair,
		TraderAddress:         traderAddr.String(),
		ExchangedQuoteAmount:  positionResp.ExchangedNotionalValue,
		ExchangedPositionSize: positionResp.ExchangedPositionSize,
		LiquidatorAddress:     liquidator.String(),
		FeeToLiquidator:       sdk.NewCoin(currentPosition.Pair.QuoteDenom(), feeToLiquidator.RoundInt()),
		FeeToEcosystemFund:    sdk.NewCoin(currentPosition.Pair.QuoteDenom(), feeToPerpEcosystemFund.RoundInt()),
		BadDebt:               sdk.NewCoin(currentPosition.Pair.QuoteDenom(), liquidationResponse.BadDebt),
		Margin:                sdk.NewCoin(currentPosition.Pair.QuoteDenom(), liquidationResponse.PositionResp.Position.Margin.RoundInt()),
		PositionNotional:      liquidationResponse.PositionResp.PositionNotional,
		PositionSize:          liquidationResponse.PositionResp.Position.Size_,
		UnrealizedPnl:         liquidationResponse.PositionResp.UnrealizedPnlAfter,
		MarkPrice:             amm.MarkPrice(),
		BlockHeight:           ctx.BlockHeight(),
		BlockTimeMs:           ctx.BlockTime().UnixMilli(),
	})

	return liquidationResponse, err
}

func (k Keeper) distributeLiquidateRewards(
	ctx sdk.Context, market types.Market, liquidateResp types.LiquidateResp) (err error) {
	// --------------------------------------------------------------
	//  Preliminary validations
	// --------------------------------------------------------------

	// validate response
	err = liquidateResp.Validate()
	if err != nil {
		return err
	}

	liquidator, err := sdk.AccAddressFromBech32(liquidateResp.Liquidator)
	if err != nil {
		return err
	}

	// validate pair
	// --------------------------------------------------------------
	// Distribution of rewards
	// --------------------------------------------------------------

	// Transfer fee from vault to PerpEF
	feeToPerpEF := liquidateResp.FeeToPerpEcosystemFund
	if feeToPerpEF.IsPositive() {
		coinToPerpEF := sdk.NewCoin(market.Pair.QuoteDenom(), feeToPerpEF)
		if err = k.BankKeeper.SendCoinsFromModuleToModule(
			ctx,
			/* from */ types.VaultModuleAccount,
			/* to */ types.PerpEFModuleAccount,
			sdk.NewCoins(coinToPerpEF),
		); err != nil {
			return err
		}
	}

	// Transfer fee from vault to liquidator
	feeToLiquidator := liquidateResp.FeeToLiquidator
	if feeToLiquidator.IsPositive() {
		err = k.Withdraw(ctx, market, liquidator, feeToLiquidator)
		if err != nil {
			return err
		}
	}

	return nil
}
