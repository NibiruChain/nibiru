package assertion

import (
	"fmt"

	"github.com/NibiruChain/collections"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/common/testutil/action"
	v2types "github.com/NibiruChain/nibiru/x/perp/types/v2"
)

type PositionChecker func(resp v2types.Position) error

type positionShouldBeEqual struct {
	Account sdk.AccAddress
	Pair    asset.Pair

	PositionCheckers []PositionChecker
}

func (p positionShouldBeEqual) Do(app *app.NibiruApp, ctx sdk.Context) (sdk.Context, error, bool) {
	position, err := app.PerpKeeperV2.Positions.Get(ctx, collections.Join(p.Pair, p.Account))
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
func Position_PositionShouldBeEqualTo(expectedPosition v2types.Position) PositionChecker {
	return func(position v2types.Position) error {
		if err := v2types.PositionsAreEqual(&expectedPosition, &position); err != nil {
			return err
		}

		return nil
	}
}

// Position_PositionSizeShouldBeEqualTo checks if the position size is equal to the expected position size
func Position_PositionSizeShouldBeEqualTo(expectedSize sdk.Dec) PositionChecker {
	return func(position v2types.Position) error {
		if position.Size_.Equal(expectedSize) {
			return nil
		}
		return fmt.Errorf("expected position size %s, got %s", expectedSize, position.Size_.String())
	}
}

func Position_PositionMarginShouldBeEqualTo(expectedMargin sdk.Dec) PositionChecker {
	return func(position v2types.Position) error {
		if position.Margin.Equal(expectedMargin) {
			return nil
		}
		return fmt.Errorf("expected position margin %s, got %s", expectedMargin, position.Margin.String())
	}
}

func Position_PositionOpenNotionalShouldBeEqualTo(expectedOpenNotional sdk.Dec) PositionChecker {
	return func(position v2types.Position) error {
		if position.OpenNotional.Equal(expectedOpenNotional) {
			return nil
		}
		return fmt.Errorf("expected position open notional %s, got %s", expectedOpenNotional, position.OpenNotional.String())
	}
}

type positionShouldNotExist struct {
	Account sdk.AccAddress
	Pair    asset.Pair
}

func (p positionShouldNotExist) Do(app *app.NibiruApp, ctx sdk.Context) (sdk.Context, error, bool) {
	_, err := app.PerpKeeperV2.Positions.Get(ctx, collections.Join(p.Pair, p.Account))
	if err == nil {
		return ctx, fmt.Errorf("position should not exist"), false
	}

	return ctx, nil, false
}

func PositionShouldNotExist(account sdk.AccAddress, pair asset.Pair) action.Action {
	return positionShouldNotExist{
		Account: account,
		Pair:    pair,
	}
}
