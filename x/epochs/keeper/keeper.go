package keeper

import (
	storetypes "github.com/cosmos/cosmos-sdk/store/types"

	"github.com/NibiruChain/collections"

	"github.com/cosmos/cosmos-sdk/codec"

	"github.com/NibiruChain/nibiru/x/epochs/types"
)

type Keeper struct {
	cdc      codec.Codec
	storeKey storetypes.StoreKey
	hooks    types.EpochHooks

	Epochs collections.Map[string, types.EpochInfo]
}

func NewKeeper(cdc codec.Codec, storeKey storetypes.StoreKey) Keeper {
	return Keeper{
		cdc:      cdc,
		storeKey: storeKey,

		Epochs: collections.NewMap[string, types.EpochInfo](storeKey, 1, collections.StringKeyEncoder, collections.ProtoValueEncoder[types.EpochInfo](cdc)),
	}
}

// SetHooks Set the epoch hooks.
func (k *Keeper) SetHooks(eh types.EpochHooks) *Keeper {
	k.hooks = eh

	return k
}
