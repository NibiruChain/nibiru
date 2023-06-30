package keeper

import (
	"encoding/json"
	"fmt"
	"strings"

	sdkerrors "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/collections"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"

	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/perp/v2/types"
)

func (k Keeper) MultiLiquidate(
	ctx sdk.Context, liquidator sdk.AccAddress, liquidationRequests []*types.MsgMultiLiquidate_Liquidation,
) ([]*types.MsgMultiLiquidateResponse_LiquidationResponse, error) {
	resps := make([]*types.MsgMultiLiquidateResponse_LiquidationResponse, len(liquidationRequests))

	var allFailed bool = true

	for reqIdx, req := range liquidationRequests {
		// NOTE: writeCachedCtx (1) emits the events on the EventManager of the
		// cachedCtx. and (2) writes that context to the commit multi store.

		// cachedCtx, writeCachedCtx := ctx.CacheContext()
		traderAddr, errAccAddress := sdk.AccAddressFromBech32(req.Trader)
		liquidatorFee, perpEfFee, err := k.liquidate(
			ctx, liquidator, req.Pair, traderAddr,
		)

		switch {
		case errAccAddress != nil:
			resps[reqIdx] = &types.MsgMultiLiquidateResponse_LiquidationResponse{
				Success: false,
				Error:   errAccAddress.Error(),
				Trader:  req.Trader,
				Pair:    req.Pair,
			}
		case err != nil:
			resps[reqIdx] = &types.MsgMultiLiquidateResponse_LiquidationResponse{
				Success: false,
				Error:   err.Error(),
				Trader:  req.Trader,
				Pair:    req.Pair,
			}
		default:
			// Success case
			allFailed = false
			resps[reqIdx] = &types.MsgMultiLiquidateResponse_LiquidationResponse{
				Success:       true,
				LiquidatorFee: &liquidatorFee,
				PerpEfFee:     &perpEfFee,
				Trader:        req.Trader,
				Pair:          req.Pair,
			}
			// writeCachedCtx()
		}
	}

	if allFailed {
		prettyResps, errPrettyResp := PrettyLiquidateResponse(resps)

		numLiquidations := len(liquidationRequests)
		errDescription := strings.Join(
			[]string{
				fmt.Sprintf("%d liquidations failed", numLiquidations),
				fmt.Sprintf("liquidate_responses: %s", prettyResps),
			},
			"\n",
		)
		if errPrettyResp != nil {
			errDescription += fmt.Sprintf("\n%s", errPrettyResp.Error())
		}
		return resps, types.ErrAllLiquidationsFailed.Wrap(errDescription)
	}

	return resps, nil
}

/*
PrettyLiquidateResponse converts a slice of liquidation responses into a
pretty formatted JSON array for each response. This helps with providing
descriptive error messages in the case of failed liquidations.

Example outputs:

```json
[

	{
	  "success": false,
	  "error": "failed liquidation A",
	  "liquidator_fee": null,
	  "perp_ef_fee": null,
	  "trader": "dummytraderA"
	},
	{
	  "success": true,
	  "error": "",
	  "liquidator_fee": { denom: "unibi", amount: "420"},
	  "perp_ef_fee": null,
	  "trader": "dummytraderB"
	}

]
```
*/
func PrettyLiquidateResponse(
	resps []*types.MsgMultiLiquidateResponse_LiquidationResponse,
) (pretty string, err error) {
	protoCodec := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
	var respJsons []json.RawMessage
	var jsonErrs string = ""
	for _, resp := range resps {
		respJsonBz, jsonErr := protoCodec.MarshalJSON(resp)
		if jsonErr != nil {
			jsonErrs += jsonErr.Error()
		}
		respJsons = append(respJsons, respJsonBz)
	}

	prettyBz, _ := json.MarshalIndent(respJsons, "", "  ")
	pretty = string(prettyBz)
	if jsonErrs != "" {
		return pretty, types.ErrParseLiquidateResponse.Wrap(jsonErrs)
	}
	return
}

/*
liquidate allows to liquidate the trader position if the margin is below the
required margin maintenance ratio.

args:
  - liquidator: the liquidator who is executing the liquidation
  - pair: the asset pair
  - trader: the trader who owns the position being liquidated

returns:
  - liquidatorFee: the amount of coins given to the liquidator
  - insuranceFundFee: the amount of coins given to the ecosystem fund
  - err: error
  - event: pointer to a typed event (proto.Message). The 'event' value
    exists when the liquidation fails and is nil when the liquidation succeeds.
*/
func (k Keeper) liquidate(
	ctx sdk.Context,
	liquidator sdk.AccAddress,
	pair asset.Pair,
	trader sdk.AccAddress,
) (liquidatorFee sdk.Coin, insuranceFundFee sdk.Coin, err error) {
	// eventLiqFailed exists when the liquidation fails and is nil when the
	// liquidation succeeds.

	market, err := k.Markets.Get(ctx, pair)
	if err != nil {
		eventLiqFailed := &types.LiquidationFailedEvent{
			Pair:       pair,
			Trader:     trader.String(),
			Liquidator: liquidator.String(),
			Reason:     types.LiquidationFailedEvent_NONEXISTENT_PAIR,
		}
		_ = ctx.EventManager().EmitTypedEvent(eventLiqFailed)
		sdkerrors.Wrapf(types.ErrPairNotFound, "pair: %s", pair)
		return
	}

	amm, err := k.AMMs.Get(ctx, pair)
	if err != nil {
		eventLiqFailed := &types.LiquidationFailedEvent{
			Pair:       pair,
			Trader:     trader.String(),
			Liquidator: liquidator.String(),
			Reason:     types.LiquidationFailedEvent_NONEXISTENT_PAIR,
		}
		_ = ctx.EventManager().EmitTypedEvent(eventLiqFailed)
		sdkerrors.Wrapf(types.ErrPairNotFound, "pair: %s", pair)
		return
	}

	position, err := k.Positions.Get(ctx, collections.Join(pair, trader))
	if err != nil {
		eventLiqFailed := &types.LiquidationFailedEvent{
			Pair:       pair,
			Trader:     trader.String(),
			Liquidator: liquidator.String(),
			Reason:     types.LiquidationFailedEvent_NONEXISTENT_POSITION,
		}
		_ = ctx.EventManager().EmitTypedEvent(eventLiqFailed)
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
		eventLiqFailed := &types.LiquidationFailedEvent{
			Pair:       pair,
			Trader:     trader.String(),
			Liquidator: liquidator.String(),
			Reason:     types.LiquidationFailedEvent_POSITION_HEALTHY,
		}
		_ = ctx.EventManager().EmitTypedEvent(eventLiqFailed)
		err = types.ErrPositionHealthy
		return
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

	insuranceFundFee = sdk.NewCoin(
		pair.QuoteDenom(),
		liquidationResponse.FeeToPerpEcosystemFund,
	)

	return liquidatorFee, insuranceFundFee, err
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
	)
	if err != nil {
		return types.LiquidateResp{}, err
	}

	// Remove the liquidation fee from the margin of the position
	liquidationFeeAmount := quoteAssetDelta.Mul(market.LiquidationFeeRatio)
	positionResp.Position.Margin = positionResp.Position.Margin.Sub(liquidationFeeAmount)
	k.Positions.Insert(ctx, collections.Join(positionResp.Position.Pair, traderAddr), positionResp.Position)

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
		err = k.WithdrawFromVault(ctx, market, liquidator, feeToLiquidator)
		if err != nil {
			return err
		}
	}

	return nil
}
