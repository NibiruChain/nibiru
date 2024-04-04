package keeper

import (
	corestoretypes "cosmossdk.io/core/store"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/runtime"

	"cosmossdk.io/collections"

	"github.com/cosmos/cosmos-sdk/codec"

	"github.com/NibiruChain/nibiru/x/epochs/types"
)

type Keeper struct {
	cdc          codec.Codec
	storeService corestoretypes.KVStoreService
	hooks        types.EpochHooks

	Epochs collections.Map[string, types.EpochInfo]
}

func NewKeeper(cdc codec.Codec, storeKey *storetypes.KVStoreKey) Keeper {
	storeService := runtime.NewKVStoreService(storeKey)
	sb := collections.NewSchemaBuilder(storeService)

	return Keeper{
		cdc:          cdc,
		storeService: storeService,

		Epochs: collections.NewMap[string, types.EpochInfo](
			sb,
			collections.NewPrefix(1),
			storeKey.String(),
			collections.StringKey,
			codec.CollValue[types.EpochInfo](cdc),
		),
	}
}

// SetHooks Set the epoch hooks.
func (k *Keeper) SetHooks(eh types.EpochHooks) *Keeper {
	k.hooks = eh

	return k
}
