package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/msgservice"
)

var (
	legacyAminoCdc = codec.NewLegacyAmino()
	ModuleCdc      = codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
)

// RegisterInterfaces register implementations
func RegisterInterfaces(registry codectypes.InterfaceRegistry) {
	registry.RegisterImplementations(
		(*sdk.Msg)(nil),
		&MsgCreateDenom{},
		&MsgChangeAdmin{},
		&MsgUpdateModuleParams{},
		&MsgMint{},
		&MsgBurn{},
		&MsgBurnNative{},
		&MsgSetDenomMetadata{},
	)

	msgservice.RegisterMsgServiceDesc(registry, &_Msg_serviceDesc)
}

func TX_MSG_TYPE_URLS() []string {
	return []string{
		"/nibiru.tokenfactory.v1.MsgCreateDenom",
		"/nibiru.tokenfactory.v1.MsgChangeAdmin",
		"/nibiru.tokenfactory.v1.MsgUpdateModuleParams",
		"/nibiru.tokenfactory.v1.MsgMint",
		"/nibiru.tokenfactory.v1.MsgBurn",
		"/nibiru.tokenfactory.v1.MsgBurnNative",
		"/nibiru.tokenfactory.v1.MsgSetDenomMetadata",
	}
}

// RegisterLegacyAminoCodec registers the necessary x/tokenfactory interfaces and
// concrete types on the provided LegacyAmino codec. These types are used for
// Amino JSON serialization and EIP-712 compatibility.
func RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	for _, ele := range []struct {
		MsgType any
		Name    string
	}{
		{&MsgCreateDenom{}, "nibiru/tokenfactory/create-denom"},
		{&MsgChangeAdmin{}, "nibiru/tokenfactory/change-admin"},
		{&MsgUpdateModuleParams{}, "nibiru/tokenfactory/update-module-params"},
		{&MsgMint{}, "nibiru/tokenfactory/mint"},
		{&MsgBurn{}, "nibiru/tokenfactory/burn"},
		{&MsgSetDenomMetadata{}, "nibiru/tokenfactory/set-denom-metadata"},
	} {
		cdc.RegisterConcrete(ele.MsgType, ele.Name, nil)
	}
}
