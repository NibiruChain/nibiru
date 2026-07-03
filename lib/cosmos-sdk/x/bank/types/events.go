package types

import (
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// bank module event types
const (
	EventTypeTransfer = "transfer"

	AttributeKeyRecipient = "recipient"
	AttributeKeySender    = sdk.AttributeKeySender

	// supply and balance tracking events name and attributes
	EventTypeCoinSpent    = "coin_spent"
	EventTypeCoinReceived = "coin_received"
	EventTypeCoinMint     = "coinbase" // NOTE(fdymylja): using mint clashes with mint module event
	EventTypeCoinBurn     = "burn"

	AttributeKeySpender  = "spender"
	AttributeKeyReceiver = "receiver"
	AttributeKeyMinter   = "minter"
	AttributeKeyBurner   = "burner"

	EventTypeWeiChange          = "wei_change"
	AttributeKeyWeiChangeAddrs  = "wei_change_addrs"
	AttributeKeyWeiChangeReason = "wei_change_reason"
)

func weiChangeReasonAttr(reason string) sdk.Attribute {
	return sdk.NewAttribute(AttributeKeyWeiChangeReason, reason)
}

var (
	WeiChangeReason_SendCoins = weiChangeReasonAttr("bank.SendCoins")
	// InputOutputCoins - uses addCoins, subUnlockedCoins
	WeiChangeReason_InputOutputCoins = weiChangeReasonAttr("bank.InputOutputCoins")

	// UndelegateCoins - uses addCoins, subUnlockedCoins
	WeiChangeReason_UndelegateCoins = weiChangeReasonAttr("bank.UndelegateCoins")

	// MintCoins - uses addCoins
	WeiChangeReason_MintCoins = weiChangeReasonAttr("bank.MintCoins")

	// BurnCoins - uses subUnlockedCoins
	WeiChangeReason_BurnCoins = weiChangeReasonAttr("bank.BurnCoins")

	// DelegateCoins - uses addCoins, setBalance
	WeiChangeReason_DelegateCoins = weiChangeReasonAttr("bank.DelegateCoins")

	// AddWei - EVM SDB only
	WeiChangeReason_AddWei = weiChangeReasonAttr("evm.AddWei")

	// SubWei - EVM SDB only
	WeiChangeReason_SubWei = weiChangeReasonAttr("evm.SubWei")
)

func WeiChangeAddrsString(addrs ...string) string {
	if len(addrs) == 0 {
		return ""
	} else if len(addrs) == 1 {
		return addrs[0]
	}
	return strings.Join(addrs, ", ")
}

// NewCoinSpentEvent constructs a new coin spent sdk.Event
func NewCoinSpentEvent(spender sdk.AccAddress, amount sdk.Coins) sdk.Event {
	return sdk.NewEvent(
		EventTypeCoinSpent,
		sdk.NewAttribute(AttributeKeySpender, spender.String()),
		sdk.NewAttribute(sdk.AttributeKeyAmount, amount.String()),
	)
}

// NewCoinReceivedEvent constructs a new coin received sdk.Event
func NewCoinReceivedEvent(receiver sdk.AccAddress, amount sdk.Coins) sdk.Event {
	return sdk.NewEvent(
		EventTypeCoinReceived,
		sdk.NewAttribute(AttributeKeyReceiver, receiver.String()),
		sdk.NewAttribute(sdk.AttributeKeyAmount, amount.String()),
	)
}

// NewCoinMintEvent construct a new coin minted sdk.Event
func NewCoinMintEvent(minter sdk.AccAddress, amount sdk.Coins) sdk.Event {
	return sdk.NewEvent(
		EventTypeCoinMint,
		sdk.NewAttribute(AttributeKeyMinter, minter.String()),
		sdk.NewAttribute(sdk.AttributeKeyAmount, amount.String()),
	)
}

// NewCoinBurnEvent constructs a new coin burned sdk.Event
func NewCoinBurnEvent(burner sdk.AccAddress, amount sdk.Coins) sdk.Event {
	return sdk.NewEvent(
		EventTypeCoinBurn,
		sdk.NewAttribute(AttributeKeyBurner, burner.String()),
		sdk.NewAttribute(sdk.AttributeKeyAmount, amount.String()),
	)
}

// NewEventWeiChange constructs a "wei_change" event.
func NewEventWeiChange(reason sdk.Attribute, addrs ...string) sdk.Event {
	return sdk.NewEvent(
		EventTypeWeiChange,
		reason,
		sdk.NewAttribute(
			AttributeKeyWeiChangeAddrs,
			WeiChangeAddrsString(addrs...),
		),
	)
}
