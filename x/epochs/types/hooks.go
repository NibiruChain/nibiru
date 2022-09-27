package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type EpochHooks interface {
	// AfterEpochEnd the first block whose timestamp is after the duration is counted as the end of the epoch
	AfterEpochEnd(ctx sdk.Context, epochIdentifier string, epochNumber uint64)
	// BeforeEpochStart new epoch is next block of epoch end block
	BeforeEpochStart(ctx sdk.Context, epochIdentifier string, epochNumber uint64)
}

var _ EpochHooks = MultiEpochHooks{}

// MultiEpochHooks combine multiple gamm hooks, all hook functions are run in array sequence.
type MultiEpochHooks []EpochHooks

func NewMultiEpochHooks(hooks ...EpochHooks) MultiEpochHooks {
	return hooks
}

// AfterEpochEnd is called when epoch is going to be ended, epochNumber is the number of epoch that is ending.
func (h MultiEpochHooks) AfterEpochEnd(ctx sdk.Context, epochIdentifier string, epochNumber uint64) {
	for i := range h {
		h[i].AfterEpochEnd(ctx, epochIdentifier, epochNumber)
	}
}

// BeforeEpochStart is called when epoch is going to be started, epochNumber is the number of epoch that is starting.
func (h MultiEpochHooks) BeforeEpochStart(ctx sdk.Context, epochIdentifier string, epochNumber uint64) {
	for i := range h {
		h[i].BeforeEpochStart(ctx, epochIdentifier, epochNumber)
	}
}
