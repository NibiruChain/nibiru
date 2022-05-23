package types

//go:generate  mockgen -destination=../../testutil/mock/perp_interfaces.go -package=mock github.com/NibiruChain/nibiru/x/perp/types AccountKeeper,BankKeeper,PricefeedKeeper,VpoolKeeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/common"

	"time"

	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	pftypes "github.com/NibiruChain/nibiru/x/pricefeed/types"
	vpooltypes "github.com/NibiruChain/nibiru/x/vpool/types"
)

// ----------------------------------------------------------
// Keeper Interfaces
// ----------------------------------------------------------

// AccountKeeper defines the expected account keeper used for simulations (noalias)
type AccountKeeper interface {
	// Methods imported from account should be defined here
	GetAccount(ctx sdk.Context, addr sdk.AccAddress) authtypes.AccountI
	GetModuleAddress(name string) sdk.AccAddress
	GetModuleAccount(ctx sdk.Context, moduleName string) authtypes.ModuleAccountI
}

// BankKeeper defines the expected interface needed to retrieve account balances.
type BankKeeper interface {
	// Methods imported from bank should be defined here
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
	BurnCoins(ctx sdk.Context, moduleName string, amt sdk.Coins) error
	GetBalance(ctx sdk.Context, addr sdk.AccAddress, denom string) sdk.Coin
}

type PricefeedKeeper interface {
	GetCurrentPrice(ctx sdk.Context, token0 string, token1 string,
	) (pftypes.CurrentPrice, error)
	GetCurrentPrices(ctx sdk.Context) pftypes.CurrentPrices
	GetRawPrices(ctx sdk.Context, marketId string) pftypes.PostedPrices
	GetPair(ctx sdk.Context, pairID string) (pftypes.Pair, bool)
	// Returns the pairs from the x/pricefeed params
	GetPairs(ctx sdk.Context) pftypes.Pairs
	GetOracle(ctx sdk.Context, pairID string, address sdk.AccAddress,
	) (sdk.AccAddress, error)
	GetOracles(ctx sdk.Context, pairID string) ([]sdk.AccAddress, error)
	SetCurrentPrices(ctx sdk.Context, token0 string, token1 string) error
}

type VpoolKeeper interface {
	/* Trades baseAssets in exchange for quoteAssets.
	The "output" asset here refers to baseAsset, which is a crypto asset like BTC.
	The quote asset is a stablecoin like NUSD.

	args:
	  - ctx: cosmos-sdk context
	  - pair: a token pair like BTC:NUSD
	  - dir: either add or remove from pool
	  - baseAssetAmount: the amount of quote asset being traded
	  - quoteAmountLimit: a limiter to ensure the trader doesn't get screwed by slippage

	ret:
	  - quoteAssetAmount: the amount of quote asset swapped
	  - err: error
	*/
	SwapBaseForQuote(
		ctx sdk.Context,
		pair common.TokenPair,
		dir vpooltypes.Direction,
		abs sdk.Dec,
		limit sdk.Dec,
	) (sdk.Dec, error)

	/* Trades quoteAssets in exchange for baseAssets.
	The "input" asset here refers to quoteAsset, which is usually a stablecoin like NUSD.
	The base asset is a crypto asset like BTC or ETH.

	args:
	- ctx: cosmos-sdk context
	- pair: a token pair like BTC:NUSD
	- dir: either add or remove from pool
	- quoteAssetAmount: the amount of quote asset being traded
	- baseAmountLimit: a limiter to ensure the trader doesn't get screwed by slippage

	ret:
	- baseAssetAmount: the amount of base asset traded from the pool
	- err: error
	*/
	SwapQuoteForBase(
		ctx sdk.Context,
		pair common.TokenPair,
		dir vpooltypes.Direction,
		quoteAssetAmount sdk.Dec,
		baseAmountLimit sdk.Dec,
	) (sdk.Dec, error)

	/* Returns the amount of quote assets required to achieve a move of baseAssetAmount in a direction,
	based on historical snapshots.
	e.g. if removing <baseAssetAmount> base assets from the pool, returns the amount of quote assets do so.

	args:
	- ctx: cosmos-sdk context
	- pair: the token pair
	- direction: add or remove
	- baseAssetAmount: amount of base asset to add or remove
	- lookbackInterval: how far back to calculate TWAP

	ret:
	- quoteAssetAmount: the amount of quote asset to make the desired move, as sdk.Dec
	- err: error
	*/
	GetBaseAssetTWAP(
		ctx sdk.Context,
		pair common.TokenPair,
		direction vpooltypes.Direction,
		baseAssetAmount sdk.Dec,
		lookbackInterval time.Duration,
	) (quoteAssetAmount sdk.Dec, err error)

	/* Returns the amount of base assets required to achieve a move of quoteAssetAmount in a direction,
	based on historical snapshots.
	e.g. if removing <quoteAssetAmount> quote assets from the pool, returns the amount of base assets do so.

	args:
	- ctx: cosmos-sdk context
	- pair: the token pair
	- direction: add or remove
	- quoteAssetAmount: amount of base asset to add or remove
	- lookbackInterval: how far back to calculate TWAP

	ret:
	- baseAssetAmount: the amount of quote asset to make the desired move, as sdk.Dec
	- err: error
	*/
	GetQuoteAssetTWAP(
		ctx sdk.Context,
		pair common.TokenPair,
		direction vpooltypes.Direction,
		quoteAssetAmount sdk.Dec,
		lookbackInterval time.Duration,
	) (baseAssetAmount sdk.Dec, err error)

	/* Returns the amount of quote assets required to achieve a move of baseAssetAmount in a direction.
	e.g. if removing <baseAssetAmount> base assets from the pool, returns the amount of quote assets do so.

	args:
	- ctx: cosmos-sdk context
	- pair: the trading token pair
	- dir: add or remove
	- baseAssetAmount: the amount of base asset

	ret:
	- quoteAmount: the amount of quote assets required to make the desired swap
	- err: error
	*/
	GetBaseAssetPrice(
		ctx sdk.Context,
		pair common.TokenPair,
		direction vpooltypes.Direction,
		baseAssetAmount sdk.Dec,
	) (quoteAssetAmount sdk.Dec, err error)

	/* Returns the amount of base assets required to achieve a move of quoteAmount in a direction.
	e.g. if removing <quoteAmount> quote assets from the pool, returns the amount of base assets do so.

	args:
	- ctx: cosmos-sdk context
	- pair: the trading token pair
	- dir: add or remove
	- quoteAmount: the amount of quote asset

	ret:
	- baseAssetAmount: the amount of base assets required to make the desired swap
	- err: error
	*/
	GetQuoteAssetPrice(
		ctx sdk.Context,
		pair common.TokenPair,
		dir vpooltypes.Direction,
		quoteAmount sdk.Dec,
	) (baseAssetAmount sdk.Dec, err error)

	/* GetSpotPrice retrieves the price of the base asset denominated in quote asset.

	The convention is the amount of quote assets required to buy one base asset.

	e.g. If the tokenPair is BTC:NUSD, the method would return sdk.Dec(40,000.00)
	because the instantaneous tangent slope on the vpool curve is 40,000.00,
	so it would cost ~40,000.00 to buy one BTC:NUSD perp.

	args:
	  - ctx: cosmos-sdk context
	  - pair: the token pair to get price for

	ret:
	  - price: the price of the token pair as sdk.Dec
	  - err: error
	*/
	GetSpotPrice(
		ctx sdk.Context,
		pair common.TokenPair,
	) (price sdk.Dec, err error)

	/* Retrieves the base asset's price from PricefeedKeeper (oracle).
	The price is denominated in quote asset, so # of quote asset to buy one base asset.

	args:
	  - ctx: cosmos-sdk context
	  - pair: token pair

	ret:
	  - price: price as sdk.Dec
	  - err: error
	*/
	GetUnderlyingPrice(
		ctx sdk.Context,
		pair common.TokenPair,
	) (price sdk.Dec, err error)

	CalcPerpTxFee(ctx sdk.Context, pair common.TokenPair, quoteAmt sdk.Int,
	) (toll sdk.Int, spread sdk.Int, err error)
	IsOverSpreadLimit(ctx sdk.Context, pair common.TokenPair) bool
	// ExistsPool returns true if pool exists, false if not.
	ExistsPool(ctx sdk.Context, pair common.TokenPair) bool
	GetSettlementPrice(ctx sdk.Context, pair common.TokenPair) (sdk.Dec, error)
}
