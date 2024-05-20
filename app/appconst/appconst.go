// Copyright (c) 2023-2024 Nibi, Inc.
package appconst

import (
	"fmt"
	"math/big"
	"runtime"
)

const (
	BinaryName = "nibiru"
	BondDenom  = "unibi"
	// AccountAddressPrefix: Bech32 prefix for Nibiru accounts.
	AccountAddressPrefix = "nibi"
)

// Runtime version vars
var (
	AppVersion = ""
	GitCommit  = ""
	BuildDate  = ""

	GoVersion = ""
	GoArch    = ""
)

// EVM Chain ID Map
var EVMChainIDs = map[string]*big.Int{
	"cataclysm-1":       big.NewInt(100),
	"nibiru-localnet-0": big.NewInt(1000),
	"nibiru-devnet-1":   big.NewInt(2000),
	"nibiru-devnet-2":   big.NewInt(3000),
	"nibiru-testnet-1":  big.NewInt(3000),
	"nibiru-testnet-2":  big.NewInt(4000),
	// other test chains will default to 10000
}
var DefaultEVMChainID = big.NewInt(10000)

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
