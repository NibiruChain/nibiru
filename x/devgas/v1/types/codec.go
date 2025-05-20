package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/msgservice"
)

func init() {
	amino.Seal()
}

var (
	amino = codec.NewLegacyAmino()

	// ModuleCdc references the global erc20 module codec. Note, the codec
	// should ONLY be used in certain instances of tests and for JSON encoding.
	//
	// The actual codec used for serialization should be provided to
	// modules/erc20 and defined at the application level.
	ModuleCdc = codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
)

const (
	// Amino names
	cancelFeeShareName   = "nibiru/MsgCancelFeeShare"
	registerFeeShareName = "nibiru/MsgRegisterFeeShare"
	updateFeeShareName   = "nibiru/MsgUpdateFeeShare"
	updateFeeShareParams = "nibiru/MsgUpdateParams"
)

// RegisterInterfaces register implementations
func RegisterInterfaces(registry codectypes.InterfaceRegistry) {
	registry.RegisterImplementations(
		(*sdk.Msg)(nil),
		&MsgRegisterFeeShare{},
		&MsgCancelFeeShare{},
		&MsgUpdateFeeShare{},
		&MsgUpdateParams{},
	)

	msgservice.RegisterMsgServiceDesc(registry, &_Msg_serviceDesc)
}

// RegisterLegacyAminoCodec registers the necessary x/FeeShare interfaces and
// concrete types on the provided LegacyAmino codec. These types are used for
// Amino JSON serialization and EIP-712 compatibility.
func RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&MsgCancelFeeShare{}, cancelFeeShareName, nil)
	cdc.RegisterConcrete(&MsgRegisterFeeShare{}, registerFeeShareName, nil)
	cdc.RegisterConcrete(&MsgUpdateFeeShare{}, updateFeeShareName, nil)
	cdc.RegisterConcrete(&MsgUpdateParams{}, updateFeeShareParams, nil)
}
