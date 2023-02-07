package wasmbinding

import (
	"encoding/json"

	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	wasmvmtypes "github.com/CosmWasm/wasmvm/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	"github.com/NibiruChain/nibiru/wasmbinding/bindings"

	"github.com/NibiruChain/nibiru/x/common/asset"
	perpkeeper "github.com/NibiruChain/nibiru/x/perp/keeper"
	perptypes "github.com/NibiruChain/nibiru/x/perp/types"
)

// CustomMessageDecorator returns decorator for custom CosmWasm bindings messages
func CustomMessageDecorator(
	perp *perpkeeper.Keeper,
) func(wasmkeeper.Messenger) wasmkeeper.Messenger {
	return func(old wasmkeeper.Messenger) wasmkeeper.Messenger {
		return &CustomMessenger{
			wrapped: old,
			perp:    perp,
		}
	}
}

type CustomMessenger struct {
	wrapped wasmkeeper.Messenger
	perp    *perpkeeper.Keeper
}

var _ wasmkeeper.Messenger = (*CustomMessenger)(nil)

// DispatchMsg executes on the contractMsg.
func (m *CustomMessenger) DispatchMsg(
	ctx sdk.Context,
	contractAddr sdk.AccAddress,
	contractIBCPortID string,
	msg wasmvmtypes.CosmosMsg,
) ([]sdk.Event, [][]byte, error) {
	if msg.Custom != nil {
		// only handle the happy path where this is really creating / minting / swapping ...
		// leave everything else for the wrapped version
		var contractMsg bindings.NibiruMsg
		if err := json.Unmarshal(msg.Custom, &contractMsg); err != nil {
			return nil, nil, sdkerrors.Wrap(err, "nibiru msg")
		}

		if contractMsg.OpenPosition != nil {
			return m.openPosition(ctx, contractAddr, contractMsg.OpenPosition)
		} else if contractMsg.ClosePosition != nil {
			return m.closePosition(ctx, contractAddr, contractMsg.ClosePosition)
		} else {
			return nil, nil, wasmvmtypes.UnsupportedRequest{Kind: "unknown Custom variant"}
		}
	}
	return m.wrapped.DispatchMsg(ctx, contractAddr, contractIBCPortID, msg)
}

func (m *CustomMessenger) openPosition(ctx sdk.Context, contractAddr sdk.AccAddress, openPosition *bindings.OpenPosition) ([]sdk.Event, [][]byte, error) {
	err := PerformOpenPosition(m.perp, ctx, contractAddr, openPosition)
	if err != nil {
		return nil, nil, sdkerrors.Wrap(err, "perform open position")
	}
	return nil, nil, nil
}

func PerformOpenPosition(perp *perpkeeper.Keeper, ctx sdk.Context, contractAddr sdk.AccAddress, openPosition *bindings.OpenPosition) error {
	if openPosition == nil {
		return wasmvmtypes.InvalidRequest{Err: "open position null open position"}
	}

	msgServer := perpkeeper.NewMsgServerImpl(*perp)

	msgOpenPosition := &perptypes.MsgOpenPosition{
		Sender:               contractAddr.String(),
		Pair:                 asset.MustNewPair(openPosition.Pair),
		Side:                 perptypes.Side(openPosition.Side),
		QuoteAssetAmount:     openPosition.QuoteAssetAmount,
		Leverage:             openPosition.Leverage,
		BaseAssetAmountLimit: openPosition.BaseAssetAmountLimit,
	}

	if err := msgOpenPosition.ValidateBasic(); err != nil {
		return sdkerrors.Wrap(err, "failed validating MsgOpenPosition")
	}

	// Open position
	_, err := msgServer.OpenPosition(sdk.WrapSDKContext(ctx), msgOpenPosition)
	if err != nil {
		return sdkerrors.Wrap(err, "opening position")
	}

	return nil
}

func (m *CustomMessenger) closePosition(ctx sdk.Context, contractAddr sdk.AccAddress, closePosition *bindings.ClosePosition) ([]sdk.Event, [][]byte, error) {
	err := PerformClosePosition(m.perp, ctx, contractAddr, closePosition)
	if err != nil {
		return nil, nil, sdkerrors.Wrap(err, "perform close position")
	}
	return nil, nil, nil
}

func PerformClosePosition(perp *perpkeeper.Keeper, ctx sdk.Context, contractAddr sdk.AccAddress, closePosition *bindings.ClosePosition) error {
	if closePosition == nil {
		return wasmvmtypes.InvalidRequest{Err: "close position null close position"}
	}

	msgServer := perpkeeper.NewMsgServerImpl(*perp)

	msgClosePosition := &perptypes.MsgClosePosition{
		Sender: contractAddr.String(),
		Pair:   asset.MustNewPair(closePosition.Pair),
	}

	if err := msgClosePosition.ValidateBasic(); err != nil {
		return sdkerrors.Wrap(err, "failed validating MsgClosePosition")
	}

	// Close position
	_, err := msgServer.ClosePosition(sdk.WrapSDKContext(ctx), msgClosePosition)
	if err != nil {
		return sdkerrors.Wrap(err, "closing position")
	}

	return nil
}
