package assertion

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/gogo/protobuf/proto"

	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/x/common/testutil/action"
	types "github.com/NibiruChain/nibiru/x/perp/v2/types"
)

// --------------------------------------------------
// Action Functions
// --------------------------------------------------

var _ action.Action = (*containsLiquidateEvent)(nil)
var _ action.Action = (*positionChangedEventShouldBeEqual)(nil)

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

// ProtoToJson converts a proto message into a JSON string using the proto codec.
// A codec defines a functionality for serializing other objects. The proto
// codec provides full Protobuf serialization compatibility.
func ProtoToJson(protoMsg proto.Message) (jsonOut string, err error) {
	protoCodec := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
	var jsonBz json.RawMessage
	jsonBz, err = protoCodec.MarshalJSON(protoMsg)
	return string(jsonBz), err
}

// EventEquals exports functions for comparing sdk.Events to concrete typed
// events implemented as proto.Message instances in Nibiru.
var EventEquals = eventEquals{}

type eventEquals struct{}

// EventHasAttribueValue parses the given ABCI event at a key to see if it
// matches (contains) the wanted value.
//
// Args:
//   - abciEvent: The event under test
//   - key: The key for which we'll check the value
//   - want: The desired value
func EventHasAttribueValue(abciEvent sdk.Event, key string, want string) error {
	attr, ok := abciEvent.GetAttribute(key)
	if !ok {
		return fmt.Errorf("abci event does not contain key: %s", key)
	}

	got := attr.Value
	if !strings.Contains(got, want) {
		return fmt.Errorf("expected %s %s, got %s", key, want, got)
	}

	return nil
}

// --------------------------------------------------
// --------------------------------------------------

type containsLiquidateEvent struct {
	ExpectedEvent types.LiquidationFailedEvent
}

func (act containsLiquidateEvent) Do(_ *app.NibiruApp, ctx sdk.Context) (
	outCtx sdk.Context, err error, isMandatory bool,
) {
	// TODO test(perp): Add action for testing the appearance of of successful
	// liquidation events.
	wantEvent := act.ExpectedEvent
	isEventContained := false
	events := ctx.EventManager().Events()
	eventsOfMatchingType := []abci.Event{}
	for idx, sdkEvent := range events {
		err := EventEquals.LiquidationFailedEvent(sdkEvent, wantEvent, idx)
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
		wantEventJson, _ := ProtoToJson(&wantEvent)
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

func (ee eventEquals) LiquidationFailedEvent(
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
		if err := EventHasAttribueValue(sdkEvent, keyWantPair.key, keyWantPair.want); err != nil {
			fieldErrs = append(fieldErrs, err.Error())
		}
	}

	if len(fieldErrs) != 1 {
		return errors.New(strings.Join(fieldErrs, ". "))
	}
	return nil
}

func (ee eventEquals) PositionChangedEvent(
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
		if err := EventHasAttribueValue(sdkEvent, keyWantPair.key, keyWantPair.want); err != nil {
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
		typedEvent, err := sdk.ParseTypedEvent(abci.Event{
			Type:       gotSdkEvent.Type,
			Attributes: gotSdkEvent.Attributes,
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

		if err := EventEquals.PositionChangedEvent(gotSdkEvent, *theEvent, eventIdx); err != nil {
			return ctx, err, false
		}
	}

	return ctx, nil, false
}
