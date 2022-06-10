/*
The "events" package implements functions to emit sdk.Events, which are
Tendermint application blockchain interface (ABCI) events.
These are returned by ABCI methods such as CheckTx, DeliverTx, and Query.

Events allow applications to associate metadata about ABCI method execution with
the transactions and blocks this metadata relates to. Events returned via these
ABCI methods do not impact Tendermint consensus in any way and instead exist to
power subscriptions and queries of Tendermint state.

For more information, see the Tendermint Core ABCI methods and types specification:
https://docs.tendermint.com/master/spec/abci/abci.html

Event Types:
- "transfer"
- "position_change"
- "position_liquidate"
- "position_settle"
- "margin_ratio_change"
- "margin_change"
- "internal_position_response"
*/
// TODO refactor: Tendermint ABCI events use snake case.
package events

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/perp/types"
)

// x/perp attributes used in multiple events
const (
	// from: receiving address of a transfer
	AttributeFromAddr = "from"
	// to: sending address of a transfer
	AttributeToAddr        = "to"
	AttributeTokenAmount   = "amount"
	AttributeTokenDenom    = "denom"
	AttributePositionOwner = "owner"
	AttributeVpool         = "vpool"
)

func NewTransferEvent(
	coin sdk.Coin, from sdk.AccAddress, to sdk.AccAddress,
) sdk.Event {
	const EventTypeTransfer = "transfer"
	return sdk.NewEvent(
		EventTypeTransfer,
		sdk.NewAttribute(AttributeFromAddr, from.String()),
		sdk.NewAttribute(AttributeToAddr, to.String()),
		sdk.NewAttribute(AttributeTokenDenom, coin.Denom),
		sdk.NewAttribute(AttributeTokenAmount, coin.Amount.String()),
	)
}

func EmitTransfer(
	ctx sdk.Context, coin sdk.Coin, from sdk.AccAddress, to sdk.AccAddress,
) {
	ctx.EventManager().EmitEvent(NewTransferEvent(coin, from, to))
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
	ctx.EventManager().EmitEvent(NewPositionChangeEvent(
		owner,
		vpool,
		margin,
		notional,
		vsizeChange,
		txFee,
		vsizeAfter,
		realizedPnlAfter,
		badDebt,
		unrealizedPnlAfter,
		liquidationPenalty,
		vPrice,
		fundingPayment,
	))
}

func NewPositionChangeEvent(
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
) sdk.Event {
	const EventTypePositionChange = "position_change"
	return sdk.NewEvent(
		EventTypePositionChange,
		sdk.NewAttribute(AttributePositionOwner, owner.String()),
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
	)
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
  feeToLiquidator sdk.Int: Commission (in margin units) received by 'liquidator'.
  badDebt sdk.Int: Bad debt (margin units) cleared by the PerpEF during the tx.
    Bad debt is negative net margin past the liquidation point of a position.
*/
func EmitPositionLiquidate(
	ctx sdk.Context,
	vpool string,
	trader sdk.AccAddress,
	notional sdk.Dec,
	vsize sdk.Dec,
	liquidator sdk.AccAddress,
	feeToLiquidator sdk.Int,
	feeToPerpEF sdk.Int,
	badDebt sdk.Dec,
) {
	ctx.EventManager().EmitEvent(NewPositionLiquidateEvent(
		vpool, trader, notional, vsize, liquidator, feeToLiquidator, feeToPerpEF,
		badDebt,
	))
}

func NewPositionLiquidateEvent(
	vpool string,
	owner sdk.AccAddress,
	notional sdk.Dec,
	vsize sdk.Dec,
	liquidator sdk.AccAddress,
	feeToLiquidator sdk.Int,
	feeToPerpEF sdk.Int,
	badDebt sdk.Dec,
) sdk.Event {
	const EventTypePositionLiquidate = "position_liquidate"
	return sdk.NewEvent(
		EventTypePositionLiquidate,
		sdk.NewAttribute(AttributeVpool, vpool),
		sdk.NewAttribute(AttributePositionOwner, owner.String()),
		sdk.NewAttribute("notional", notional.String()),
		sdk.NewAttribute("vsize", vsize.String()),
		sdk.NewAttribute("liquidator", liquidator.String()),
		sdk.NewAttribute("feeToLiquidator", feeToLiquidator.String()),
		sdk.NewAttribute("feeToPerpEF", feeToPerpEF.String()),
		sdk.NewAttribute("badDebt", badDebt.String()),
	)
}

/* EmitPositionSettle emits an event when a position is settled.

Args:
  ctx sdk.Context: Carries information about the current state of the application.
  vpool string: Identifier for the virtual pool of the position.
  trader string: Owner of the position.
  settled sdk.Coin: Settled coin as dictated by the settlement price of the vpool.
*/
func EmitPositionSettle(
	ctx sdk.Context,
	vpool string,
	trader string,
	settled sdk.Coins,
) {
	ctx.EventManager().EmitEvent(NewPositionSettleEvent(
		vpool, trader, settled,
	))
}

func NewPositionSettleEvent(
	vpool string,
	trader string,
	settled sdk.Coins,
) sdk.Event {
	const EventTypePositionSettle = "position_settle"
	return sdk.NewEvent(
		EventTypePositionSettle,
		sdk.NewAttribute(AttributeVpool, vpool),
		sdk.NewAttribute(AttributePositionOwner, trader),
		sdk.NewAttribute("settled_coins", settled.String()),
	)
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
	ctx.EventManager().EmitEvent(NewMarginRatioChangeEvent(marginRatio))
}

func NewMarginRatioChangeEvent(
	marginRatio sdk.Dec,
) sdk.Event {
	const EventTypeMarginRatioChange = "margin_ratio_change"
	return sdk.NewEvent(
		EventTypeMarginRatioChange,
		sdk.NewAttribute("margin_ratio", marginRatio.String()),
	)
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
	traderAddr sdk.AccAddress,
	vpool string,
	marginAmt sdk.Int,
	fundingPayment sdk.Dec,
) {
	ctx.EventManager().EmitEvent(NewMarginChangeEvent(
		traderAddr, vpool, marginAmt, fundingPayment),
	)
}

func NewMarginChangeEvent(
	traderAddr sdk.AccAddress,
	vpool string,
	marginAmt sdk.Int,
	fundingPayment sdk.Dec,
) sdk.Event {
	const EventTypeMarginChange = "margin_change"
	return sdk.NewEvent(
		EventTypeMarginChange,
		sdk.NewAttribute(AttributePositionOwner, traderAddr.String()),
		sdk.NewAttribute(AttributeVpool, vpool),
		sdk.NewAttribute("margin_amt", marginAmt.String()),
		sdk.NewAttribute("funding_payment", fundingPayment.String()),
	)
}

// --------------------------------------------------------------------

/* EmitInternalPositionResponseEvent emits an sdk.Event to track the position
response ('PositionResp') outputs returned by: 'closePositionEntirely',
'closeAndOpenReversePosition', 'increasePosition', and 'decreasePosition'.
*/
func EmitInternalPositionResponseEvent(
	ctx sdk.Context, positionResp *types.PositionResp, function string) {
	ctx.EventManager().EmitEvent(NewInternalPositionResponseEvent(
		positionResp, function),
	)
}

/* NewInternalPositionResponseEvent returns an sdk.Event to track the position
response ('PositionResp') outputs returned by: 'closePositionEntirely',
'closeAndOpenReversePosition', 'increasePosition', and 'decreasePosition'.
*/
func NewInternalPositionResponseEvent(
	positionResp *types.PositionResp, function string,
) sdk.Event {
	pos := positionResp.Position
	return sdk.NewEvent(
		"internal_position_response",
		sdk.NewAttribute(AttributePositionOwner, pos.TraderAddress),
		sdk.NewAttribute(AttributeVpool, pos.Pair),
		sdk.NewAttribute("pos_margin", pos.Margin.String()),
		sdk.NewAttribute("pos_open_notional", pos.OpenNotional.String()),
		sdk.NewAttribute("bad_debt", positionResp.BadDebt.String()),
		sdk.NewAttribute("exchanged_position_size", positionResp.ExchangedPositionSize.String()),
		sdk.NewAttribute("funding_payment", positionResp.FundingPayment.String()),
		sdk.NewAttribute("realized_pnl", positionResp.RealizedPnl.String()),
		sdk.NewAttribute("margin_to_vault", positionResp.MarginToVault.String()),
		sdk.NewAttribute("unrealized_pnl_after", positionResp.UnrealizedPnlAfter.String()),
		sdk.NewAttribute("function", function),
	)
}
