package testutil

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/cosmos/gogoproto/proto"

	"github.com/NibiruChain/nibiru/x/common/set"

	abci "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

// FilterNewEvents returns only the new events from afterEvents that were not present in beforeEvents
func FilterNewEvents(beforeEvents, afterEvents sdk.Events) sdk.Events {
	newEvents := make(sdk.Events, 0)

	for _, afterEvent := range afterEvents {
		found := false
		for _, beforeEvent := range beforeEvents {
			if reflect.DeepEqual(afterEvent, beforeEvent) {
				found = true
				break
			}
		}
		if !found {
			newEvents = append(newEvents, afterEvent)
		}
	}

	return newEvents
}

// AssertEventsPresent: Errors if the given event type is not present in events
func AssertEventPresent(events sdk.Events, eventType string) error {
	foundTypes := set.New[string]()
	for _, event := range events {
		if event.Type == eventType {
			return nil
		}
		foundTypes.Add(event.Type)
	}
	return fmt.Errorf("event \"%s\" not found within set: %s", eventType, foundTypes.ToSlice())
}

// AssertEventsPresent: Errors if the given event types are not present in events
func AssertEventsPresent(events sdk.Events, eventTypes []string) (err error) {
	for _, eventType := range eventTypes {
		err := AssertEventPresent(events, eventType)
		if err != nil {
			return err
		}
	}
	return
}

// RequireNotHasTypedEvent: Error if an event type matches the proto.Message name
func RequireNotHasTypedEvent(t require.TestingT, ctx sdk.Context, event proto.Message) {
	name := proto.MessageName(event)
	for _, ev := range ctx.EventManager().Events() {
		if ev.Type == name {
			t.Errorf("unexpected event found: %s", name)
		}
	}
}

func RequireContainsTypedEvent(t require.TestingT, ctx sdk.Context, event proto.Message) {
	eventType := proto.MessageName(event)
	foundEvents := []proto.Message{}
	for _, abciEvent := range ctx.EventManager().Events() {
		if abciEvent.Type != eventType {
			continue
		}
		typedEvent, err := sdk.ParseTypedEvent(abci.Event{
			Type:       abciEvent.Type,
			Attributes: abciEvent.Attributes,
		})
		require.NoError(t, err)

		if reflect.DeepEqual(typedEvent, event) {
			return
		} else {
			foundEvents = append(foundEvents, typedEvent)
		}
	}

	t.Errorf("event not found, event: %+v, found events: %+v", event, foundEvents)
}

// EventHasAttributeValue parses the given ABCI event at a key to see if it
// matches (contains) the wanted value.
//
// Args:
//   - abciEvent: The event under test
//   - key: The key for which we'll check the value
//   - want: The desired value
func EventHasAttributeValue(abciEvent sdk.Event, key string, want string) error {
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
