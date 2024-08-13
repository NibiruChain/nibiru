package testutil_test

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/v2/x/common/denoms"
	"github.com/NibiruChain/nibiru/v2/x/common/testutil"
	"github.com/NibiruChain/nibiru/v2/x/common/testutil/testapp"
)

func (s *TestSuite) TestEventsUtils() {
	bapp, ctx := testapp.NewNibiruTestAppAndContext()

	// Events on the ctx before we broadcast any txs
	var beforeEvents sdk.Events = ctx.EventManager().Events()

	newCoins := func(coinsStr string) sdk.Coins {
		out, err := sdk.ParseCoinsNormalized(coinsStr)
		if err != nil {
			panic(err)
		}
		return out
	}

	funds := sdk.NewCoins(sdk.NewInt64Coin(denoms.NIBI, 5_000_000))
	_, addrs := testutil.PrivKeyAddressPairs(2)
	senderAddr, otherAddr := addrs[0], addrs[1]
	err := testapp.FundAccount(bapp.BankKeeper, ctx, senderAddr, funds)
	s.NoError(err)

	s.NoError(
		bapp.BankKeeper.SendCoins(ctx, senderAddr, otherAddr, newCoins("12unibi")),
	)

	// Events on the ctx after broadcasting tx
	var sdkEvents sdk.Events = ctx.EventManager().Events()

	s.Run("AssertEventsPresent", func() {
		err = testutil.AssertEventsPresent(sdkEvents,
			[]string{"transfer", "coin_received", "message", "coin_spent"},
		)
		s.NoError(err)
		s.Error(
			testutil.AssertEventsPresent(sdkEvents, []string{"foobar"}),
		)
	})

	s.Run("EventHasAttributeValue", func() {
		var transferEvent sdk.Event
		for _, abciEvent := range sdkEvents {
			if abciEvent.Type == "transfer" {
				transferEvent = abciEvent
			}
		}
		for _, err := range []error{
			testutil.EventHasAttributeValue(transferEvent, "sender", senderAddr.String()),
			testutil.EventHasAttributeValue(transferEvent, "recipient", otherAddr.String()),
			testutil.EventHasAttributeValue(transferEvent, "amount", "12unibi"),
		} {
			s.NoError(err)
		}
	})

	s.Run("FilterNewEvents", func() {
		newEvents := testutil.FilterNewEvents(beforeEvents, sdkEvents)
		lenBefore := len(beforeEvents)
		lenAfter := len(sdkEvents)
		lenNew := len(newEvents)
		s.Equal(lenAfter-lenNew, lenBefore)

		expectedNewEvents := sdkEvents[lenBefore:lenAfter]
		s.Len(expectedNewEvents, lenNew)
		s.ElementsMatch(newEvents, expectedNewEvents)
	})
}
