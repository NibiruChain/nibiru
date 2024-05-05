// Copyright (c) 2023-2024 Nibi, Inc.
package eth

import (
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdktx "github.com/cosmos/cosmos-sdk/types/tx"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
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
