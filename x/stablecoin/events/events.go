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

func EmitMintStable(
	ctx sdk.Context, coin sdk.Coin,
) {
	event := sdk.NewEvent(
		EventTypeMintStable,
		sdk.NewAttribute(AttributeTokenDenom, coin.Denom),
		sdk.NewAttribute(AttributeTokenAmount, coin.Amount.String()),
	)
	ctx.EventManager().EmitEvents(sdk.Events{event})
}

func EmitBurnStable(
	ctx sdk.Context, coin sdk.Coin, from sdk.AccAddress, to sdk.AccAddress,
) {
	event := sdk.NewEvent(
		EventTypeBurnStable,
		sdk.NewAttribute(AttributeTokenDenom, coin.Denom),
		sdk.NewAttribute(AttributeTokenAmount, coin.Amount.String()),
	)
	ctx.EventManager().EmitEvents(sdk.Events{event})
}

func EmitMintMtrx(
	ctx sdk.Context, coin sdk.Coin, from sdk.AccAddress, to sdk.AccAddress,
) {
	event := sdk.NewEvent(
		EventTypeMintMtrx,
		sdk.NewAttribute(AttributeTokenDenom, coin.Denom),
		sdk.NewAttribute(AttributeTokenAmount, coin.Amount.String()),
	)
	ctx.EventManager().EmitEvents(sdk.Events{event})
}

func EmitBurnMtrx(
	ctx sdk.Context, coin sdk.Coin, from sdk.AccAddress, to sdk.AccAddress,
) {
	event := sdk.NewEvent(
		EventTypeBurnMtrx,
		sdk.NewAttribute(AttributeTokenDenom, coin.Denom),
		sdk.NewAttribute(AttributeTokenAmount, coin.Amount.String()),
	)
	ctx.EventManager().EmitEvents(sdk.Events{event})
}
