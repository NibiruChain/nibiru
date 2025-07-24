package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

var (
	legacyAminoCdc = codec.NewLegacyAmino()
	ModuleCdc      = codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
)

// NOTE: This is required for the GetSignBytes function
func init() {
	sdk.RegisterLegacyAminoCodec(legacyAminoCdc)
}
