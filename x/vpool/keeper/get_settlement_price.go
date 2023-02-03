package keeper

import (
	"errors"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/common/asset"
)

// TODO(mercilex): implement
func (k Keeper) GetSettlementPrice(ctx sdk.Context, pair asset.Pair) (sdk.Dec, error) {
	return sdk.Dec{}, errors.New("GetSettlementPrice has not been implemented yet.")
}
