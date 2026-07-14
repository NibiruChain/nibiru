package tendermint

import (
	sdk "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types"

	clienttypes "github.com/NibiruChain/nibiru/v2/lib/ibc-go/modules/core/02-client/types"
	"github.com/NibiruChain/nibiru/v2/lib/ibc-go/modules/core/exported"
)

// ExportMetadata exports all the consensus metadata in the client store so they can be included in clients genesis
// and imported by a ClientKeeper
func (cs ClientState) ExportMetadata(store sdk.KVStore) []exported.GenesisMetadata {
	gm := make([]exported.GenesisMetadata, 0)
	IterateConsensusMetadata(store, func(key, val []byte) bool {
		gm = append(gm, clienttypes.NewGenesisMetadata(key, val))
		return false
	})
	if len(gm) == 0 {
		return nil
	}
	return gm
}
