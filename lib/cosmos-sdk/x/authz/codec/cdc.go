package codec

import (
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/codec"
	cryptocodec "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/crypto/codec"
	sdk "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types"
)

var (
	Amino     = codec.NewLegacyAmino()
	ModuleCdc = codec.NewAminoCodec(Amino)
)

func init() {
	cryptocodec.RegisterCrypto(Amino)
	codec.RegisterEvidences(Amino)
	sdk.RegisterLegacyAminoCodec(Amino)
}
