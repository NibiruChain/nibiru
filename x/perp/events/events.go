/*
The "events" package implements functions to emit sdk.Events, which are
Tendermint application blockchain interface (ABCI) events.
These are returned by ABCI methods such as CheckTx, DeliverTx, and Query.

Events allow applications to associate metadata about ABCI method execution with
the transactions and blocks this metadata relates to. Events returned via these
ABCI methods do not impact Tendermint consensus in any way and instead exist to
power subscriptions and queries of Tendermint state.
*/
package events

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// x/perp attributes used in multiple events
const (
	// from: receiving address of a transfer
	AttributeFromAddr = "from"
	// to: sending address of a transfer
	AttributeToAddr         = "to"
	AttributeTokenAmount    = "amount"
	AttributeTokenDenom     = "denom"
	AttributePosittionOwner = "owner"
	AttributeVpool          = "vpool"
)

func EmitTransfer(
	ctx sdk.Context, coin sdk.Coin, from string, to string,
) {
	const EventTypeTransfer = "transfer"
	ctx.EventManager().EmitEvent(sdk.NewEvent(
		EventTypeTransfer,
		sdk.NewAttribute(AttributeFromAddr, from),
		sdk.NewAttribute(AttributeToAddr, to),
		sdk.NewAttribute(AttributeTokenDenom, coin.Denom),
		sdk.NewAttribute(AttributeTokenAmount, coin.Amount.String()),
	))
}

/* EmitPositionChange emits an event when a position (vpool-trader) is changed.

Args:
  ctx sdk.Context: Carries information about the current state of the application.
  owner sdk.AccAddress: owner of the position.
  vpool string: identifier of the corresponding virtual pool for the position
  margin sdk.Int: amount of quote token (y) backing the position.
  notional sdk.Dec: margin * leverage * vPrice. 'notional' is the virtual size times
    the virtual price on 'vpool'.
  vsizeChange sdk.Dec: magnitude of the change to vsize. The vsize is the amount
  	of base assets for the position, margin * leverage * priceBasePerQuote.
  txFee sdk.Int: transaction fee paid
  vsizeAfter sdk.Dec: position virtual size after the change
  realizedPnlAfter: realize profits and losses after the change
  badDebt sdk.Int: Amount of bad debt cleared by the PerpEF during the change.
    Bad debt is negative net margin past the liquidation point of a position.
  unrealizedPnlAfter: unrealized profits and losses after the change
  liquidationPenalty: amt of margin (y) lost due to liquidation
  vPrice sdk.Dec: vPrice defined as yRes / xRes for a vpool, where yRes is the
    quote reserves and xRes is the base reserves.
  fundingPayment sdk.Dec: A funding payment made or received by the trader on
    the current position. 'fundingPayment' is positive if 'owner' is the sender
	and negative if 'owner' is the receiver of the payment. Its magnitude is
	abs(vSize * fundingRate). Funding payments act to converge the mark price
	(vPrice) and index price (average price on major exchanges).

// TODO Q: Is there a way to split this into different events without creating too
// much complexity?
*/
func EmitPositionChange(
	ctx sdk.Context,
	owner sdk.AccAddress,
	vpool string,
	margin sdk.Int,
	notional sdk.Dec,
	vsizeChange sdk.Dec,
	txFee sdk.Int,
	vsizeAfter sdk.Dec,
	realizedPnlAfter sdk.Dec,
	badDebt sdk.Dec,
	unrealizedPnlAfter sdk.Dec,
	liquidationPenalty sdk.Int,
	vPrice sdk.Dec,
	fundingPayment sdk.Dec,
) {
	const EventTypePositionChange = "position_change"
	ctx.EventManager().EmitEvent(sdk.NewEvent(
		EventTypePositionChange,
		sdk.NewAttribute(AttributePosittionOwner, owner.String()),
		sdk.NewAttribute(AttributeVpool, vpool),
		sdk.NewAttribute("margin", margin.String()),
		sdk.NewAttribute("notional", notional.String()),
		sdk.NewAttribute("vsizeChange", vsizeChange.String()),
		sdk.NewAttribute("txFee", txFee.String()),
		sdk.NewAttribute("vsizeAfter", vsizeAfter.String()),
		sdk.NewAttribute("realizedPnlAfter", realizedPnlAfter.String()),
		sdk.NewAttribute("badDebt", badDebt.String()),
		sdk.NewAttribute("unrealizedPnlAfter", unrealizedPnlAfter.String()),
		sdk.NewAttribute("liquidationPenalty", liquidationPenalty.String()),
		sdk.NewAttribute("vPrice", vPrice.String()),
		sdk.NewAttribute("fundingPayment", fundingPayment.String()),
	))
}

/* EmitPositionLiquidate emits an event when a liquidation occurs.

Args:
  ctx sdk.Context: Carries information about the current state of the application.
  vpool string: identifier of the corresponding virtual pool for the position
  owner sdk.AccAddress: owner of the position.
  notional sdk.Dec: margin * leverage * vPrice. 'notional' is the virtual size times
    the virtual price on 'vpool'.
  vsize sdk.Dec: virtual amount of base assets for the position, which would be
    margin * leverage * priceBasePerQuote.
  liquidator sdk.AccAddress: Address of the account that executed the tx.
  liquidationFee sdk.Int: Commission (in margin units) received by 'liquidator'.
  badDebt sdk.Int: Bad debt (margin units) cleared by the PerpEF during the tx.
    Bad debt is negative net margin past the liquidation point of a position.
*/
func EmitPositionLiquidate(
	ctx sdk.Context,
	vpool string,
	owner sdk.AccAddress,
	notional sdk.Dec,
	vsize sdk.Dec,
	liquidator sdk.AccAddress,
	liquidationFee sdk.Int,
	badDebt sdk.Dec,
) {
	const EventTypePositionLiquidate = "position_liquidate"
	ctx.EventManager().EmitEvent(sdk.NewEvent(
		EventTypePositionLiquidate,
		sdk.NewAttribute(AttributeVpool, vpool),
		sdk.NewAttribute(AttributePosittionOwner, owner.String()),
		sdk.NewAttribute("notional", notional.String()),
		sdk.NewAttribute("vsize", vsize.String()),
		sdk.NewAttribute("liquidator", liquidator.String()),
		sdk.NewAttribute("liquidationFee", liquidationFee.String()),
		sdk.NewAttribute("badDebt", badDebt.String()),
	))
}

/* EmitPositionSettle emits an event when a position is settled.

Args:
  ctx sdk.Context: Carries information about the current state of the application.
  vpool string: Identifier for the virtual pool of the position.
  owner sdk.AccAddress: Owner of the position.
  settled sdk.Coin: Settled coin as dictated by the settlement price of the vpool.
*/
func EmitPositionSettle(
	ctx sdk.Context,
	vpool string,
	owner sdk.AccAddress,
	settled sdk.Coin,
) {
	const EventTypePositionSettle = "position_settle"
	ctx.EventManager().EmitEvent(sdk.NewEvent(
		EventTypePositionSettle,
		sdk.NewAttribute(AttributeVpool, vpool),
		sdk.NewAttribute(AttributePosittionOwner, owner.String()),
		sdk.NewAttribute("settle_amt", settled.Amount.String()),
		sdk.NewAttribute("settle_denom", settled.Denom),
	))
}

/* EmitMarginRatioChange emits an event when the protocol margin ratio changes.

Args:
  ctx sdk.Context: Carries information about the current state of the application.
  vpool string: Identifier for the virtual pool of the position.
*/
func EmitMarginRatioChange(
	ctx sdk.Context,
	marginRatio sdk.Dec,
) {
	const EventTypeMarginRatioChange = "margin_ratio_change"
	ctx.EventManager().EmitEvent(sdk.NewEvent(
		EventTypeMarginRatioChange,
		sdk.NewAttribute("margin_ratio", marginRatio.String()),
	))
}

/* EmitMarginChange emits an event when the protocol margin ratio changes.

Args:
  ctx sdk.Context: Carries information about the current state of the application.
  owner sdk.AccAddress: Owner of the position.
  vpool string: Identifier for the virtual pool of the position.
  marginAmt sdk.Int: Delta of the position margin. If positive, margin was added.
  fundingPayment sdk.Int: The position 'owner' may realize a funding payment if
    there is no bad debt upon margin removal based on the delta of the position
	margin and latest cumulative premium fraction.
*/
func EmitMarginChange(
	ctx sdk.Context,
	owner sdk.AccAddress,
	vpool string,
	marginAmt sdk.Int,
	fundingPayment sdk.Dec,
) {
	const EventTypeMarginChange = "margin_change"
	ctx.EventManager().EmitEvent(sdk.NewEvent(
		EventTypeMarginChange,
		sdk.NewAttribute(AttributePosittionOwner, owner.String()),
		sdk.NewAttribute(AttributeVpool, vpool),
		sdk.NewAttribute("margin_amt", marginAmt.String()),
		sdk.NewAttribute("funding_payment", fundingPayment.String()),
	))
}
