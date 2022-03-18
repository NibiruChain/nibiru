package keeper

import (
	"context"
	ammv1 "github.com/MatrixDao/matrix/api/amm"
	"github.com/MatrixDao/matrix/x/amm/types"
	"github.com/cosmos/cosmos-sdk/orm/model/ormdb"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/protobuf/reflect/protoreflect"
)

var PoolSchema = ormdb.ModuleSchema{
	FileDescriptors: map[uint32]protoreflect.FileDescriptor{
		1: ammv1.File_amm_amm_proto,
	},
}

func NewKeeper(store ammv1.AmmStore) Keeper {
	return Keeper{
		store: store,
	}
}

type Keeper struct {
	store ammv1.AmmStore
}

// SwapInput swaps pair token
func (k Keeper) SwapInput(dir ammv1.Direction, amount sdk.Coin) (sdk.Int, error) {
	if amount.Denom != types.StableDenom {
		return sdk.ZeroInt(), types.ErrStableNotSupported
	}

	if amount.Amount.Equal(sdk.ZeroInt()) {
		return sdk.ZeroInt(), nil
	}

	if dir == ammv1.Direction_REMOVE_FROM_AMM {
	}

	return sdk.NewInt(1234), nil
}

func (k Keeper) getQuoteAssetReserve(pair string) sdk.Int {
	return sdk.ZeroInt()
}

func (k Keeper) CreatePool(ctx context.Context, pair string) error {
	pool := &ammv1.Pool{
		Pair:              pair,
		QuoteAssetReserve: "1234",
	}

	return k.store.PoolTable().Save(ctx, pool)
}

func (k Keeper) GetPool(ctx context.Context, pair string) (*ammv1.Pool, error) {
	return k.store.PoolTable().Get(ctx, pair)
}
