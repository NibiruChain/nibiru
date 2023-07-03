package assertion

import (
	"errors"
	"fmt"
	"strings"

	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/gogoproto/proto"

	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/x/common/testutil"
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
	expectedEvent types.LiquidationFailedEvent
}

func (act containsLiquidateEvent) Do(_ *app.NibiruApp, ctx sdk.Context) (
	outCtx sdk.Context, err error, isMandatory bool,
) {
	foundEvent := false
	events := ctx.EventManager().Events()
	matchingEvents := []abci.Event{}
	for _, sdkEvent := range events {
		if sdkEvent.Type != proto.MessageName(&act.expectedEvent) {
			continue
		}

		err := assertLiquidationFailedEvent(sdkEvent, act.expectedEvent)
		if err == nil {
			foundEvent = true
			break
		}

		abciEvent := abci.Event{
			Type:       sdkEvent.Type,
			Attributes: sdkEvent.Attributes,
		}
		matchingEvents = append(matchingEvents, abciEvent)
	}

	if foundEvent {
		// happy path
		return ctx, nil, true
	}

	// Show descriptive error messages if the expected event is missing
	expectedEventBz, _ := codec.ProtoMarshalJSON(&act.expectedEvent, nil)
	return ctx, errors.New(
		strings.Join([]string{
			fmt.Sprintf("expected the context event manager to contain event: %s.", string(expectedEventBz)),
			fmt.Sprintf("found %v events:", len(events)),
			fmt.Sprintf("events of matching type:\n%v", sdk.StringifyEvents(matchingEvents).String()),
		}, "\n"),
	), false
}

// ContainsLiquidateEvent checks if a typed event (proto.Message) is contained in the
// event manager of the app context.
func ContainsLiquidateEvent(
	expectedEvent types.LiquidationFailedEvent,
) action.Action {
	return containsLiquidateEvent{
		expectedEvent: expectedEvent,
	}
}

func assertLiquidationFailedEvent(
	sdkEvent sdk.Event, liquidationFailedEvent types.LiquidationFailedEvent,
) error {
	fieldErrs := []string{}

	for _, eventField := range []struct {
		key  string
		want string
	}{
		{"pair", liquidationFailedEvent.Pair.String()},
		{"trader", liquidationFailedEvent.Trader},
		{"liquidator", liquidationFailedEvent.Liquidator},
		{"reason", liquidationFailedEvent.Reason.String()},
	} {
		if err := testutil.EventHasAttributeValue(sdkEvent, eventField.key, eventField.want); err != nil {
			fieldErrs = append(fieldErrs, err.Error())
		}
	}

	if len(fieldErrs) > 0 {
		return errors.New(strings.Join(fieldErrs, ". "))
	}

	return nil
}

func assertPositionChangedEvent(
	sdkEvent sdk.Event, positionChangedEvent types.PositionChangedEvent,
) error {
	badDebtBz, err := codec.ProtoMarshalJSON(&positionChangedEvent.BadDebt, nil)
	if err != nil {
		panic(err)
	}
	transactionFeeBz, err := codec.ProtoMarshalJSON(&positionChangedEvent.TransactionFee, nil)
	if err != nil {
		panic(err)
	}

	fieldErrs := []string{}

	for _, eventField := range []struct {
		key  string
		want string
	}{
		{"position_notional", positionChangedEvent.PositionNotional.String()},
		{"transaction_fee", string(transactionFeeBz)},
		{"bad_debt", string(badDebtBz)},
		{"realized_pnl", positionChangedEvent.RealizedPnl.String()},
		{"funding_payment", positionChangedEvent.FundingPayment.String()},
		{"block_height", fmt.Sprintf("%v", positionChangedEvent.BlockHeight)},
		{"margin_to_user", positionChangedEvent.MarginToUser.String()},
		{"change_reason", string(positionChangedEvent.ChangeReason)},
	} {
		if err := testutil.EventHasAttributeValue(sdkEvent, eventField.key, eventField.want); err != nil {
			fieldErrs = append(fieldErrs, err.Error())
		}
	}

	if len(fieldErrs) > 0 {
		return errors.New(strings.Join(fieldErrs, ". "))
	}

	return nil
}

type positionChangedEventShouldBeEqual struct {
	expectedEvent *types.PositionChangedEvent
}

func (p positionChangedEventShouldBeEqual) Do(_ *app.NibiruApp, ctx sdk.Context) (sdk.Context, error, bool) {
	for _, sdkEvent := range ctx.EventManager().Events() {
		if sdkEvent.Type != proto.MessageName(p.expectedEvent) {
			continue
		}
		typedEvent, err := sdk.ParseTypedEvent(abci.Event{
			Type:       sdkEvent.Type,
			Attributes: sdkEvent.Attributes,
		})
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

		if err := assertPositionChangedEvent(sdkEvent, *p.expectedEvent); err != nil {
			return ctx, err, false
		}
	}

	return ctx, nil, false
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
