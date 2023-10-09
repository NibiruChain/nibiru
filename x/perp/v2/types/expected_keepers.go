package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/epochs/types"

	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
)

// ----------------------------------------------------------
// Keeper Interfaces
// ----------------------------------------------------------

// AccountKeeper defines the expected account keeper used for simulations (noalias)
type AccountKeeper interface {
	GetAccount(ctx sdk.Context, addr sdk.AccAddress) authtypes.AccountI
	GetModuleAddress(name string) sdk.AccAddress
	GetModuleAccount(ctx sdk.Context, moduleName string) authtypes.ModuleAccountI
}

// BankKeeper defines the expected interface needed to retrieve account balances.
type BankKeeper interface {
	SpendableCoins(ctx sdk.Context, addr sdk.AccAddress) sdk.Coins
	MintCoins(ctx sdk.Context, moduleName string, amt sdk.Coins) error
	SendCoinsFromAccountToModule(
		ctx sdk.Context, senderAddr sdk.AccAddress, recipientModule string,
		amt sdk.Coins,
	) error
	SendCoinsFromModuleToModule(
		ctx sdk.Context, senderModule string, recipientModule string, amt sdk.Coins) error
	SendCoinsFromModuleToAccount(
		ctx sdk.Context, senderModule string, recipientAddr sdk.AccAddress,
		amt sdk.Coins,
	) error
	GetBalance(ctx sdk.Context, addr sdk.AccAddress, denom string) sdk.Coin
	GetAllBalances(ctx sdk.Context, addr sdk.AccAddress) sdk.Coins
}

type OracleKeeper interface {
	GetExchangeRate(ctx sdk.Context, pair asset.Pair) (sdk.Dec, error)
	GetExchangeRateTwap(ctx sdk.Context, pair asset.Pair) (sdk.Dec, error)
	SetPrice(ctx sdk.Context, pair asset.Pair, price sdk.Dec)
}

type EpochKeeper interface {
	// GetEpochInfo returns epoch info by identifier.
	GetEpochInfo(ctx sdk.Context, identifier string) (types.EpochInfo, error)
}

type StakingKeeper interface {
	BondDenom(ctx sdk.Context) string
}
