package types

//go:generate  mockgen -destination=../../testutil/mock/perp_interfaces.go -package=mock github.com/NibiruChain/nibiru/x/perp/types AccountKeeper,BankKeeper,PriceKeeper,VpoolKeeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth/types"

	"github.com/NibiruChain/nibiru/x/common"

	pftypes "github.com/NibiruChain/nibiru/x/pricefeed/types"
	pooltypes "github.com/NibiruChain/nibiru/x/vpool/types"
)

// ----------------------------------------------------------
// Keeper Interfaces
// ----------------------------------------------------------

// AccountKeeper defines the expected account keeper used for simulations (noalias)
type AccountKeeper interface {
	// Methods imported from account should be defined here
	GetAccount(ctx sdk.Context, addr sdk.AccAddress) types.AccountI
	GetModuleAddress(name string) sdk.AccAddress
	GetModuleAccount(ctx sdk.Context, moduleName string) types.ModuleAccountI
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
	SendCoinsFromModuleToAccount(
		ctx sdk.Context, senderModule string, recipientAddr sdk.AccAddress,
		amt sdk.Coins,
	) error
	BurnCoins(ctx sdk.Context, moduleName string, amt sdk.Coins) error
	GetBalance(ctx sdk.Context, addr sdk.AccAddress, denom string) sdk.Coin
}

type PriceKeeper interface {
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
	SwapInput(
		ctx sdk.Context,
		pair common.TokenPair,
		dir pooltypes.Direction,
		quoteAssetAmount sdk.Dec,
		baseAmountLimit sdk.Dec,
	) (sdk.Dec, error)

	SwapOutput(
		ctx sdk.Context,
		pair common.TokenPair,
		dir pooltypes.Direction,
		abs sdk.Dec,
		limit sdk.Dec,
	) (sdk.Int, error)
	GetOutputTWAP(ctx sdk.Context, pair common.TokenPair, dir pooltypes.Direction, abs sdk.Int,
	) (sdk.Dec, error)
	GetOutputPrice(ctx sdk.Context, pair common.TokenPair, dir pooltypes.Direction, abs sdk.Dec,
	) (sdk.Dec, error)
	GetSpotPrice(ctx sdk.Context, pair common.TokenPair) (sdk.Dec, error)
	GetUnderlyingPrice(ctx sdk.Context, pair common.TokenPair) (sdk.Dec, error)
	CalcFee(ctx sdk.Context, pair common.TokenPair, quoteAmt sdk.Int) (toll sdk.Int, spread sdk.Int, err error)
	// ExistsPool returns true if pool exists, false if not.
	ExistsPool(ctx sdk.Context, pair common.TokenPair) bool
}
