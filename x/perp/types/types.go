package types

import (
	"errors"

	"github.com/NibiruChain/nibiru/x/common"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	ModuleName           = "perp"
	VaultModuleAccount   = "vault"
	PerpEFModuleAccount  = "perp_ef"
	FeePoolModuleAccount = "fee_pool"
)

var (
	ErrNotFound         = errors.New("not found")
	ErrPositionNotFound = errors.New("no position found")
	ErrPairNotFound     = errors.New("pair doesn't have live vpool")
)

func ZeroPosition(ctx sdk.Context, vpair common.TokenPair, trader string) *Position {
	return &Position{
		Address:                             trader,
		Pair:                                vpair.String(),
		Size_:                               sdk.ZeroInt(),
		Margin:                              sdk.ZeroInt(),
		OpenNotional:                        sdk.ZeroInt(),
		LastUpdateCumulativePremiumFraction: sdk.ZeroInt(),
		LiquidityHistoryIndex:               0,
		BlockNumber:                         ctx.BlockHeight(),
	}
}
