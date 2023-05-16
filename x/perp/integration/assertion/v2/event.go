package assertion

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/gogo/protobuf/proto"
	abci "github.com/tendermint/tendermint/abci/types"

	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/x/common/testutil/action"
	v2types "github.com/NibiruChain/nibiru/x/perp/v2/types"
)

type positionChangedEventShouldBeEqual struct {
	ExpectedEvent *v2types.PositionChangedEvent
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

		theEvent, ok := typedEvent.(*v2types.PositionChangedEvent)
		if !ok {
			return ctx, fmt.Errorf("expected event is not of type PositionChangedEvent"), false
		}

		if theEvent.Pair != p.ExpectedEvent.Pair {
			return ctx, fmt.Errorf("expected pair %s, got %s", p.ExpectedEvent.Pair, theEvent.Pair), false
		}

		if theEvent.TraderAddress != p.ExpectedEvent.TraderAddress {
			return ctx, fmt.Errorf("expected trader address %s, got %s", p.ExpectedEvent.TraderAddress, theEvent.TraderAddress), false
		}

		if !theEvent.Margin.Equal(p.ExpectedEvent.Margin) {
			return ctx, fmt.Errorf("expected margin %s, got %s", p.ExpectedEvent.Margin, theEvent.Margin), false
		}

		if !theEvent.PositionNotional.Equal(p.ExpectedEvent.PositionNotional) {
			return ctx, fmt.Errorf("expected position notional %s, got %s", p.ExpectedEvent.PositionNotional, theEvent.PositionNotional), false
		}

		if !theEvent.ExchangedSize.Equal(p.ExpectedEvent.ExchangedSize) {
			return ctx, fmt.Errorf("expected exchanged size %s, got %s", p.ExpectedEvent.ExchangedSize, theEvent.ExchangedSize), false
		}

		if !theEvent.ExchangedNotional.Equal(p.ExpectedEvent.ExchangedNotional) {
			return ctx, fmt.Errorf("expected exchanged notional %s, got %s", p.ExpectedEvent.ExchangedNotional, theEvent.ExchangedNotional), false
		}

		if !theEvent.TransactionFee.Equal(p.ExpectedEvent.TransactionFee) {
			return ctx, fmt.Errorf("expected transaction fee %s, got %s", p.ExpectedEvent.TransactionFee, theEvent.TransactionFee), false
		}

		if !theEvent.PositionSize.Equal(p.ExpectedEvent.PositionSize) {
			return ctx, fmt.Errorf("expected position size %s, got %s", p.ExpectedEvent.PositionSize, theEvent.PositionSize), false
		}

		if !theEvent.RealizedPnl.Equal(p.ExpectedEvent.RealizedPnl) {
			return ctx, fmt.Errorf("expected realized pnl %s, got %s", p.ExpectedEvent.RealizedPnl, theEvent.RealizedPnl), false
		}

		if !theEvent.UnrealizedPnlAfter.Equal(p.ExpectedEvent.UnrealizedPnlAfter) {
			return ctx, fmt.Errorf("expected unrealized pnl after %s, got %s", p.ExpectedEvent.UnrealizedPnlAfter, theEvent.UnrealizedPnlAfter), false
		}

		if !theEvent.BadDebt.Equal(p.ExpectedEvent.BadDebt) {
			return ctx, fmt.Errorf("expected bad debt %s, got %s", p.ExpectedEvent.BadDebt, theEvent.BadDebt), false
		}

		if !theEvent.FundingPayment.Equal(p.ExpectedEvent.FundingPayment) {
			return ctx, fmt.Errorf("expected funding payment %s, got %s", p.ExpectedEvent.FundingPayment, theEvent.FundingPayment), false
		}

		if theEvent.BlockHeight != p.ExpectedEvent.BlockHeight {
			return ctx, fmt.Errorf("expected block height %d, got %d", p.ExpectedEvent.BlockHeight, theEvent.BlockHeight), false
		}

		if theEvent.BlockTimeMs != p.ExpectedEvent.BlockTimeMs {
			return ctx, fmt.Errorf("expected block time ms %d, got %d", p.ExpectedEvent.BlockTimeMs, theEvent.BlockTimeMs), false
		}
	}

	return ctx, nil, false
}

// PositionChangedEventShouldBeEqual checks that the position changed event is equal to the expected event.
func PositionChangedEventShouldBeEqual(
	expectedEvent *v2types.PositionChangedEvent,
) action.Action {
	return positionChangedEventShouldBeEqual{
		ExpectedEvent: expectedEvent,
	}
}
