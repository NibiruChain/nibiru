package evmtrader

import (
	"github.com/NibiruChain/nibiru/v2/x/evm/embeds"
	"github.com/ethereum/go-ethereum/accounts/abi"
)

// getERC20ABI returns the ERC20 ABI.
func getERC20ABI() abi.ABI {
	return *embeds.SmartContract_ERC20MinterWithMetadataUpdates.ABI
}

// getWasmPrecompileABI returns the WASM precompile ABI.
func getWasmPrecompileABI() abi.ABI {
	return *embeds.SmartContract_Wasm.ABI
}
