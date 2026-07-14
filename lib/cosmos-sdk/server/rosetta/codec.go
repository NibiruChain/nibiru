package rosetta

import (
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/codec"
	codectypes "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/codec/types"
	cryptocodec "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/crypto/codec"
	authcodec "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/auth/types"
	bankcodec "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/bank/types"
)

// MakeCodec generates the codec required to interact
// with the cosmos APIs used by the rosetta gateway
func MakeCodec() (*codec.ProtoCodec, codectypes.InterfaceRegistry) {
	ir := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(ir)

	authcodec.RegisterInterfaces(ir)
	bankcodec.RegisterInterfaces(ir)
	cryptocodec.RegisterInterfaces(ir)

	return cdc, ir
}
