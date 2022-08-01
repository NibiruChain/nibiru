package simulation

import (
	"bytes"
	"fmt"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/types/kv"

	"github.com/NibiruChain/nibiru/x/pricefeed/types"
)

var (
	oraclesNamespace     = []byte("oracles")
	activePairsNamespace = []byte("active pairs")
)

// NewDecodeStore returns a decoder function closure that unmarshals the KVPair's
// Value to the corresponding pricefeed type.
func NewDecodeStore(cdc codec.BinaryCodec) func(kvA, kvB kv.Pair) string {
	return func(kvA, kvB kv.Pair) string {
		switch {
		// prefix stores
		case bytes.HasPrefix(kvA.Key, oraclesNamespace):
			var oraclesA, oraclesB types.OraclesMarshaler
			cdc.MustUnmarshal(kvA.Value, &oraclesA)
			cdc.MustUnmarshal(kvB.Value, &oraclesB)
			return fmt.Sprintf("%v\n%v", oraclesA, oraclesB)
		case bytes.HasPrefix(kvA.Key, activePairsNamespace):
			var activePairsA, activePairsB types.ActivePairMarshaler
			cdc.MustUnmarshal(kvA.Value, &activePairsA)
			cdc.MustUnmarshal(kvB.Value, &activePairsB)
			return fmt.Sprintf("%v\n%v", activePairsA, activePairsB)
		// prefix stores

		case bytes.Equal(kvA.Key[:1], types.RawPriceFeedPrefix):
			var priceA, priceB types.PostedPrice
			cdc.MustUnmarshal(kvA.Value, &priceA)
			cdc.MustUnmarshal(kvB.Value, &priceB)
			return fmt.Sprintf("%v\n%v", priceA, priceB)

		case bytes.Equal(kvA.Key[:1], types.CurrentPricePrefix):
			var priceA, priceB types.CurrentPrice
			cdc.MustUnmarshal(kvA.Value, &priceA)
			cdc.MustUnmarshal(kvB.Value, &priceB)
			return fmt.Sprintf("%v\n%v", priceA, priceB)

		case bytes.Equal(kvA.Key[:1], types.PriceSnapshotPrefix):
			var snapA, snapB types.PriceSnapshot
			cdc.MustUnmarshal(kvA.Value, &snapA)
			cdc.MustUnmarshal(kvB.Value, &snapB)
			return fmt.Sprintf("%v\n%v", snapA, snapB)

		default:
			panic(fmt.Sprintf("invalid pricefeed key prefix %X", kvA.Key[:1]))
		}
	}
}
