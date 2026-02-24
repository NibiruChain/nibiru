package evmtrader

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/NibiruChain/nibiru/v2/x/evm/embeds"
	"github.com/NibiruChain/nibiru/sai-trading/abis"
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

var (
	perpVaultEvmInterfaceABI     abi.ABI
	perpVaultEvmInterfaceABIOnce sync.Once
)

func getPerpVaultEvmInterfaceABI() abi.ABI {
	perpVaultEvmInterfaceABIOnce.Do(func() {
		var artifact struct {
			ABI json.RawMessage `json:"abi"`
		}
		if err := json.Unmarshal(abis.PerpVaultEvmInterface, &artifact); err != nil {
			panic(fmt.Sprintf("parse PerpVaultEvmInterface artifact: %v", err))
		}

		parsed, err := abi.JSON(strings.NewReader(string(artifact.ABI)))
		if err != nil {
			panic(fmt.Sprintf("parse PerpVaultEvmInterface ABI: %v", err))
		}
		perpVaultEvmInterfaceABI = parsed
	})
	return perpVaultEvmInterfaceABI
}
