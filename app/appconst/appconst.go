// Package appconst defines global constants and utility functions
// used throughout the Nibiru application.
package appconst

// Copyright (c) 2023-2024 Nibi, Inc.

import (
	"fmt"
	"math/big"
	"runtime"
	"strings"

	gethcommon "github.com/ethereum/go-ethereum/common"

	wasm "github.com/CosmWasm/wasmd/x/wasm/types"
	db "github.com/cometbft/cometbft-db"
	"github.com/cosmos/cosmos-sdk/version"
)

const (
	BinaryName = "nibiru"

	// DENOM_UNIBI is the micro-denomination used by x/bank for NIBI balances.
	// One unibi = 10^{-6} NIBI. [DENOM_UNIBI] is the Bank Coin denomination for
	// staking, governance, and gas.
	//
	// The aggregate, 18 decimal, EVM-facing precision comes from combining this
	// with the wei remainder store at 10^{12} wei per unibi. Equivalently, wei
	// is defined as attoNIBI. Thus,
	// 1 NIBI = 10^{18} attoNIBI = 10 = 10^{18} wei = 10^{6} microNIBI.
	DENOM_UNIBI = "unibi"

	// AccountAddressPrefix: Bech32 prefix for Nibiru accounts.
	AccountAddressPrefix = "nibi"

	// ADDR_LEN_EOA is the length (number of bytes) of an [e]xternally [o]wned
	// [a]ccount.
	ADDR_LEN_EOA = 20 // wasm.SDKAddrLen
	// ADDR_LEN_WASM_CONTRACT is the length (number of bytes) of a wasm contract
	// address.
	ADDR_LEN_WASM_CONTRACT = wasm.ContractAddrLen
)

var (
	DefaultDBBackend     db.BackendType = db.PebbleDBBackend
	HavePebbleDBBuildTag bool

	// MAINNET_WNIBI_ADDR is the (real) hex address of WNIBI on mainnet. NIBI acts as
	// "ether" in the Nibiru EVM state. WNIBI is the Nibiru equivalent of WETH on
	// Ethereum.
	MAINNET_WNIBI_ADDR = gethcommon.HexToAddress("0x0CaCF669f8446BeCA826913a3c6B96aCD4b02a97")

	// MAINNET_STNIBI_ADDR is the (real) hex address of stNIBI on mainnet.
	MAINNET_STNIBI_ADDR = gethcommon.HexToAddress("0xcA0a9Fb5FBF692fa12fD13c0A900EC56Bb3f0a7b")
)

// RuntimeVersion returns a formatted string with versioning and build metadata,
// including the Nibiru version, Git commit, Go runtime, architecture, and build tags.
func RuntimeVersion() string {
	info := version.NewInfo()
	nibiruVersion := info.Version
	if len(nibiruVersion) == 0 {
		nibiruVersion = "dev"
	}
	goVersion := runtime.Version()
	goArch := runtime.GOARCH
	return fmt.Sprintf(
		"Nibiru %s: Compiled at Git commit %s using Go %s, arch %s, and build tags (%s)",
		nibiruVersion,
		info.GitCommit,
		goVersion,
		goArch,
		strings.TrimRight(info.BuildTags, ","), // build tags have a trailing comma
	)
}

// EIP 155 Chain IDs for Nibiru
const (
	ETH_CHAIN_ID_MAINNET int64 = 6900

	ETH_CHAIN_ID_TESTNET_1 int64 = 7210
	ETH_CHAIN_ID_TESTNET_2 int64 = 6911
	ETH_CHAIN_ID_TESTNET_3 int64 = 6912

	ETH_CHAIN_ID_DEVNET_1 int64 = 6920
	ETH_CHAIN_ID_DEVNET_2 int64 = 6921
	ETH_CHAIN_ID_DEVNET_3 int64 = 6922

	ETH_CHAIN_ID_LOCALNET_0 int64 = 6930
	ETH_CHAIN_ID_LOCALNET_1 int64 = 6931
	ETH_CHAIN_ID_LOCALNET_2 int64 = 6932
	ETH_CHAIN_ID_LOCALNET_3 int64 = 6933

	ETH_CHAIN_ID_DEFAULT int64 = 6930
)

// knownEthChainIDMap maps `sdk.Context` chain IDs to their corresponding EIP-155
// Ethereum Chain IDs, which must be positive integers.
var knownEthChainIDMap = map[string]int64{
	"cataclysm-1": ETH_CHAIN_ID_MAINNET,

	"nibiru-testnet-1": ETH_CHAIN_ID_TESTNET_1,
	"nibiru-testnet-2": ETH_CHAIN_ID_TESTNET_2,
	"nibiru-testnet-3": ETH_CHAIN_ID_TESTNET_3,

	"nibiru-devnet-1": ETH_CHAIN_ID_DEVNET_1,
	"nibiru-devnet-2": ETH_CHAIN_ID_DEVNET_2,
	"nibiru-devnet-3": ETH_CHAIN_ID_DEVNET_3,

	"nibiru-localnet-0": ETH_CHAIN_ID_LOCALNET_0,
	"nibiru-localnet-1": ETH_CHAIN_ID_LOCALNET_1,
	"nibiru-localnet-2": ETH_CHAIN_ID_LOCALNET_2,
	"nibiru-localnet-3": ETH_CHAIN_ID_LOCALNET_3,
}

// GetEthChainID: Maps the given chain ID from the block's `sdk.Context` to an
// EVM Chain ID (`*big.Int`).
func GetEthChainID(ctxChainID string) (ethChainID *big.Int) {
	ethChainIdInt, found := knownEthChainIDMap[ctxChainID]
	if !found {
		return big.NewInt(ETH_CHAIN_ID_DEFAULT)
	}
	return big.NewInt(ethChainIdInt)
}
