package keeper

import (
	"fmt"

	"github.com/NibiruChain/collections"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/perp/v2/types"
)

// AddMargin adds margin to an existing position, effectively deleveraging it.
// Adding margin increases the margin ratio of the corresponding position.
//
// args:
//   - ctx: the cosmos-sdk context
//   - pair: the asset pair
//   - traderAddr: the trader's address
//   - marginToAdd: the amount of margin to add. Must be positive.
//
// ret:
//   - res: the response
//   - err: error if any
func (k Keeper) AddMargin(
	ctx sdk.Context, pair asset.Pair, traderAddr sdk.AccAddress, marginToAdd sdk.Coin,
) (res *types.MsgAddMarginResponse, err error) {
	market, err := k.GetMarket(ctx, pair)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", types.ErrPairNotFound, pair)
	}
	amm, err := k.GetAMM(ctx, pair)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", types.ErrPairNotFound, pair)
	}

	if marginToAdd.Denom != amm.Pair.QuoteDenom() {
		return nil, fmt.Errorf("invalid margin denom: %s", marginToAdd.Denom)
	}

	position, err := k.Positions.Get(ctx, collections.Join(pair, traderAddr))
	if err != nil {
		return nil, err
	}

	fundingPayment := FundingPayment(position, market.LatestCumulativePremiumFraction)
	remainingMargin := position.Margin.Add(sdk.NewDecFromInt(marginToAdd.Amount)).Sub(fundingPayment)

	if remainingMargin.IsNegative() {
		return nil, types.ErrBadDebt.Wrapf("applying funding payment would result in negative remaining margin: %s", remainingMargin)
	}

	if err = k.BankKeeper.SendCoinsFromAccountToModule(
		ctx,
		/* from */ traderAddr,
		/* to */ types.VaultModuleAccount,
		/* amount */ sdk.NewCoins(marginToAdd),
	); err != nil {
		return nil, err
	}

	// apply funding payment and add margin
	position.Margin = remainingMargin
	position.LatestCumulativePremiumFraction = market.LatestCumulativePremiumFraction
	position.LastUpdatedBlockNumber = ctx.BlockHeight()
	k.Positions.Insert(ctx, collections.Join(position.Pair, traderAddr), position)

	positionNotional, err := PositionNotionalSpot(amm, position)
	if err != nil {
		return nil, err
	}

	if err = ctx.EventManager().EmitTypedEvent(
		&types.PositionChangedEvent{
			FinalPosition:    position,
			PositionNotional: positionNotional,
			TransactionFee:   sdk.NewCoin(pair.QuoteDenom(), sdk.ZeroInt()), // always zero when adding margin
			RealizedPnl:      sdk.ZeroDec(),                                 // always zero when adding margin
			BadDebt:          sdk.NewCoin(pair.QuoteDenom(), sdk.ZeroInt()), // always zero when adding margin
			FundingPayment:   fundingPayment,
			BlockHeight:      ctx.BlockHeight(),
			MarginToUser:     marginToAdd.Amount.Neg(),
			ChangeReason:     types.ChangeReason_AddMargin,
		},
	); err != nil {
		return nil, err
	}

	return &types.MsgAddMarginResponse{
		FundingPayment: fundingPayment,
		Position:       &position,
	}, nil
}

/*
	RemoveMargin further leverages an existing position by directly removing

the margin (collateral) that backs it from the vault. This also decreases the
margin ratio of the position.

Fails if the position goes underwater.

args:
  - ctx: the cosmos-sdk context
  - pair: the asset pair
  - traderAddr: the trader's address
  - margin: the amount of margin to withdraw. Must be positive.

ret:
  - marginOut: the amount of margin removed
  - fundingPayment: the funding payment that was applied with this position interaction
  - err: error if any
*/
func (k Keeper) RemoveMargin(
	ctx sdk.Context, pair asset.Pair, traderAddr sdk.AccAddress, marginToRemove sdk.Coin,
) (res *types.MsgRemoveMarginResponse, err error) {
	// fetch objects from state
	market, err := k.GetMarket(ctx, pair)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", types.ErrPairNotFound, pair)
	}

	amm, err := k.GetAMM(ctx, pair)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", types.ErrPairNotFound, pair)
	}
	if marginToRemove.Denom != amm.Pair.QuoteDenom() {
		return nil, fmt.Errorf("invalid margin denom: %s", marginToRemove.Denom)
	}

	position, err := k.Positions.Get(ctx, collections.Join(pair, traderAddr))
	if err != nil {
		return nil, err
	}

	// ensure we have enough free collateral
	spotNotional, err := PositionNotionalSpot(amm, position)
	if err != nil {
		return nil, err
	}
	twapNotional, err := k.PositionNotionalTWAP(ctx, position, market.TwapLookbackWindow)
	if err != nil {
		return nil, err
	}
	minPositionNotional := sdk.MinDec(spotNotional, twapNotional)

	// account for funding payment
	fundingPayment := FundingPayment(position, market.LatestCumulativePremiumFraction)
	remainingMargin := position.Margin.Sub(fundingPayment)

	// account for negative PnL
	unrealizedPnl := UnrealizedPnl(position, minPositionNotional)
	if unrealizedPnl.IsNegative() {
		remainingMargin = remainingMargin.Add(unrealizedPnl)
	}

	if remainingMargin.LT(sdk.NewDecFromInt(marginToRemove.Amount)) {
		return nil, types.ErrBadDebt.Wrapf(
			"not enough free collateral to remove margin; remainingMargin %s, marginToRemove %s", remainingMargin, marginToRemove,
		)
	}

	// apply funding payment and remove margin
	position.Margin = position.Margin.Sub(fundingPayment).Sub(sdk.NewDecFromInt(marginToRemove.Amount))
	position.LatestCumulativePremiumFraction = market.LatestCumulativePremiumFraction
	position.LastUpdatedBlockNumber = ctx.BlockHeight()

	err = k.checkMarginRatio(ctx, market, amm, position)
	if err != nil {
		return nil, err
	}

	if err = k.WithdrawFromVault(ctx, market, traderAddr, marginToRemove.Amount); err != nil {
		return nil, err
	}
	k.Positions.Insert(ctx, collections.Join(position.Pair, traderAddr), position)

	if err = ctx.EventManager().EmitTypedEvent(
		&types.PositionChangedEvent{
			FinalPosition:    position,
			PositionNotional: spotNotional,
			TransactionFee:   sdk.NewCoin(pair.QuoteDenom(), sdk.ZeroInt()), // always zero when removing margin
			RealizedPnl:      sdk.ZeroDec(),                                 // always zero when removing margin
			BadDebt:          sdk.NewCoin(pair.QuoteDenom(), sdk.ZeroInt()), // always zero when removing margin
			FundingPayment:   fundingPayment,
			BlockHeight:      ctx.BlockHeight(),
			MarginToUser:     marginToRemove.Amount,
			ChangeReason:     types.ChangeReason_RemoveMargin,
		},
	); err != nil {
		return nil, err
	}

	return &types.MsgRemoveMarginResponse{
		FundingPayment: fundingPayment,
		Position:       &position,
	}, nil
}
