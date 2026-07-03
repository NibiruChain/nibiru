package keeper

import (
	storetypes "github.com/cosmos/cosmos-sdk/store/types"

	"github.com/NibiruChain/nibiru/v2/x/collections"

	"github.com/cosmos/cosmos-sdk/codec"

	"github.com/NibiruChain/nibiru/v2/x/epochs"
)

type Keeper struct {
	cdc      codec.Codec
	storeKey storetypes.StoreKey
	hooks    epochs.EpochHooks

	Epochs collections.Map[string, epochs.EpochInfo]
}

func NewKeeper(cdc codec.Codec, storeKey storetypes.StoreKey) *Keeper {
	return &Keeper{
		cdc:      cdc,
		storeKey: storeKey,

		Epochs: collections.NewMap[string, epochs.EpochInfo](storeKey, 1, collections.StringKeyEncoder, collections.ProtoValueEncoder[epochs.EpochInfo](cdc)),
	}
}

// SetHooks Set the epoch hooks.
func (k *Keeper) SetHooks(eh epochs.EpochHooks) {
	k.hooks = eh
}
