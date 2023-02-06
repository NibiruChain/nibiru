package types // noalias

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	"github.com/NibiruChain/nibiru/x/common/asset"
	spottypes "github.com/NibiruChain/nibiru/x/spot/types"
)

// AccountKeeper defines the expected account keeper used for simulations (noalias)
type AccountKeeper interface {
	GetModuleAddress(name string) sdk.AccAddress
	GetModuleAccount(ctx sdk.Context, moduleName string) authtypes.ModuleAccountI
	GetAccount(ctx sdk.Context, addr sdk.AccAddress) authtypes.AccountI
	SetAccount(sdk.Context, authtypes.AccountI)
}

// BankKeeper defines the expected interface needed to retrieve account balances.
type BankKeeper interface {
	SpendableCoins(ctx sdk.Context, addr sdk.AccAddress) sdk.Coins
	MintCoins(ctx sdk.Context, moduleName string, amt sdk.Coins) error
	SendCoinsFromAccountToModule(
		ctx sdk.Context, senderAddr sdk.AccAddress, recipientModule string,
		amt sdk.Coins,
	) error
	SendCoinsFromModuleToAccount(
		ctx sdk.Context, senderModule string, recipientAddr sdk.AccAddress,
		amt sdk.Coins,
	) error
	BurnCoins(ctx sdk.Context, moduleName string, amt sdk.Coins) error
	GetSupply(ctx sdk.Context, denom string) sdk.Coin
	GetBalance(ctx sdk.Context, addr sdk.AccAddress, denom string) sdk.Coin
	GetAllBalances(ctx sdk.Context, addr sdk.AccAddress) sdk.Coins
}

type OracleKeeper interface {
	GetExchangeRate(ctx sdk.Context, pair asset.Pair) (sdk.Dec, error)
	GetExchangeRateTwap(ctx sdk.Context, pair asset.Pair) (sdk.Dec, error)
}

type SpotKeeper interface {
	FetchPoolFromPair(ctx sdk.Context, denomA string, denomB string,
	) (pool spottypes.Pool, err error)
	FetchPool(ctx sdk.Context, poolId uint64) (pool spottypes.Pool, err error)
}
