package events

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	EventTypePoolCreated = "pool_created"
	EventTypePoolJoined  = "pool_joined"
	EventTypePoolExited  = "pool_exited"

	AttributeCreator      = "creator"
	AttributeSender       = "sender"
	AttributePoolId       = "pool_id"
	AttributeTokensIn     = "tokens_in"
	AttributeNumSharesOut = "pool_shares_out"
	AttributeNumRemCoins  = "rem_coins"
	AttributeTokensOut    = "tokens_out"
	AttributeNumSharesIn  = "pool_shares_in"
)

func EmitPoolJoinedEvent(
	ctx sdk.Context,
	sender sdk.AccAddress,
	poolId uint64,
	tokensIn sdk.Coins,
	numSharesOut sdk.Coin,
	remCoins sdk.Coins,
) {
	ctx.EventManager().EmitEvent(
		NewPoolJoinedEvent(
			sender,
			poolId,
			tokensIn,
			numSharesOut,
			remCoins,
		),
	)
}

func NewPoolJoinedEvent(
	sender sdk.AccAddress,
	poolId uint64,
	tokensIn sdk.Coins,
	numSharesOut sdk.Coin,
	remCoins sdk.Coins,
) sdk.Event {
	return sdk.NewEvent(
		EventTypePoolJoined,
		sdk.NewAttribute(AttributeSender, sender.String()),
		sdk.NewAttribute(AttributePoolId, fmt.Sprintf("%d", poolId)),
		sdk.NewAttribute(AttributeTokensIn, tokensIn.String()),
		sdk.NewAttribute(AttributeNumSharesOut, numSharesOut.String()),
		sdk.NewAttribute(AttributeNumRemCoins, remCoins.String()),
	)
}

func EmitPoolCreatedEvent(ctx sdk.Context, sender sdk.AccAddress, poolId uint64) {
	ctx.EventManager().EmitEvent(
		NewPoolCreatedEvent(sender, poolId),
	)
}

func NewPoolCreatedEvent(sender sdk.AccAddress, poolId uint64) sdk.Event {
	return sdk.NewEvent(
		EventTypePoolCreated,
		sdk.NewAttribute(AttributeCreator, sender.String()),
		sdk.NewAttribute(AttributePoolId, fmt.Sprintf("%d", poolId)),
	)
}

func EmitPoolExitedEvent(
	ctx sdk.Context,
	sender sdk.AccAddress,
	poolId uint64,
	numSharesIn sdk.Coin,
	tokensOut sdk.Coins,
) {
	ctx.EventManager().EmitEvent(
		NewPoolExitedEvent(
			sender,
			poolId,
			numSharesIn,
			tokensOut,
		),
	)
}

func NewPoolExitedEvent(
	sender sdk.AccAddress,
	poolId uint64,
	numSharesIn sdk.Coin,
	tokensOut sdk.Coins,
) sdk.Event {
	return sdk.NewEvent(
		EventTypePoolExited,
		sdk.NewAttribute(AttributeSender, sender.String()),
		sdk.NewAttribute(AttributePoolId, fmt.Sprintf("%d", poolId)),
		sdk.NewAttribute(AttributeNumSharesIn, numSharesIn.String()),
		sdk.NewAttribute(AttributeTokensOut, tokensOut.String()),
	)
}
