package assertion

import (
	"errors"
	"fmt"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/gogo/protobuf/proto"
	abci "github.com/tendermint/tendermint/abci/types"

	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/x/common/testutil/action"
	types "github.com/NibiruChain/nibiru/x/perp/v2/types"
)

type positionChangedEventShouldBeEqual struct {
	ExpectedEvent *types.PositionChangedEvent
}

func (p positionChangedEventShouldBeEqual) Do(_ *app.NibiruApp, ctx sdk.Context) (sdk.Context, error, bool) {
	for _, abciEvent := range ctx.EventManager().Events() {
		if abciEvent.Type != proto.MessageName(p.ExpectedEvent) {
			continue
		}
		typedEvent, err := sdk.ParseTypedEvent(abci.Event{
			Type:       abciEvent.Type,
			Attributes: abciEvent.Attributes,
		})
		if err != nil {
			return ctx, err, false
		}

		theEvent, ok := typedEvent.(*types.PositionChangedEvent)
		if !ok {
			return ctx, fmt.Errorf("expected event is not of type PositionChangedEvent"), false
		}

		if err := types.PositionsAreEqual(&p.ExpectedEvent.FinalPosition, &theEvent.FinalPosition); err != nil {
			return ctx, err, false
		}

		fieldErrs := []string{}
		if !theEvent.PositionNotional.Equal(p.ExpectedEvent.PositionNotional) {
			err := fmt.Errorf("expected position notional %s, got %s", p.ExpectedEvent.PositionNotional, theEvent.PositionNotional)
			fieldErrs = append(fieldErrs, err.Error())
		}

		if !theEvent.TransactionFee.Equal(p.ExpectedEvent.TransactionFee) {
			err := fmt.Errorf("expected transaction fee %s, got %s", p.ExpectedEvent.TransactionFee, theEvent.TransactionFee)
			fieldErrs = append(fieldErrs, err.Error())
		}

		if !theEvent.RealizedPnl.Equal(p.ExpectedEvent.RealizedPnl) {
			err := fmt.Errorf("expected realized pnl %s, got %s", p.ExpectedEvent.RealizedPnl, theEvent.RealizedPnl)
			fieldErrs = append(fieldErrs, err.Error())
		}

		if !theEvent.BadDebt.Equal(p.ExpectedEvent.BadDebt) {
			err := fmt.Errorf("expected bad debt %s, got %s", p.ExpectedEvent.BadDebt, theEvent.BadDebt)
			fieldErrs = append(fieldErrs, err.Error())
		}

		if !theEvent.FundingPayment.Equal(p.ExpectedEvent.FundingPayment) {
			err := fmt.Errorf("expected funding payment %s, got %s", p.ExpectedEvent.FundingPayment, theEvent.FundingPayment)
			fieldErrs = append(fieldErrs, err.Error())
		}

		if theEvent.BlockHeight != p.ExpectedEvent.BlockHeight {
			err := fmt.Errorf("expected block height %d, got %d", p.ExpectedEvent.BlockHeight, theEvent.BlockHeight)
			fieldErrs = append(fieldErrs, err.Error())
		}

		if !theEvent.ExchangedMargin.Equal(p.ExpectedEvent.ExchangedMargin) {
			err := fmt.Errorf("expected exchanged margin %s, got %s",
				p.ExpectedEvent.ExchangedMargin, theEvent.ExchangedMargin)
			fieldErrs = append(fieldErrs, err.Error())
		}

		if len(fieldErrs) != 0 {
			err := strings.Join(fieldErrs, "\n")
			return ctx, errors.New(err), false
		}
	}

	return ctx, nil, false
}

// PositionChangedEventShouldBeEqual checks that the position changed event is equal to the expected event.
func PositionChangedEventShouldBeEqual(
	expectedEvent *types.PositionChangedEvent,
) action.Action {
	return positionChangedEventShouldBeEqual{
		ExpectedEvent: expectedEvent,
	}
}
