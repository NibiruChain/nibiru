package action

import (
	"errors"
	"fmt"

	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/common/denoms"
	"github.com/NibiruChain/nibiru/x/common/testutil"
	"github.com/NibiruChain/nibiru/x/common/testutil/action"
	perpkeeper "github.com/NibiruChain/nibiru/x/perp/v2/keeper"
	"github.com/NibiruChain/nibiru/x/perp/v2/types"
)

// MarketOrder opens a position with the given parameters.
//
// responseCheckers are optional functions that can be used to check expected response.
func MarketOrder(
	trader sdk.AccAddress,
	pair asset.Pair,
	dir types.Direction,
	margin sdkmath.Int,
	leverage sdk.Dec,
	baseAssetLimit sdk.Dec,
	responseCheckers ...MarketOrderResponseChecker,
) action.Action {
	return &openPositionAction{
		trader:           trader,
		pair:             pair,
		dir:              dir,
		margin:           margin,
		leverage:         leverage,
		baseAssetLimit:   baseAssetLimit,
		responseCheckers: responseCheckers,
	}
}

type openPositionAction struct {
	trader         sdk.AccAddress
	pair           asset.Pair
	dir            types.Direction
	margin         sdkmath.Int
	leverage       sdk.Dec
	baseAssetLimit sdk.Dec

	responseCheckers []MarketOrderResponseChecker
}

func (o openPositionAction) Do(app *app.NibiruApp, ctx sdk.Context) (sdk.Context, error) {
	resp, err := app.PerpKeeperV2.MarketOrder(
		ctx, o.pair, o.dir, o.trader,
		o.margin, o.leverage, o.baseAssetLimit,
	)
	if err != nil {
		return ctx, err
	}

	if o.responseCheckers != nil {
		for _, check := range o.responseCheckers {
			err = check(resp)
			if err != nil {
				return ctx, err
			}
		}
	}

	return ctx, nil
}

type openPositionFailsAction struct {
	trader         sdk.AccAddress
	pair           asset.Pair
	dir            types.Direction
	margin         sdkmath.Int
	leverage       sdk.Dec
	baseAssetLimit sdk.Dec
	expectedErr    error
}

func (o openPositionFailsAction) Do(app *app.NibiruApp, ctx sdk.Context) (sdk.Context, error) {
	_, err := app.PerpKeeperV2.MarketOrder(
		ctx, o.pair, o.dir, o.trader,
		o.margin, o.leverage, o.baseAssetLimit,
	)

	if !errors.Is(err, o.expectedErr) {
		return ctx, fmt.Errorf("expected error %s, got %s", o.expectedErr, err)
	}

	return ctx, nil
}

func MarketOrderFails(
	trader sdk.AccAddress,
	pair asset.Pair,
	dir types.Direction,
	margin sdkmath.Int,
	leverage sdk.Dec,
	baseAssetLimit sdk.Dec,
	expectedErr error,
) action.Action {
	return &openPositionFailsAction{
		trader:         trader,
		pair:           pair,
		dir:            dir,
		margin:         margin,
		leverage:       leverage,
		baseAssetLimit: baseAssetLimit,
		expectedErr:    expectedErr,
	}
}

// Open Position Response Checkers
type MarketOrderResponseChecker func(resp *types.PositionResp) error

// MarketOrderResp_PositionShouldBeEqual checks that the position included in the response is equal to the expected position response.
func MarketOrderResp_PositionShouldBeEqual(expected types.Position) MarketOrderResponseChecker {
	return func(actual *types.PositionResp) error {
		return types.PositionsAreEqual(&expected, &actual.Position)
	}
}

// MarketOrderResp_ExchangeNotionalValueShouldBeEqual checks that the exchanged notional value included in the response is equal to the expected value.
func MarketOrderResp_ExchangeNotionalValueShouldBeEqual(expected sdk.Dec) MarketOrderResponseChecker {
	return func(actual *types.PositionResp) error {
		if !actual.ExchangedNotionalValue.Equal(expected) {
			return fmt.Errorf("expected exchanged notional value %s, got %s", expected, actual.ExchangedNotionalValue)
		}

		return nil
	}
}

// MarketOrderResp_ExchangedPositionSizeShouldBeEqual checks that the exchanged position size included in the response is equal to the expected value.
func MarketOrderResp_ExchangedPositionSizeShouldBeEqual(expected sdk.Dec) MarketOrderResponseChecker {
	return func(actual *types.PositionResp) error {
		if !actual.ExchangedPositionSize.Equal(expected) {
			return fmt.Errorf("expected exchanged position size %s, got %s", expected, actual.ExchangedPositionSize)
		}

		return nil
	}
}

// MarketOrderResp_BadDebtShouldBeEqual checks that the bad debt included in the response is equal to the expected value.
func MarketOrderResp_BadDebtShouldBeEqual(expected sdk.Dec) MarketOrderResponseChecker {
	return func(actual *types.PositionResp) error {
		if !actual.BadDebt.Equal(expected) {
			return fmt.Errorf("expected bad debt %s, got %s", expected, actual.BadDebt)
		}

		return nil
	}
}

// MarketOrderResp_FundingPaymentShouldBeEqual checks that the funding payment included in the response is equal to the expected value.
func MarketOrderResp_FundingPaymentShouldBeEqual(expected sdk.Dec) MarketOrderResponseChecker {
	return func(actual *types.PositionResp) error {
		if !actual.FundingPayment.Equal(expected) {
			return fmt.Errorf("expected funding payment %s, got %s", expected, actual.FundingPayment)
		}

		return nil
	}
}

// MarketOrderResp_RealizedPnlShouldBeEqual checks that the realized pnl included in the response is equal to the expected value.
func MarketOrderResp_RealizedPnlShouldBeEqual(expected sdk.Dec) MarketOrderResponseChecker {
	return func(actual *types.PositionResp) error {
		if !actual.RealizedPnl.Equal(expected) {
			return fmt.Errorf("expected realized pnl %s, got %s", expected, actual.RealizedPnl)
		}

		return nil
	}
}

// MarketOrderResp_UnrealizedPnlAfterShouldBeEqual checks that the unrealized pnl after included in the response is equal to the expected value.
func MarketOrderResp_UnrealizedPnlAfterShouldBeEqual(expected sdk.Dec) MarketOrderResponseChecker {
	return func(actual *types.PositionResp) error {
		if !actual.UnrealizedPnlAfter.Equal(expected) {
			return fmt.Errorf("expected unrealized pnl after %s, got %s", expected, actual.UnrealizedPnlAfter)
		}

		return nil
	}
}

// MarketOrderResp_MarginToVaultShouldBeEqual checks that the margin to vault included in the response is equal to the expected value.
func MarketOrderResp_MarginToVaultShouldBeEqual(expected sdk.Dec) MarketOrderResponseChecker {
	return func(actual *types.PositionResp) error {
		if !actual.MarginToVault.Equal(expected) {
			return fmt.Errorf("expected margin to vault %s, got %s", expected, actual.MarginToVault)
		}

		return nil
	}
}

// MarketOrderResp_PositionNotionalShouldBeEqual checks that the position notional included in the response is equal to the expected value.
func MarketOrderResp_PositionNotionalShouldBeEqual(expected sdk.Dec) MarketOrderResponseChecker {
	return func(actual *types.PositionResp) error {
		if !actual.PositionNotional.Equal(expected) {
			return fmt.Errorf("expected position notional %s, got %s", expected, actual.PositionNotional)
		}

		return nil
	}
}

// Close Position

type closePositionAction struct {
	Account sdk.AccAddress
	Pair    asset.Pair
}

func (c closePositionAction) Do(app *app.NibiruApp, ctx sdk.Context) (sdk.Context, error) {
	_, err := app.PerpKeeperV2.ClosePosition(ctx, c.Pair, c.Account)
	if err != nil {
		return ctx, err
	}

	return ctx, nil
}

// ClosePosition closes a position for the given account and pair.
func ClosePosition(account sdk.AccAddress, pair asset.Pair) action.Action {
	return &closePositionAction{
		Account: account,
		Pair:    pair,
	}
}

type closePositionFailsAction struct {
	Account sdk.AccAddress
	Pair    asset.Pair

	expectedErr error
}

func (c closePositionFailsAction) Do(app *app.NibiruApp, ctx sdk.Context) (sdk.Context, error) {
	_, err := app.PerpKeeperV2.ClosePosition(ctx, c.Pair, c.Account)

	if !errors.Is(err, c.expectedErr) {
		return ctx, fmt.Errorf("expected error %s, got %s", c.expectedErr, err)
	}

	return ctx, nil
}

func ClosePositionFails(account sdk.AccAddress, pair asset.Pair, expectedErr error) action.Action {
	return &closePositionFailsAction{
		Account:     account,
		Pair:        pair,
		expectedErr: expectedErr,
	}
}

// Manually insert position, skipping open position logic
type insertPosition struct {
	position types.Position
}

func (i insertPosition) Do(app *app.NibiruApp, ctx sdk.Context) (sdk.Context, error) {
	traderAddr := sdk.MustAccAddressFromBech32(i.position.TraderAddress)
	app.PerpKeeperV2.SavePosition(ctx, i.position.Pair, 1, traderAddr, i.position)
	return ctx, nil
}

// InsertPosition: Adds a position into state without a corresponding market
// order.
func InsertPosition(modifiers ...positionModifier) action.Action {
	position := types.Position{
		Pair:                            asset.Registry.Pair(denoms.BTC, denoms.USDC),
		TraderAddress:                   testutil.AccAddress().String(),
		Size_:                           sdk.ZeroDec(),
		Margin:                          sdk.ZeroDec(),
		OpenNotional:                    sdk.ZeroDec(),
		LatestCumulativePremiumFraction: sdk.ZeroDec(),
		LastUpdatedBlockNumber:          0,
	}

	for _, modifier := range modifiers {
		modifier(&position)
	}

	return insertPosition{
		position: position,
	}
}

type positionModifier func(position *types.Position)

func WithPair(pair asset.Pair) positionModifier {
	return func(position *types.Position) {
		position.Pair = pair
	}
}

func WithTrader(addr sdk.AccAddress) positionModifier {
	return func(position *types.Position) {
		position.TraderAddress = addr.String()
	}
}

func WithMargin(margin sdk.Dec) positionModifier {
	return func(position *types.Position) {
		position.Margin = margin
	}
}

func WithOpenNotional(openNotional sdk.Dec) positionModifier {
	return func(position *types.Position) {
		position.OpenNotional = openNotional
	}
}

func WithSize(size sdk.Dec) positionModifier {
	return func(position *types.Position) {
		position.Size_ = size
	}
}

func WithLatestCumulativePremiumFraction(latestCumulativePremiumFraction sdk.Dec) positionModifier {
	return func(position *types.Position) {
		position.LatestCumulativePremiumFraction = latestCumulativePremiumFraction
	}
}

func WithLastUpdatedBlockNumber(lastUpdatedBlockNumber int64) positionModifier {
	return func(position *types.Position) {
		position.LastUpdatedBlockNumber = lastUpdatedBlockNumber
	}
}

type partialClose struct {
	trader sdk.AccAddress
	pair   asset.Pair
	amount sdk.Dec
}

func (p partialClose) Do(app *app.NibiruApp, ctx sdk.Context) (sdk.Context, error) {
	txMsg := &types.MsgPartialClose{
		Sender: p.trader.String(),
		Pair:   p.pair,
		Size_:  p.amount,
	}
	goCtx := sdk.WrapSDKContext(ctx)
	_, err := perpkeeper.NewMsgServerImpl(app.PerpKeeperV2).PartialClose(
		goCtx, txMsg)
	if err != nil {
		return ctx, err
	}

	return ctx, nil
}

func PartialClose(trader sdk.AccAddress, pair asset.Pair, amount sdk.Dec) action.Action {
	return partialClose{
		trader: trader,
		pair:   pair,
		amount: amount,
	}
}

type partialCloseFails struct {
	trader sdk.AccAddress
	pair   asset.Pair
	amount sdk.Dec

	expectedErr error
}

func (p partialCloseFails) IsNotMandatory() {}

func (p partialCloseFails) Do(app *app.NibiruApp, ctx sdk.Context) (sdk.Context, error) {
	txMsg := &types.MsgPartialClose{
		Sender: p.trader.String(),
		Pair:   p.pair,
		Size_:  p.amount,
	}
	goCtx := sdk.WrapSDKContext(ctx)
	_, err := perpkeeper.NewMsgServerImpl(app.PerpKeeperV2).PartialClose(
		goCtx, txMsg,
	)

	if !errors.Is(err, p.expectedErr) {
		return ctx, fmt.Errorf("expected error %s, got %s", p.expectedErr, err)
	}

	return ctx, nil
}

func PartialCloseFails(trader sdk.AccAddress, pair asset.Pair, amount sdk.Dec, expectedErr error) action.Action {
	return partialCloseFails{
		trader:      trader,
		pair:        pair,
		amount:      amount,
		expectedErr: expectedErr,
	}
}
