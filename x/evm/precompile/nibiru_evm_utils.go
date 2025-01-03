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

// EvmEventAbciEvent is the string key used to retrieve the "AbciEvent" Ethereum
// ABI event for a precompiled contract that implements the "INibiruEvm"
// interface from "Nibiru/x/evm/embeds/contracts/NibiruEvmUtils.sol".
const EvmEventAbciEvent = "AbciEvent"

// EmitEventAbciEvents adds a sequence of ABCI events to the EVM state DB so that
// they can be emitted at the end of the "EthereumTx". These events are indexed
// by their ABCI event type and help communicate non-EVM events in Ethereum-based
// block explorers and indexers by saving the event attributes in JSON form.
//
// Instead of ABI packing the non-indexed argument, this function encodes the
// [gethcore.Log].Data as a JSON string directly to optimize readbility in
// explorers without requiring the reader to decode using an ABI.
//
// Simply use ["encoding/hex".DecodeString] with the "0x" prefix removed to read
// the ABCI event.
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

		// eventType is the first (and only) indexed field
		topics[1] = EventTopicFromString(abciEvent.Type)

		attrsBz := AttrsToJSON(append([]abci.EventAttribute{
			{Key: "eventType", Value: abciEvent.Type},
		}, abciEvent.Attributes...))
		nonIndexedArgs, _ := event.Inputs.NonIndexed().Pack(string(attrsBz))
		db.AddLog(&gethcore.Log{
			Address:     emittingAddr,
			Topics:      topics,
			Data:        nonIndexedArgs,
			BlockNumber: blockNumber,
		})
	}
}

// AttrsToJSON creates a deterministic JSON encoding for the key-value tuples.
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

// EventTopicFromBytes creates a "Topic" hash for an EVM event log.
// An event topic is a 32-byte field used to index specific fields in a smart
// contract event. Topics make it possible to efficiently filter for and search
// events in transaction logs.
func EventTopicFromBytes(bz []byte) (topic gethcommon.Hash) {
	hash := crypto.Keccak256Hash(bz)
	copy(topic[:], hash[:])
	return topic
}

// EventTopicFromString creates a "Topic" hash for an EVM event log.
// An event topic is a 32-byte field used to index specific fields in a smart
// contract event. Topics make it possible to efficiently filter for and search
// events in transaction logs.
func EventTopicFromString(str string) (topic gethcommon.Hash) {
	return EventTopicFromBytes([]byte(str))
}
