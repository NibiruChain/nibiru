package simulation

import (
	"fmt"
	"github.com/NibiruChain/nibiru/collections"

	gogotypes "github.com/gogo/protobuf/types"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/kv"

	"github.com/NibiruChain/nibiru/x/oracle/types"
)

// NewDecodeStore returns a decoder function closure that unmarshals the KVPair's
// Value to the corresponding oracle type.
func NewDecodeStore(cdc codec.Codec) func(kvA, kvB kv.Pair) string {
	return func(kvA, kvB kv.Pair) string {
		switch kvA.Key[0] {
		case 1:
			return fmt.Sprintf("%v\n%v", collections.DecValueEncoder.ValueDecode(kvA.Value), collections.DecValueEncoder.ValueDecode(kvB.Value))
		case 2:
			return fmt.Sprintf("%v\n%v", sdk.AccAddress(kvA.Value), sdk.AccAddress(kvB.Value))
		case 3:
			var counterA, counterB gogotypes.UInt64Value
			cdc.MustUnmarshal(kvA.Value, &counterA)
			cdc.MustUnmarshal(kvB.Value, &counterB)
			return fmt.Sprintf("%v\n%v", counterA.Value, counterB.Value)
		case 4:
			var prevoteA, prevoteB types.AggregateExchangeRatePrevote
			cdc.MustUnmarshal(kvA.Value, &prevoteA)
			cdc.MustUnmarshal(kvB.Value, &prevoteB)
			return fmt.Sprintf("%v\n%v", prevoteA, prevoteB)
		case 5:
			var voteA, voteB types.AggregateExchangeRateVote
			cdc.MustUnmarshal(kvA.Value, &voteA)
			cdc.MustUnmarshal(kvB.Value, &voteB)
			return fmt.Sprintf("%v\n%v", voteA, voteB)
		case 6:
			return fmt.Sprintf("%s\n%s", types.ExtractPairFromPairKey(kvA.Key), types.ExtractPairFromPairKey(kvB.Key))
		default:
			panic(fmt.Sprintf("invalid oracle key prefix %X", kvA.Key[:1]))
		}
	}
}
