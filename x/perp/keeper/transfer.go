package keeper

import (
	"github.com/NibiruChain/nibiru/x/common"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Transfer moves the provided coins from the module address to the trader.
// It returns the actual moved coins as we do not know which is the quote asset.
func (k Keeper) Transfer(ctx sdk.Context, pair common.TokenPair, trader sdk.AccAddress, amount sdk.Int) (transferred sdk.Coins, err error) {
	panic("impl")
}
