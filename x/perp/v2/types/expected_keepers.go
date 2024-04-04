package types

import (
	"context"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/epochs/types"
)

// ----------------------------------------------------------
// Keeper Interfaces
// ----------------------------------------------------------

// AccountKeeper defines the expected account keeper used for simulations (noalias)
type AccountKeeper interface {
	GetAccount(ctx context.Context, addr sdk.AccAddress) sdk.AccountI
	GetModuleAddress(name string) sdk.AccAddress
	GetModuleAccount(ctx context.Context, moduleName string) sdk.ModuleAccountI
}

// BankKeeper defines the expected interface needed to retrieve account balances.
type BankKeeper interface {
	SpendableCoins(ctx context.Context, addr sdk.AccAddress) sdk.Coins
	MintCoins(ctx context.Context, moduleName string, amt sdk.Coins) error
	SendCoinsFromAccountToModule(
		ctx context.Context, senderAddr sdk.AccAddress, recipientModule string,
		amt sdk.Coins,
	) error
	SendCoinsFromModuleToModule(
		ctx context.Context, senderModule string, recipientModule string, amt sdk.Coins) error
	SendCoinsFromModuleToAccount(
		ctx context.Context, senderModule string, recipientAddr sdk.AccAddress,
		amt sdk.Coins,
	) error
	GetBalance(ctx context.Context, addr sdk.AccAddress, denom string) sdk.Coin
	GetAllBalances(ctx context.Context, addr sdk.AccAddress) sdk.Coins
}

type OracleKeeper interface {
	GetExchangeRate(ctx sdk.Context, pair asset.Pair) (math.LegacyDec, error)
	GetExchangeRateTwap(ctx sdk.Context, pair asset.Pair) (math.LegacyDec, error)
	SetPrice(ctx sdk.Context, pair asset.Pair, price math.LegacyDec)
}

type EpochKeeper interface {
	// GetEpochInfo returns epoch info by identifier.
	GetEpochInfo(ctx sdk.Context, identifier string) (types.EpochInfo, error)
}

type SudoKeeper interface {
	// CheckPermissions Checks if a contract is contained within the set of sudo
	// contracts defined in the x/sudo module. These smart contracts are able to
	// execute certain permissioned functions.
	CheckPermissions(contract sdk.AccAddress, ctx sdk.Context) error
}
