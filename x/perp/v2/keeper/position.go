package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/collections"
	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/perp/v2/types"
)

func (k Keeper) GetPosition(ctx sdk.Context, pair asset.Pair, version uint64, account sdk.AccAddress) (types.Position, error) {
	position, err := k.Positions.Get(ctx, collections.Join(collections.Join(pair, version), account))
	if err != nil {
		return types.Position{}, types.ErrPositionNotFound
	}

	return position, nil
}

func (k Keeper) DeletePosition(ctx sdk.Context, pair asset.Pair, version uint64, account sdk.AccAddress) error {
	err := k.Positions.Delete(ctx, collections.Join(collections.Join(pair, version), account))
	if err != nil {
		return types.ErrPositionNotFound
	}

	return nil
}

func (k Keeper) SavePosition(ctx sdk.Context, pair asset.Pair, version uint64, account sdk.AccAddress, position types.Position) {
	k.Positions.Insert(ctx, collections.Join(collections.Join(position.Pair, version), account), position)
}
