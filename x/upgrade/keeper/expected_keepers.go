package keeper

import (
	sdk "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types"
	wasmtypes "github.com/NibiruChain/nibiru/v2/x/wasm/types"
)

// ContractBridge is the host boundary used by x-upgrade. App wiring supplies
// governance-permission code creation/instantiation and ordinary execute.
// Depending on this interface keeps the upgrade module independent of the
// concrete Wasm keeper, including in Wasm's own native test applications.
type ContractBridge interface {
	QuerySmartBounded(sdk.Context, sdk.AccAddress, []byte, uint64) ([]byte, uint64, error)
	Sudo(sdk.Context, sdk.AccAddress, []byte) ([]byte, error)
	SudoWithErrorCause(sdk.Context, sdk.AccAddress, []byte) ([]byte, error)
	IterateContractsByCreator(sdk.Context, sdk.AccAddress, func(sdk.AccAddress) bool)
	Execute(sdk.Context, sdk.AccAddress, sdk.AccAddress, []byte, sdk.Coins) ([]byte, error)
	CreateCode(sdk.Context, sdk.AccAddress, []byte, *wasmtypes.AccessConfig) (uint64, []byte, error)
	Instantiate2(sdk.Context, uint64, sdk.AccAddress, sdk.AccAddress, []byte, string, sdk.Coins, []byte, bool) (sdk.AccAddress, []byte, error)
	PredictContractAddress([]byte, sdk.AccAddress, []byte) sdk.AccAddress
}
