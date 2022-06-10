package events

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/stablecoin/types"
)

// x/stablecoin attributes for events
const (
	AttributeFromAddr    = "from"
	AttributeToAddr      = "to"
	AttributeTokenAmount = "amount"
	AttributeTokenDenom  = "denom"
)

func EmitTransfer(
	ctx sdk.Context, coin sdk.Coin, from string, to string,
) error {
	protoEvent := &types.EventTransfer{
		Coin: coin,
		From: from,
		To:   to,
	}
	return ctx.EventManager().EmitTypedEvent(protoEvent)
}

// EmitMintStable emits an event when a Nibiru Stablecoin is minted.
func EmitMintStable(ctx sdk.Context, coin sdk.Coin) error {
	protoEvent := &types.EventMintStable{
		Amount: coin.Amount,
	}
	return ctx.EventManager().EmitTypedEvent(protoEvent)
}

// EmitBurnStable emits an event when a Nibiru Stablecoin is burned.
func EmitBurnStable(ctx sdk.Context, coin sdk.Coin) error {
	protoEvent := &types.EventBurnStable{
		Amount: coin.Amount,
	}
	return ctx.EventManager().EmitTypedEvent(protoEvent)
}

// EmitMintNIBI emits an event when NIBI is minted.
func EmitMintNIBI(ctx sdk.Context, coin sdk.Coin) error {
	protoEvent := &types.EventMintNIBI{
		Amount: coin.Amount,
	}
	return ctx.EventManager().EmitTypedEvent(protoEvent)
}

// EmitBurnNIBI emits an event when NIBI is burned.
func EmitBurnNIBI(ctx sdk.Context, coin sdk.Coin) error {
	protoEvent := &types.EventBurnNIBI{
		Amount: coin.Amount,
	}
	return ctx.EventManager().EmitTypedEvent(protoEvent)
}

// EmitRecollateralize emits an event when a 'Recollateralize' occurs.
func EmitRecollateralize(
	ctx sdk.Context, inCoin sdk.Coin, outCoin sdk.Coin, caller string,
	collRatio sdk.Dec,
) error {
	protoEvent := &types.EventRecollateralize{
		Caller:    caller,
		InCoin:    inCoin,
		OutCoin:   outCoin,
		CollRatio: collRatio,
	}
	return ctx.EventManager().EmitTypedEvent(protoEvent)
}

// EmitBuyback emits an event when a 'Buyback' occurs.
func EmitBuyback(
	ctx sdk.Context, inCoin sdk.Coin, outCoin sdk.Coin, caller string,
	collRatio sdk.Dec,
) error {
	protoEvent := &types.EventBuyback{
		Caller:    caller,
		InCoin:    inCoin,
		OutCoin:   outCoin,
		CollRatio: collRatio,
	}
	return ctx.EventManager().EmitTypedEvent(protoEvent)
}
