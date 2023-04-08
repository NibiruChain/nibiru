package simulation

import (
	"bytes"
	"fmt"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/types/kv"

	"github.com/NibiruChain/nibiru/x/perp/amm/types"
)

// NewDecodeStore returns a decoder function closure that unmarshals the KVPair's
// Value to the corresponding nft type.
func NewDecodeStore(cdc codec.BinaryCodec) func(kvA, kvB kv.Pair) string {
	return func(kvA, kvB kv.Pair) string {
		switch {
		case bytes.Equal(kvA.Key[:1], []byte{0x0}):
			var vpoolA, vpoolB types.Vpool
			cdc.MustUnmarshal(kvA.Value, &vpoolA)
			cdc.MustUnmarshal(kvB.Value, &vpoolB)
			return fmt.Sprintf("%v\n%v", vpoolA, vpoolB)
		case bytes.Equal(kvA.Key[:1], []byte{0x1}):
			var snapshotA, snapshotB types.ReserveSnapshot
			cdc.MustUnmarshal(kvA.Value, &snapshotA)
			cdc.MustUnmarshal(kvB.Value, &snapshotB)
			return fmt.Sprintf("%v\n%v", snapshotA, snapshotB)
		default:
			panic(fmt.Sprintf("invalid vpool key %X", kvA.Key))
		}
	}
}
