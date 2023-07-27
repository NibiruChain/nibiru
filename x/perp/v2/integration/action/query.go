package action

import (
	"errors"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/collections"

	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/common/testutil/action"
	"github.com/NibiruChain/nibiru/x/perp/v2/keeper"
	types "github.com/NibiruChain/nibiru/x/perp/v2/types"
)

type queryPosition struct {
	pair             asset.Pair
	traderAddress    sdk.AccAddress
	responseCheckers []QueryPositionChecker
}

func (q queryPosition) Do(app *app.NibiruApp, ctx sdk.Context) (sdk.Context, error, bool) {
	queryServer := keeper.NewQuerier(app.PerpKeeperV2)

	resp, err := queryServer.QueryPosition(sdk.WrapSDKContext(ctx), &types.QueryPositionRequest{
		Pair:   q.pair,
		Trader: q.traderAddress.String(),
	})
	if err != nil {
		return ctx, err, false
	}

	for _, checker := range q.responseCheckers {
		if err := checker(*resp); err != nil {
			return ctx, err, false
		}
	}

	return ctx, nil, false
}

func QueryPosition(pair asset.Pair, traderAddress sdk.AccAddress, responseCheckers ...QueryPositionChecker) action.Action {
	return queryPosition{
		pair:             pair,
		traderAddress:    traderAddress,
		responseCheckers: responseCheckers,
	}
}

type QueryPositionChecker func(resp types.QueryPositionResponse) error

func QueryPosition_PositionEquals(expected types.Position) QueryPositionChecker {
	return func(resp types.QueryPositionResponse) error {
		return types.PositionsAreEqual(&expected, &resp.Position)
	}
}

func QueryPosition_PositionNotionalEquals(expected sdk.Dec) QueryPositionChecker {
	return func(resp types.QueryPositionResponse) error {
		if !expected.Equal(resp.PositionNotional) {
			return fmt.Errorf("expected position notional %s, got %s", expected, resp.PositionNotional)
		}
		return nil
	}
}

func QueryPosition_UnrealizedPnlEquals(expected sdk.Dec) QueryPositionChecker {
	return func(resp types.QueryPositionResponse) error {
		if !expected.Equal(resp.UnrealizedPnl) {
			return fmt.Errorf("expected unrealized pnl %s, got %s", expected, resp.UnrealizedPnl)
		}
		return nil
	}
}

func QueryPosition_MarginRatioEquals(expected sdk.Dec) QueryPositionChecker {
	return func(resp types.QueryPositionResponse) error {
		if !expected.Equal(resp.MarginRatio) {
			return fmt.Errorf("expected margin ratio %s, got %s", expected, resp.MarginRatio)
		}
		return nil
	}
}

type queryAllPositions struct {
	traderAddress       sdk.AccAddress
	allResponseCheckers [][]QueryPositionChecker
}

func (q queryAllPositions) Do(app *app.NibiruApp, ctx sdk.Context) (sdk.Context, error, bool) {
	queryServer := keeper.NewQuerier(app.PerpKeeperV2)

	resp, err := queryServer.QueryPositions(sdk.WrapSDKContext(ctx), &types.QueryPositionsRequest{
		Trader: q.traderAddress.String(),
	})
	if err != nil {
		return ctx, err, false
	}

	for i, positionCheckers := range q.allResponseCheckers {
		for _, checker := range positionCheckers {
			if err := checker(resp.Positions[i]); err != nil {
				return ctx, err, false
			}
		}
	}

	return ctx, nil, false
}

func QueryPositions(traderAddress sdk.AccAddress, responseCheckers ...[]QueryPositionChecker) action.Action {
	return queryAllPositions{
		traderAddress:       traderAddress,
		allResponseCheckers: responseCheckers,
	}
}

type queryPositionNotFound struct {
	pair          asset.Pair
	traderAddress sdk.AccAddress
}

func (q queryPositionNotFound) Do(app *app.NibiruApp, ctx sdk.Context) (sdk.Context, error, bool) {
	queryServer := keeper.NewQuerier(app.PerpKeeperV2)

	_, err := queryServer.QueryPosition(sdk.WrapSDKContext(ctx), &types.QueryPositionRequest{
		Pair:   q.pair,
		Trader: q.traderAddress.String(),
	})
	if !errors.Is(err, collections.ErrNotFound) {
		return ctx, fmt.Errorf("expected position not found, but found a position for pair %s, trader %s", q.pair, q.traderAddress), false
	}

	return ctx, nil, false
}

func QueryPositionNotFound(pair asset.Pair, traderAddress sdk.AccAddress) action.Action {
	return queryPositionNotFound{
		pair:          pair,
		traderAddress: traderAddress,
	}
}

type queryMarkets struct {
	allResponseCheckers []QueryMarketsChecker
}

func (q queryMarkets) Do(app *app.NibiruApp, ctx sdk.Context) (sdk.Context, error, bool) {
	queryServer := keeper.NewQuerier(app.PerpKeeperV2)

	resp, err := queryServer.QueryMarkets(sdk.WrapSDKContext(ctx), &types.QueryMarketsRequest{})
	if err != nil {
		return ctx, err, false
	}

	for _, marketsCheckers := range q.allResponseCheckers {
		if err := marketsCheckers(resp.AmmMarkets); err != nil {
			return ctx, err, false
		}
	}

	return ctx, nil, false
}

func QueryMarkets(responseCheckers ...QueryMarketsChecker) action.Action {
	return queryMarkets{
		allResponseCheckers: responseCheckers,
	}
}

type QueryMarketsChecker func(resp []types.AmmMarket) error

func QueryMarkets_MarketsShouldContain(expectedMarket types.Market) QueryMarketsChecker {
	return func(resp []types.AmmMarket) error {
		for _, market := range resp {
			if types.MarketsAreEqual(&expectedMarket, &market.Market) == nil {
				return nil
			}
		}
		marketsStr := make([]string, len(resp))
		for i, market := range resp {
			marketsStr[i] = market.Market.String()
		}
		return fmt.Errorf("expected markets to contain %s but found %s", expectedMarket.String(), marketsStr)
	}
}

type queryModuleAccounts struct {
	traderAddress       sdk.AccAddress
	allResponseCheckers []QueryModuleAccountsChecker
}

func (q queryModuleAccounts) Do(app *app.NibiruApp, ctx sdk.Context) (sdk.Context, error, bool) {
	queryServer := keeper.NewQuerier(app.PerpKeeperV2)

	resp, err := queryServer.ModuleAccounts(sdk.WrapSDKContext(ctx), &types.QueryModuleAccountsRequest{})
	if err != nil {
		return ctx, err, false
	}

	for _, accountsCheckers := range q.allResponseCheckers {
		if err := accountsCheckers(resp.Accounts); err != nil {
			return ctx, err, false
		}
	}

	return ctx, nil, false
}

func QueryModuleAccounts(responseCheckers ...QueryModuleAccountsChecker) action.Action {
	return queryModuleAccounts{allResponseCheckers: responseCheckers}
}

type QueryModuleAccountsChecker func(resp []types.AccountWithBalance) error

func QueryModuleAccounts_ModulesBalanceShouldBe(expectedBalance map[string]sdk.Coins) QueryModuleAccountsChecker {
	return func(resp []types.AccountWithBalance) error {
		for name, balance := range expectedBalance {
			found := false
			for _, account := range resp {
				if account.Name == name {
					found = true
					if !account.Balance.IsEqual(balance) {
						return fmt.Errorf("expected module %s to have balance %s, got %s", name, balance, account.Balance)
					}
				}
			}
			if !found {
				return fmt.Errorf("expected module %s to have balance %s but not found", name, balance)
			}
		}
		return nil
	}
}
