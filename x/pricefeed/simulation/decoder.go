package simulation

import (
	"bytes"
	"fmt"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/types/kv"

	"github.com/NibiruChain/nibiru/x/pricefeed/types"
)

// NewDecodeStore returns a decoder function closure that unmarshals the KVPair's
// Value to the corresponding gov type.
func NewDecodeStore(cdc codec.Codec) func(kvA, kvB kv.Pair) string {
	return func(kvA, kvB kv.Pair) string {
		switch {
		case bytes.Equal(kvA.Key[:1], []byte{0}):
			var currentPriceA, currentPriceB types.CurrentPrice
			cdc.MustUnmarshal(kvA.Value, &currentPriceA)
			cdc.MustUnmarshal(kvB.Value, &currentPriceB)
			return fmt.Sprintf("%v\n%v", currentPriceA, currentPriceB)
		case bytes.Equal(kvA.Key[:1], []byte{1}):
			var postedPriceA, postedPriceB types.PostedPrice
			cdc.MustUnmarshal(kvA.Value, &postedPriceA)
			cdc.MustUnmarshal(kvB.Value, &postedPriceB)
			return fmt.Sprintf("%v\n%v", postedPriceA, postedPriceB)
		case bytes.Equal(kvA.Key[:1], []byte{2}):
			var priceSnapshotA, priceSnapshotB types.PriceSnapshot
			cdc.MustUnmarshal(kvA.Value, &priceSnapshotA)
			cdc.MustUnmarshal(kvB.Value, &priceSnapshotB)
			return fmt.Sprintf("%v\n%v", priceSnapshotA, priceSnapshotB)
		default:
			panic(fmt.Sprintf("invalid pricefeed key prefix %X", kvA.Key[:1]))
		}
	}
}
