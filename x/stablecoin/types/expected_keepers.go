package types // noalias

import (
	dextypes "github.com/NibiruChain/nibiru/x/dex/types"
	pftypes "github.com/NibiruChain/nibiru/x/pricefeed/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth/types"
)

// AccountKeeper defines the expected account keeper used for simulations (noalias)
type AccountKeeper interface {
	GetModuleAddress(name string) sdk.AccAddress
	GetModuleAccount(ctx sdk.Context, moduleName string) types.ModuleAccountI
	GetAccount(ctx sdk.Context, addr sdk.AccAddress) types.AccountI
	SetAccount(sdk.Context, types.AccountI)
}

// BankKeeper defines the expected interface needed to retrieve account balances.
type BankKeeper interface {
	SpendableCoins(ctx sdk.Context, addr sdk.AccAddress) sdk.Coins
	MintCoins(ctx sdk.Context, moduleName string, amt sdk.Coins) error
	SendCoinsFromAccountToModule(ctx sdk.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins) error
	SendCoinsFromModuleToAccount(ctx sdk.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error
	BurnCoins(ctx sdk.Context, moduleName string, amt sdk.Coins) error
	GetSupply(ctx sdk.Context, denom string) sdk.Coin
	GetBalance(ctx sdk.Context, addr sdk.AccAddress, denom string) sdk.Coin
	GetAllBalances(ctx sdk.Context, addr sdk.AccAddress) sdk.Coins
}

type PriceKeeper interface {
	GetCurrentTWAPPrice(ctx sdk.Context, token0 string, token1 string) (pftypes.CurrentTWAP, error)
	GetCurrentPrice(ctx sdk.Context, token0 string, token1 string) (pftypes.CurrentPrice, error)
	GetCurrentPrices(ctx sdk.Context) pftypes.CurrentPrices
	GetRawPrices(ctx sdk.Context, marketId string) pftypes.PostedPrices
	GetPair(ctx sdk.Context, pairID string) (pftypes.Pair, bool)
	GetPairs(ctx sdk.Context) pftypes.Pairs
	GetOracle(ctx sdk.Context, pairID string, address sdk.AccAddress) (sdk.AccAddress, error)
	GetOracles(ctx sdk.Context, pairID string) ([]sdk.AccAddress, error)
	SetCurrentPrices(ctx sdk.Context, token0 string, token1 string) error
}

type DexKeeper interface {
	GetFromPair(ctx sdk.Context, denomA string, denomB string) (poolId uint64, err error)
	FetchPool(ctx sdk.Context, poolId uint64) (pool dextypes.Pool, err error)
}
