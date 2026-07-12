package migrations

import (
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/codec"
	sdk "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types"
	sdkerrors "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types/errors"

	clienttypes "github.com/NibiruChain/nibiru/v2/lib/ibc-go/modules/core/02-client/types"
	"github.com/NibiruChain/nibiru/v2/lib/ibc-go/modules/core/exported"
	ibctm "github.com/NibiruChain/nibiru/v2/lib/ibc-go/modules/light-clients/07-tendermint"
)

// PruneExpiredConsensusStates prunes all expired tendermint consensus states. This function
// may optionally be called during in-place store migrations. The ibc store key must be provided.
func PruneExpiredConsensusStates(ctx sdk.Context, cdc codec.BinaryCodec, clientKeeper ClientKeeper) (int, error) {
	var clientIDs []string
	clientKeeper.IterateClientStates(ctx, []byte(exported.Tendermint), func(clientID string, _ exported.ClientState) bool {
		clientIDs = append(clientIDs, clientID)
		return false
	})

	// keep track of the total consensus states pruned so chains can
	// understand how much space is saved when the migration is run
	var totalPruned int

	for _, clientID := range clientIDs {
		clientStore := clientKeeper.ClientStore(ctx, clientID)

		clientState, ok := clientKeeper.GetClientState(ctx, clientID)
		if !ok {
			return 0, sdkerrors.Wrapf(clienttypes.ErrClientNotFound, "clientID %s", clientID)
		}

		tmClientState, ok := clientState.(*ibctm.ClientState)
		if !ok {
			return 0, sdkerrors.Wrap(clienttypes.ErrInvalidClient, "client state is not tendermint even though client id contains 07-tendermint")
		}

		totalPruned += ibctm.PruneAllExpiredConsensusStates(ctx, clientStore, cdc, tmClientState)
	}

	clientLogger := clientKeeper.Logger(ctx)
	clientLogger.Info("pruned expired tendermint consensus states", "total", totalPruned)

	return totalPruned, nil
}
