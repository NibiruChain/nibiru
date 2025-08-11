// Copyright (c) 2023-2024 Nibi, Inc.
package encoding

import (
	"cosmossdk.io/simapp/params"
	amino "github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/std"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth/tx"

	"github.com/NibiruChain/nibiru/v2/app"
	"github.com/NibiruChain/nibiru/v2/eth"
	cryptocodec "github.com/NibiruChain/nibiru/v2/eth/crypto/codec"
)

// MakeConfig creates an EncodingConfig for testing
func MakeConfig() params.EncodingConfig {
	interfaceRegistry := types.NewInterfaceRegistry()
	protoCodec := amino.NewProtoCodec(interfaceRegistry)

	encodingConfig := params.EncodingConfig{
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
