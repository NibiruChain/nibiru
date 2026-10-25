//go:build nibiru_nondeterminism_canary

package evmstate

import (
	"encoding/binary"
	"time"

	sdk "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types"
)

// nondeterminismCanaryKey sits outside the EVM collections namespaces, which
// are single small bytes (see evm/const.go).
var nondeterminismCanaryKey = []byte("nondeterminism_canary")

// nondeterminismCanary is a planted bug for `just chaos determinism`. It
// writes this node's wall-clock time to the EVM store on every EndBlock, so
// each validator commits a different app hash. NEVER ship a binary built
// with the nibiru_nondeterminism_canary tag.
func (k *Keeper) nondeterminismCanary(ctx sdk.Context) {
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, uint64(time.Now().UnixNano()))
	ctx.KVStore(k.storeKey).Set(nondeterminismCanaryKey, bz)
}
