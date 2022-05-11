package keeper

import (
	"github.com/NibiruChain/nibiru/x/common"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// TODO(mercilex): implement
func (k Keeper) GetSettlementPrice(ctx sdk.Context, pair common.TokenPair) (sdk.Dec, error) {
	panic("impl")
}
