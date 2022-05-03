package v1

import (
	pftypes "github.com/NibiruChain/nibiru/x/pricefeed/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth/types"
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

// ----------------------------------------------------------
// Vpool Interface
// ----------------------------------------------------------

type VirtualPoolDirection uint8

const (
	VirtualPoolDirection_AddToAMM = iota
	VirtualPoolDirection_RemoveFromAMM
)

type VirtualPool interface {
	Pair() string
	QuoteTokenDenom() string
	SwapInput(
		ctx sdk.Context, ammDir VirtualPoolDirection, inputAmount,
		minOutputAmount sdk.Int, canOverFluctuationLimit bool,
	) (sdk.Int, error)
	SwapOutput(
		ctx sdk.Context, dir VirtualPoolDirection, abs sdk.Int, limit sdk.Int,
	) (sdk.Int, error)
	GetOpenInterestNotionalCap(ctx sdk.Context) (sdk.Int, error)
	GetMaxHoldingBaseAsset(ctx sdk.Context) (sdk.Int, error)
	GetOutputTWAP(ctx sdk.Context, dir VirtualPoolDirection, abs sdk.Int,
	) (sdk.Int, error)
	GetOutputPrice(ctx sdk.Context, dir VirtualPoolDirection, abs sdk.Int,
	) (sdk.Int, error)
	GetUnderlyingPrice(ctx sdk.Context) (sdk.Int, error)
	GetSpotPrice(ctx sdk.Context) (sdk.Int, error)
	CalcFee(notional sdk.Int) (sdk.Int, sdk.Int, error)
}
