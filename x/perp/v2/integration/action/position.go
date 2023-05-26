package action

import (
	"errors"
	"fmt"

	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/collections"

	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/common/denoms"
	"github.com/NibiruChain/nibiru/x/common/testutil"
	"github.com/NibiruChain/nibiru/x/common/testutil/action"
	"github.com/NibiruChain/nibiru/x/perp/v2/types"
)

// OpenPosition opens a position with the given parameters.
//
// responseCheckers are optional functions that can be used to check expected response.
func OpenPosition(
	trader sdk.AccAddress,
	pair asset.Pair,
	dir types.Direction,
	margin sdkmath.Int,
	leverage sdk.Dec,
	baseAssetLimit sdk.Dec,
	responseCheckers ...OpenPositionResponseChecker,
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

	responseCheckers []OpenPositionResponseChecker
}

func (o openPositionAction) Do(app *app.NibiruApp, ctx sdk.Context) (sdk.Context, error, bool) {
	resp, err := app.PerpKeeperV2.OpenPosition(
		ctx, o.pair, o.dir, o.trader,
		o.margin, o.leverage, o.baseAssetLimit,
	)
	if err != nil {
		return ctx, err, true
	}

	if o.responseCheckers != nil {
		for _, check := range o.responseCheckers {
			err = check(resp)
			if err != nil {
				return ctx, err, false
			}
		}
	}

	return ctx, nil, true
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

func (o openPositionFailsAction) Do(app *app.NibiruApp, ctx sdk.Context) (sdk.Context, error, bool) {
	_, err := app.PerpKeeperV2.OpenPosition(
		ctx, o.pair, o.dir, o.trader,
		o.margin, o.leverage, o.baseAssetLimit,
	)

	if !errors.Is(err, o.expectedErr) {
		return ctx, fmt.Errorf("expected error %s, got %s", o.expectedErr, err), true
	}

	return ctx, nil, true
}

func OpenPositionFails(
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
type OpenPositionResponseChecker func(resp *types.PositionResp) error

// OpenPositionResp_PositionShouldBeEqual checks that the position included in the response is equal to the expected position response.
func OpenPositionResp_PositionShouldBeEqual(expected types.Position) OpenPositionResponseChecker {
	return func(actual *types.PositionResp) error {
		if err := types.PositionsAreEqual(&expected, &actual.Position); err != nil {
			return err
		}

		return nil
	}
}

// OpenPositionResp_ExchangeNotionalValueShouldBeEqual checks that the exchanged notional value included in the response is equal to the expected value.
func OpenPositionResp_ExchangeNotionalValueShouldBeEqual(expected sdk.Dec) OpenPositionResponseChecker {
	return func(actual *types.PositionResp) error {
		if !actual.ExchangedNotionalValue.Equal(expected) {
			return fmt.Errorf("expected exchanged notional value %s, got %s", expected, actual.ExchangedNotionalValue)
		}

		return nil
	}
}

// OpenPositionResp_ExchangedPositionSizeShouldBeEqual checks that the exchanged position size included in the response is equal to the expected value.
func OpenPositionResp_ExchangedPositionSizeShouldBeEqual(expected sdk.Dec) OpenPositionResponseChecker {
	return func(actual *types.PositionResp) error {
		if !actual.ExchangedPositionSize.Equal(expected) {
			return fmt.Errorf("expected exchanged position size %s, got %s", expected, actual.ExchangedPositionSize)
		}

		return nil
	}
}

// OpenPositionResp_BadDebtShouldBeEqual checks that the bad debt included in the response is equal to the expected value.
func OpenPositionResp_BadDebtShouldBeEqual(expected sdk.Dec) OpenPositionResponseChecker {
	return func(actual *types.PositionResp) error {
		if !actual.BadDebt.Equal(expected) {
			return fmt.Errorf("expected bad debt %s, got %s", expected, actual.BadDebt)
		}

		return nil
	}
}

// OpenPositionResp_FundingPaymentShouldBeEqual checks that the funding payment included in the response is equal to the expected value.
func OpenPositionResp_FundingPaymentShouldBeEqual(expected sdk.Dec) OpenPositionResponseChecker {
	return func(actual *types.PositionResp) error {
		if !actual.FundingPayment.Equal(expected) {
			return fmt.Errorf("expected funding payment %s, got %s", expected, actual.FundingPayment)
		}

		return nil
	}
}

// OpenPositionResp_RealizedPnlShouldBeEqual checks that the realized pnl included in the response is equal to the expected value.
func OpenPositionResp_RealizedPnlShouldBeEqual(expected sdk.Dec) OpenPositionResponseChecker {
	return func(actual *types.PositionResp) error {
		if !actual.RealizedPnl.Equal(expected) {
			return fmt.Errorf("expected realized pnl %s, got %s", expected, actual.RealizedPnl)
		}

		return nil
	}
}

// OpenPositionResp_UnrealizedPnlAfterShouldBeEqual checks that the unrealized pnl after included in the response is equal to the expected value.
func OpenPositionResp_UnrealizedPnlAfterShouldBeEqual(expected sdk.Dec) OpenPositionResponseChecker {
	return func(actual *types.PositionResp) error {
		if !actual.UnrealizedPnlAfter.Equal(expected) {
			return fmt.Errorf("expected unrealized pnl after %s, got %s", expected, actual.UnrealizedPnlAfter)
		}

		return nil
	}
}

// OpenPositionResp_MarginToVaultShouldBeEqual checks that the margin to vault included in the response is equal to the expected value.
func OpenPositionResp_MarginToVaultShouldBeEqual(expected sdk.Dec) OpenPositionResponseChecker {
	return func(actual *types.PositionResp) error {
		if !actual.MarginToVault.Equal(expected) {
			return fmt.Errorf("expected margin to vault %s, got %s", expected, actual.MarginToVault)
		}

		return nil
	}
}

// OpenPositionResp_PositionNotionalShouldBeEqual checks that the position notional included in the response is equal to the expected value.
func OpenPositionResp_PositionNotionalShouldBeEqual(expected sdk.Dec) OpenPositionResponseChecker {
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

func (c closePositionAction) Do(app *app.NibiruApp, ctx sdk.Context) (sdk.Context, error, bool) {
	_, err := app.PerpKeeperV2.ClosePosition(ctx, c.Pair, c.Account)
	if err != nil {
		return ctx, err, true
	}

	return ctx, nil, true
}

// ClosePosition closes a position for the given account and pair.
func ClosePosition(account sdk.AccAddress, pair asset.Pair) action.Action {
	return &closePositionAction{
		Account: account,
		Pair:    pair,
	}
}

// Manually insert position, skipping open position logic

type insertPosition struct {
	position types.Position
}

func (i insertPosition) Do(app *app.NibiruApp, ctx sdk.Context) (sdk.Context, error, bool) {
	traderAddr := sdk.MustAccAddressFromBech32(i.position.TraderAddress)
	app.PerpKeeperV2.Positions.Insert(ctx, collections.Join(i.position.Pair, traderAddr), i.position)
	return ctx, nil, true
}

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
