package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/msgservice"
)

func RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&MsgCreatePool{}, "spot/CreatePool", nil)
	cdc.RegisterConcrete(&MsgJoinPool{}, "spot/JoinPool", nil)
	cdc.RegisterConcrete(&MsgExitPool{}, "spot/ExitPool", nil)
	cdc.RegisterConcrete(&MsgSwapAssets{}, "spot/SwapAssets", nil)
}

func RegisterInterfaces(registry cdctypes.InterfaceRegistry) {
	registry.RegisterImplementations(
		/* interface */ (*sdk.Msg)(nil),
		/* implementations */
		&MsgCreatePool{},
		&MsgJoinPool{},
		&MsgExitPool{},
		&MsgSwapAssets{},
	)

	msgservice.RegisterMsgServiceDesc(registry, &_Msg_serviceDesc)
}

var ModuleCdc = codec.NewProtoCodec(cdctypes.NewInterfaceRegistry())
