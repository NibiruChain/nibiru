package app

import (
	"github.com/NibiruChain/nibiru/app/codec"
)

// EncodingConfig specifies the concrete encoding types to use for a given app.
// This is provided for compatibility between protobuf and amino implementations.
type EncodingConfig = codec.EncodingConfig

// RegisterModuleBasics registers an EncodingConfig for amino based test configuration.
func RegisterModuleBasics(encodingConfig EncodingConfig) EncodingConfig {
	ModuleBasics.RegisterLegacyAminoCodec(encodingConfig.Amino)
	ModuleBasics.RegisterInterfaces(encodingConfig.InterfaceRegistry)
	return encodingConfig
}

// MakeEncodingConfigAndRegister creates an EncodingConfig for an amino based test configuration.
func MakeEncodingConfigAndRegister() EncodingConfig {
	encodingConfig := codec.MakeEncodingConfig()
	return RegisterModuleBasics(encodingConfig)
}
