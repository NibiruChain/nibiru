package assertion

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	abci "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/gogoproto/proto"

	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/x/common/testutil/action"
	types "github.com/NibiruChain/nibiru/x/perp/v2/types"
)

var _ action.Action = (*containsLiquidateEvent)(nil)
var _ action.Action = (*positionChangedEventShouldBeEqual)(nil)

// TODO test(perp): Add action for testing the appearance of of successful
// liquidation events.

// --------------------------------------------------
// --------------------------------------------------

type containsLiquidateEvent struct {
	expectedEvent *types.LiquidationFailedEvent
}

func (act containsLiquidateEvent) Do(_ *app.NibiruApp, ctx sdk.Context) (
	outCtx sdk.Context, err error, isMandatory bool,
) {
	foundEvent := false
	matchingEvents := []abci.Event{}

	for _, sdkEvent := range ctx.EventManager().Events() {
		if sdkEvent.Type != proto.MessageName(act.expectedEvent) {
			continue
		}

		abciEvent := abci.Event{
			Type:       sdkEvent.Type,
			Attributes: sdkEvent.Attributes,
		}

		typedEvent, err := sdk.ParseTypedEvent(abciEvent)
		if err != nil {
			return ctx, err, false
		}

		liquidationFailedEvent, ok := typedEvent.(*types.LiquidationFailedEvent)
		if !ok {
			return ctx,
				fmt.Errorf("expected event of type %s, got %s", proto.MessageName(act.expectedEvent), abciEvent.Type),
				false
		}

		if reflect.DeepEqual(act.expectedEvent, liquidationFailedEvent) {
			foundEvent = true
			break
		}

		matchingEvents = append(matchingEvents, abciEvent)
	}

	if foundEvent {
		// happy path
		return ctx, nil, true
	}

	// Show descriptive error messages if the expected event is missing
	expected, _ := sdk.TypedEventToEvent(act.expectedEvent)
	return ctx, errors.New(
		strings.Join([]string{
			fmt.Sprintf("expected: %+v.", sdk.StringifyEvents([]abci.Event{abci.Event(expected)})),
			fmt.Sprintf("found %v events:", len(ctx.EventManager().Events())),
			fmt.Sprintf("events of matching type:\n%v", sdk.StringifyEvents(matchingEvents).String()),
		}, "\n"),
	), false
}

// ContainsLiquidateEvent checks if a typed event (proto.Message) is contained in the
// event manager of the app context.
func ContainsLiquidateEvent(
	expectedEvent *types.LiquidationFailedEvent,
) action.Action {
	return containsLiquidateEvent{
		expectedEvent: expectedEvent,
	}
}

type positionChangedEventShouldBeEqual struct {
	expectedEvent *types.PositionChangedEvent
}

func (p positionChangedEventShouldBeEqual) Do(_ *app.NibiruApp, ctx sdk.Context) (sdk.Context, error, bool) {
	for _, sdkEvent := range ctx.EventManager().Events() {
		if sdkEvent.Type != proto.MessageName(p.expectedEvent) {
			continue
		}

		abciEvent := abci.Event{
			Type:       sdkEvent.Type,
			Attributes: sdkEvent.Attributes,
		}

		typedEvent, err := sdk.ParseTypedEvent(abciEvent)
		if err != nil {
			return ctx, err, false
		}

		positionChangedEvent, ok := typedEvent.(*types.PositionChangedEvent)
		if !ok {
			return ctx, fmt.Errorf("expected event is not of type PositionChangedEvent"), false
		}

		if err := types.PositionsAreEqual(&p.expectedEvent.FinalPosition, &positionChangedEvent.FinalPosition); err != nil {
			return ctx, err, false
		}

		if !reflect.DeepEqual(p.expectedEvent, positionChangedEvent) {
			expected, _ := sdk.TypedEventToEvent(p.expectedEvent)
			return ctx, fmt.Errorf(`expected event is not equal to the actual event.
want:
%+v
got:
%+v`, sdk.StringifyEvents([]abci.Event{abci.Event(expected)}), sdk.StringifyEvents([]abci.Event{abciEvent})), false
		}

		return ctx, nil, false
	}

	return ctx, fmt.Errorf("unable to find desired event of type %s", proto.MessageName(p.expectedEvent)), false
}

// PositionChangedEventShouldBeEqual checks that the position changed event is
// equal to the expected event.
func PositionChangedEventShouldBeEqual(
	expectedEvent *types.PositionChangedEvent,
) action.Action {
	return positionChangedEventShouldBeEqual{
		expectedEvent: expectedEvent,
	}
}
