package keeper

import (
	"github.com/MatrixDao/matrix/x/amm/types"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func NewKeeper(storeKey storetypes.StoreKey) Keeper {
	return Keeper{storeKey: storeKey}
}

type Keeper struct {
	storeKey storetypes.StoreKey
}

// SwapInput swaps pair token
func (k Keeper) SwapInput(dir types.Direction, amount sdk.Coin) error {
	return nil
}
