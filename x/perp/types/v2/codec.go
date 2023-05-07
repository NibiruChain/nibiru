package v2

import (
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/msgservice"
)

func RegisterCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&MsgAddMargin{}, "perp/add_margin", nil)
	cdc.RegisterConcrete(&MsgRemoveMargin{}, "perp/remove_margin", nil)
	cdc.RegisterConcrete(&MsgOpenPosition{}, "perp/open_position", nil)
	cdc.RegisterConcrete(&MsgClosePosition{}, "perp/close_position", nil)
	cdc.RegisterConcrete(&MsgDonateToEcosystemFund{}, "perp/donate_to_ef", nil)
	cdc.RegisterConcrete(&MsgMultiLiquidate{}, "perp/multi_liquidate", nil)
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
