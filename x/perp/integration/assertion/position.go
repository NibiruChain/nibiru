package assertion

import (
	"fmt"
	"github.com/NibiruChain/collections"
	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/common/testutil/action"
	"github.com/NibiruChain/nibiru/x/perp/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type PositionChecker func(resp types.Position) error

type positionShouldBeEqual struct {
	Account sdk.AccAddress
	Pair    asset.Pair

	PositionCheckers []PositionChecker
}

func (p positionShouldBeEqual) Do(app *app.NibiruApp, ctx sdk.Context) (sdk.Context, error) {
	position, err := app.PerpKeeper.Positions.Get(ctx, collections.Join(p.Pair, p.Account))
	if err != nil {
		return ctx, err
	}

	for _, checker := range p.PositionCheckers {
		if err := checker(position); err != nil {
			return ctx, err
		}
	}

	return ctx, nil
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

// PositionChekers

// Position_PositionShouldBeEqualTo checks if the position is equal to the expected position
func Position_PositionShouldBeEqualTo(expectedPosition types.Position) PositionChecker {
	return func(position types.Position) error {
		if err := types.PositionsAreEqual(&expectedPosition, &position); err != nil {
			return err
		}

		return nil
	}
}

// Position_PositionSizeShouldBeEqualTo checks if the position size is equal to the expected position size
func Position_PositionSizeShouldBeEqualTo(expectedSize sdk.Dec) PositionChecker {
	return func(position types.Position) error {
		if position.Size_.Equal(expectedSize) {
			return nil
		}
		return fmt.Errorf("expected position size %s, got %s", expectedSize, position.Size_.String())
	}
}
