package action

import (
	"fmt"

	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/common/testutil/action"
	"github.com/NibiruChain/nibiru/x/perp/keeper/v2"
	v2types "github.com/NibiruChain/nibiru/x/perp/types/v2"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type queryPosition struct {
	pair             asset.Pair
	traderAddress    sdk.AccAddress
	responseCheckers []queryPositionChecker
}

func (q queryPosition) Do(app *app.NibiruApp, ctx sdk.Context) (sdk.Context, error, bool) {
	queryServer := keeper.NewQuerier(app.PerpKeeperV2)

	resp, err := queryServer.QueryPosition(sdk.WrapSDKContext(ctx), &v2types.QueryPositionRequest{
		Pair:   q.pair,
		Trader: q.traderAddress.String(),
	})
	if err != nil {
		return ctx, err, false
	}

	for _, checker := range q.responseCheckers {
		err := checker(resp)
		if err != nil {
			return ctx, err, false
		}
	}

	return ctx, nil, false
}

func QueryPosition(pair asset.Pair, traderAddress sdk.AccAddress, responseCheckers ...queryPositionChecker) action.Action {
	return queryPosition{
		pair:             pair,
		traderAddress:    traderAddress,
		responseCheckers: responseCheckers,
	}
}

type queryPositionChecker func(resp *v2types.QueryPositionResponse) error

func QueryPosition_PositionEquals(expected v2types.Position) queryPositionChecker {
	return func(resp *v2types.QueryPositionResponse) error {
		return v2types.PositionsAreEqual(&expected, resp.Position)
	}
}

func QueryPosition_PositionNotionalEquals(expected sdk.Dec) queryPositionChecker {
	return func(resp *v2types.QueryPositionResponse) error {
		if !expected.Equal(resp.PositionNotional) {
			return fmt.Errorf("expected position notional %s, got %s", expected, resp.PositionNotional)
		}
		return nil
	}
}

func QueryPosition_UnrealizedPnlEquals(expected sdk.Dec) queryPositionChecker {
	return func(resp *v2types.QueryPositionResponse) error {
		if !expected.Equal(resp.UnrealizedPnl) {
			return fmt.Errorf("expected unrealized pnl %s, got %s", expected, resp.UnrealizedPnl)
		}
		return nil
	}
}

func QueryPosition_MarginRatioEquals(expected sdk.Dec) queryPositionChecker {
	return func(resp *v2types.QueryPositionResponse) error {
		if !expected.Equal(resp.MarginRatio) {
			return fmt.Errorf("expected margin ratio %s, got %s", expected, resp.MarginRatio)
		}
		return nil
	}
}
