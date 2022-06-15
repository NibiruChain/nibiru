package types

import (
	"errors"
	"fmt"

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
	ErrPositionNotFound     = errors.New("no position found")
	ErrPairNotFound         = errors.New("pair doesn't have live vpool")
	ErrPairMetadataNotFound = errors.New("pair doesn't have metadata")
	ErrPositionZero         = errors.New("position is zero")
	// failed to remove margin; position has bad debt
	ErrFailedRemoveMarginCanCauseBadDebt = errors.New("failed to remove margin; position would have bad debt if removed")
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
