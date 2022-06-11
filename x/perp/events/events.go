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
	AttributePositionOwner = "owner"
	AttributeVpool         = "vpool"
)

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
