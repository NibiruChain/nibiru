package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/msgservice"
	authzcodec "github.com/cosmos/cosmos-sdk/x/authz/codec"
)

var (
	legacyAminoCdc = codec.NewLegacyAmino()
	ModuleCdc      = codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
)

// NOTE: This is required for the GetSignBytes function
func init() {
	RegisterLegacyAminoCodec(legacyAminoCdc)

	sdk.RegisterLegacyAminoCodec(legacyAminoCdc)

	// Register all Amino interfaces and concrete types on the authz Amino codec
	// so that this can later be used to properly serialize MsgGrant and MsgExec
	// instances.
	RegisterLegacyAminoCodec(authzcodec.Amino)
}

// RegisterInterfaces register implementations
func RegisterInterfaces(registry codectypes.InterfaceRegistry) {
	registry.RegisterImplementations(
		(*sdk.Msg)(nil),
		&MsgSetFeeTokens{},
	)

	msgservice.RegisterMsgServiceDesc(registry, &_Msg_serviceDesc)
}

func TX_MSG_TYPE_URLS() []string {
	return []string{
		"/nibiru.txfees.v1.MsgSetFeeTokens",
	}
}

func RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	for _, ele := range []struct {
		MsgType any
		Name    string
	}{
		{&MsgSetFeeTokens{}, "nibiru/tokenfactory/set-fee-tokens"},
	} {
		cdc.RegisterConcrete(ele.MsgType, ele.Name, nil)
	}
}
