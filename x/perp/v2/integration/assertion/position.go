package assertion

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/common/testutil/action"
	types "github.com/NibiruChain/nibiru/x/perp/v2/types"
)

type PositionChecker func(resp types.Position) error

type positionShouldBeEqual struct {
	Account sdk.AccAddress
	Pair    asset.Pair

	PositionCheckers []PositionChecker
}

func (p positionShouldBeEqual) Do(app *app.NibiruApp, ctx sdk.Context) (sdk.Context, error, bool) {
	position, err := app.PerpKeeperV2.GetPosition(ctx, p.Pair, 1, p.Account)
	if err != nil {
		return ctx, err, false
	}
	for _, checker := range p.PositionCheckers {
		if err := checker(position); err != nil {
			return ctx, err, false
		}
	}

	return ctx, nil, false
}

func PositionShouldBeEqual(
	account sdk.AccAddress, pair asset.Pair, positionCheckers ...PositionChecker,
) action.Action {
	return positionShouldBeEqual{
		Account: account,
		Pair:    pair,

		PositionCheckers: positionCheckers,
	}
}

// PositionCheckers

// Position_PositionShouldBeEqualTo checks if the position is equal to the expected position
func Position_PositionShouldBeEqualTo(expectedPosition types.Position) PositionChecker {
	return func(position types.Position) error {
		if err := types.PositionsAreEqual(&expectedPosition, &position); err != nil {
			return err
		}

		return nil
	}
}

type positionShouldNotExist struct {
	Account sdk.AccAddress
	Pair    asset.Pair
	Version uint64
}

func (p positionShouldNotExist) Do(app *app.NibiruApp, ctx sdk.Context) (sdk.Context, error, bool) {
	_, err := app.PerpKeeperV2.GetPosition(ctx, p.Pair, p.Version, p.Account)
	if err == nil {
		return ctx, fmt.Errorf("position should not exist, but it does with pair %s", p.Pair), false
	}

	return ctx, nil, false
}

func PositionShouldNotExist(account sdk.AccAddress, pair asset.Pair, version uint64) action.Action {
	return positionShouldNotExist{
		Account: account,
		Pair:    pair,
		Version: version,
	}
}

type positionShouldExist struct {
	Account sdk.AccAddress
	Pair    asset.Pair
	Version uint64
}

func (p positionShouldExist) Do(app *app.NibiruApp, ctx sdk.Context) (sdk.Context, error, bool) {
	_, err := app.PerpKeeperV2.GetPosition(ctx, p.Pair, p.Version, p.Account)
	if err != nil {
		return ctx, fmt.Errorf("position should exist, but it does not with pair %s", p.Pair), false
	}

	return ctx, nil, false
}

func PositionShouldExist(account sdk.AccAddress, pair asset.Pair, version uint64) action.Action {
	return positionShouldExist{
		Account: account,
		Pair:    pair,
		Version: version,
	}
}
