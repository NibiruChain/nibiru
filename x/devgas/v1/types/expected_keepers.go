package types

import (
	"context"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"

	// "github.com/cosmos/cosmos-sdk/client"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// AccountKeeper defines the expected interface needed to retrieve account info.
type AccountKeeper interface {
	GetModuleAddress(moduleName string) sdk.AccAddress
	GetModuleAccount(ctx context.Context, name string) sdk.ModuleAccountI

	HasAccount(ctx context.Context, addr sdk.AccAddress) bool
	GetAccount(ctx context.Context, addr sdk.AccAddress) (account sdk.AccountI)
}

// BankKeeper defines the expected interface needed to retrieve account balances.
type BankKeeper interface {
	SendCoinsFromModuleToAccount(ctx context.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error
	SendCoinsFromAccountToModule(ctx context.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins) error
	BurnCoins(ctx context.Context, moduleName string, amt sdk.Coins) error
	GetBalance(ctx context.Context, addr sdk.AccAddress, denom string) sdk.Coin
	GetAllBalances(ctx context.Context, addr sdk.AccAddress) sdk.Coins
}

// WasmKeeper defines the expected interface needed to retrieve cosmwasm contracts.
type WasmKeeper interface {
	GetContractInfo(ctx context.Context, contractAddr sdk.AccAddress) (wasmtypes.ContractInfo, error)
}
