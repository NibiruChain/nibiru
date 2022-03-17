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
func (k Keeper) SwapInput(dir types.Direction, amount sdk.Coin) (sdk.Int, error) {
	if amount.Denom != types.StableDenom {
		return sdk.ZeroInt(), types.ErrStableNotSupported
	}

	if amount.Amount.Equal(sdk.ZeroInt()) {
		return sdk.ZeroInt(), nil
	}

	if dir == types.REMOVE_FROM_AMM {
	}

	return sdk.NewInt(1234), nil
}

func (k Keeper) getQuoteAssetReserve(pair string) sdk.Int {
}
