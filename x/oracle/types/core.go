package types

import sdk "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types"

// IsPeriodLastBlock returns true if we are at the last block of the period
func IsPeriodLastBlock(ctx sdk.Context, blocksPerPeriod uint64) bool {
	return ((uint64)(ctx.BlockHeight())+1)%blocksPerPeriod == 0
}
