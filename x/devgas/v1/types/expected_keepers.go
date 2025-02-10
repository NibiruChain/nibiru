package types

import (
	wasm "github.com/CosmWasm/wasmd/x/wasm/types"

	// "github.com/cosmos/cosmos-sdk/client"
	sdk "github.com/cosmos/cosmos-sdk/types"
	auth "github.com/cosmos/cosmos-sdk/x/auth/types"
)

// AccountKeeper defines the expected interface needed to retrieve account info.
type AccountKeeper interface {
	GetModuleAddress(moduleName string) sdk.AccAddress
	GetModuleAccount(ctx sdk.Context, name string) auth.ModuleAccountI

	HasAccount(ctx sdk.Context, addr sdk.AccAddress) bool
	GetAccount(ctx sdk.Context, addr sdk.AccAddress) (account auth.AccountI)
}

// BankKeeper defines the expected interface needed to retrieve account balances.
type BankKeeper interface {
	SendCoinsFromModuleToAccount(ctx sdk.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error
	SendCoinsFromAccountToModule(ctx sdk.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins) error
	BurnCoins(ctx sdk.Context, moduleName string, amt sdk.Coins) error
	GetBalance(ctx sdk.Context, addr sdk.AccAddress, denom string) sdk.Coin
	GetAllBalances(ctx sdk.Context, addr sdk.AccAddress) sdk.Coins
}

// WasmKeeper defines the expected interface needed to retrieve cosmwasm contracts.
type WasmKeeper interface {
	GetContractInfo(ctx sdk.Context, contractAddr sdk.AccAddress) (wasm.ContractInfo, error)
}
