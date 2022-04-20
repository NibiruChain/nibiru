package events

import (
	"fmt"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	attributeSender       = "sender"
	EventTypeJoinPool     = "join_pool"
	attributePoolId       = "pool_id"
	attributeTokensIn     = "tokens_in"
	attributeNumSharesOut = "shares_out"
	attributeNumRemCoins  = "rem_coins"
)

func EmitJoinPool(
	ctx sdk.Context,
	sender sdk.AccAddress,
	poolId uint64,
	tokensIn sdk.Coins,
	numSharesOut sdk.Coin,
	remCoins sdk.Coins,
) {
	ctx.EventManager().EmitEvent(sdk.NewEvent(
		EventTypeJoinPool,
		sdk.NewAttribute(attributeSender, sender.String()),
		sdk.NewAttribute(attributePoolId, fmt.Sprintf("%d", poolId)),
		sdk.NewAttribute(attributeTokensIn, tokensIn.String()),
		sdk.NewAttribute(attributeNumSharesOut, numSharesOut.String()),
		sdk.NewAttribute(attributeNumRemCoins, remCoins.String()),
	))
}
