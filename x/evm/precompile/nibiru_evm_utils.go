package precompile

import (
	"bytes"
	"fmt"

	abci "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	gethcommon "github.com/ethereum/go-ethereum/common"
	gethcore "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/NibiruChain/nibiru/v2/x/common/set"
	"github.com/NibiruChain/nibiru/v2/x/evm/embeds"
	"github.com/NibiruChain/nibiru/v2/x/evm/statedb"
)

const EvmEventAbciEvent = "AbciEvent"

// EmitEventAbciEvents adds a sequence of ABCI events to the EVM state DB so that
// they can be emitted at the end of the "EthereumTx". These events are indexed
// by their ABCI event type and help communicate non-EVM events in Ethereum-based
// block explorers and indexers by saving the event attributes in JSON form.
// Note that event parsing is handled [gethabi.UnpackIntoMap]
func EmitEventAbciEvents(
	ctx sdk.Context,
	db *statedb.StateDB,
	abciEvents []sdk.Event,
	emittingAddr gethcommon.Address,
) {
	blockNumber := uint64(ctx.BlockHeight())
	event := embeds.SmartContract_Wasm.ABI.Events[EvmEventAbciEvent]
	for _, abciEvent := range abciEvents {
		// Why 2 topics? Because 2 = event ID + number of indexed event fields
		topics := make([]gethcommon.Hash, 2)
		topics[0] = event.ID

		// eventType is the first (and only) indexed event
		topics[1] = EventTopicFromString(abciEvent.Type)

		attrsBz := AttrsToJSON(append([]abci.EventAttribute{
			{Key: "eventType", Value: abciEvent.Type},
		}, abciEvent.Attributes...))
		nonIndexedArgs, _ := event.Inputs.NonIndexed().Pack(attrsBz)
		db.AddLog(&gethcore.Log{
			Address:     emittingAddr,
			Topics:      topics,
			Data:        nonIndexedArgs,
			BlockNumber: blockNumber,
		})
	}
}

// AttrsToJSON creates a deterministic JSON encoding for the
func AttrsToJSON(attrs []abci.EventAttribute) []byte {
	if len(attrs) == 0 {
		return []byte("")
	}
	keysSeen := set.New[string]()

	// Create JSON object from the key-value tuples
	var buf bytes.Buffer
	buf.WriteByte('{')
	for i, attr := range attrs {
		// Keys must be unique to guarantee valid JSON object
		if keysSeen.Has(attr.Key) {
			continue
		}
		keysSeen.Add(attr.Key)

		if i > 0 {
			buf.WriteByte(',')
		}

		// Quote key and value
		_, _ = fmt.Fprintf(&buf, `"%s":"%s"`, attr.Key, attr.Value)
	}
	buf.WriteByte('}')

	return buf.Bytes()
}

// EventTopicFromBytes creates an [abi.Event]
func EventTopicFromBytes(bz []byte) (topic gethcommon.Hash) {
	hash := crypto.Keccak256Hash(bz)
	copy(topic[:], hash[:])
	return topic
}

func EventTopicFromString(str string) (topic gethcommon.Hash) {
	return EventTopicFromBytes([]byte(str))
}
