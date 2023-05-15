package v2

import (
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/msgservice"
)

func RegisterCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&MsgAddMargin{}, "perpv2/add_margin", nil)
	cdc.RegisterConcrete(&MsgRemoveMargin{}, "perpv2/remove_margin", nil)
	cdc.RegisterConcrete(&MsgOpenPosition{}, "perpv2/open_position", nil)
	cdc.RegisterConcrete(&MsgClosePosition{}, "perpv2/close_position", nil)
	cdc.RegisterConcrete(&MsgDonateToEcosystemFund{}, "perpv2/donate_to_ef", nil)
	cdc.RegisterConcrete(&MsgMultiLiquidate{}, "perpv2/multi_liquidate", nil)
}

func RegisterInterfaces(registry cdctypes.InterfaceRegistry) {
	registry.RegisterImplementations(
		/* interface */ (*sdk.Msg)(nil),
		/* implementations */
		&MsgRemoveMargin{},
		&MsgAddMargin{},
		&MsgOpenPosition{},
		&MsgClosePosition{},
		&MsgMultiLiquidate{},
	)

	msgservice.RegisterMsgServiceDesc(registry, &_Msg_serviceDesc)
}

var (
	ModuleCdc = codec.NewProtoCodec(cdctypes.NewInterfaceRegistry())
)
