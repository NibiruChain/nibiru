package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// EpochHooks defines a set of lifecycle hooks to occur in the ABCI BeginBlock
// hooks based on temporal epochs.
type EpochHooks interface {
	// AfterEpochEnd the first block whose timestamp is after the duration is
	// counted as the end of the epoch
	AfterEpochEnd(ctx sdk.Context, epochIdentifier string, epochNumber uint64)
	// BeforeEpochStart new epoch is next block of epoch end block
	BeforeEpochStart(ctx sdk.Context, epochIdentifier string, epochNumber uint64)
}

var _ EpochHooks = MultiEpochHooks{}

// MultiEpochHooks combines multiple [EpochHooks]. All hook functions are
// executed sequentially in the order of the slice.
type MultiEpochHooks []EpochHooks

func NewMultiEpochHooks(hooks ...EpochHooks) MultiEpochHooks {
	return hooks
}

// AfterEpochEnd runs logic at the end of an epoch.
//
//   - epochIdentifier: The unique identifier of specific epoch. Ex: "30 min", "1 min".
//   - epochNumber: Counter for the specific epoch type identified.
func (h MultiEpochHooks) AfterEpochEnd(ctx sdk.Context, epochIdentifier string, epochNumber uint64) {
	for i := range h {
		h[i].AfterEpochEnd(ctx, epochIdentifier, epochNumber)
	}
}

// BeforeEpochStart runs logic in the ABCI BeginBlocker right before an epoch
// starts.
//
//   - epochIdentifier: The unique identifier of specific epoch. Ex: "30 min", "1 min".
//   - epochNumber: Counter for the specific epoch type identified.
func (h MultiEpochHooks) BeforeEpochStart(ctx sdk.Context, epochIdentifier string, epochNumber uint64) {
	for i := range h {
		h[i].BeforeEpochStart(ctx, epochIdentifier, epochNumber)
	}
}
