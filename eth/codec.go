// Copyright (c) 2023-2024 Nibi, Inc.
package eth

import (
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdktx "github.com/cosmos/cosmos-sdk/types/tx"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	"github.com/NibiruChain/nibiru/app/appconst"
)

const (
	EthBaseDenom = appconst.BondDenom
	// EIP155ChainID_Testnet: Chain ID for a testnet Nibiru following the
	// format proposed by Vitalik in EIP155.
	EIP155ChainID_Testnet = "nibirutest_420-1"

	DefaultGasPrice = 20

	// ProtocolVersion: Latest supported version of the Ethereum protocol.
	// Matches the message types and expected APIs.
	// As of April, 2024, the highest protocol version on Ethereum mainnet is
	// "eth/68".
	// See https://github.com/ethereum/devp2p/blob/master/caps/eth.md#change-log
	// for the historical summary of each version.
	ProtocolVersion = 65
)

// Transaction extension protobuf type URLs
const (
	TYPE_URL_WEB3_TX        = "/eth.types.v1.ExtensionOptionsWeb3Tx"
	TYPE_URL_DYNAMIC_FEE_TX = "/eth.types.v1.ExtensionOptionDynamicFeeTx"
)

// RegisterInterfaces registers the tendermint concrete client-related
// implementations and interfaces.
func RegisterInterfaces(registry codectypes.InterfaceRegistry) {
	// proto name: "cosmos.auth.v1beta1.AccountI"
	registry.RegisterImplementations(
		(*authtypes.AccountI)(nil),
		&EthAccount{},
		// Also impl by: [
		//   &authtypes.BaseAccount{},
		//   &authtypes.ModuleAccount{},
		// ]
	)

	// proto name: "cosmos.auth.v1beta1.GenesisAccount"
	registry.RegisterImplementations(
		(*authtypes.GenesisAccount)(nil),
		&EthAccount{},
		// Also impl by: [
		//   &authtypes.BaseAccount{},
		//   &authtypes.ModuleAccount{},
		// ]
	)

	// proto name: "cosmos.tx.v1beta1.TxExtensionOptionI"
	registry.RegisterImplementations(
		(*sdktx.TxExtensionOptionI)(nil),
		&ExtensionOptionsWeb3Tx{},
		&ExtensionOptionDynamicFeeTx{},
	)
}
