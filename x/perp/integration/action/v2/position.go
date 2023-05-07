package action

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/collections"
	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/common/denoms"
	"github.com/NibiruChain/nibiru/x/common/testutil"
	"github.com/NibiruChain/nibiru/x/common/testutil/action"
	v2types "github.com/NibiruChain/nibiru/x/perp/types/v2"
)

// Manually insert position, skipping open position logic

type insertPosition struct {
	position v2types.Position
}

func (i insertPosition) Do(app *app.NibiruApp, ctx sdk.Context) (sdk.Context, error, bool) {
	traderAddr := sdk.MustAccAddressFromBech32(i.position.TraderAddress)
	app.PerpKeeperV2.Positions.Insert(ctx, collections.Join(i.position.Pair, traderAddr), i.position)
	return ctx, nil, true
}

func InsertPosition(modifiers ...positionModifier) action.Action {
	position := v2types.Position{
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

type positionModifier func(position *v2types.Position)

func WithPair(pair asset.Pair) positionModifier {
	return func(position *v2types.Position) {
		position.Pair = pair
	}
}

func WithTrader(addr sdk.AccAddress) positionModifier {
	return func(position *v2types.Position) {
		position.TraderAddress = addr.String()
	}
}

func WithMargin(margin sdk.Dec) positionModifier {
	return func(position *v2types.Position) {
		position.Margin = margin
	}
}

func WithOpenNotional(openNotional sdk.Dec) positionModifier {
	return func(position *v2types.Position) {
		position.OpenNotional = openNotional
	}
}

func WithSize(size sdk.Dec) positionModifier {
	return func(position *v2types.Position) {
		position.Size_ = size
	}
}

func WithLatestCumulativePremiumFraction(latestCumulativePremiumFraction sdk.Dec) positionModifier {
	return func(position *v2types.Position) {
		position.LatestCumulativePremiumFraction = latestCumulativePremiumFraction
	}
}

func WithLastUpdatedBlockNumber(lastUpdatedBlockNumber int64) positionModifier {
	return func(position *v2types.Position) {
		position.LastUpdatedBlockNumber = lastUpdatedBlockNumber
	}
}
