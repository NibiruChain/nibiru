package wasmext

import (
	"github.com/NibiruChain/nibiru/v2/x/evm"

	"cosmossdk.io/errors"
	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	wasm "github.com/CosmWasm/wasmd/x/wasm/types"
	wasmvmtypes "github.com/CosmWasm/wasmvm/types"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/codec"
	sdkcodec "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// NibiruWasmOptions: Wasm Options are extension points to instantiate the Wasm
// keeper with non-default values
func NibiruWasmOptions(
	grpcQueryRouter *baseapp.GRPCQueryRouter,
	appCodec codec.Codec,
	msgHandlerArgs MsgHandlerArgs,
) []wasmkeeper.Option {
	wasmQueryOption := wasmkeeper.WithQueryPlugins(&wasmkeeper.QueryPlugins{
		Stargate: wasmkeeper.AcceptListStargateQuerier(
			WasmAcceptedStargateQueries(),
			grpcQueryRouter,
			appCodec,
		),
	})

	wasmMsgHandlerOption := wasmkeeper.WithMessageHandler(WasmMessageHandler(msgHandlerArgs))

	return []wasmkeeper.Option{
		wasmQueryOption,
		wasmMsgHandlerOption,
	}
}

func (h SDKMessageHandler) handleSdkMessage(ctx sdk.Context, contractAddr sdk.Address, msg sdk.Msg) (*sdk.Result, error) {
	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	// make sure this account can send it
	for _, acct := range msg.GetSigners() {
		if !acct.Equals(contractAddr) {
			return nil, errors.Wrap(sdkerrors.ErrUnauthorized, "contract doesn't have permission")
		}
	}

	msgTypeUrl := sdk.MsgTypeURL(msg)
	if msgTypeUrl == sdk.MsgTypeURL(new(evm.MsgEthereumTx)) {
		return nil, errors.Wrap(sdkerrors.ErrUnauthorized, "Wasm VM to EVM call pattern is not yet supported")
	}

	// find the handler and execute it
	if handler := h.router.Handler(msg); handler != nil {
		// ADR 031 request type routing
		msgResult, err := handler(ctx, msg)
		return msgResult, err
	}
	// legacy sdk.Msg routing
	// Assuming that the app developer has migrated all their Msgs to
	// proto messages and has registered all `Msg services`, then this
	// path should never be called, because all those Msgs should be
	// registered within the `msgServiceRouter` already.
	return nil, errors.Wrapf(sdkerrors.ErrUnknownRequest, "can't route message %+v", msg)
}

type MsgHandlerArgs struct {
	Router           MessageRouter
	Ics4Wrapper      wasm.ICS4Wrapper
	ChannelKeeper    wasm.ChannelKeeper
	CapabilityKeeper wasm.CapabilityKeeper
	BankKeeper       wasm.Burner
	Unpacker         sdkcodec.AnyUnpacker
	PortSource       wasm.ICS20TransferPortSource
}

// SDKMessageHandler can handles messages that can be encoded into sdk.Message types and routed.
type SDKMessageHandler struct {
	router   MessageRouter
	encoders msgEncoder
}

// MessageRouter ADR 031 request type routing
type MessageRouter interface {
	Handler(msg sdk.Msg) baseapp.MsgServiceHandler
}

// msgEncoder is an extension point to customize encodings
type msgEncoder interface {
	// Encode converts wasmvm message to n cosmos message types
	Encode(ctx sdk.Context, contractAddr sdk.AccAddress, contractIBCPortID string, msg wasmvmtypes.CosmosMsg) ([]sdk.Msg, error)
}

// WasmMessageHandler is a replacement constructor for
// [wasmkeeper.NewDefaultMessageHandler] inside of [wasmkeeper.NewKeeper].
func WasmMessageHandler(
	args MsgHandlerArgs,
) wasmkeeper.Messenger {
	encoders := wasmkeeper.DefaultEncoders(args.Unpacker, args.PortSource)
	return wasmkeeper.NewMessageHandlerChain(
		NewSDKMessageHandler(args.Router, encoders),
		wasmkeeper.NewIBCRawPacketHandler(args.Ics4Wrapper, args.ChannelKeeper, args.CapabilityKeeper),
		wasmkeeper.NewBurnCoinMessageHandler(args.BankKeeper),
	)
}

func NewSDKMessageHandler(router MessageRouter, encoders msgEncoder) SDKMessageHandler {
	return SDKMessageHandler{
		router:   router,
		encoders: encoders,
	}
}

func (h SDKMessageHandler) DispatchMsg(ctx sdk.Context, contractAddr sdk.AccAddress, contractIBCPortID string, msg wasmvmtypes.CosmosMsg) (events []sdk.Event, data [][]byte, err error) {
	sdkMsgs, err := h.encoders.Encode(ctx, contractAddr, contractIBCPortID, msg)
	if err != nil {
		return nil, nil, err
	}
	for _, sdkMsg := range sdkMsgs {
		res, err := h.handleSdkMessage(ctx, contractAddr, sdkMsg)
		if err != nil {
			return nil, nil, err
		}
		// append data
		data = append(data, res.Data)
		// append events
		sdkEvents := make([]sdk.Event, len(res.Events))
		for i := range res.Events {
			sdkEvents[i] = sdk.Event(res.Events[i])
		}
		events = append(events, sdkEvents...)
	}
	return
}
