package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// ----------------------------------------------------------
// Vpool Interface
// ----------------------------------------------------------

type VirtualPoolDirection uint8

const (
	VirtualPoolDirection_AddToAMM = iota
	VirtualPoolDirection_RemoveFromAMM
)

type IVirtualPool interface {
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
	GetUnderlyingPrice(ctx sdk.Context) (sdk.Dec, error)
	GetSpotPrice(ctx sdk.Context) (sdk.Int, error)
	/* CalcFee calculates the total tx fee for exchanging 'quoteAmt' of tokens on
	the exchange.

	Args:
	  quoteAmt (sdk.Int):

	Returns:
	  toll (sdk.Int): Amount of tokens transferred to the the fee pool.
	  spread (sdk.Int): Amount of tokens transferred to the PerpEF.
	*/
	CalcFee(quoteAmt sdk.Int) (toll sdk.Int, spread sdk.Int, err error)
}
