package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type DexHooks interface {
	// the first block whose timestamp is after the duration is counted as the end of the epoch
	AfterPoolCreation(ctx sdk.Context, pool Pool)
}

var _ DexHooks = MultiEpochHooks{}

// combine multiple gamm hooks, all hook functions are run in array sequence.
type MultiEpochHooks []DexHooks

func NewMultiEpochHooks(hooks ...DexHooks) MultiEpochHooks {
	return hooks
}

// AfterPoolCreation is called after a new pool is created
func (h MultiEpochHooks) AfterPoolCreation(ctx sdk.Context, pool Pool) {
	for i := range h {
		h[i].AfterPoolCreation(ctx, pool)
	}
}
