package action

import (
	"errors"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkquery "github.com/cosmos/cosmos-sdk/types/query"

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
		return ctx, fmt.Errorf(
			"expected position not found, but found a position for pair %s, trader %s",
			q.pair,
			q.traderAddress,
		), false
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
	versioned           bool
	allResponseCheckers []QueryMarketsChecker
}

func (q queryMarkets) Do(app *app.NibiruApp, ctx sdk.Context) (sdk.Context, error, bool) {
	queryServer := keeper.NewQuerier(app.PerpKeeperV2)

	resp, err := queryServer.QueryMarkets(sdk.WrapSDKContext(ctx), &types.QueryMarketsRequest{
		Versioned: q.versioned,
	})
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

// QueryMarkets queries all markets, versioned contains active and inactive markets
func QueryMarkets(versioned bool, responseCheckers ...QueryMarketsChecker) action.Action {
	return queryMarkets{
		versioned:           versioned,
		allResponseCheckers: responseCheckers,
	}
}

type QueryMarketsChecker func(resp []types.AmmMarket) error

func QueryMarkets_MarketsShouldContain(expectedMarket types.Market) QueryMarketsChecker {
	return func(resp []types.AmmMarket) error {
		for _, market := range resp {
			if types.MarketsAreEqual(expectedMarket, market.Market) == nil {
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

func QueryMarkets_ShouldLength(length int) QueryMarketsChecker {
	return func(resp []types.AmmMarket) error {
		if len(resp) != length {
			return fmt.Errorf("expected markets to have length %d, got %d", length, len(resp))
		}

		return nil
	}
}

type queryModuleAccounts struct {
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

// ---------------------------------------------------------
// QueryPositionStore
// ---------------------------------------------------------

func QueryPositionStore(
	pageReq *sdkquery.PageRequest, wantErr bool, checks ...QueryPositionStoreChecks,
) action.Action {
	return queryPositionStore{
		pageReq: pageReq,
		wantErr: wantErr,
		checks:  checks,
	}
}

func (q queryPositionStore) Do(
	app *app.NibiruApp, ctx sdk.Context,
) (newCtx sdk.Context, err error, isMandatory bool) {
	queryServer := keeper.NewQuerier(app.PerpKeeperV2)

	gotResp, err := queryServer.QueryPositionStore(
		sdk.WrapSDKContext(ctx),
		&types.QueryPositionStoreRequest{Pagination: q.pageReq},
	)
	if q.wantErr && err != nil {
		return action.ActionResp(ctx, nil) // pass
	} else if !q.wantErr && err != nil {
		return action.ActionResp(ctx, err) // fail
	}

	for _, checker := range q.checks {
		if err := checker(*gotResp); err != nil {
			return action.ActionResp(ctx, err)
		}
	}
	return action.ActionResp(ctx, nil)
}

type queryPositionStore struct {
	pageReq *sdkquery.PageRequest
	wantErr bool
	checks  []QueryPositionStoreChecks
}

type QueryPositionStoreChecks func(resp types.QueryPositionStoreResponse) error

func CheckPositionStore_NumPositions(num int) QueryPositionStoreChecks {
	return func(got types.QueryPositionStoreResponse) error {
		gotNumPos := len(got.Positions)
		if num != gotNumPos {
			return fmt.Errorf("expected num positions: %v, got: %v", num, gotNumPos)
		}
		return nil
	}
}
