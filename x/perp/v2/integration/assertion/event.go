package assertion

import (
	"errors"
	"fmt"
	"strings"

	abci "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/gogo/protobuf/proto"

	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/x/common/testutil"
	"github.com/NibiruChain/nibiru/x/common/testutil/action"
	types "github.com/NibiruChain/nibiru/x/perp/v2/types"
)

var _ action.Action = (*containsLiquidateEvent)(nil)
var _ action.Action = (*positionChangedEventShouldBeEqual)(nil)

// TODO test(perp): Add action for testing the appearance of of successful
// liquidation events.

// PositionChangedEventShouldBeEqual checks that the position changed event is
// equal to the expected event.
func PositionChangedEventShouldBeEqual(
	expectedEvent *types.PositionChangedEvent,
) action.Action {
	return positionChangedEventShouldBeEqual{
		ExpectedEvent: expectedEvent,
	}
}

// ContainsLiquidateEvent checks if a typed event (proto.Message) is contained in the
// event manager of the app context.
func ContainsLiquidateEvent(
	expectedEvent types.LiquidationFailedEvent,
) action.Action {
	return containsLiquidateEvent{
		ExpectedEvent: expectedEvent,
	}
}

// eventEquals exports functions for comparing sdk.Events to concrete typed
// events implemented as proto.Message instances in Nibiru.
var eventEquals = iEventEquals{}

type iEventEquals struct{}

// --------------------------------------------------
// --------------------------------------------------

type containsLiquidateEvent struct {
	ExpectedEvent types.LiquidationFailedEvent
}

func (act containsLiquidateEvent) Do(_ *app.NibiruApp, ctx sdk.Context) (
	outCtx sdk.Context, err error, isMandatory bool,
) {
	wantEvent := act.ExpectedEvent
	isEventContained := false
	events := ctx.EventManager().Events()
	eventsOfMatchingType := []abci.Event{}
	for idx, sdkEvent := range events {
		err := eventEquals.LiquidationFailedEvent(sdkEvent, wantEvent, idx)
		if err == nil {
			isEventContained = true
			break
		} else if sdkEvent.Type != "nibiru.perp.v2.LiquidationFailedEvent" {
			continue
		} else if sdkEvent.Type == "nibiru.perp.v2.LiquidationFailedEvent" && err != nil {
			abciEvent := abci.Event{
				Type:       sdkEvent.Type,
				Attributes: sdkEvent.Attributes,
			}
			eventsOfMatchingType = append(eventsOfMatchingType, abciEvent)
		}
	}

	if isEventContained {
		// happy path
		return ctx, nil, true
	} else {
		// Show descriptive error messages if the expected event is missing
		wantEventJson, _ := testutil.ProtoToJson(&wantEvent)
		var matchingEvents string = sdk.StringifyEvents(eventsOfMatchingType).String()
		return ctx, errors.New(
			strings.Join([]string{
				fmt.Sprintf("expected the context event manager to contain event: %s.", wantEventJson),
				fmt.Sprintf("found %v events:", len(events)),
				fmt.Sprintf("events of matching type:\n%v", matchingEvents),
			}, "\n"),
		), false
	}
}

func (ee iEventEquals) LiquidationFailedEvent(
	sdkEvent sdk.Event, tevent types.LiquidationFailedEvent, eventIdx int,
) error {
	fieldErrs := []string{fmt.Sprintf("[DEBUG eventIdx: %v]", eventIdx)}

	for _, keyWantPair := range []struct {
		key  string
		want string
	}{
		{"pair", tevent.Pair.String()},
		{"trader", tevent.Trader},
		{"liquidator", tevent.Liquidator},
		{"reason", tevent.Reason.String()},
	} {
		if err := testutil.EventHasAttribueValue(sdkEvent, keyWantPair.key, keyWantPair.want); err != nil {
			fieldErrs = append(fieldErrs, err.Error())
		}
	}

	if len(fieldErrs) != 1 {
		return errors.New(strings.Join(fieldErrs, ". "))
	}
	return nil
}

func (ee iEventEquals) PositionChangedEvent(
	sdkEvent sdk.Event, tevent types.PositionChangedEvent, eventIdx int,
) error {
	fieldErrs := []string{fmt.Sprintf("[DEBUG eventIdx: %v]", eventIdx)}

	for _, keyWantPair := range []struct {
		key  string
		want string
	}{
		{"position_notional", tevent.PositionNotional.String()},
		{"transaction_fee", tevent.TransactionFee.String()},
		{"bad_debt", tevent.BadDebt.String()},
		{"realized_pnl", tevent.RealizedPnl.String()},
		{"funding_payment", tevent.FundingPayment.String()},
		{"block_height", fmt.Sprintf("%v", tevent.BlockHeight)},
		{"margin_to_user", tevent.MarginToUser.String()},
		{"change_reason", string(tevent.ChangeReason)},
	} {
		if err := testutil.EventHasAttribueValue(sdkEvent, keyWantPair.key, keyWantPair.want); err != nil {
			fieldErrs = append(fieldErrs, err.Error())
		}
	}

	if len(fieldErrs) != 1 {
		return errors.New(strings.Join(fieldErrs, ". "))
	}
	return nil
}

type positionChangedEventShouldBeEqual struct {
	ExpectedEvent *types.PositionChangedEvent
}

func (p positionChangedEventShouldBeEqual) Do(_ *app.NibiruApp, ctx sdk.Context) (sdk.Context, error, bool) {
	for eventIdx, gotSdkEvent := range ctx.EventManager().Events() {
		if gotSdkEvent.Type != proto.MessageName(p.ExpectedEvent) {
			continue
		}
		gotProtoMessage, err := sdk.ParseTypedEvent(abci.Event{
			Type:       gotSdkEvent.Type,
			Attributes: gotSdkEvent.Attributes,
		})
		if err != nil {
			return ctx, err, false
		}

		gotTypedEvent, ok := gotProtoMessage.(*types.PositionChangedEvent)
		if !ok {
			return ctx, fmt.Errorf("expected event is not of type PositionChangedEvent"), false
		}

		if err := types.PositionsAreEqual(&p.ExpectedEvent.FinalPosition, &gotTypedEvent.FinalPosition); err != nil {
			return ctx, err, false
		}

		if err := eventEquals.PositionChangedEvent(gotSdkEvent, *gotTypedEvent, eventIdx); err != nil {
			return ctx, err, false
		}
	}

	return ctx, nil, false
}
