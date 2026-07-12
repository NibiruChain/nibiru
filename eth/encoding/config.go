// Copyright (c) 2023-2024 Nibi, Inc.
package encoding

import (
	"github.com/NibiruChain/nibiru/v2/eth/eip712"
	amino "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/codec"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/codec/types"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/std"
	sdk "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/auth/tx"

	"github.com/NibiruChain/nibiru/v2/app"
	"github.com/NibiruChain/nibiru/v2/eth"
	cryptocodec "github.com/NibiruChain/nibiru/v2/eth/crypto/codec"
)

// MakeConfig creates an EncodingConfig for testing
func MakeConfig() eip712.EncodingConfig {
	interfaceRegistry := types.NewInterfaceRegistry()
	protoCodec := amino.NewProtoCodec(interfaceRegistry)

	encodingConfig := eip712.EncodingConfig{
		InterfaceRegistry: interfaceRegistry,
		Codec:             protoCodec,
		Amino:             amino.NewLegacyAmino(),
		TxConfig:          tx.NewTxConfig(protoCodec, tx.DefaultSignModes),
	}

	sdk.RegisterLegacyAminoCodec(encodingConfig.Amino)
	cryptocodec.RegisterCrypto(encodingConfig.Amino)
	amino.RegisterEvidences(encodingConfig.Amino)
	app.ModuleBasics.RegisterLegacyAminoCodec(encodingConfig.Amino)

	std.RegisterInterfaces(interfaceRegistry)
	cryptocodec.RegisterInterfaces(interfaceRegistry)
	eth.RegisterInterfaces(interfaceRegistry)
	app.ModuleBasics.RegisterInterfaces(encodingConfig.InterfaceRegistry)

	return encodingConfig
}
