package genmsg

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	v1 "github.com/NibiruChain/nibiru/x/genmsg/v1"
)

func anyToMsg(ir types.InterfaceRegistry, anyMsg *types.Any) (sdk.Msg, error) {
	var sdkMsg sdk.Msg
	err := ir.UnpackAny(anyMsg, &sdkMsg)
	if err != nil {
		return nil, err
	}
	if err = sdkMsg.ValidateBasic(); err != nil {
		return nil, err
	}
	return sdkMsg, nil
}

func validateGenesis(cdc codec.JSONCodec, genesis *v1.GenesisState) error {
	interfaceRegistryProvider, ok := cdc.(interface {
		InterfaceRegistry() types.InterfaceRegistry
	})
	if !ok {
		return fmt.Errorf("codec does not implement InterfaceRegistry")
	}
	interfaceRegistry := interfaceRegistryProvider.InterfaceRegistry()
	// check if all messages are known by the codec
	for i, anyMsg := range genesis.Messages {
		if _, err := anyToMsg(interfaceRegistry, anyMsg); err != nil {
			return fmt.Errorf("at index %d: %w", i, err)
		}
	}
	return nil
}

func initGenesis(context sdk.Context, cdc codec.JSONCodec, router MessageRouter, genesis *v1.GenesisState) error {
	interfaceRegistryProvider, ok := cdc.(interface {
		InterfaceRegistry() types.InterfaceRegistry
	})
	if !ok {
		return fmt.Errorf("codec does not implement InterfaceRegistry")
	}
	interfaceRegistry := interfaceRegistryProvider.InterfaceRegistry()

	// execute all messages in order
	for i, anyMsg := range genesis.Messages {
		msg, err := anyToMsg(interfaceRegistry, anyMsg)
		if err != nil {
			return fmt.Errorf("at index %d: message decoding: %w", i, err)
		}
		handler := router.Handler(msg)
		if handler == nil {
			return fmt.Errorf("at index %d: no handler for message %T %s", i, msg, msg)
		}
		// resp, err := handler(context, msg)
		_, err = handler(context, msg)
		if err != nil {
			return fmt.Errorf("at index %d: message processing: %w", i, err)
		}
		// log.Printf("message %d processed %s: %s", i, msg, resp.String())
	}
	return nil
}
