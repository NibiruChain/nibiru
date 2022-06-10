package testutil

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/gogo/protobuf/proto"
	"github.com/stretchr/testify/require"
	abci "github.com/tendermint/tendermint/abci/types"
)

func RequireEqualWithMessage(
	t require.TestingT, expected interface{}, actual interface{}, varName string) {
	require.Equalf(t, expected, actual,
		"Expected '%s': %d,\nActual '%s': %d",
		varName, expected, varName, actual)
}

func RequireNotHasTypedEvent(t require.TestingT, ctx sdk.Context, event proto.Message) {
	name := proto.MessageName(event)
	for _, ev := range ctx.EventManager().Events() {
		if ev.Type == name {
			t.Errorf("unexpected event found: %s", name)
		}
	}
}

func RequireHasTypedEvent(t require.TestingT, ctx sdk.Context, event proto.Message) {
	for _, abciEvent := range ctx.EventManager().Events() {
		if abciEvent.Type != proto.MessageName(event) {
			continue
		}
		typedEvent, err := sdk.ParseTypedEvent(abci.Event{
			Type:       abciEvent.Type,
			Attributes: abciEvent.Attributes,
		})
		require.NoError(t, err)

		require.Equal(t, event, typedEvent, "events do not match")
		return
	}

	t.Errorf("event not found")
}
