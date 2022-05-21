package types

import (
	"errors"
	fmt "fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	"github.com/NibiruChain/nibiru/x/common"
)

const (
	ModuleName           = "perp"
	VaultModuleAccount   = "vault"
	PerpEFModuleAccount  = "perp_ef"
	FeePoolModuleAccount = "fee_pool"
)

// x/perp module sentinel errors
var (
	ErrMarginHighEnough = sdkerrors.Register(ModuleName, 1,
		"Margin is higher than required maintenant margin ratio")
	ErrPositionNotFound = errors.New("no position found")
	ErrPairNotFound     = errors.New("pair doesn't have live vpool")
	ErrPositionZero     = errors.New("position is zero")
)

func ZeroPosition(ctx sdk.Context, vpair common.TokenPair, trader string) *Position {
	return &Position{
		Address:                             trader,
		Pair:                                vpair.String(),
		Size_:                               sdk.ZeroDec(),
		Margin:                              sdk.ZeroDec(),
		OpenNotional:                        sdk.ZeroDec(),
		LastUpdateCumulativePremiumFraction: sdk.ZeroDec(),
		LiquidityHistoryIndex:               0,
		BlockNumber:                         ctx.BlockHeight(),
	}
}

func (l *LiquidateResp) Validate() error {
	for _, field := range []sdk.Dec{
		l.BadDebt, l.FeeToLiquidator, l.FeeToPerpEcosystemFund} {
		if field.IsNil() {
			return fmt.Errorf(
				`invalid liquidationOutput: %v,
				must not have nil fields`, l.String())
		}
	}
	return nil
}
