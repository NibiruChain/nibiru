package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"cosmossdk.io/collections"

	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/perp/v2/types"
)

// GetPosition returns the position or types.ErrPositionNotFound if it does not exist.
func (k Keeper) GetPosition(ctx sdk.Context, pair asset.Pair, version uint64, account sdk.AccAddress) (types.Position, error) {
	position, err := k.Positions.Get(ctx, collections.Join(collections.Join(pair, version), account))
	if err != nil {
		return types.Position{}, types.ErrPositionNotFound
	}

	return position, nil
}

func (k Keeper) DeletePosition(ctx sdk.Context, pair asset.Pair, version uint64, account sdk.AccAddress) error {
	err := k.Positions.Remove(ctx, collections.Join(collections.Join(pair, version), account))
	if err != nil {
		return types.ErrPositionNotFound
	}

	return nil
}

func (k Keeper) SavePosition(ctx sdk.Context, pair asset.Pair, version uint64, account sdk.AccAddress, position types.Position) {
	k.Positions.Set(ctx, collections.Join(collections.Join(position.Pair, version), account), position)
}
