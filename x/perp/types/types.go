package types

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/common"
)

const (
	ModuleName           = "perp"
	VaultModuleAccount   = "vault"
	PerpEFModuleAccount  = "perp_ef"
	FeePoolModuleAccount = "fee_pool"
)

func ZeroPosition(ctx sdk.Context, tokenPair common.AssetPair, traderAddr sdk.AccAddress) *Position {
	return &Position{
		TraderAddress:                       traderAddr.String(),
		Pair:                                tokenPair.String(),
		Size_:                               sdk.ZeroDec(),
		Margin:                              sdk.ZeroDec(),
		OpenNotional:                        sdk.ZeroDec(),
		LastUpdateCumulativePremiumFraction: sdk.ZeroDec(),
		BlockNumber:                         ctx.BlockHeight(),
	}
}

func (l *LiquidateResp) Validate() error {
	nilFieldError := fmt.Errorf(
		`invalid liquidationOutput: %v,
				must not have nil fields`, l.String())

	// nil sdk.Int check
	for _, field := range []sdk.Int{
		l.FeeToLiquidator, l.FeeToPerpEcosystemFund} {
		if field.IsNil() {
			return nilFieldError
		}
	}

	// nil sdk.Dec check
	for _, field := range []sdk.Dec{l.BadDebt} {
		if field.IsNil() {
			return nilFieldError
		}
	}

	if _, err := sdk.AccAddressFromBech32(l.Liquidator); err != nil {
		return err
	}

	return nil
}
