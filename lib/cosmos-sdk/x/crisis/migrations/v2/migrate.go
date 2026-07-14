package v2

import (
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/codec"
	storetypes "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/store/types"
	sdk "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types"
	sdkerrors "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types/errors"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/crisis/exported"
)

const (
	ModuleName = "crisis"
)

var (
	ConstantFee    = []byte("ConstantFee")
	ConstantFeeKey = []byte{0x01}
)

// MigrateStore migrates the x/crisis module state from the consensus version 1 to
// version 2. Specifically, it takes the `ConstantFee` parameter that is currently stored
// and managed by the x/params module and stores it directly into the x/crisis
// module state.
func MigrateStore(ctx sdk.Context, storeKey storetypes.StoreKey, legacySubspace exported.Subspace, cdc codec.BinaryCodec) error {
	store := ctx.KVStore(storeKey)
	var currConstantFee sdk.Coin
	legacySubspace.Get(ctx, ConstantFee, &currConstantFee)

	if !currConstantFee.IsValid() {
		return sdkerrors.ErrInvalidCoins.Wrap("constant fee")
	}

	bz, err := cdc.Marshal(&currConstantFee)
	if err != nil {
		return err
	}

	store.Set(ConstantFeeKey, bz)

	return nil
}
