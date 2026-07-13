//go:build !test_amino
// +build !test_amino

package params

import (
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/codec"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/codec/types"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/auth/tx"
)

// MakeTestEncodingConfig creates an EncodingConfig for a non-amino based test configuration.
// This function should be used only internally (in the SDK).
// App user should'nt create new codecs - use the app.AppCodec instead.
// [DEPRECATED]
func MakeTestEncodingConfig() EncodingConfig {
	cdc := codec.NewLegacyAmino()
	interfaceRegistry := types.NewInterfaceRegistry()
	protoCdc := codec.NewProtoCodec(interfaceRegistry)

	return EncodingConfig{
		InterfaceRegistry: interfaceRegistry,
		Codec:             protoCdc,
		TxConfig:          tx.NewTxConfig(protoCdc, tx.DefaultSignModes),
		Amino:             cdc,
	}
}
