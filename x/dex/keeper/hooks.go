package keeper

import (
	"github.com/NibiruChain/nibiru/x/dex/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k Keeper) AfterPoolCreation(ctx sdk.Context, pool types.Pool) {
	k.hooks.AfterPoolCreation(ctx, pool)
}
