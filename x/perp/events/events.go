/*
The "events" package implements functions to emit sdk.Events, which are
application blockchain interface (ABCI) events.
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

/*
EmitPositionChange emits an event when a position (vpool-trader) is changed.

Args:
  ctx sdk.Context:
  owner sdk.AccAddress:
  vpool string: identifier of the position\'s corresponding virtual pool
  margin sdk.Int: amount of quote token (y) backing the position.
  leveragedMargin sdk.Dec: margin * leverage
  vsizeChange sdk.Dec: magnitude of the change to vsize
  txFee sdk.Int: transaction fee payed
  vSizeAfter sdk.Dec: position virtual size after the change
  realizedPnlAfter: realize profits and losses after the change
  badDebt sdk.Dec: // TODO
  unrealizedPnlAfter: unrealized profits and losses after the change
  liquidationPenalty: amt of margin (y) lost due to liquidation
  vPrice sdk.Dec: vPrice defined as yRes / xRes for a vpool, where yRes is the
    quote reserves and xRes is the base reserves.
  fundingPayment sdk.Dec: A funding payment made or received by the trader on
    the current position. 'fundingPayment' is positive if 'owner' is the sender
	and negative if 'owner' is the receiver of the payment. It's magnitude is
	abs(vSize * fundingRate). Funding payments act to converge the mark price
	(vPrice) and index price (average price on majorexchanges).
// TODO When does EmitPositionChange happen?
// TODO Is there a way to split this into different events without creating too
// much complexity?
*/
func EmitPositionChange(
	ctx sdk.Context,
	owner sdk.AccAddress,
	vpool string,
	margin sdk.Int,
	leveragedMargin sdk.Dec,
	vsizeChange sdk.Dec,
	txFee sdk.Int,
	vSizeAfter sdk.Dec,
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
		sdk.NewAttribute("leveragedMargin", leveragedMargin.String()),
		sdk.NewAttribute("vsizeChange", vsizeChange.String()),
		sdk.NewAttribute("txFee", txFee.String()),
		sdk.NewAttribute("vSizeAfter", vSizeAfter.String()),
		sdk.NewAttribute("realizedPnlAfter", realizedPnlAfter.String()),
		sdk.NewAttribute("badDebt", badDebt.String()),
		sdk.NewAttribute("unrealizedPnlAfter", unrealizedPnlAfter.String()),
		sdk.NewAttribute("liquidationPenalty", liquidationPenalty.String()),
		sdk.NewAttribute("vPrice", vPrice.String()),
		sdk.NewAttribute("fundingPayment", fundingPayment.String()),
	))
}
