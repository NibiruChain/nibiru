package exchange

import (
	"context"
	"fmt"
	derivativesv1 "github.com/MatrixDao/matrix/api/derivatives"
	vammv1 "github.com/MatrixDao/matrix/api/vamm"
	"github.com/MatrixDao/matrix/x/derivatives/types"
	"github.com/cosmos/cosmos-sdk/orm/types/ormerrors"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type VirtualAMM interface {
	Pair() string
	vammv1.VirtualAMMClient
}

type parsedPosition struct {
	address string
	pair    string

	size                                 sdk.Int
	margin                               sdk.Int
	openNotional                         sdk.Int
	lastUpdatedCumulativePremiumFraction sdk.Int
	liquidityHistoryIndex                int64
	blockNumber                          int64
}

type positionUpdate struct {
	position *parsedPosition

	exchangedQuoteAssetAmount sdk.Int
	badDebt                   sdk.Int
	exchangedPositionSide     sdk.Int
	fundingPayment            sdk.Int
	realizedPnL               sdk.Int
	marginToVault             sdk.Int
	unrealizedPnLAfter        sdk.Int
}

type Exchange struct {
	state derivativesv1.StateStore
}

func (e Exchange) OpenPosition(
	ctx context.Context, amm VirtualAMM,
	trader string, side vammv1.Direction, quoteAssetAmount sdk.Int, leverage sdk.Int, baseAssetAmountLimit sdk.Int,
) error {

	// exchange must not be stopped
	if err := e.stopped(ctx); err != nil {
		return err
	}
	// TODO validate token amount != 0;
	// TODO validate leverage != 0;
	// TODO validate margin is correct;
	// TODO validate VirtualAMM is not restricted.

	position, positionExists := e.getPosition(ctx, amm, trader)
	var (
		updatedPosition *positionUpdate
		err             error
	)
	switch {
	case
		!positionExists, // if position does not exist
		position.size.IsPositive() && side == vammv1.Direction_DIRECTION_LONG,  // or current size is LONG and the user wants to LONG again
		position.size.IsNegative() && side == vammv1.Direction_DIRECTION_SHORT: // or current size is SHORT and the user wants to SHORT again
		// then it means we're trying to increase position
		updatedPosition, err = e.increasePosition(
			ctx, amm, side, position,
			quoteAssetAmount.Mul(leverage), baseAssetAmountLimit, leverage)
	// every other possible case is a position decrease
	default:
		e.decreasePosition()
	}

	panic("impl")
}

func (e Exchange) increasePosition(
	ctx context.Context, amm VirtualAMM,
	side vammv1.Direction, position *parsedPosition,
	openNotional sdk.Int, minimumSize sdk.Int, leverage sdk.Int,
) (update *positionUpdate, err error) {
	update = new(positionUpdate)

	update.exchangedPositionSide, err = swapInput(ctx, amm, side, openNotional, minimumSize, false)
	if err != nil {
		return nil, err
	}
	newSize := position.size.Add(update.exchangedPositionSide)

	// check max size was not exceeded

	// tODO(mercilex) there's a reference for whitelist traders who can have unlimited pos
	maxHoldingBaseAssetResp, err := amm.GetMaxHoldingBaseAsset(ctx, &vammv1.GetMaxHoldingBaseAssetRequest{})
	if err != nil {
		return nil, err
	}
	maxHoldingBaseAsset, ok := sdk.NewIntFromString(maxHoldingBaseAssetResp.Max)
	if !ok {
		panic(fmt.Errorf("AMM returned invalid int %s", maxHoldingBaseAssetResp.Max)) // TODO(mercilex): inefficient back and forth int conversion
	}

	if !maxHoldingBaseAsset.IsZero() && maxHoldingBaseAsset.LT(newSize.Abs()) {
		return nil, types.ErrPositionSideLimit
	}

	increaseMarginRequirement := openNotional.Quo(leverage) // TODO(mercilex): this is highly dependent on how leverage looks like
	remainMargin, _, fundingPayment, latestCumulativePremiumFraction, err := e.calculateRemainingMarginWithFundingPayment(ctx, amm, position, increaseMarginRequirement)
	if err != nil {
		return nil, err
	}

	panic("finish")

}

func (e Exchange) decreasePosition() error {
	panic("impl")
}

func (e Exchange) stopped(ctx context.Context) error {
	params, err := e.state.ParamsTable().Get(ctx)
	if err != nil {
		// not initialized
		panic(fmt.Errorf("derivatives exchange is not initialized correctly: %w", err))
	}

	if params.Stopped {
		return types.ErrNotRunning
	}

	return nil
}

func (e Exchange) getPosition(ctx context.Context, amm VirtualAMM, trader string) (*parsedPosition, bool) {
	pos, err := e.state.PositionTable().Get(ctx, trader, amm.Pair())
	switch {
	// in case position exists we parse it for sdk.Int
	case err == nil:
		// TODO(mercilex): understand if any of the following params can be empty
		size, ok := sdk.NewIntFromString(pos.Size)
		if !ok {
			panic(fmt.Errorf("invalid size: %s", pos.Size))
		}

		margin, ok := sdk.NewIntFromString(pos.Margin)
		if !ok {
			panic(fmt.Errorf("invalid margin: %s", pos.Margin))
		}

		openNotional, ok := sdk.NewIntFromString(pos.OpenNotional)
		if !ok {
			panic(fmt.Errorf("invalid open notional: %s", pos.OpenNotional))
		}

		lastUpdatedCumulativePremiumFraction, ok := sdk.NewIntFromString(pos.LastUpdateCumulativePremiumFraction)
		if !ok {
			panic(fmt.Errorf("invalid last updated cumulative premium fraction: %s", pos.LastUpdateCumulativePremiumFraction))
		}
		return &parsedPosition{
			address:                              pos.Address,
			pair:                                 pos.Pair,
			size:                                 size,
			margin:                               margin,
			openNotional:                         openNotional,
			lastUpdatedCumulativePremiumFraction: lastUpdatedCumulativePremiumFraction,
			liquidityHistoryIndex:                pos.LiquidityHistoryIndex,
			blockNumber:                          pos.BlockNumber,
		}, true

	// pos does not exist
	case ormerrors.IsNotFound(err):
		return nil, false
	// anything else is a panic
	default:
		panic(err)
	}
}
