// Copyright (c) 2023-2024 Nibi, Inc.
package appconst

import (
	"fmt"
	"math/big"
	"runtime"

	db "github.com/cometbft/cometbft-db"
)

const (
	BinaryName = "nibiru"
	BondDenom  = "unibi"
	// AccountAddressPrefix: Bech32 prefix for Nibiru accounts.
	AccountAddressPrefix = "nibi"
)

var (
	DefaultDBBackend db.BackendType = db.PebbleDBBackend
)

// Runtime version vars
var (
	AppVersion = ""
	GitCommit  = ""
	BuildDate  = ""

	GoVersion = ""
	GoArch    = ""
)

func init() {
	if len(AppVersion) == 0 {
		AppVersion = "dev"
	}

	GoVersion = runtime.Version()
	GoArch = runtime.GOARCH
}

func RuntimeVersion() string {
	return fmt.Sprintf(
		"Version %s (%s)\nCompiled at %s using Go %s (%s)",
		AppVersion,
		GitCommit,
		BuildDate,
		GoVersion,
		GoArch,
	)
}

// EIP 155 Chain IDs exported for tests.
const (
	ETH_CHAIN_ID_MAINNET int64 = 420
	ETH_CHAIN_ID_LOCAL   int64 = 256
	ETH_CHAIN_ID_DEVNET  int64 = 500
	ETH_CHAIN_ID_DEFAULT int64 = 3000
)

var knownEthChainIDMap = map[string]int64{
	"cataclysm-1": ETH_CHAIN_ID_MAINNET,

	"nibiru-localnet-0": ETH_CHAIN_ID_LOCAL,
	"nibiru-localnet-1": ETH_CHAIN_ID_LOCAL,
	"nibiru-localnet-2": ETH_CHAIN_ID_LOCAL,
	"nibiru-localnet-3": ETH_CHAIN_ID_LOCAL,

	"nibiru-testnet-0": ETH_CHAIN_ID_DEVNET,
	"nibiru-testnet-1": ETH_CHAIN_ID_DEVNET,
	"nibiru-testnet-2": ETH_CHAIN_ID_DEVNET,
	"nibiru-testnet-3": ETH_CHAIN_ID_DEVNET,

	"nibiru-devnet-0": ETH_CHAIN_ID_DEVNET,
	"nibiru-devnet-1": ETH_CHAIN_ID_DEVNET,
	"nibiru-devnet-2": ETH_CHAIN_ID_DEVNET,
	"nibiru-devnet-3": ETH_CHAIN_ID_DEVNET,
}

// GetEthChainID: Maps the given chain ID from the block's `sdk.Context` to an
// EVM Chain ID (`*big.Int`).
func GetEthChainID(ctxChainID string) (ethChainID *big.Int) {
	ethChainIdInt, found := knownEthChainIDMap[ctxChainID]
	if !found {
		ethChainID = big.NewInt(ETH_CHAIN_ID_DEFAULT)
	} else {
		ethChainID = big.NewInt(ethChainIdInt)
	}
	return ethChainID
}
