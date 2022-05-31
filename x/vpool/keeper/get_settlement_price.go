package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/common"
)

// TODO(mercilex): implement
func (k Keeper) GetSettlementPrice(ctx sdk.Context, pair common.AssetPair) (sdk.Dec, error) {
	panic("impl")
}
