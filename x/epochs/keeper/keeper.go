package keeper

import (
	"fmt"

	storetypes "cosmossdk.io/store/types"

	"cosmossdk.io/collections"

	"github.com/cometbft/cometbft/libs/log"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

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

// SetHooks Set the gamm hooks.
func (k *Keeper) SetHooks(eh types.EpochHooks) *Keeper {
	if k.hooks != nil {
		panic("cannot set epochs hooks twice")
	}

	k.hooks = eh

	return k
}

func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", fmt.Sprintf("x/%s", types.ModuleName))
}
