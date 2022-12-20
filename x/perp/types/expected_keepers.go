package types

//go:generate  mockgen -destination=../../testutil/mock/perp_interfaces.go -package=mock github.com/NibiruChain/nibiru/x/perp/types AccountKeeper,BankKeeper,PricefeedKeeper,VpoolKeeper,EpochKeeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/epochs/types"

	"time"

	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	vpooltypes "github.com/NibiruChain/nibiru/x/vpool/types"
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
}

type PricefeedKeeper interface {
	GetCurrentPrice(ctx sdk.Context, token0 string, token1 string) (sdk.Dec, error)
	GatherRawPrices(ctx sdk.Context, token0 string, token1 string) error
	IsActivePair(ctx sdk.Context, pairID string) bool
	GetCurrentTWAP(ctx sdk.Context, token0 string, token1 string) (sdk.Dec, error)
}

type VpoolKeeper interface {
	SwapBaseForQuote(
		ctx sdk.Context,
		pair common.AssetPair,
		dir vpooltypes.Direction,
		baseAssetAmount sdk.Dec,
		quoteAmountLimit sdk.Dec,
		skipFluctuationLimitCheck bool,
	) (sdk.Dec, error)

	SwapQuoteForBase(
		ctx sdk.Context,
		pair common.AssetPair,
		dir vpooltypes.Direction,
		quoteAssetAmount sdk.Dec,
		baseAmountLimit sdk.Dec,
		skipFluctuationLimitCheck bool,
	) (sdk.Dec, error)

	GetBaseAssetTWAP(
		ctx sdk.Context,
		pair common.AssetPair,
		direction vpooltypes.Direction,
		baseAssetAmount sdk.Dec,
		lookbackInterval time.Duration,
	) (quoteAssetAmount sdk.Dec, err error)

	GetBaseAssetPrice(
		ctx sdk.Context,
		pair common.AssetPair,
		direction vpooltypes.Direction,
		baseAssetAmount sdk.Dec,
	) (quoteAssetAmount sdk.Dec, err error)

	GetQuoteAssetPrice(
		ctx sdk.Context,
		pair common.AssetPair,
		dir vpooltypes.Direction,
		quoteAmount sdk.Dec,
	) (baseAssetAmount sdk.Dec, err error)

	GetMarkPrice(
		ctx sdk.Context,
		pair common.AssetPair,
	) (price sdk.Dec, err error)

	GetMarkPriceTWAP(
		ctx sdk.Context,
		pair common.AssetPair,
		lookbackInterval time.Duration,
	) (quoteAssetAmount sdk.Dec, err error)

	GetAllPools(ctx sdk.Context) []vpooltypes.Vpool

	IsOverSpreadLimit(ctx sdk.Context, pair common.AssetPair) bool
	GetMaintenanceMarginRatio(ctx sdk.Context, pair common.AssetPair) sdk.Dec
	GetMaxLeverage(ctx sdk.Context, pair common.AssetPair) sdk.Dec
	ExistsPool(ctx sdk.Context, pair common.AssetPair) bool
	GetSettlementPrice(ctx sdk.Context, pair common.AssetPair) (sdk.Dec, error)
	GetLastSnapshot(ctx sdk.Context, pool vpooltypes.Vpool) (vpooltypes.ReserveSnapshot, error)
}

type EpochKeeper interface {
	// GetEpochInfo returns epoch info by identifier.
	GetEpochInfo(ctx sdk.Context, identifier string) types.EpochInfo
}
