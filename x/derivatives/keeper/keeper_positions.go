package keeper

import (
	"errors"
	"fmt"

	derivativesv1 "github.com/MatrixDao/matrix/api/derivatives"
	"github.com/MatrixDao/matrix/x/derivatives/types"
	"github.com/cosmos/cosmos-sdk/orm/types/ormerrors"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type Direction int

const (
	DirectionNotDefined Direction = iota
	DirectionLong
	DirectionShort
)

type MsgOpenPosition struct {
	Symbol    string
	Amount    string
	Leverage  int
	Direction Direction
	Sender    string
}

func (m *MsgOpenPosition) ValidateBasic() error {
	if m.Symbol == "" {
		return fmt.Errorf("empty symbol")
	}
	if m.Amount.IsZero() {
		return fmt.Errorf("empty amount")
	}
	if m.Leverage <= 0 {
		return fmt.Errorf("invalid leverage")
	}
	if m.Direction == DirectionNotDefined {
		return fmt.Errorf("undefined direction")
	}
	if m.Direction != DirectionLong && m.Direction != DirectionShort {
		return fmt.Errorf("unknown direction")
	}

	return nil
}

type MsgOpenPositionResponse struct {
}

func (k *Keeper) OpenPosition(ctx sdk.Context, msg *MsgOpenPosition) (*MsgOpenPositionResponse, error) {
	// we check if the exchange is running
	if k.Stopped(ctx) {
		return nil, types.ErrNotRunning
	}
	// get maximum leverage for pair which checks also if the pair exists
	if msg.Leverage > k.MaximumLeverageForPair(ctx, msg.Symbol) {
		return nil, types.ErrLeverage
	}
	// TODO check if 	requireMoreMarginRatio()
	position, err := k.store.PositionTable().Get(ctx, msg.Sender, msg.Symbol)
	// check error type
	switch {
	// case new position
	case errors.Is(err, ormerrors.NotFound):
		return k.newPosition(ctx, nil, msg)
	// case existing position
	case errors.Is(err, nil):
		return k.changePosition(ctx, position)
	default:
		return nil, err
	}
}

func (k Keeper) newPosition(ctx sdk.Context, meta *derivativesv1.PairMetadata, msg *MsgOpenPosition) (*MsgOpenPositionResponse, error) {
	// first we transfer money from the user to the derivatives exchange TODO
	switch msg.Direction {
	case DirectionLong:
		// we mint virtual collateral based on the collateral asset
	case DirectionShort:
		// hardest case we need to calculate given the leverage
		// how much
	}
	// then we mint the corresponding amount of virtual collateral
	// then we swap based on the direction
	return &MsgOpenPositionResponse{}, nil
}

func (k Keeper) changePosition(ctx sdk.Context, position *derivativesv1.Position) (*MsgOpenPositionResponse, error) {
	panic("implement")
}

func (k *Keeper) MaximumLeverageForPair(ctx sdk.Context, pair string) int {
	// TODO
	return 1000
}
