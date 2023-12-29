package action

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/common/testutil/action"
	"github.com/NibiruChain/nibiru/x/perp/v2/types"
)

// closeMarket
type closeMarket struct {
	pair asset.Pair
}

func (c closeMarket) Do(app *app.NibiruApp, ctx sdk.Context) (sdk.Context, error) {
	err := app.PerpKeeperV2.Sudo().CloseMarket(ctx, c.pair)
	if err != nil {
		return ctx, err
	}

	return ctx, nil
}

func CloseMarket(pair asset.Pair) action.Action {
	return closeMarket{pair: pair}
}

// closeMarketShouldFail
type closeMarketShouldFail struct {
	pair asset.Pair
}

func (c closeMarketShouldFail) Do(app *app.NibiruApp, ctx sdk.Context) (sdk.Context, error) {
	err := app.PerpKeeperV2.Sudo().CloseMarket(ctx, c.pair)
	if err == nil {
		return ctx, err
	}

	return ctx, nil
}

func CloseMarketShouldFail(pair asset.Pair) action.Action {
	return closeMarketShouldFail{pair: pair}
}

// settlePosition
type settlePosition struct {
	pair             asset.Pair
	version          uint64
	trader           sdk.AccAddress
	responseCheckers []SettlePositionChecker
}

func (c settlePosition) Do(app *app.NibiruApp, ctx sdk.Context) (sdk.Context, error) {
	resp, err := app.PerpKeeperV2.SettlePosition(ctx, c.pair, c.version, c.trader)
	if err != nil {
		return ctx, err
	}

	for _, checker := range c.responseCheckers {
		if err := checker(*resp); err != nil {
			return ctx, err
		}
	}

	return ctx, nil
}

func SettlePosition(pair asset.Pair, version uint64, trader sdk.AccAddress, responseCheckers ...SettlePositionChecker) action.Action {
	return settlePosition{pair: pair, version: version, trader: trader, responseCheckers: responseCheckers}
}

type SettlePositionChecker func(resp types.PositionResp) error

func SettlePositionChecker_PositionEquals(expected types.Position) SettlePositionChecker {
	return func(resp types.PositionResp) error {
		return types.PositionsAreEqual(&expected, &resp.Position)
	}
}

func SettlePositionChecker_MarginToVault(expectedMarginToVault sdk.Dec) SettlePositionChecker {
	return func(resp types.PositionResp) error {
		if expectedMarginToVault.Equal(resp.MarginToVault) {
			return nil
		} else {
			return fmt.Errorf("expected margin to vault %s, got %s", expectedMarginToVault, resp.MarginToVault)
		}
	}
}

func SettlePositionChecker_BadDebt(expectedBadDebt sdk.Dec) SettlePositionChecker {
	return func(resp types.PositionResp) error {
		if expectedBadDebt.Equal(resp.BadDebt) {
			return nil
		} else {
			return fmt.Errorf("expected bad debt %s, got %s", expectedBadDebt, resp.BadDebt)
		}
	}
}

// settlePositionShouldFail
type settlePositionShouldFail struct {
	pair    asset.Pair
	version uint64
	trader  sdk.AccAddress
}

func (c settlePositionShouldFail) Do(app *app.NibiruApp, ctx sdk.Context) (sdk.Context, error) {
	_, err := app.PerpKeeperV2.SettlePosition(ctx, c.pair, c.version, c.trader)
	if err == nil {
		return ctx, err
	}

	return ctx, nil
}

func SettlePositionShouldFail(pair asset.Pair, version uint64, trader sdk.AccAddress) action.Action {
	return settlePositionShouldFail{pair: pair, version: version, trader: trader}
}
