package action

import (
	"fmt"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/pkg/errors"

	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/common/testutil/action"
)

type closeMarket struct {
	pair asset.Pair
}

func (c closeMarket) Do(app *app.NibiruApp, ctx sdk.Context) (sdk.Context, error, bool) {
	err := app.PerpKeeperV2.CloseMarket(ctx, c.pair)
	if err != nil {
		return ctx, err, false
	}

	return ctx, nil, true
}

func CloseMarket(pair asset.Pair) action.Action {
	return closeMarket{pair: pair}
}

type settlePosition struct {
	account sdk.AccAddress
	pair    asset.Pair
	version uint64
}

func (s settlePosition) Do(app *app.NibiruApp, ctx sdk.Context) (sdk.Context, error, bool) {
	err := app.PerpKeeperV2.SettlePosition(ctx, s.account, s.pair, s.version)
	if err != nil {
		return ctx, err, false
	}

	return ctx, nil, true
}

func SettlePosition(account sdk.AccAddress, pair asset.Pair, version uint64) action.Action {
	return settlePosition{account: account, pair: pair, version: version}
}

type settlePositionShouldFail struct {
	account sdk.AccAddress
	pair    asset.Pair
	version uint64

	expectedErr error
}

func (s settlePositionShouldFail) Do(app *app.NibiruApp, ctx sdk.Context) (sdk.Context, error, bool) {
	err := app.PerpKeeperV2.SettlePosition(ctx, s.account, s.pair, s.version)

	if !errors.Is(err, s.expectedErr) {
		return ctx, fmt.Errorf("expected error: %v, got: %v", s.expectedErr, err), false
	}

	return ctx, nil, true
}

func SettlePositionShouldFail(account sdk.AccAddress, pair asset.Pair, version uint64, expectedErr error) action.Action {
	return settlePositionShouldFail{account: account, pair: pair, version: version, expectedErr: expectedErr}
}
