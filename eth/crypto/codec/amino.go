// Copyright (c) 2023-2024 Nibi, Inc.
package codec

import (
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/legacy"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"

	"github.com/NibiruChain/nibiru/v2/eth/crypto/ethsecp256k1"
)

// RegisterCrypto registers all crypto dependency types with the provided Amino
// codec.
func RegisterCrypto(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&ethsecp256k1.PubKey{},
		ethsecp256k1.PubKeyName, nil)
	cdc.RegisterConcrete(&ethsecp256k1.PrivKey{},
		ethsecp256k1.PrivKeyName, nil)

	keyring.RegisterLegacyAminoCodec(cdc)

	// NOTE: update SDK's amino codec to include the ethsecp256k1 keys.
	// DO NOT REMOVE unless deprecated on the SDK.
	legacy.Cdc = cdc
}
