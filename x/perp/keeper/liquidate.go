package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/collections"

	"github.com/NibiruChain/nibiru/x/common/asset"
	perpammtypes "github.com/NibiruChain/nibiru/x/perp/amm/types"
	"github.com/NibiruChain/nibiru/x/perp/types"
)

/*
	Liquidate allows to liquidate the trader position if the margin is below the

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
func (k Keeper) Liquidate(
	ctx sdk.Context,
	liquidator sdk.AccAddress,
	pair asset.Pair,
	trader sdk.AccAddress,
) (liquidatorFee sdk.Coin, perpEcosystemFundFee sdk.Coin, err error) {
	vpool, err := k.VpoolKeeper.GetPool(ctx, pair)
	if err != nil {
		return sdk.Coin{}, sdk.Coin{}, types.ErrPairNotFound
	}

	if err != nil {
		_ = ctx.EventManager().EmitTypedEvent(&types.LiquidationFailedEvent{ // nolint:errcheck
			Pair:       pair,
			Trader:     trader.String(),
			Liquidator: liquidator.String(),
			Reason:     types.LiquidationFailedEvent_NONEXISTENT_PAIR,
		})
		return
	}

	position, err := k.Positions.Get(ctx, collections.Join(pair, trader))
	if err != nil {
		_ = ctx.EventManager().EmitTypedEvent(&types.LiquidationFailedEvent{ // nolint:errcheck
			Pair:       pair,
			Trader:     trader.String(),
			Liquidator: liquidator.String(),
			Reason:     types.LiquidationFailedEvent_NONEXISTENT_POSITION,
		})
		return
	}

	marginRatio, err := k.GetMarginRatio(
		ctx,
		vpool,
		position,
		types.MarginCalculationPriceOption_MAX_PNL,
	)
	if err != nil {
		return
	}

	isOverSpreadLimit, err := k.VpoolKeeper.IsOverSpreadLimit(ctx, pair)
	if err != nil {
		return
	}
	if isOverSpreadLimit {
		marginRatioBasedOnOracle, err := k.GetMarginRatio(
			ctx, vpool, position, types.MarginCalculationPriceOption_INDEX)
		if err != nil {
			return liquidatorFee, perpEcosystemFundFee, err
		}

		marginRatio = sdk.MaxDec(marginRatio, marginRatioBasedOnOracle)
	}

	params := k.GetParams(ctx)

	maintenanceMarginRatio, err := k.VpoolKeeper.GetMaintenanceMarginRatio(ctx, pair)
	if err != nil {
		return
	}
	err = validateMarginRatio(marginRatio, maintenanceMarginRatio, false)
	if err != nil {
		_ = ctx.EventManager().EmitTypedEvent(&types.LiquidationFailedEvent{ // nolint:errcheck
			Pair:       pair,
			Trader:     trader.String(),
			Liquidator: liquidator.String(),
			Reason:     types.LiquidationFailedEvent_POSITION_HEALTHY,
		})
		return
	}

	marginRatioBasedOnSpot, err := k.GetMarginRatio(
		ctx, vpool, position, types.MarginCalculationPriceOption_SPOT)
	if err != nil {
		return
	}

	var liquidationResponse types.LiquidateResp
	if marginRatioBasedOnSpot.GTE(params.LiquidationFeeRatio) {
		liquidationResponse, err = k.ExecutePartialLiquidation(ctx, liquidator, &position)
	} else {
		liquidationResponse, err = k.ExecuteFullLiquidation(ctx, liquidator, &position)
	}
	if err != nil {
		return
	}

	liquidatorFee = sdk.NewCoin(
		pair.QuoteDenom(),
		liquidationResponse.FeeToLiquidator,
	)

	perpEcosystemFundFee = sdk.NewCoin(
		pair.QuoteDenom(),
		liquidationResponse.FeeToPerpEcosystemFund,
	)

	return liquidatorFee, perpEcosystemFundFee, nil
}

/*
ExecuteFullLiquidation Fully liquidates a position. It is assumed that the margin ratio has already been
checked prior to calling this method.

args:
  - ctx: cosmos-sdk context
  - liquidator: the liquidator's address
  - position: the position to liquidate

ret:
  - liquidationResp: a response object containing the results of the liquidation
  - err: error
*/
func (k Keeper) ExecuteFullLiquidation(
	ctx sdk.Context, liquidator sdk.AccAddress, position *types.Position,
) (liquidationResp types.LiquidateResp, err error) {
	params := k.GetParams(ctx)

	vpool, err := k.VpoolKeeper.GetPool(ctx, position.Pair)
	if err != nil {
		return types.LiquidateResp{}, types.ErrPairNotFound
	}

	traderAddr, err := sdk.AccAddressFromBech32(position.TraderAddress)
	if err != nil {
		return types.LiquidateResp{}, err
	}

	_, positionResp, err := k.closePositionEntirely(
		ctx,
		vpool,
		/* currentPosition */ *position,
		/* quoteAssetAmountLimit */ sdk.ZeroDec(),
		/* skipFluctuationLimitCheck */ true,
	)
	if err != nil {
		return types.LiquidateResp{}, err
	}

	remainMargin := positionResp.MarginToVault.Abs()

	feeToLiquidator := params.LiquidationFeeRatio.
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
			position.Pair.QuoteDenom(),
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
	err = k.distributeLiquidateRewards(ctx, liquidationResp)
	if err != nil {
		return types.LiquidateResp{}, err
	}

	markPrice, err := k.VpoolKeeper.GetMarkPrice(ctx, position.Pair)
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
		MarkPrice:             markPrice,
		BlockHeight:           ctx.BlockHeight(),
		BlockTimeMs:           ctx.BlockTime().UnixMilli(),
	})

	return liquidationResp, err
}

func (k Keeper) distributeLiquidateRewards(
	ctx sdk.Context, liquidateResp types.LiquidateResp) (err error) {
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
	pair := liquidateResp.PositionResp.Position.Pair
	err = k.requireVpool(ctx, pair)
	if err != nil {
		return err
	}

	// --------------------------------------------------------------
	// Distribution of rewards
	// --------------------------------------------------------------

	// Transfer fee from vault to PerpEF
	feeToPerpEF := liquidateResp.FeeToPerpEcosystemFund
	if feeToPerpEF.IsPositive() {
		coinToPerpEF := sdk.NewCoin(
			pair.QuoteDenom(), feeToPerpEF)
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
		err = k.Withdraw(ctx, pair.QuoteDenom(), liquidator, feeToLiquidator)
		if err != nil {
			return err
		}
	}

	return nil
}

// ExecutePartialLiquidation partially liquidates a position
func (k Keeper) ExecutePartialLiquidation(
	ctx sdk.Context, liquidator sdk.AccAddress, currentPosition *types.Position,
) (types.LiquidateResp, error) {
	params := k.GetParams(ctx)

	vpool, err := k.VpoolKeeper.GetPool(ctx, currentPosition.Pair)
	if err != nil {
		return types.LiquidateResp{}, types.ErrPairNotFound
	}

	traderAddr, err := sdk.AccAddressFromBech32(currentPosition.TraderAddress)
	if err != nil {
		return types.LiquidateResp{}, err
	}

	var baseAssetDir perpammtypes.Direction
	if currentPosition.Size_.IsPositive() {
		baseAssetDir = perpammtypes.Direction_ADD_TO_POOL
	} else {
		baseAssetDir = perpammtypes.Direction_REMOVE_FROM_POOL
	}

	partiallyLiquidatedPositionNotional, err := k.VpoolKeeper.GetBaseAssetPrice(
		vpool,
		baseAssetDir,
		/* abs= */ currentPosition.Size_.Mul(params.PartialLiquidationRatio),
	)
	if err != nil {
		return types.LiquidateResp{}, err
	}

	_, positionResp, err := k.decreasePosition(
		/* ctx */ ctx,
		vpool,
		/* currentPosition */ *currentPosition,
		/* quoteAssetAmount */ partiallyLiquidatedPositionNotional,
		/* baseAmtLimit */ sdk.ZeroDec(),
		/* skipFluctuationLimitCheck */ true,
	)
	if err != nil {
		return types.LiquidateResp{}, err
	}

	// Remove the liquidation fee from the margin of the position
	liquidationFeeAmount := positionResp.ExchangedNotionalValue.
		Mul(params.LiquidationFeeRatio)
	positionResp.Position.Margin = positionResp.Position.Margin.
		Sub(liquidationFeeAmount)
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
	err = k.distributeLiquidateRewards(ctx, liquidationResponse)
	if err != nil {
		return types.LiquidateResp{}, err
	}

	markPrice, err := k.VpoolKeeper.GetMarkPrice(ctx, currentPosition.Pair)
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
		MarkPrice:             markPrice,
		BlockHeight:           ctx.BlockHeight(),
		BlockTimeMs:           ctx.BlockTime().UnixMilli(),
	})

	return liquidationResponse, err
}

func (k Keeper) MultiLiquidate(
	ctx sdk.Context, liquidator sdk.AccAddress, liquidationRequests []*types.MsgMultiLiquidate_Liquidation,
) ([]*types.MsgMultiLiquidateResponse_LiquidationResponse, error) {
	if !k.isWhitelistedLiquidator(ctx, liquidator) {
		return nil, types.ErrUnauthorized.Wrapf("%s is not a whitelisted liquidator", liquidator.String())
	}

	resp := make([]*types.MsgMultiLiquidateResponse_LiquidationResponse, len(liquidationRequests))

	var allFailed bool = true
	for i, req := range liquidationRequests {
		traderAddr := sdk.MustAccAddressFromBech32(req.Trader)
		cachedCtx, commit := ctx.CacheContext()
		liquidatorFee, perpEfFee, err := k.Liquidate(cachedCtx, liquidator, req.Pair, traderAddr)

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
		return nil, types.ErrAllLiquidationsFailed.Wrapf("%d liquidations failed", len(liquidationRequests))
	}

	return resp, nil
}

func (k Keeper) isWhitelistedLiquidator(ctx sdk.Context, addr sdk.AccAddress) bool {
	addrStr := addr.String()
	params := k.GetParams(ctx)
	// TODO(k-yang): look into an O(1) lookup data structure here
	for _, whitelisted := range params.WhitelistedLiquidators {
		if addrStr == whitelisted {
			return true
		}
	}
	return false
}
