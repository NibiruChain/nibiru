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

// PositionChangedEventShouldBeEqual checks that the position changed event is equal to the expected event.
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

type containsLiquidateEvent struct {
	ExpectedEvent types.LiquidationFailedEvent
}

func ProtoToJson(protoMsg proto.Message) (jsonOut string, err error) {
	protoCodec := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
	var jsonBz json.RawMessage
	jsonBz, err = protoCodec.MarshalJSON(protoMsg)
	return string(jsonBz), err
}

func (act containsLiquidateEvent) Do(_ *app.NibiruApp, ctx sdk.Context) (
	outCtx sdk.Context, err error, isMandatory bool,
) {
	// TODO test(perp): Add support for testing the appearance of of successful
	// liquidation events.
	typedEvent := act.ExpectedEvent
	eventContained := false
	errDescriptions := []string{}
	events := ctx.EventManager().Events()
	for idx, abciEvent := range events {
		err := EventEquals.LiquidationFailedEvent(abciEvent, typedEvent, idx)
		if err == nil {
			eventContained = true
			errDescriptions = []string{}
			break
		} else if abciEvent.Type != "nibiru.perp.v2.LiquidationFailedEvent" {
			continue
		} else if abciEvent.Type == "nibiru.perp.v2.LiquidationFailedEvent" && err != nil {
			errDescriptions = append(errDescriptions, err.Error())
		}
	}

	if eventContained {
		// happy path
		return ctx, nil, true
	} else {
		// Show descriptive error messages if the expected event is missing
		teventJson, _ := ProtoToJson(&typedEvent)
		return ctx, fmt.Errorf(
			`expected the context event manager to contain event: %s.
			found %v events: 
			description: %v`,
			teventJson, len(events), errDescriptions,
		), false
	}
}

var EventEquals = eventEquals{}

type eventEquals struct{}

func (ee eventEquals) LiquidationFailedEvent(
	abciEvent sdk.Event, tevent types.LiquidationFailedEvent, eventIdx int,
) error {
	fieldErrs := []string{fmt.Sprintf("DEBUG [eventIdx: %v]", eventIdx)}
	if err := EventHasAttribueValue(abciEvent, "pair", tevent.Pair.String()); err != nil {
		fieldErrs = append(fieldErrs, err.Error())
	}
	if err := EventHasAttribueValue(abciEvent, "trader", tevent.Trader); err != nil {
		fieldErrs = append(fieldErrs, err.Error())
	}
	if err := EventHasAttribueValue(abciEvent, "liquidator", tevent.Liquidator); err != nil {
		fieldErrs = append(fieldErrs, err.Error())
	}
	if err := EventHasAttribueValue(abciEvent, "reason", tevent.Reason.String()); err != nil {
		fieldErrs = append(fieldErrs, err.Error())
	}

	if len(fieldErrs) != 1 {
		return errors.New(strings.Join(fieldErrs, "\n"))
	}
	return nil
}

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

		if !theEvent.MarginToUser.Equal(p.ExpectedEvent.MarginToUser) {
			err := fmt.Errorf("expected exchanged margin %s, got %s",
				p.ExpectedEvent.MarginToUser, theEvent.MarginToUser)
			fieldErrs = append(fieldErrs, err.Error())
		}

		if theEvent.ChangeReason != p.ExpectedEvent.ChangeReason {
			err := fmt.Errorf("expected change type %s, got %s",
				p.ExpectedEvent.ChangeReason, theEvent.ChangeReason)
			fieldErrs = append(fieldErrs, err.Error())
		}

		if len(fieldErrs) != 0 {
			err := strings.Join(fieldErrs, "\n")
			return ctx, errors.New(err), false
		}
	}

	return ctx, nil, false
}
