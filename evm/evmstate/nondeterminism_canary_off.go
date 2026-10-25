//go:build !nibiru_nondeterminism_canary

package evmstate

import (
	sdk "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types"
)

// nondeterminismCanary is a no-op in normal builds. The
// nibiru_nondeterminism_canary build tag swaps in a planted bug used to check
// that `just chaos determinism` catches app hash divergence.
func (k *Keeper) nondeterminismCanary(_ sdk.Context) {}
