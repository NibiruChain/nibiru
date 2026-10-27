//go:build !nibiru_fault_injection

package evmstate

import (
	sdk "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types"
)

// faultInjection is a no-op in normal builds. The nibiru_fault_injection
// build tag swaps in a planted bug used to check that
// `just chaos determinism` catches app hash divergence.
func (k *Keeper) faultInjection(_ sdk.Context) {}
