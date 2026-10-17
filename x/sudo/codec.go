package sudo

import (
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/codec"
	cdctypes "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/codec/types"
	sdk "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types/msgservice"
)

func RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&MsgEditSudoers{}, "sudo/edit_sudoers", nil)
	cdc.RegisterConcrete(&MsgUpdateRoleMembers{}, "sudo/update_role_members", nil)
}

func RegisterInterfaces(registry cdctypes.InterfaceRegistry) {
	registry.RegisterImplementations(
		/* interface */ (*sdk.Msg)(nil),
		/* implementations */
		&MsgEditSudoers{},
		&MsgUpdateRoleMembers{},
		&MsgChangeRoot{},
		&MsgEditZeroGasActors{},
	)

	msgservice.RegisterMsgServiceDesc(registry, &_Msg_serviceDesc)
}

var ModuleCdc = codec.NewProtoCodec(cdctypes.NewInterfaceRegistry())
