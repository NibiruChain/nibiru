//go:build nibiru_fault_injection

package evmstate

import (
	"encoding/binary"
	"time"

	sdk "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types"
)

// faultInjectionKey sits outside the EVM collections namespaces, which are
// single small bytes (see evm/const.go).
var faultInjectionKey = []byte("fault_injection")

// faultInjection is a planted bug for `just chaos determinism-selftest`. It
// writes this node's wall-clock time to the EVM store on every EndBlock, so
// each validator commits a different app hash. NEVER ship a binary built
// with the nibiru_fault_injection tag.
func (k *Keeper) faultInjection(ctx sdk.Context) {
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, uint64(time.Now().UnixNano()))
	ctx.KVStore(k.storeKey).Set(faultInjectionKey, bz)
}
