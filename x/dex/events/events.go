package events

import (
	"fmt"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	EventTypePoolCreated = "pool_created"
	EventTypePoolJoined = "pool_joined"

	AttributeCreator      = "creator"

	AttributeSender       = "sender"
	AttributePoolId       = "pool_id"
	AttributeTokensIn     = "tokens_in"
	AttributeNumSharesOut = "shares_out"
	AttributeNumRemCoins  = "rem_coins"
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
