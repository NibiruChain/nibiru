package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/v2/x/devgas/v1/types"
)

// GetFeeShare returns the FeeShare for a registered contract
func (k Keeper) GetFeeShare(
	ctx sdk.Context,
	contract sdk.Address,
) (devGas types.FeeShare, isFound bool) {
	isFound = true
	devGas, err := k.DevGasStore.Get(ctx, contract.String())
	if err != nil {
		isFound = false
	}
	return devGas, isFound
}

// SetFeeShare stores the FeeShare for a registered contract, then iterates
// over every registered Indexer and instructs them to create the relationship
// between the primary key PK and the object v.
func (k Keeper) SetFeeShare(ctx sdk.Context, feeshare types.FeeShare) {
	k.DevGasStore.Insert(ctx, feeshare.ContractAddress, feeshare)
}

// IsFeeShareRegistered checks if a contract was registered for receiving
// transaction fees
func (k Keeper) IsFeeShareRegistered(
	ctx sdk.Context,
	contract sdk.Address,
) (isRegistered bool) {
	_, err := k.DevGasStore.Get(ctx, contract.String())
	return err == nil
}
