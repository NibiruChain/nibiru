package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/msgservice"
)

func RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&MsgAddMargin{}, "perpv2/add_margin", nil)
	cdc.RegisterConcrete(&MsgRemoveMargin{}, "perpv2/remove_margin", nil)
	cdc.RegisterConcrete(&MsgMarketOrder{}, "perpv2/market_order", nil)
	cdc.RegisterConcrete(&MsgClosePosition{}, "perpv2/close_position", nil)
	cdc.RegisterConcrete(&MsgPartialClose{}, "perpv2/partial_close", nil)
	cdc.RegisterConcrete(&MsgDonateToEcosystemFund{}, "perpv2/donate_to_ef", nil)
	cdc.RegisterConcrete(&MsgMultiLiquidate{}, "perpv2/multi_liquidate", nil)
	cdc.RegisterConcrete(&MsgChangeCollateralDenom{}, "perpv2/change_collateral_denom", nil)
	cdc.RegisterConcrete(&MsgShiftPegMultiplier{}, "perpv2/shift_peg_multiplier", nil)
	cdc.RegisterConcrete(&MsgShiftSwapInvariant{}, "perpv2/shift_swap_invariant", nil)
}

func RegisterInterfaces(registry cdctypes.InterfaceRegistry) {
	registry.RegisterImplementations(
		/* interface */ (*sdk.Msg)(nil),
		/* implementations */
		&MsgRemoveMargin{},
		&MsgAddMargin{},
		&MsgMarketOrder{},
		&MsgClosePosition{},
		&MsgPartialClose{},
		&MsgMultiLiquidate{},
		&MsgChangeCollateralDenom{},
		&MsgShiftPegMultiplier{},
		&MsgShiftSwapInvariant{},
	)

	msgservice.RegisterMsgServiceDesc(registry, &_Msg_serviceDesc)
}

var ModuleCdc = codec.NewProtoCodec(cdctypes.NewInterfaceRegistry())
