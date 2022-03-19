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
func (k Keeper) SwapInput(pair string, dir ammv1.Direction, quoteAssetAmount sdk.Int) (sdk.Int, error) {
	if !k.ExistsPool(context.Background(), pair) {
		return sdk.Int{}, types.ErrPairNotSupported
	}

	if quoteAssetAmount.Equal(sdk.ZeroInt()) {
		return sdk.ZeroInt(), nil
	}

	if dir == ammv1.Direction_REMOVE_FROM_AMM {
		pool, err := k.getPool(pair)
		if err != nil {
			return sdk.Int{}, err
		}

		enoughReserve, err := types.PoolHasEnoughQuoteReserve(pool, quoteAssetAmount)
		if err != nil {
			return sdk.Int{}, err
		}
		if !enoughReserve {
			return sdk.Int{}, types.ErrOvertradingLimit

		}
	}

	return sdk.NewInt(1234), nil
}

// getPool returns the pool
func (k Keeper) getPool(pair string) (*ammv1.Pool, error) {
	p, err := k.store.PoolTable().Get(context.Background(), pair)
	if err != nil {
		return nil, err
	}

	return p, nil
}

// CreatePool creates a pool for a specific pair.
func (k Keeper) CreatePool(
	ctx context.Context,
	pair string,
	tradeLimitRatio sdk.Int, // integer with 6 decimals, 1_000_000 means 1.0
	quoteAssetReserve sdk.Int,
) error {
	pool := &ammv1.Pool{
		Pair:              pair,
		TradeLimitRatio:   tradeLimitRatio.String(),
		QuoteAssetReserve: quoteAssetReserve.String(),
	}

	return k.store.PoolTable().Save(ctx, pool)
}

// ExistsPool returns true if pool exists, false if not.
func (k Keeper) ExistsPool(ctx context.Context, pair string) bool {
	has, _ := k.store.PoolTable().Has(ctx, pair)

	return has
}
