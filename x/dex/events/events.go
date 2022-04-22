package events

import (
	"fmt"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	AttributeSender       = "sender"
	EventTypeJoinPool     = "join_pool"
	AttributePoolId       = "pool_id"
	AttributeTokensIn     = "tokens_in"
	AttributeNumSharesOut = "shares_out"
	AttributeNumRemCoins  = "rem_coins"
)

func EmitJoinPool(
	ctx sdk.Context,
	sender sdk.AccAddress,
	poolId uint64,
	tokensIn sdk.Coins,
	numSharesOut sdk.Coin,
	remCoins sdk.Coins,
) {
	ctx.EventManager().EmitEvent(
		NewJoinPoolEvent(
			sender,
			poolId,
			tokensIn,
			numSharesOut,
			remCoins,
		),
	)
}

func NewJoinPoolEvent(
	sender sdk.AccAddress,
	poolId uint64,
	tokensIn sdk.Coins,
	numSharesOut sdk.Coin,
	remCoins sdk.Coins,
) sdk.Event {
	return sdk.NewEvent(
		EventTypeJoinPool,
		sdk.NewAttribute(AttributeSender, sender.String()),
		sdk.NewAttribute(AttributePoolId, fmt.Sprintf("%d", poolId)),
		sdk.NewAttribute(AttributeTokensIn, tokensIn.String()),
		sdk.NewAttribute(AttributeNumSharesOut, numSharesOut.String()),
		sdk.NewAttribute(AttributeNumRemCoins, remCoins.String()),
	)
}
