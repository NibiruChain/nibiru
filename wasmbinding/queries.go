package wasmbinding

import (
	"fmt"

	"github.com/NibiruChain/nibiru/x/common/asset"
	perpkeeper "github.com/NibiruChain/nibiru/x/perp/keeper"
	perptypes "github.com/NibiruChain/nibiru/x/perp/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type QueryPlugin struct {
	perpKeeper *perpkeeper.Keeper
}

// NewQueryPlugin returns a reference to a new QueryPlugin.
func NewQueryPlugin(pk *perpkeeper.Keeper) *QueryPlugin {
	return &QueryPlugin{
		perpKeeper: pk,
	}
}

// GetPosition is a query to get position info per trader per pair
func (qp QueryPlugin) GetPosition(ctx sdk.Context, trader string, pair string) (*perptypes.QueryPositionResponse, error) {
	_ctx := sdk.WrapSDKContext(ctx)

	if trader == "" || pair == "" {
		return nil, fmt.Errorf("invalid trader or pair")
	}

	querier := perpkeeper.NewQuerier(*qp.perpKeeper)
	return querier.QueryPosition(_ctx, &perptypes.QueryPositionRequest{
		Pair:   asset.MustNewPair(pair),
		Trader: trader,
	})
}

// GetPosition is a query to get position info per trader
func (qp QueryPlugin) GetPositions(ctx sdk.Context, trader string) (*perptypes.QueryPositionsResponse, error) {
	_ctx := sdk.WrapSDKContext(ctx)

	if trader == "" {
		return nil, fmt.Errorf("invalid trader")
	}

	querier := perpkeeper.NewQuerier(*qp.perpKeeper)
	return querier.QueryPositions(_ctx, &perptypes.QueryPositionsRequest{
		Trader: trader,
	})
}
