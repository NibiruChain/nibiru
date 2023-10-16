package app

import (
	"github.com/NibiruChain/nibiru/app/codec"
	"github.com/cosmos/cosmos-sdk/std"
)

type EncodingConfig = codec.EncodingConfig

// MakeEncodingConfig creates an EncodingConfig for an amino based test configuration.
func MakeEncodingConfig() codec.EncodingConfig {
	encodingConfig := codec.MakeEncodingConfig()
	std.RegisterLegacyAminoCodec(encodingConfig.Amino)
	std.RegisterInterfaces(encodingConfig.InterfaceRegistry)
	ModuleBasics.RegisterLegacyAminoCodec(encodingConfig.Amino)
	ModuleBasics.RegisterInterfaces(encodingConfig.InterfaceRegistry)
	return encodingConfig
}
