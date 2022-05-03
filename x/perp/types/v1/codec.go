package v1

import (
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/msgservice"

	grpc "google.golang.org/grpc"
)

func RegisterCodec(cdc *codec.LegacyAmino) {
	// cdc.RegisterConcrete(&MsgFoo{}, "perp/Foo", nil)
}

func RegisterInterfaces(registry cdctypes.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil)) // &MsgFoo{},

	var _Msg_serviceDesc grpc.ServiceDesc // placeholder until msgs are implemented
	msgservice.RegisterMsgServiceDesc(registry, &_Msg_serviceDesc)
}

var (
	Amino     = codec.NewLegacyAmino()
	ModuleCdc = codec.NewProtoCodec(cdctypes.NewInterfaceRegistry())
)
