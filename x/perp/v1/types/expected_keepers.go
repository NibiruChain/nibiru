package types

//go:generate  mockgen -destination=../../common/testutil/mock/perp_interfaces.go -package=mock github.com/NibiruChain/nibiru/x/perp/types AccountKeeper,BankKeeper,OracleKeeper,PerpAmmKeeper,EpochKeeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/epochs/types"

	"time"

	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	perpammtypes "github.com/NibiruChain/nibiru/x/perp/v1/amm/types"
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

type PerpAmmKeeper interface {
	SwapBaseForQuote(
		ctx sdk.Context,
		market perpammtypes.Market,
		dir perpammtypes.Direction,
		baseAssetAmount sdk.Dec,
		quoteAmountLimit sdk.Dec,
		skipFluctuationLimitCheck bool,
	) (perpammtypes.Market, sdk.Dec, error)

	SwapQuoteForBase(
		ctx sdk.Context,
		market perpammtypes.Market,
		dir perpammtypes.Direction,
		quoteAssetAmount sdk.Dec,
		baseAmountLimit sdk.Dec,
		skipFluctuationLimitCheck bool,
	) (perpammtypes.Market, sdk.Dec, error)

	GetBaseAssetTWAP(
		ctx sdk.Context,
		pair asset.Pair,
		direction perpammtypes.Direction,
		baseAssetAmount sdk.Dec,
		lookbackInterval time.Duration,
	) (quoteAssetAmount sdk.Dec, err error)

	GetBaseAssetPrice(
		market perpammtypes.Market,
		direction perpammtypes.Direction,
		baseAssetAmount sdk.Dec,
	) (quoteAssetAmount sdk.Dec, err error)

	GetMarkPrice(
		ctx sdk.Context,
		pair asset.Pair,
	) (price sdk.Dec, err error)

	GetMarkPriceTWAP(
		ctx sdk.Context,
		pair asset.Pair,
		lookbackInterval time.Duration,
	) (quoteAssetAmount sdk.Dec, err error)

	GetAllPools(ctx sdk.Context) []perpammtypes.Market
	GetPool(ctx sdk.Context, pair asset.Pair) (perpammtypes.Market, error)

	IsOverSpreadLimit(ctx sdk.Context, pair asset.Pair) (bool, error)
	GetMaintenanceMarginRatio(ctx sdk.Context, pair asset.Pair) (sdk.Dec, error)
	ExistsPool(ctx sdk.Context, pair asset.Pair) bool
	GetSettlementPrice(ctx sdk.Context, pair asset.Pair) (sdk.Dec, error)
	GetLastSnapshot(ctx sdk.Context, pool perpammtypes.Market) (perpammtypes.ReserveSnapshot, error)

	EditPoolPegMultiplier(ctx sdk.Context, pair asset.Pair, pegMultiplier sdk.Dec) error
	EditSwapInvariant(
		ctx sdk.Context, pair asset.Pair, multiplier sdk.Dec,
	) (newMarket perpammtypes.Market, err error)
}

type EpochKeeper interface {
	// GetEpochInfo returns epoch info by identifier.
	GetEpochInfo(ctx sdk.Context, identifier string) types.EpochInfo
}
