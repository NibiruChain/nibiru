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
  - ecosystemFundFee: the amount of coins given to the ecosystem fund
  - err: error
  - event: pointer to a typed event (proto.Message). The 'event' value
    exists when the liquidation fails and is nil when the liquidation succeeds.
*/
func (k Keeper) liquidate(
	ctx sdk.Context,
	liquidator sdk.AccAddress,
	pair asset.Pair,
	trader sdk.AccAddress,
) (liquidatorFee sdk.Coin, ecosystemFundFee sdk.Coin, err error) {
	// eventLiqFailed exists when the liquidation fails and is nil when the
	// liquidation succeeds.

	market, err := k.GetMarket(ctx, pair)
	if err != nil {
		eventLiqFailed := &types.LiquidationFailedEvent{
			Pair:       pair,
			Trader:     trader.String(),
			Liquidator: liquidator.String(),
			Reason:     types.LiquidationFailedEvent_NONEXISTENT_PAIR,
		}
		_ = ctx.EventManager().EmitTypedEvent(eventLiqFailed)
		err = sdkerrors.Wrapf(types.ErrPairNotFound, "pair: %s", pair)
		return
	}

	amm, err := k.GetAMM(ctx, pair)
	if err != nil {
		eventLiqFailed := &types.LiquidationFailedEvent{
			Pair:       pair,
			Trader:     trader.String(),
			Liquidator: liquidator.String(),
			Reason:     types.LiquidationFailedEvent_NONEXISTENT_PAIR,
		}
		_ = ctx.EventManager().EmitTypedEvent(eventLiqFailed)
		err = sdkerrors.Wrapf(types.ErrPairNotFound, "pair: %s", pair)
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
	if spotMarginRatio.GTE(market.LiquidationFeeRatio) {
		liquidatorFee, ecosystemFundFee, err = k.executePartialLiquidation(ctx, market, amm, liquidator, &position)
	} else {
		liquidatorFee, ecosystemFundFee, err = k.executeFullLiquidation(ctx, market, amm, liquidator, &position)
	}
	if err != nil {
		return sdk.Coin{}, sdk.Coin{}, err
	}

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
) (liquidatorfee sdk.Coin, ecosystemFundFee sdk.Coin, err error) {
	_, positionResp, err := k.closePositionEntirely(
		ctx,
		market,
		amm,
		/* currentPosition */ *position,
		/* quoteAssetAmountLimit */ sdk.ZeroDec(),
	)
	if err != nil {
		return sdk.Coin{}, sdk.Coin{}, err
	}

	remainMargin := positionResp.MarginToVault.Abs()

	liquidatorFeeAmount := market.LiquidationFeeRatio.
		Mul(positionResp.ExchangedNotionalValue).
		QuoInt64(2)
	totalBadDebt := positionResp.BadDebt

	if liquidatorFeeAmount.GT(remainMargin) {
		// if the remainMargin is not enough for liquidationFee, count it as bad debt
		totalBadDebt = totalBadDebt.Add(liquidatorFeeAmount.Sub(remainMargin))
		remainMargin = sdk.ZeroDec()
	} else {
		// Otherwise, the remaining margin will be transferred to ecosystemFund
		remainMargin = remainMargin.Sub(liquidatorFeeAmount)
	}

	// Realize bad debt
	if totalBadDebt.IsPositive() {
		if err = k.realizeBadDebt(
			ctx,
			market,
			totalBadDebt.RoundInt(),
		); err != nil {
			return sdk.Coin{}, sdk.Coin{}, err
		}
	}

	ecosystemFundFeeAmount := sdk.ZeroDec()
	if remainMargin.IsPositive() {
		ecosystemFundFeeAmount = remainMargin
	}

	quoteDenom := market.Pair.QuoteDenom()

	liquidatorfee = sdk.NewCoin(quoteDenom, liquidatorFeeAmount.RoundInt())
	ecosystemFundFee = sdk.NewCoin(quoteDenom, ecosystemFundFeeAmount.RoundInt())

	err = k.distributeLiquidateRewards(
		ctx,
		market,
		liquidator,
		liquidatorfee,
		ecosystemFundFee,
	)
	if err != nil {
		return sdk.Coin{}, sdk.Coin{}, err
	}

	_ = ctx.EventManager().EmitTypedEvent(&types.PositionLiquidatedEvent{
		PositionChangedEvent: types.PositionChangedEvent{
			FinalPosition:    positionResp.Position,
			PositionNotional: positionResp.PositionNotional,
			TransactionFee:   sdk.NewCoin(position.Pair.QuoteDenom(), sdk.ZeroInt()), // no transaction fee for liquidation
			RealizedPnl:      positionResp.RealizedPnl,
			BadDebt:          sdk.NewCoin(position.Pair.QuoteDenom(), totalBadDebt.RoundInt()),
			FundingPayment:   positionResp.FundingPayment,
			BlockHeight:      ctx.BlockHeight(),
			MarginToUser:     sdk.ZeroInt(), // no margin to user for full liquidation
			ChangeReason:     types.ChangeReason_FullLiquidation,
		},
		LiquidatorAddress:  liquidator.String(),
		FeeToLiquidator:    sdk.NewCoin(position.Pair.QuoteDenom(), liquidatorFeeAmount.RoundInt()),
		FeeToEcosystemFund: sdk.NewCoin(position.Pair.QuoteDenom(), ecosystemFundFeeAmount.RoundInt()),
	})

	return liquidatorfee, ecosystemFundFee, err
}

// executePartialLiquidation partially liquidates a position
func (k Keeper) executePartialLiquidation(
	ctx sdk.Context, market types.Market, amm types.AMM, liquidator sdk.AccAddress, position *types.Position,
) (liquidatorFee sdk.Coin, ecosystemFundFee sdk.Coin, err error) {
	traderAddr, err := sdk.AccAddressFromBech32(position.TraderAddress)
	if err != nil {
		return sdk.Coin{}, sdk.Coin{}, err
	}

	var dir types.Direction
	if position.Size_.IsPositive() {
		dir = types.Direction_SHORT
	} else {
		dir = types.Direction_LONG
	}

	quoteReserveDelta, err := amm.GetQuoteReserveAmt(position.Size_.Mul(market.PartialLiquidationRatio), dir)
	if err != nil {
		return sdk.Coin{}, sdk.Coin{}, err
	}
	quoteAssetDelta := amm.FromQuoteReserveToAsset(quoteReserveDelta)

	_, positionResp, err := k.decreasePosition(
		/* ctx */ ctx,
		market,
		amm,
		/* currentPosition */ *position,
		/* quoteAssetAmount */ quoteAssetDelta,
		/* baseAmtLimit */ sdk.ZeroDec(),
	)
	if err != nil {
		return sdk.Coin{}, sdk.Coin{}, err
	}

	// Remove the liquidation fee from the margin of the position
	liquidationFeeAmount := quoteAssetDelta.Mul(market.LiquidationFeeRatio)
	positionResp.Position.Margin = positionResp.Position.Margin.Sub(liquidationFeeAmount)
	k.Positions.Insert(ctx, collections.Join(positionResp.Position.Pair, traderAddr), positionResp.Position)

	// Compute splits for the liquidation fee
	feeToLiquidator := liquidationFeeAmount.QuoInt64(2)
	feeToPerpEcosystemFund := liquidationFeeAmount.Sub(feeToLiquidator)

	err = k.distributeLiquidateRewards(ctx, market, liquidator,
		sdk.NewCoin(market.Pair.QuoteDenom(), feeToPerpEcosystemFund.RoundInt()),
		sdk.NewCoin(market.Pair.QuoteDenom(), feeToLiquidator.RoundInt()),
	)
	if err != nil {
		return sdk.Coin{}, sdk.Coin{}, err
	}

	_ = ctx.EventManager().EmitTypedEvent(&types.PositionLiquidatedEvent{
		PositionChangedEvent: types.PositionChangedEvent{
			FinalPosition:    positionResp.Position,
			PositionNotional: positionResp.PositionNotional,
			TransactionFee:   sdk.NewCoin(position.Pair.QuoteDenom(), sdk.ZeroInt()), // no transaction fee for liquidation
			RealizedPnl:      positionResp.RealizedPnl,
			BadDebt:          sdk.NewCoin(position.Pair.QuoteDenom(), sdk.ZeroInt()), // no bad debt for partial liquidation
			FundingPayment:   positionResp.FundingPayment,
			BlockHeight:      ctx.BlockHeight(),
			MarginToUser:     sdk.ZeroInt(), // no margin to user for partial liquidation
			ChangeReason:     types.ChangeReason_PartialLiquidation,
		},
		LiquidatorAddress:  liquidator.String(),
		FeeToLiquidator:    sdk.NewCoin(position.Pair.QuoteDenom(), feeToLiquidator.RoundInt()),
		FeeToEcosystemFund: sdk.NewCoin(position.Pair.QuoteDenom(), feeToPerpEcosystemFund.RoundInt()),
	})

	return liquidatorFee, ecosystemFundFee, err
}

func (k Keeper) distributeLiquidateRewards(
	ctx sdk.Context, market types.Market, liquidator sdk.AccAddress, liquidatorFee sdk.Coin, ecosystemFundFee sdk.Coin,
) (err error) {
	// --------------------------------------------------------------
	// Distribution of rewards
	// --------------------------------------------------------------

	// Transfer fee from vault to PerpEF
	if ecosystemFundFee.IsPositive() {
		if err = k.BankKeeper.SendCoinsFromModuleToModule(
			ctx,
			/* from */ types.VaultModuleAccount,
			/* to */ types.PerpEFModuleAccount,
			sdk.NewCoins(ecosystemFundFee),
		); err != nil {
			return err
		}
	}

	// Transfer fee from vault to liquidator
	if liquidatorFee.IsPositive() {
		err = k.WithdrawFromVault(ctx, market, liquidator, liquidatorFee.Amount)
		if err != nil {
			return err
		}
	}

	return nil
}
