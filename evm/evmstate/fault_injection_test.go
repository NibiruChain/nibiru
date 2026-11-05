//go:build nibiru_fault_injection

package evmstate_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	abci "github.com/NibiruChain/nibiru/v2/bft/abci/types"
	"github.com/NibiruChain/nibiru/v2/evm"
	"github.com/NibiruChain/nibiru/v2/evm/evmtest"
)

// TestFaultInjectionChainGuard checks that the planted EndBlock write only
// happens on local test chains.
func TestFaultInjectionChainGuard(t *testing.T) {
	for _, tc := range []struct {
		chainID string
		written bool
	}{
		{chainID: "cataclysm-1", written: false},
		{chainID: "nibiru-testnet-2", written: false},
		{chainID: "nibiru-localnet-3", written: true},
	} {
		t.Run(tc.chainID, func(t *testing.T) {
			deps := evmtest.NewTestDeps()
			ctx := deps.Ctx().WithChainID(tc.chainID)
			deps.EvmKeeper.EndBlock(ctx, abci.RequestEndBlock{})

			store := ctx.KVStore(deps.App.GetKey(evm.StoreKey))
			require.Equal(t, tc.written, store.Has([]byte("fault_injection")))
		})
	}
}
