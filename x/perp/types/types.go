package types

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/common/asset"
)

const (
	ModuleName           = "perp"
	VaultModuleAccount   = "vault"
	PerpEFModuleAccount  = "perp_ef"
	FeePoolModuleAccount = "fee_pool"
)

// x/perp module sentinel errors
var (
	ErrMarginRatioTooHigh                = sdkerrors.Register(ModuleName, 1, "margin ratio is too healthy to liquidate")
	ErrPairNotFound                      = sdkerrors.Register(ModuleName, 2, "pair doesn't have live market")
	ErrPositionZero                      = sdkerrors.Register(ModuleName, 3, "position is zero")
	ErrFailedRemoveMarginCanCauseBadDebt = sdkerrors.Register(ModuleName, 4, "failed to remove margin; position would have bad debt if removed")
	ErrQuoteAmountIsZero                 = sdkerrors.Register(ModuleName, 5, "quote amount cannot be zero")
	ErrLeverageIsZero                    = sdkerrors.Register(ModuleName, 6, "leverage cannot be zero")
	ErrMarginRatioTooLow                 = sdkerrors.Register(ModuleName, 7, "margin ratio did not meet maintenance margin ratio")
	ErrLeverageIsTooHigh                 = sdkerrors.Register(ModuleName, 8, "leverage cannot be higher than market parameter")
	ErrUnauthorized                      = sdkerrors.Register(ModuleName, 9, "operation not authorized")
	ErrAllLiquidationsFailed             = sdkerrors.Register(ModuleName, 10, "all liquidations failed")
)

func ZeroPosition(ctx sdk.Context, tokenPair asset.Pair, traderAddr sdk.AccAddress) Position {
	return Position{
		TraderAddress:                   traderAddr.String(),
		Pair:                            tokenPair,
		Size_:                           sdk.ZeroDec(),
		Margin:                          sdk.ZeroDec(),
		OpenNotional:                    sdk.ZeroDec(),
		LatestCumulativePremiumFraction: sdk.ZeroDec(),
		BlockNumber:                     ctx.BlockHeight(),
	}
}

func (l *LiquidateResp) Validate() error {
	nilFieldError := fmt.Errorf("invalid liquidationOutput, must not have nil fields")

	// nil sdk.Int check
	for _, field := range []sdk.Int{
		l.FeeToLiquidator, l.FeeToPerpEcosystemFund} {
		if field.IsNil() {
			return nilFieldError
		}
	}

	// nil sdk.Int check
	for _, field := range []sdk.Int{l.BadDebt} {
		if field.IsNil() {
			return nilFieldError
		}
	}

	if _, err := sdk.AccAddressFromBech32(l.Liquidator); err != nil {
		return err
	}

	return nil
}

func PositionsAreEqual(expected, actual *Position) error {
	if expected.Pair != actual.Pair {
		return fmt.Errorf("expected position pair %s, got %s", expected.Pair, actual.Pair)
	}

	if expected.TraderAddress != actual.TraderAddress {
		return fmt.Errorf("expected position trader address %s, got %s", expected.TraderAddress, actual.TraderAddress)
	}

	if !expected.Margin.Equal(actual.Margin) {
		return fmt.Errorf("expected position margin %s, got %s", expected.Margin, actual.Margin)
	}

	if !expected.OpenNotional.Equal(actual.OpenNotional) {
		return fmt.Errorf("expected position open notional %s, got %s", expected.OpenNotional, actual.OpenNotional)
	}

	if !expected.Size_.Equal(actual.Size_) {
		return fmt.Errorf("expected position size %s, got %s", expected.Size_, actual.Size_)
	}

	if expected.BlockNumber != actual.BlockNumber {
		return fmt.Errorf("expected position block number %d, got %d", expected.BlockNumber, actual.BlockNumber)
	}

	if !expected.LatestCumulativePremiumFraction.Equal(actual.LatestCumulativePremiumFraction) {
		return fmt.Errorf(
			"expected position latest cumulative premium fraction %s, got %s",
			expected.LatestCumulativePremiumFraction,
			actual.LatestCumulativePremiumFraction,
		)
	}
	return nil
}

var ModuleAccounts = []string{
	ModuleName,
	VaultModuleAccount,
	PerpEFModuleAccount,
	FeePoolModuleAccount,
	common.TreasuryPoolModuleAccount,
}
