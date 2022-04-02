package events

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// x/stablecoin events
const (
	EventTypeMintStable = "mint_stable"
	EventTypeBurnStable = "burn_stable"
	EventTypeMintMtrx   = "mint_mtrx"
	EventTypeBurnMtrx   = "burn_mtrx"
	EventTypeTransfer   = "transfer"

	AttributeFromAddr    = "from"
	AttributeToAddr      = "to"
	AttributeTokenAmount = "amount"
	AttributeTokenDenom  = "denom"
)

func EmitTransfer(
	ctx sdk.Context, coin sdk.Coin, from sdk.AccAddress, to sdk.AccAddress,
) {
	event := sdk.NewEvent(
		EventTypeTransfer,
		sdk.NewAttribute(AttributeFromAddr, from.String()),
		sdk.NewAttribute(AttributeToAddr, to.String()),
		sdk.NewAttribute(AttributeTokenDenom, coin.Denom),
		sdk.NewAttribute(AttributeTokenAmount, coin.Amount.String()),
	)
	ctx.EventManager().EmitEvents(sdk.Events{event})
}

func _mintOrBurnEvent(eventType string, coin sdk.Coin) sdk.Event {
	event := sdk.NewEvent(
		eventType,
		sdk.NewAttribute(AttributeTokenDenom, coin.Denom),
		sdk.NewAttribute(AttributeTokenAmount, coin.Amount.String()),
	)
	return event
}

func EmitMintStable(
	ctx sdk.Context, coin sdk.Coin,
) {
	event := _mintOrBurnEvent(EventTypeMintStable, coin)
	ctx.EventManager().EmitEvents(sdk.Events{event})
}

func EmitBurnStable(
	ctx sdk.Context, coin sdk.Coin, from sdk.AccAddress, to sdk.AccAddress,
) {
	event := _mintOrBurnEvent(EventTypeBurnStable, coin)
	ctx.EventManager().EmitEvents(sdk.Events{event})
}

func EmitMintMtrx(
	ctx sdk.Context, coin sdk.Coin, from sdk.AccAddress, to sdk.AccAddress,
) {
	event := _mintOrBurnEvent(EventTypeMintMtrx, coin)
	ctx.EventManager().EmitEvents(sdk.Events{event})
}

func EmitBurnMtrx(
	ctx sdk.Context, coin sdk.Coin, from sdk.AccAddress, to sdk.AccAddress,
) {
	event := _mintOrBurnEvent(EventTypeBurnMtrx, coin)
	ctx.EventManager().EmitEvents(sdk.Events{event})
}
