package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/perp/events"
	"github.com/NibiruChain/nibiru/x/perp/types"
	vpooltypes "github.com/NibiruChain/nibiru/x/vpool/types"
)

type LiquidateResp struct {
	BadDebt                sdk.Dec
	FeeToLiquidator        sdk.Dec
	FeeToPerpEcosystemFund sdk.Dec
	Liquidator             sdk.AccAddress
	PositionResp           *types.PositionResp
}

func (l *LiquidateResp) String() string {
	return fmt.Sprintf(`
	LiquidateResp {
		BadDebt: %v,
		FeeToLiquidator: %v,
		FeeToPerpEcosystemFund: %v,
		PositionResp: %v,
		Liquidator: %v,
	}
	`,
		l.BadDebt.String(),
		l.FeeToLiquidator.String(),
		l.FeeToPerpEcosystemFund.String(),
		l.PositionResp.String(),
		l.Liquidator.String(),
	)
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

// ExecuteFullLiquidation fully liquidates a position.
func (k Keeper) ExecuteFullLiquidation(
	ctx sdk.Context, liquidator sdk.AccAddress, position *types.Position,
) (err error) {
	params := k.GetParams(ctx)

	positionResp, err := k.closePositionEntirely(
		ctx,
		/* currentPosition */ *position,
		/* quoteAssetAmountLimit */ sdk.ZeroDec())
	if err != nil {
		return err
	}

	remainMargin := positionResp.MarginToVault.Abs()

	feeToLiquidator := params.GetLiquidationFeeAsDec().
		Mul(positionResp.ExchangedQuoteAssetAmount).
		QuoInt64(2)
	totalBadDebt := positionResp.BadDebt

	if feeToLiquidator.GT(remainMargin) {
		// if the remainMargin is not enough for liquidationFee, count it as bad debt
		liquidationBadDebt := feeToLiquidator.Sub(remainMargin)
		totalBadDebt = totalBadDebt.Add(liquidationBadDebt)
	} else {
		// Otherwise, the remaining margin rest will be transferred to ecosystemFund
		remainMargin = remainMargin.Sub(feeToLiquidator)
	}

	feeToPerpEcosystemFund := sdk.ZeroDec()
	if remainMargin.IsPositive() {
		feeToPerpEcosystemFund = remainMargin
	}

	err = k.distributeLiquidateRewards(ctx, LiquidateResp{
		BadDebt:                totalBadDebt,
		FeeToLiquidator:        feeToLiquidator,
		FeeToPerpEcosystemFund: feeToPerpEcosystemFund,
		Liquidator:             liquidator,
		PositionResp:           positionResp,
	})
	if err != nil {
		return err
	}

	return nil
}

func (k Keeper) distributeLiquidateRewards(
	ctx sdk.Context, liquidateResp LiquidateResp) (err error) {
	// --------------------------------------------------------------
	//  Preliminary validations
	// --------------------------------------------------------------

	// validate response
	err = liquidateResp.Validate()
	if err != nil {
		return err
	}

	// validate liquidator
	liquidator, err := sdk.AccAddressFromBech32(liquidateResp.Liquidator.String())
	if err != nil {
		return err
	}

	// validate pair
	pair, err := common.NewTokenPairFromStr(liquidateResp.PositionResp.Position.Pair)
	if err != nil {
		return err
	}
	err = k.requireVpool(ctx, pair)
	if err != nil {
		return err
	}

	// --------------------------------------------------------------
	// Distribution of rewards
	// --------------------------------------------------------------

	vaultAddr := k.AccountKeeper.GetModuleAddress(types.VaultModuleAccount)
	perpEFAddr := k.AccountKeeper.GetModuleAddress(types.PerpEFModuleAccount)

	// Transfer fee from vault to PerpEF
	feeToPerpEF := liquidateResp.FeeToPerpEcosystemFund.RoundInt()
	if feeToPerpEF.IsPositive() {
		coinToPerpEF := sdk.NewCoin(
			pair.GetQuoteTokenDenom(), feeToPerpEF)
		err = k.BankKeeper.SendCoinsFromModuleToModule(
			ctx,
			/* from */ types.VaultModuleAccount,
			/* to */ types.PerpEFModuleAccount,
			sdk.NewCoins(coinToPerpEF),
		)
		if err != nil {
			return err
		}
		events.EmitTransfer(ctx,
			/* coin */ coinToPerpEF,
			/* from */ vaultAddr.String(),
			/* to */ perpEFAddr.String(),
		)
	}

	// Transfer fee from PerpEF to liquidator
	feeToLiquidator := liquidateResp.FeeToLiquidator.RoundInt()
	if feeToLiquidator.IsPositive() {
		coinToLiquidator := sdk.NewCoin(
			pair.GetQuoteTokenDenom(), feeToLiquidator)
		err = k.BankKeeper.SendCoinsFromModuleToAccount(
			ctx,
			/* from */ types.PerpEFModuleAccount,
			/* to */ liquidator,
			sdk.NewCoins(coinToLiquidator),
		)
		if err != nil {
			return err
		}
		events.EmitTransfer(ctx,
			/* coin */ coinToLiquidator,
			/* from */ perpEFAddr.String(),
			/* to */ liquidator.String(),
		)
	}

	return nil
}

/* CreatePartialLiquidation returns the 'LiquidateResp' of a partial liquidation.

Args:
- ctx (sdk.Context): Carries information about the current state of the application.
- position: the position that will be partially liquidated

Returns:
- (*LiquidateResp): fees, bad debt, and position response for the partial liquidation
- (error): An error if one is raised.
*/
func (k Keeper) CreatePartialLiquidation(
	ctx sdk.Context, position *types.Position,
) (*LiquidateResp, error) {
	// Get position direction: long or short
	var dir vpooltypes.Direction
	if position.Size_.GTE(sdk.ZeroDec()) {
		dir = vpooltypes.Direction_ADD_TO_POOL
	} else {
		dir = vpooltypes.Direction_REMOVE_FROM_POOL
	}

	trader, err := sdk.AccAddressFromBech32(position.Address)
	if err != nil {
		return nil, err
	}

	// validate pair
	pair, err := common.NewTokenPairFromStr(position.Pair)
	if err != nil {
		return nil, err
	}
	err = k.requireVpool(ctx, pair)
	if err != nil {
		return nil, err
	}

	// Compute the notional of the portion of position that's being liquidated
	params := k.GetParams(ctx)
	partiallyLiquidatedPositionNotional, err := k.VpoolKeeper.GetBaseAssetPrice(
		ctx, pair, dir,
		/* baseAssetAmount */ position.Size_.Mul(params.GetPartialLiquidationRatioAsDec()).Abs(),
	)
	if err != nil {
		return nil, err
	}

	// Partially close (i.e. open reverse) the position
	positionResp, err := k.openReversePosition(
		/* ctx */ ctx,
		/* currentPosition */ *position,
		/* quoteAssetAmount */ partiallyLiquidatedPositionNotional,
		/* leverage */ sdk.OneDec(),
		/* baseAssetAmountLimit */ sdk.ZeroDec(),
		/* canOverFluctuationLimit */ true,
	)
	if err != nil {
		return nil, err
	}

	// Compute the liquidation penality, of which half goes to the liquidator
	// and half goes to the ecosystem fund.
	fullLiquidationFee := params.GetLiquidationFeeAsDec().
		Mul(positionResp.ExchangedQuoteAssetAmount)
	feeToLiquidator := fullLiquidationFee.QuoInt64(2)
	feeToPerpEF := fullLiquidationFee.Sub(feeToLiquidator)

	positionResp.Position.Margin = positionResp.Position.Margin.Sub(fullLiquidationFee)
	k.SetPosition(ctx, pair, trader.String(), positionResp.Position)

	return &LiquidateResp{
		FeeToPerpEcosystemFund: feeToPerpEF,
		FeeToLiquidator:        feeToLiquidator,
		PositionResp:           positionResp,
		BadDebt:                positionResp.BadDebt,
	}, nil
}
